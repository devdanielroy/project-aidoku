// Package breakdown implements the third Claude API invocation step
// (§3 stage 4.3): a full Japanese explanation of a chunk — sentence
// structure, vocabulary, grammar, and meaning — shown to the learner
// after they've answered its three questions. See AIDOKU_DESIGN.md §2
// step 5 / §4.
//
// Deliberately simpler than question generation (§7c): a chunk gets one
// breakdown, a single free-form Japanese text blob (matching
// db/schema.sql's breakdown.content and the Flutter app's
// app/lib/models/breakdown.dart — Breakdown{id, content}), not a fixed
// set of typed fields to unmarshal and validate. There's no JSON
// envelope on the wire either: the model is asked to output the
// Japanese text directly.
//
// Same no-rule-based-fallback stance as question generation, and for
// the same reason: writing a good explanation isn't mechanical, so
// there's nothing sensible to fall back to. A chunk that fails
// generation after retrying is an error to surface for manual
// regeneration, not a chunk published with a worse substitute — see
// AIDOKU_DESIGN.md §7a's "nothing reaches a paying user unreviewed".
package breakdown

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/types"
)

const (
	// DefaultModel is used when Generator.Model is empty. Same reasoning
	// as chunk grouping and question generation: run once per chunk, not
	// per user, so a stronger model is worth it — see AIDOKU_DESIGN.md
	// §7a.
	DefaultModel = "claude-sonnet-5"

	// DefaultMaxTokens bounds the LLM's response size. A full breakdown
	// (multiple Japanese sections covering structure/vocab/grammar/
	// meaning) runs longer than the three short questions in §7c, so
	// this is double that stage's default.
	DefaultMaxTokens = 4096

	// maxRetries matches question generation (§7c) — one more than
	// Stage B's retry budget (§7b), since there's no fallback to land on
	// here either.
	maxRetries = 2
)

// llmCaller is the subset of *anthropic.Client that Generator needs.
// Defined on the consumer side so tests can supply a fake with no
// network dependency — same pattern as chunk.llmCaller and
// question.llmCaller.
type llmCaller interface {
	CreateMessage(ctx context.Context, req anthropic.MessagesRequest) (*anthropic.MessagesResponse, error)
}

// Generator runs the breakdown-generation stage: turning one chunk into
// its full Japanese explanation.
type Generator struct {
	Client    llmCaller
	Model     string      // defaults to DefaultModel if empty
	MaxTokens int         // defaults to DefaultMaxTokens if zero
	Logger    *log.Logger // defaults to log.Default() if nil; records failed attempts for review
}

// NewGenerator returns a Generator backed by client, using default
// model/token/logger settings.
func NewGenerator(client *anthropic.Client) *Generator {
	return &Generator{Client: client}
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

// GenerateBreakdown produces chunk's full Japanese breakdown — the
// content that goes straight into db.SaveBreakdown / the breakdown
// table's content column. See the package doc for why a failure here is
// a returned error, not a silent fallback.
func (g *Generator) GenerateBreakdown(ctx context.Context, chunk types.Chunk) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, err := g.callLLM(ctx, chunk)
		if err == nil {
			if err = validateBreakdown(content); err == nil {
				return content, nil
			}
		}
		lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
		g.logger().Printf("breakdown: chunk %d: %v", chunk.Index, lastErr)
	}
	return "", fmt.Errorf("breakdown: failed to generate a valid breakdown for chunk %d after %d attempt(s): %w", chunk.Index, maxRetries+1, lastErr)
}

func (g *Generator) callLLM(ctx context.Context, chunk types.Chunk) (string, error) {
	payload, err := json.Marshal(chunk)
	if err != nil {
		return "", fmt.Errorf("marshal chunk: %w", err)
	}

	resp, err := g.Client.CreateMessage(ctx, anthropic.MessagesRequest{
		Model:     g.model(),
		MaxTokens: g.maxTokens(),
		System:    systemPrompt,
		Messages:  []anthropic.Message{{Role: "user", Content: string(payload)}},
	})
	if err != nil {
		return "", fmt.Errorf("call anthropic: %w", err)
	}

	text, err := resp.Text()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// systemPrompt fixes the breakdown's content and house style — matching
// the format already validated in the Flutter mock content
// (app/assets/mock/pride_and_prejudice.json) — and the plain-text (not
// JSON) output contract.
var systemPrompt = `You are a stage in an offline book-processing pipeline for a language-learning app. The learner is a Japanese speaker (L1) learning English (L2) through literature. You will receive a single JSON object describing one reading "chunk": {"index": <int>, "text": <string>, "char_count": <int>}. The learner has already read this chunk unassisted and answered three questions about it (vocab, grammar, comprehension). Your job is to write the full breakdown they see next: a thorough explanation of the passage, in Japanese.

Cover, as relevant to this specific chunk:
- Sentence structure: how the sentence(s) are built — clauses, constructions, word order — quoting the exact English span you're explaining.
- Vocabulary: notable or high-value words/phrases worth knowing, with their meaning as used here.
- Grammar: patterns or constructions worth explaining (tense, modals, reported speech, inversion, conditionals, etc.).
- Meaning/interpretation: what the passage actually conveys — including tone, irony, or subtext where present, not just a literal paraphrase.
- Cultural or stylistic notes, only when genuinely relevant (an idiom, a period-specific reference, an authorial technique) — don't manufacture one if there's nothing to say.

Not every chunk needs every section. A short, grammatically simple chunk might only need a brief vocabulary and meaning note — match the depth of explanation to what the passage actually contains, don't pad it out.

Format, matching this app's established house style:
- Section headers in Japanese, wrapped in 【】 — use exactly these labels for these sections when you include them: 【文構造】 for sentence structure, 【語彙】 for vocabulary, 【文法】 for grammar, 【意味】 for meaning. Use a different 【...】 label only for a cultural/stylistic note that doesn't fit those four.
- Separate sections with a blank line.
- Quote exact English spans from the chunk's text in double quotes when referring to them.
- List multiple vocabulary items with a "・" bullet per item.
- Write entirely in natural, explanatory Japanese — no English prose outside of quoted spans copied from the passage itself.

Illustrative example only — showing format, not content to reuse:
【文構造】"It is a truth universally acknowledged, that ..." は "It is + 過去分詞 + that節" という形式主語構文で、「〜ということは広く認められた真実である」という意味です。

【語彙】
・acknowledged「認められている」
・in want of a wife「妻を欲している」

【文法】"must" は義務ではなく論理的な推測(「〜にちがいない」)を表します。

【意味】表面上は一般論を装っていますが、実際には当時の結婚観への皮肉です。

Output ONLY the Japanese breakdown text itself, and nothing else — no JSON, no markdown code fences, no English preamble, no meta-commentary before or after it.`

// containsJapanese matches any Hiragana, Katakana, or CJK Unified
// Ideograph rune — a cheap sanity check that the response is actually
// written in Japanese, per the hard requirement in AIDOKU_DESIGN.md §2
// step 5.
var containsJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`)

// validateBreakdown confirms content is non-empty and genuinely contains
// Japanese text. It deliberately does not check for the specific 【...】
// section structure the system prompt asks for: which sections a given
// chunk actually needs varies (see the prompt), so a rigid structural
// check would reject legitimate, thinner breakdowns for simple chunks
// along with genuinely malformed ones — better to catch those in the QA
// pass (§3 stage 5) than to force every breakdown into the same shape
// here.
func validateBreakdown(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("empty breakdown")
	}
	if !containsJapanese.MatchString(content) {
		return errors.New("breakdown does not appear to contain any Japanese text")
	}
	return nil
}
