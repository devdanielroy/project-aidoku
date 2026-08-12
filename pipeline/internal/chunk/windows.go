package chunk

import "aidoku/pipeline/internal/types"

// WindowTargetChars is the soft target window size for a single Stage B
// call, per AIDOKU_DESIGN.md §3a's "book-length handling: process in
// overlapping windows (e.g. ~3,000-character windows... ) to fit context
// limits". "Overlapping" is aspirational, not yet true here — see
// SplitIntoWindows.
const WindowTargetChars = 3000

// SplitIntoWindows groups sentences into windows of up to targetChars
// each, for separate Stage B calls against a whole book (GroupSentences-
// IntoChunks itself only groups whatever window it's given — something
// upstream has to do the windowing). A sentence never spans two windows,
// same hard rule as chunk grouping itself; mechanically identical to
// GreedyGroup, just producing windows of sentences instead of chunks of
// sentence indices.
//
// This is the plain, non-overlapping version. AIDOKU_DESIGN.md §3a
// describes windows overlapping at the edges, with boundary decisions
// reconciled using the pass with fuller context — that refinement isn't
// designed yet, let alone implemented, so a chunk that would ideally
// span a window boundary instead gets cut there by whichever window it
// falls into. Acceptable for this pipeline's actual v0 target (a short
// story — see pipeline/books.txt): few, if any, windows in practice, so
// few or no boundaries to get wrong in the first place. Revisit once a
// longer book makes this matter.
func SplitIntoWindows(sentences []types.SentenceInput, targetChars int) [][]types.SentenceInput {
	if len(sentences) == 0 {
		return nil
	}

	var windows [][]types.SentenceInput
	var current []types.SentenceInput
	currentChars := 0

	flush := func() {
		if len(current) == 0 {
			return
		}
		windows = append(windows, current)
		current = nil
		currentChars = 0
	}

	for _, s := range sentences {
		if len(current) > 0 && currentChars+s.CharCount > targetChars {
			flush()
		}
		current = append(current, s)
		currentChars += s.CharCount
	}
	flush()

	return windows
}
