// Package types holds data shapes shared across pipeline stages, so each
// stage's package doesn't need to depend on another stage's package just to
// know what it produces or consumes.
//
// See AIDOKU_DESIGN.md §7b for the design these mirror.
package types

// SentenceInput is one sentence produced by Stage A (deterministic
// segmentation) and consumed by Stage B (LLM chunk grouping). Text is the
// exact original source text for the sentence — untouched, never
// reconstructed or paraphrased.
type SentenceInput struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	CharCount int    `json:"char_count"`
}

// ChunkGroup is one chunk's worth of sentence indices, as decided by Stage
// B. SentenceIndices must be contiguous and ascending (e.g. [4,5,6]) — the
// LLM groups sentences into chunks, it never reorders or splits them.
type ChunkGroup struct {
	ChunkIndex      int   `json:"chunk_index"`
	SentenceIndices []int `json:"sentence_indices"`
}

// ChunkGroupingResponse is Stage B's full response: an ordered, complete,
// non-overlapping partition of every sentence in the input window into
// chunks.
type ChunkGroupingResponse struct {
	Chunks []ChunkGroup `json:"chunks"`
}
