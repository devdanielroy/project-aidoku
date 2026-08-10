// Package chunk implements Stage B of the book processor: grouping
// Stage-A sentences into reading chunks via an LLM call that returns only
// grouping indices, never sentence text. See AIDOKU_DESIGN.md §3a/§7b.
//
// Dialogue-turn atomicity (keeping a whole same-speaker quote together in
// one chunk) is an open design question and is deliberately NOT enforced
// here yet — see AIDOKU_DESIGN.md §3a/§7. This package implements only the
// plain length-targeted grouping rules from §7b; a v0 punt, revisit once
// there's real chunked output from an actual book to judge against.
package chunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/types"
)

const (
	// TargetChunkChars and ChunkCharTolerance mirror the ~240±60 char soft
	// target from AIDOKU_DESIGN.md §3a.
	TargetChunkChars   = 240
	ChunkCharTolerance = 60

	// DefaultModel is used when Grouper.Model is empty. Chunk grouping is a
	// reasoning task run once per book, not per user, so a stronger model
	// is worth it — see AIDOKU_DESIGN.md §7a.
	DefaultModel = "claude-sonnet-5"

	// DefaultMaxTokens bounds the LLM's response size. A grouping response
	// is just nested integers, so this comfortably covers even a large
	// window's worth of chunks.
	DefaultMaxTokens = 4096

	// maxRetries is how many additional attempts are made against the LLM
	// after an invalid or failed first response, before falling back to
	// GreedyGroup. See AIDOKU_DESIGN.md §7b step 5 ("retry once, then fall
	// back").
	maxRetries = 1
)

// llmCaller is the subset of *anthropic.Client that Grouper needs. Defined
// on the consumer side so tests can supply a fake with no network
// dependency.
type llmCaller interface {
	CreateMessage(ctx context.Context, req anthropic.MessagesRequest) (*anthropic.MessagesResponse, error)
}

// Grouper runs Stage B: turning a window of sentences into a chunk
// grouping.
type Grouper struct {
	Client    llmCaller
	Model     string      // defaults to DefaultModel if empty
	MaxTokens int         // defaults to DefaultMaxTokens if zero
	Logger    *log.Logger // defaults to log.Default() if nil; records fallback events for later review
}

// NewGrouper returns a Grouper backed by client, using default
// model/token/logger settings.
func NewGrouper(client *anthropic.Client) *Grouper {
	return &Grouper{Client: client}
}

func (g *Grouper) model() string {
	if g.Model != "" {
		return g.Model
	}
	return DefaultModel
}

func (g *Grouper) maxTokens() int {
	if g.MaxTokens != 0 {
		return g.MaxTokens
	}
	return DefaultMaxTokens
}

func (g *Grouper) logger() *log.Logger {
	if g.Logger != nil {
		return g.Logger
	}
	return log.Default()
}

// GroupSentencesIntoChunks groups sentences — a window of a book's Stage-A
// output, in ascending contiguous Index order — into chunks. It calls the
// LLM, validates that the response is a clean partition, retries once on
// failure, and falls back to GreedyGroup if the LLM still hasn't produced a
// valid grouping. See AIDOKU_DESIGN.md §7b.
//
// GroupSentencesIntoChunks itself never returns an error for LLM
// failures — the fallback always succeeds — so a non-nil error here means
// something is wrong with the input itself, not the LLM call. Fallback
// events are logged (not returned as errors) so they can be reviewed.
func (g *Grouper) GroupSentencesIntoChunks(ctx context.Context, sentences []types.SentenceInput) (types.ChunkGroupingResponse, error) {
	if len(sentences) == 0 {
		return types.ChunkGroupingResponse{}, nil
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := g.callLLM(ctx, sentences)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}
		if err := ValidatePartition(sentences, resp); err != nil {
			lastErr = fmt.Errorf("attempt %d: invalid partition: %w", attempt+1, err)
			continue
		}
		return resp, nil
	}

	g.logger().Printf("chunk: falling back to greedy grouping for %d sentence(s) (window starting at index %d) after LLM grouping failed: %v",
		len(sentences), sentences[0].Index, lastErr)
	return GreedyGroup(sentences), nil
}

func (g *Grouper) callLLM(ctx context.Context, sentences []types.SentenceInput) (types.ChunkGroupingResponse, error) {
	payload, err := json.Marshal(sentences)
	if err != nil {
		return types.ChunkGroupingResponse{}, fmt.Errorf("marshal sentences: %w", err)
	}

	resp, err := g.Client.CreateMessage(ctx, anthropic.MessagesRequest{
		Model:     g.model(),
		MaxTokens: g.maxTokens(),
		System:    systemPrompt,
		Messages:  []anthropic.Message{{Role: "user", Content: string(payload)}},
	})
	if err != nil {
		return types.ChunkGroupingResponse{}, fmt.Errorf("call anthropic: %w", err)
	}

	text, err := resp.Text()
	if err != nil {
		return types.ChunkGroupingResponse{}, err
	}

	var parsed types.ChunkGroupingResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return types.ChunkGroupingResponse{}, fmt.Errorf("unmarshal LLM response as JSON: %w (raw response: %s)", err, text)
	}
	return parsed, nil
}

