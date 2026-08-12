// Package clean implements the second pipeline stage: stripping Project
// Gutenberg's license header/footer boilerplate from raw ingested text,
// and normalizing whitespace/encoding. See AIDOKU_DESIGN.md §3 step 2.
//
// Scope note: Clean itself only removes Gutenberg's own wrapper —
// everything before the "*** START OF ... PROJECT GUTENBERG EBOOK ***"
// line and everything from the matching "*** END OF ... ***" line
// onward. It does NOT attempt to strip front matter *within* the book
// itself (title pages, prefaces, tables of contents), which some
// Gutenberg editions include before the actual novel text begins — the
// Pride and Prejudice testdata fixture used in this package's tests is a
// real example, with roughly 30KB of illustrated-edition preface before
// "Chapter I." Stripping that generically isn't something Clean can do
// (see Trim, which handles it a different way: human-supplied anchors
// rather than detection). Illustration placeholders are handled
// differently again — not stripped, but condensed; see
// condenseIllustrations.
package clean

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	startMarker      = regexp.MustCompile(`(?i)\*\*\*\s*START OF (?:THE|THIS) PROJECT GUTENBERG EBOOK.*?\*\*\*`)
	endMarker        = regexp.MustCompile(`(?i)\*\*\*\s*END OF (?:THE|THIS) PROJECT GUTENBERG EBOOK.*?\*\*\*`)
	paragraphBreakRe = regexp.MustCompile(`\n{2,}`)
)

// utf8BOM is U+FEFF, sometimes present as the first character of a text
// file to mark its byte order / encoding. Written as an escape rather
// than a literal character — Go only permits a literal BOM as the very
// first byte of a source file.
const utf8BOM = "\uFEFF"

// Clean strips Project Gutenberg's license header and footer from raw,
// and normalizes line endings and excess blank lines. Returns an error if
// the standard Gutenberg START/END markers aren't found, rather than
// silently passing through unrecognized boilerplate — or worse, actual
// book content mistaken for boilerplate — into the rest of the pipeline;
// an unexpected format should surface for a human to look at.
func Clean(raw string) (string, error) {
	normalized := normalizeLineEndings(raw)

	startLoc := startMarker.FindStringIndex(normalized)
	if startLoc == nil {
		return "", fmt.Errorf("clean: no Project Gutenberg START marker found")
	}
	endLoc := endMarker.FindStringIndex(normalized)
	if endLoc == nil {
		return "", fmt.Errorf("clean: no Project Gutenberg END marker found")
	}
	if endLoc[0] < startLoc[1] {
		return "", fmt.Errorf("clean: END marker appears before START marker")
	}

	body := normalized[startLoc[1]:endLoc[0]]
	body = strings.TrimSpace(body)
	body = removeBareIllustrations(body)
	body, err := condenseIllustrations(body)
	if err != nil {
		return "", fmt.Errorf("clean: %w", err)
	}
	body = dewrapParagraphs(body)
	return body, nil
}

func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimPrefix(s, utf8BOM)
}

// dewrapParagraphs undoes Project Gutenberg's fixed-width line wrapping.
// Plain-text Gutenberg files hard-wrap prose to ~70 columns using single
// newlines within a paragraph (not real line breaks — an artifact of the
// .txt format) and a blank line between paragraphs. Left alone, that
// wrapping ends up baked into individual sentences verbatim (e.g. "That
// is an\nuncommon advantage..."), which is faithful to the raw file but
// not to how the sentence actually reads.
//
// This joins each paragraph's wrapped lines back into one flowing line —
// trimming each line's leading/trailing whitespace first, so indentation
// (e.g. centered title-page text) doesn't leave a gap mid-sentence — and
// normalizes the separator between paragraphs to exactly one blank line.
func dewrapParagraphs(body string) string {
	var paragraphs []string
	for _, p := range paragraphBreakRe.Split(body, -1) {
		var lines []string
		for line := range strings.SplitSeq(p, "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
		if len(lines) == 0 {
			continue // a paragraph that was pure whitespace/artifacts
		}
		paragraphs = append(paragraphs, strings.Join(lines, " "))
	}
	return strings.Join(paragraphs, "\n\n")
}

// Trim narrows text — Clean's output — down to the span between
// firstLine and lastLine, inclusive, discarding everything outside it:
// front matter (prefaces, illustration lists, title pages) before
// firstLine, and back matter (colophons, "THE END" notices, appendices)
// after lastLine. Both anchors are book-specific and human-supplied (see
// pipeline/books.txt) rather than detected — there's no reliable generic
// way to recognize "front/back matter" across arbitrarily different
// editions, as Clean's own package doc explains.
//
// firstLine and lastLine must each match exactly once in text. Zero
// matches means the anchor is wrong (a typo, the wrong edition, or text
// that wasn't put through the same Clean normalization the anchor was
// chosen against); more than one match means the anchor isn't specific
// enough to safely use as a unique cut point. Either way, that's an
// error to surface for a human to fix the catalog entry, not something
// to guess through.
func Trim(text, firstLine, lastLine string) (string, error) {
	if n := strings.Count(text, firstLine); n != 1 {
		return "", fmt.Errorf("clean: first-line anchor matches %d time(s) in text, want exactly 1: %q", n, firstLine)
	}
	if n := strings.Count(text, lastLine); n != 1 {
		return "", fmt.Errorf("clean: last-line anchor matches %d time(s) in text, want exactly 1: %q", n, lastLine)
	}

	start := strings.Index(text, firstLine)
	end := strings.Index(text, lastLine) + len(lastLine)
	if end <= start {
		return "", fmt.Errorf("clean: last-line anchor appears before first-line anchor")
	}

	return text[start:end], nil
}
