// Package question implements the question-generation pipeline stage: one
// vocab, one grammar, and one comprehension question per chunk — one of
// §3's Claude API Invocation Steps ("Generate questions") / §4. It
// follows the same general pattern as
// Stage B chunk grouping (§7b) — structured JSON request/response,
// strict validation, retry on failure — with one deliberate difference:
// there is no rule-based fallback here.
//
// Chunk grouping (Stage B) can fall back to a deterministic greedy
// grouper because it's a mechanical, structural task — a rule-based
// approximation is genuinely adequate. Writing a good vocab/grammar/
// comprehension question is not mechanical; there's no reasonable
// non-LLM fallback for it. So GenerateQuestions retries a few times and,
// if it still can't produce a valid set, returns an error instead of
// publishing something low-quality — consistent with AIDOKU_DESIGN.md
// §7a's "nothing reaches a paying user unreviewed": a chunk that fails
// generation should be flagged for manual regeneration/QA, not silently
// patched over.
package question

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/types"
)

const (
	// DefaultModel is used when Generator.Model is empty. Question writing
	// is a reasoning/creative task run once per chunk, not per user, so a
	// stronger model is worth it — see AIDOKU_DESIGN.md §7a.
	DefaultModel = "claude-sonnet-5"

	// DefaultMaxTokens bounds the LLM's response size. Three short
	// multiple-choice questions with brief explanations comfortably fit.
	DefaultMaxTokens = 2048

	// optionsPerQuestion is the fixed multiple-choice width used
	// throughout the app (see app/lib/models/question.dart and the mock
	// content it was validated against).
	optionsPerQuestion = 4

	// maxRetries is how many additional attempts are made against the LLM
	// after an invalid or failed first response before GenerateQuestions
	// gives up and returns an error. One more than Stage B's retry budget
	// (§7b) since there's no fallback to land on here — see package doc.
	maxRetries = 2
)

// llmCaller is the subset of *anthropic.Client that Generator needs.
// Defined on the consumer side so tests can supply a fake with no network
// dependency.
type llmCaller interface {
	CreateMessage(ctx context.Context, req anthropic.MessagesRequest) (*anthropic.MessagesResponse, error)
}

// Generator runs the question-generation stage: turning one chunk into
// its three questions.
type Generator struct {
	Client    llmCaller
	Model     string      // defaults to DefaultModel if empty
	MaxTokens int         // defaults to DefaultMaxTokens if zero
	Logger    *log.Logger // defaults to log.Default() if nil; records failed attempts for review

	// LanguagePair is required, not defaulted — GenerateQuestions
	// returns an error immediately if it's left unset, rather than
	// silently assuming a pair. See langpair's package doc for why
	// there's no default.
	LanguagePair langpair.LanguagePair
}

// NewGenerator returns a Generator backed by client and pair, using
// default model/token/logger settings.
func NewGenerator(client *anthropic.Client, pair langpair.LanguagePair) *Generator {
	return &Generator{Client: client, LanguagePair: pair}
}

func (g *Generator) model() string {
	if g.Model != "" {
		return g.Model
	}
	return DefaultModel
}

func (g *Generator) maxTokens() int {
	if g.MaxTokens != 0 {
		return g.MaxTokens
	}
	return DefaultMaxTokens
}

func (g *Generator) logger() *log.Logger {
	if g.Logger != nil {
		return g.Logger
	}
	return log.Default()
}

// GenerateQuestions produces exactly one vocab, one grammar, and one
// comprehension question for chunk, returned in that canonical order
// regardless of what order the LLM listed them in. See the package doc
// for why a failure here is a returned error, not a silent fallback —
// including when LanguagePair itself was never configured.
func (g *Generator) GenerateQuestions(ctx context.Context, chunk types.Chunk) ([]types.Question, error) {
	if g.LanguagePair.Target == "" || g.LanguagePair.Native == "" {
		return nil, fmt.Errorf("question: Generator.LanguagePair is not set (see langpair.ByCode)")
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		set, err := g.callLLM(ctx, chunk)
		if err == nil {
			var questions []types.Question
			questions, err = ValidateQuestionSet(chunk, set)
			if err == nil {
				return questions, nil
			}
		}
		lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
		g.logger().Printf("question: chunk %d: %v", chunk.Index, lastErr)
	}
	return nil, fmt.Errorf("question: failed to generate a valid question set for chunk %d after %d attempt(s): %w", chunk.Index, maxRetries+1, lastErr)
}

