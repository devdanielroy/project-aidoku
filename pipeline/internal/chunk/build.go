package chunk

import (
	"strings"

	"aidoku/pipeline/internal/types"
)

// BuildChunks reconstructs each chunk's text by concatenating its
// sentences (joined by a single space) in the order given by resp.
// Guaranteed byte-identical to the source text, sentence-for-sentence,
// since Stage A's sentence text is never rewritten and Stage B only ever
// decides grouping — see AIDOKU_DESIGN.md §3a.
//
// resp is assumed to already be a validated partition of sentences (see
// ValidatePartition) — BuildChunks does not re-validate it, and will
// panic on a sentence index that isn't present in sentences.
func BuildChunks(sentences []types.SentenceInput, resp types.ChunkGroupingResponse) []types.Chunk {
	bySentenceIndex := make(map[int]types.SentenceInput, len(sentences))
	for _, s := range sentences {
		bySentenceIndex[s.Index] = s
	}

	chunks := make([]types.Chunk, len(resp.Chunks))
	for i, group := range resp.Chunks {
		var b strings.Builder
		for j, idx := range group.SentenceIndices {
			if j > 0 {
				b.WriteByte(' ')
			}
			sentence, ok := bySentenceIndex[idx]
			if !ok {
				panic("chunk: BuildChunks: sentence index not found in sentences — resp was not a validated partition of sentences")
			}
			b.WriteString(sentence.Text)
		}
		text := b.String()
		chunks[i] = types.Chunk{
			Index:     group.ChunkIndex,
			Text:      text,
			CharCount: len([]rune(text)),
		}
	}
	return chunks
}
