package clean

// CleanJapanese is Clean's JP_EN-pair sibling — see langpair.JP_EN and
// AIDOKU_DESIGN.md §7i. Everything about stripping Project Gutenberg's
// license wrapper and handling "[Illustration: ...]" placeholders is
// identical to Clean: those are Gutenberg's own transcription
// conventions, written in English regardless of the book's own
// language, so there's nothing to vary. The one real difference is
// dewrapParagraphs's rejoin behavior — Japanese has no spaces between
// words, so wrapped lines are rejoined with no separator at all,
// instead of Clean's " " — see dewrapParagraphs's own doc comment.
func CleanJapanese(raw string) (string, error) {
	return clean(raw, "")
}