func (g *Generator) callLLM(ctx context.Context, chunk types.Chunk) (rawQuestionSet, error) {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return rawQuestionSet{}, fmt.Errorf("marshal chunk: %w", err)
	}

	resp, err := g.Client.CreateMessage(ctx, anthropic.MessagesRequest{
		Model:     g.model(),
		MaxTokens: g.maxTokens(),
		System:    buildSystemPrompt(g.LanguagePair),
		Messages:  []anthropic.Message{{Role: "user", Content: string(payload)}},
	})
	if err != nil {
		return rawQuestionSet{}, fmt.Errorf("call anthropic: %w", err)
	}

	text, err := resp.Text()
	if err != nil {
		return rawQuestionSet{}, err
	}

	var parsed rawQuestionSet
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return rawQuestionSet{}, fmt.Errorf("unmarshal LLM response as JSON: %w (raw response: %s)", err, text)
	}
	return parsed, nil
}

// buildSystemPrompt fixes the three question types' rules, the
// highlight contract, and the exact JSON output schema — parameterized
// by pair for which language the learner is studying (Target) vs.
// their own (Native), the language everything the reader sees
// (prompts/options/explanations) is written in. See AIDOKU_DESIGN.md §2
// step 5 and §7's "i18n architecture" open question.
func buildSystemPrompt(pair langpair.LanguagePair) string {
	target, native := langpair.DisplayName(pair.Target), langpair.DisplayName(pair.Native)
	return fmt.Sprintf(`You are a stage in an offline book-processing pipeline for a language-learning app. The learner is a %[2]s speaker (L1) learning %[1]s (L2) through literature. You will receive a single JSON object describing one reading "chunk": {"index": <int>, "text": <string>, "char_count": <int>}.

Your job: write exactly three multiple-choice questions testing this chunk — one "vocab", one "grammar", one "comprehension" — each testing something different, so they don't overlap with each other.

For "vocab":
- Pick one notable or high-value word (or short fixed phrase, e.g. a phrasal verb) from the chunk that's worth testing.
- "highlight" must be that exact word/phrase, copied byte-for-byte from the chunk's "text" (same case, same punctuation) — it must appear verbatim as a substring of it.
- "prompt" asks what the underlined word/phrase means, in %[2]s, without repeating the surrounding sentence — the reader will see it underlined directly in the passage, so don't re-quote it.
- "explanation" is 1-3 %[2]s sentences explaining the word's meaning as used here.

For "grammar":
- Identify one grammar pattern or construction actually present in the chunk (e.g. a modal, a tense, reported speech, a conditional, an inversion, a relative clause).
- "highlight" must be the exact span of text demonstrating that pattern, copied byte-for-byte from the chunk's "text".
- "prompt" asks what the underlined grammar expresses, in %[2]s, without re-quoting the passage.
- "explanation" is 1-3 %[2]s sentences explaining the grammar point.

For "comprehension":
- Test whether the reader understood what actually happened or was said in the chunk — not one specific word or span.
- Do not include "highlight" at all for this question (omit the field entirely) — this question is about the whole chunk, not one word or phrase.
- "prompt" asks what the chunk conveys or what happened in it, in %[2]s.
- "explanation" is 1-3 %[2]s sentences confirming what happened.

Rules that apply to all three:
- "options" is always exactly %[3]d entries, in %[2]s, all distinct and all plausible — no throwaway wrong answers. Always list the correct option first, as options[0] ("answer_index": 0) — the client app randomizes the displayed order before showing it to the reader, so you don't need to vary or think about position.
- Do not restate large portions of the passage inside any prompt — the passage itself is always visible to the reader alongside the question.
- "highlight" (vocab/grammar) is always in %[1]s, copied verbatim from the chunk's own text, never translated into %[2]s — these instructions being written in %[2]s doesn't mean the %[1]s text you're quoting from should be.

Output ONLY a single JSON object with this exact shape, and nothing else — no prose, no explanation, no markdown code fences:
{"questions": [
  {"type": "vocab", "prompt": "...", "options": ["...", "...", "...", "..."], "answer_index": 0, "explanation": "...", "highlight": "..."},
  {"type": "grammar", "prompt": "...", "options": ["...", "...", "...", "..."], "answer_index": 0, "explanation": "...", "highlight": "..."},
  {"type": "comprehension", "prompt": "...", "options": ["...", "...", "...", "..."], "answer_index": 0, "explanation": "..."}
]}`, target, native, optionsPerQuestion)
}