// systemPrompt fixes the target chunk length, the sentence-boundary-only
// rule, and the exact JSON output schema — see AIDOKU_DESIGN.md §7b.
var systemPrompt = fmt.Sprintf(`You are a stage in an offline book-processing pipeline for a language-learning app. You will receive a JSON array of sentences from a window of a book. Each sentence has an "index" (its position in the book), its exact "text", and its "char_count".

Your only job is to group these sentences into chunks, in order, targeting approximately %d characters per chunk. This is a soft target: a sentence is never split, so if a single sentence is longer than the target it becomes its own chunk; otherwise aim to keep chunks within roughly ±%d characters of the target.

Rules:
- Every sentence index must appear in exactly one chunk.
- Each chunk's sentence_indices must be a contiguous, ascending run (e.g. [4,5,6], never [4,6] or [6,5,4]).
- Chunks must be listed in ascending order and cover the input with no gaps and no overlaps — chunk 0 starts at the first sentence index, and each following chunk starts immediately after the previous chunk ends.
- The input sentences are already correctly segmented, including dialogue: do not try to re-split or merge sentence text. You are only choosing where between sentences to cut chunks.
- When a choice between two similarly-sized breaks is otherwise close, prefer the one that falls at a paragraph or topic boundary — but the character-count target takes priority over this preference.

Output ONLY a single JSON object with this exact shape, and nothing else — no prose, no explanation, no markdown code fences:
{"chunks": [{"chunk_index": 0, "sentence_indices": [0, 1, 2]}, {"chunk_index": 1, "sentence_indices": [3]}]}`, TargetChunkChars, ChunkCharTolerance)

// ValidatePartition confirms resp is a valid, complete, ordered partition
// of sentences: every sentence index appears exactly once, indices within
// each chunk are contiguous and ascending, and chunks themselves are
// ascending and gapless. See AIDOKU_DESIGN.md §3a/§7b.
func ValidatePartition(sentences []types.SentenceInput, resp types.ChunkGroupingResponse) error {
	n := len(sentences)
	if n == 0 {
		if len(resp.Chunks) != 0 {
			return fmt.Errorf("expected no chunks for an empty sentence list, got %d", len(resp.Chunks))
		}
		return nil
	}

	for i := 1; i < n; i++ {
		if sentences[i].Index != sentences[i-1].Index+1 {
			return fmt.Errorf("input sentences are not contiguous/ascending at position %d (index %d follows %d)", i, sentences[i].Index, sentences[i-1].Index)
		}
	}
	firstIndex := sentences[0].Index
	lastIndex := sentences[n-1].Index

	if len(resp.Chunks) == 0 {
		return errors.New("grouping response has no chunks")
	}

	want := firstIndex
	for ci, group := range resp.Chunks {
		if group.ChunkIndex != ci {
			return fmt.Errorf("chunk at position %d has chunk_index %d, want %d (chunks must be listed in ascending order starting at 0)", ci, group.ChunkIndex, ci)
		}
		if len(group.SentenceIndices) == 0 {
			return fmt.Errorf("chunk %d has no sentence indices", ci)
		}
		for pos, idx := range group.SentenceIndices {
			if idx != want {
				return fmt.Errorf("chunk %d sentence position %d has index %d, want %d (sentence_indices must be contiguous, ascending, and cover every sentence exactly once with no gaps or overlaps)", ci, pos, idx, want)
			}
			want++
		}
	}
	if want != lastIndex+1 {
		return fmt.Errorf("grouping covers indices [%d,%d), want [%d,%d)", firstIndex, want, firstIndex, lastIndex+1)
	}
	return nil
}

// GreedyGroup is the deterministic rule-based fallback grouper: accumulate
// sentences into the current chunk until adding the next one would exceed
// TargetChunkChars+ChunkCharTolerance, then cut. A single sentence longer
// than the limit still becomes its own chunk, since it's never rejected
// from an empty chunk. See AIDOKU_DESIGN.md §3a.
func GreedyGroup(sentences []types.SentenceInput) types.ChunkGroupingResponse {
	if len(sentences) == 0 {
		return types.ChunkGroupingResponse{}
	}

	var chunks []types.ChunkGroup
	var current []int
	currentChars := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		chunks = append(chunks, types.ChunkGroup{ChunkIndex: len(chunks), SentenceIndices: current})
		current = nil
		currentChars = 0
	}

	limit := TargetChunkChars + ChunkCharTolerance
	for _, s := range sentences {
		if len(current) > 0 && currentChars+s.CharCount > limit {
			flush()
		}
		current = append(current, s.Index)
		currentChars += s.CharCount
	}
	flush()

	return types.ChunkGroupingResponse{Chunks: chunks}
}
