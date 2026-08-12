package chunk

import (
	"testing"

	"aidoku/pipeline/internal/segment"
	"aidoku/pipeline/internal/types"
)

func TestBuildChunks(t *testing.T) {
	sents := sentences(5, 0, 10) // "Sentence 0." .. "Sentence 4.", 10 chars each per the fixture

	resp := types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
		{ChunkIndex: 0, SentenceIndices: []int{0, 1}},
		{ChunkIndex: 1, SentenceIndices: []int{2}},
		{ChunkIndex: 2, SentenceIndices: []int{3, 4}},
	}}

	got := BuildChunks(sents, resp)

	want := []types.Chunk{
		{Index: 0, Text: "Sentence 0. Sentence 1.", CharCount: 23},
		{Index: 1, Text: "Sentence 2.", CharCount: 11},
		{Index: 2, Text: "Sentence 3. Sentence 4.", CharCount: 23},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d chunks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chunk %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestBuildChunks_Empty(t *testing.T) {
	got := BuildChunks(nil, types.ChunkGroupingResponse{})
	if len(got) != 0 {
		t.Fatalf("got %d chunks, want 0", len(got))
	}
}

// TestBuildChunks_RealText verifies chunk reconstruction end to end
// against Segment's real output (not the synthetic `sentences` fixture),
// confirming the design doc's claim that reconstructed chunk text is
// byte-identical to the corresponding slice of the source text.
func TestBuildChunks_RealText(t *testing.T) {
	text := `It is a truth universally acknowledged, that a single man in possession of a good fortune, must be in want of a wife. However little known the feelings or views of such a man may be on his first entering a neighbourhood, this truth is so well fixed in the minds of the surrounding families, that he is considered the rightful property of some one or other of their daughters.`

	sents := segment.Segment(text)
	resp := GreedyGroup(sents)
	chunks := BuildChunks(sents, resp)

	reconstructed := ""
	for i, c := range chunks {
		if i > 0 {
			reconstructed += " "
		}
		reconstructed += c.Text
	}

	if reconstructed != text {
		t.Fatalf("reconstructed text does not match source:\ngot:  %q\nwant: %q", reconstructed, text)
	}

	for _, c := range chunks {
		if c.CharCount != len([]rune(c.Text)) {
			t.Errorf("chunk %d: CharCount = %d, want %d", c.Index, c.CharCount, len([]rune(c.Text)))
		}
	}
}