// rawQuestion is the unvalidated shape read directly off the wire. Type
// is a plain string here (not types.QuestionType) because an
// unrecognized value must be caught by validation, not silently zeroed
// out or rejected by json.Unmarshal in a way that's hard to diagnose.
type rawQuestion struct {
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	AnswerIndex int      `json:"answer_index"`
	Explanation string   `json:"explanation"`
	Highlight   string   `json:"highlight,omitempty"`
}

type rawQuestionSet struct {
	Questions []rawQuestion `json:"questions"`
}

// ValidateQuestionSet confirms set contains exactly one well-formed
// question of each type, and converts it to the canonical
// []types.Question form (ordered vocab, grammar, comprehension). See
// AIDOKU_DESIGN.md §4 and the package doc.
func ValidateQuestionSet(chunk types.Chunk, set rawQuestionSet) ([]types.Question, error) {
	if len(set.Questions) != 3 {
		return nil, fmt.Errorf("expected 3 questions, got %d", len(set.Questions))
	}

	byType := make(map[types.QuestionType]rawQuestion, 3)
	for i, q := range set.Questions {
		t := types.QuestionType(q.Type)
		switch t {
		case types.QuestionTypeVocab, types.QuestionTypeGrammar, types.QuestionTypeComprehension:
		default:
			return nil, fmt.Errorf("question %d: unknown type %q", i, q.Type)
		}
		if _, dup := byType[t]; dup {
			return nil, fmt.Errorf("question %d: duplicate type %q", i, q.Type)
		}
		byType[t] = q
	}

	order := []types.QuestionType{types.QuestionTypeVocab, types.QuestionTypeGrammar, types.QuestionTypeComprehension}
	questions := make([]types.Question, 0, 3)
	for _, t := range order {
		raw, ok := byType[t]
		if !ok {
			return nil, fmt.Errorf("missing a %q question", t)
		}
		q, err := validateOne(chunk, t, raw)
		if err != nil {
			return nil, fmt.Errorf("%q question: %w", t, err)
		}
		questions = append(questions, q)
	}
	return questions, nil
}

func validateOne(chunk types.Chunk, t types.QuestionType, raw rawQuestion) (types.Question, error) {
	if strings.TrimSpace(raw.Prompt) == "" {
		return types.Question{}, fmt.Errorf("empty prompt")
	}
	if len(raw.Options) != optionsPerQuestion {
		return types.Question{}, fmt.Errorf("expected %d options, got %d", optionsPerQuestion, len(raw.Options))
	}
	seen := make(map[string]bool, len(raw.Options))
	for i, opt := range raw.Options {
		trimmed := strings.TrimSpace(opt)
		if trimmed == "" {
			return types.Question{}, fmt.Errorf("option %d is empty", i)
		}
		if seen[trimmed] {
			return types.Question{}, fmt.Errorf("option %d duplicates another option: %q", i, trimmed)
		}
		seen[trimmed] = true
	}
	// systemPrompt asks the model to always put the correct answer at
	// options[0], so it only has to write distractors — not also track
	// where it hid the right one — and the client is expected to
	// randomize display order before showing options to the user. We
	// still accept any in-range index here rather than enforcing 0: the
	// contract that actually matters is "answer_index correctly points
	// to the correct option," not its position.
	if raw.AnswerIndex < 0 || raw.AnswerIndex >= len(raw.Options) {
		return types.Question{}, fmt.Errorf("answer_index %d out of range [0,%d)", raw.AnswerIndex, len(raw.Options))
	}
	if strings.TrimSpace(raw.Explanation) == "" {
		return types.Question{}, fmt.Errorf("empty explanation")
	}

	switch t {
	case types.QuestionTypeVocab, types.QuestionTypeGrammar:
		if raw.Highlight == "" {
			return types.Question{}, fmt.Errorf("missing highlight")
		}
		if !strings.Contains(chunk.Text, raw.Highlight) {
			return types.Question{}, fmt.Errorf("highlight %q does not appear verbatim in the chunk text", raw.Highlight)
		}
	case types.QuestionTypeComprehension:
		if raw.Highlight != "" {
			return types.Question{}, fmt.Errorf("comprehension questions must not have a highlight, got %q", raw.Highlight)
		}
	}

	return types.Question{
		Type:        t,
		Prompt:      raw.Prompt,
		Options:     raw.Options,
		AnswerIndex: raw.AnswerIndex,
		Explanation: raw.Explanation,
		Highlight:   raw.Highlight,
	}, nil
}
