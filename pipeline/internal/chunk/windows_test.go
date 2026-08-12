package chunk

import (
	"testing"

	"aidoku/pipeline/internal/types"
)

func TestSplitIntoWindows(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		if got := SplitIntoWindows(nil, 3000); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("everything fits in one window", func(t *testing.T) {
		sents := sentences(5, 0, 60) // running total 300, well under a 3000 target
		windows := SplitIntoWindows(sents, 3000)
		if len(windows) != 1 {
			t.Fatalf("got %d window(s), want 1", len(windows))
		}
		if len(windows[0]) != 5 {
			t.Errorf("window has %d sentence(s), want 5", len(windows[0]))
		}
	})

	t.Run("cuts once the target would be exceeded", func(t *testing.T) {
		sents := sentences(6, 0, 60) // 6th sentence would push the total to 360 > 300
		windows := SplitIntoWindows(sents, 300)
		if len(windows) != 2 {
			t.Fatalf("got %d window(s), want 2", len(windows))
		}
		if len(windows[0]) != 5 {
			t.Errorf("first window has %d sentence(s), want 5", len(windows[0]))
		}
		if len(windows[1]) != 1 {
			t.Errorf("second window has %d sentence(s), want 1", len(windows[1]))
		}
	})

	t.Run("a single oversized sentence still becomes its own window", func(t *testing.T) {
		sents := []types.SentenceInput{
			{Index: 0, Text: "short.", CharCount: 50},
			{Index: 1, Text: "long.", CharCount: 500}, // alone, way over the target
			{Index: 2, Text: "short.", CharCount: 50},
		}
		windows := SplitIntoWindows(sents, 300)
		if len(windows) != 3 {
			t.Fatalf("got %d window(s), want 3 (the oversized sentence forces a cut on both sides)", len(windows))
		}
	})

	t.Run("every sentence appears exactly once, in order, across all windows", func(t *testing.T) {
		sents := sentences(20, 0, 137) // an awkward size relative to the target, on purpose
		windows := SplitIntoWindows(sents, 1000)

		var gotIndices []int
		for _, w := range windows {
			for _, s := range w {
				gotIndices = append(gotIndices, s.Index)
			}
		}
		if len(gotIndices) != len(sents) {
			t.Fatalf("windows contain %d sentence(s) total, want %d", len(gotIndices), len(sents))
		}
		for i, idx := range gotIndices {
			if idx != i {
				t.Fatalf("sentence at flattened position %d has index %d, want %d (sentences must stay contiguous and in order)", i, idx, i)
			}
		}
	})

	t.Run("no window exceeds the target except for a single oversized sentence", func(t *testing.T) {
		sents := sentences(30, 0, 90)
		windows := SplitIntoWindows(sents, 1000)

		for i, w := range windows {
			total := 0
			for _, s := range w {
				total += s.CharCount
			}
			if total > 1000 && len(w) > 1 {
				t.Errorf("window %d has %d chars across %d sentences, want <=1000 unless it's a single oversized sentence", i, total, len(w))
			}
		}
	})
}
