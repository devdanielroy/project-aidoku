package clean

// Clean strips Project Gutenberg's license header and footer from raw,
// and normalizes line endings and excess blank lines. Returns an error if
// the standard Gutenberg START/END markers aren't found, rather than
// silently passing through unrecognized boilerplate — or worse, actual
// book content mistaken for boilerplate — into the rest of the pipeline;
// an unexpected format should surface for a human to look at.
//
// This is the EN_JP pair's version — for source text where words are
// space-separated (English). See CleanJapanese (clean_japanese.go) for
// the JP_EN pair's sibling; the two share everything (see the shared
// clean helper) except how dewrapParagraphs rejoins a paragraph's
// hard-wrapped lines.
func Clean(raw string) (string, error) {
	return clean(raw, " ")
}
