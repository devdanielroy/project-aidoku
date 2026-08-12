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

// Chunk is a published reading chunk: a ChunkGroup's sentences
// reconstructed back into text (see chunk.BuildChunks). Mirrors the Chunk
// entity in AIDOKU_DESIGN.md §4, minus storage-assigned fields (id,
// book_id) that don't exist yet at this stage of the pipeline.
type Chunk struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	CharCount int    `json:"char_count"`
}

// QuestionType is one of the three question kinds every chunk is tested
// with. See AIDOKU_DESIGN.md §2 step 4 / §4.
type QuestionType string

const (
	QuestionTypeVocab         QuestionType = "vocab"
	QuestionTypeGrammar       QuestionType = "grammar"
	QuestionTypeComprehension QuestionType = "comprehension"
)

// Question is one multiple-choice question tied to a chunk. Mirrors the
// Question entity in AIDOKU_DESIGN.md §4, and the shape the Flutter
// client's mock data already uses (see app/assets/mock/*.json) —
// options/answer take a fixed multiple-choice shape (Options +
// AnswerIndex), minus storage-assigned fields (id, chunk_id).
type Question struct {
	Type        QuestionType `json:"type"`
	Prompt      string       `json:"prompt"`
	Options     []string     `json:"options"`
	AnswerIndex int          `json:"answer_index"`
	Explanation string       `json:"explanation"`
	// Highlight is the exact substring of the chunk's text this question
	// is about — underlined in the passage in place of the prompt
	// re-quoting it. Empty for comprehension questions, which are about
	// the whole chunk rather than one word or phrase.
	Highlight string `json:"highlight,omitempty"`
}
