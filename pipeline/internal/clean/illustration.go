package clean

import (
	"fmt"
	"strings"
)

// illustrationOpenTag is Project Gutenberg's convention for marking where
// an image appeared in the original printed book, since plain text can't
// show the image itself: "[Illustration: <description/caption>]",
// sometimes with a "[_Copyright ... _]" notice nested inside before the
// closing bracket. Real example from this package's testdata:
//
//	[Illustration:
//
//	“He came down to see the place”
//
//	[_Copyright 1894 by George Allen._]]
const illustrationOpenTag = "[Illustration:"

// condenseIllustrations finds each "[Illustration: ...]" block and
// collapses its internal whitespace down to a single line, e.g. the
// example above becomes:
//
//	[Illustration: “He came down to see the place” [_Copyright 1894 by George Allen._]]
//
// Nothing is stripped — the caption/copyright text is real content, kept
// in case a future feature displays the actual illustration (sourced
// separately) using this text to identify which one. Condensing it to a
// single line is what makes it possible to treat as one atomic unit
// downstream: Stage A (segment.Segment) has no terminal punctuation to
// find inside an uncondensed, multi-line "[Illustration: ...]" block, so
// left alone it silently glues onto whatever real sentence follows it —
// see AIDOKU_DESIGN.md §7d.
//
// Matching is bracket-depth-aware, not just "up to the next ']'": the
// block can contain its own nested brackets (the copyright notice above
// is itself "[_..._]"), so the first ']' encountered isn't necessarily
// the one that closes the illustration block.
// bareIllustrationTag is Project Gutenberg's other, simpler illustration
// convention: just "[Illustration]" on its own, marking that an image
// appeared there without describing it at all. Unlike the captioned
// "[Illustration: ...]" form above, there's no identifying text worth
// keeping for a possible future "show the real image" feature — so
// removeBareIllustrations drops it entirely rather than condensing it.
const bareIllustrationTag = "[Illustration]"

// removeBareIllustrations deletes every occurrence of bareIllustrationTag.
// Whatever blank space it leaves behind is cleaned up naturally once
// dewrapParagraphs runs — it already skips a paragraph that turns out to
// be pure whitespace once its content is gone.
//
// A plain substring match is safe here (no bracket-depth matching needed
// like condenseIllustrations): "[Illustration]" never appears as a
// substring of a captioned "[Illustration: ...]" block, since a colon
// always follows "Illustration" there.
func removeBareIllustrations(body string) string {
	return strings.ReplaceAll(body, bareIllustrationTag, "")
}

// wordJoin is inserted between two lines that were only separated by
// Gutenberg's line-wrapping — same parameter, same reasoning, as
// dewrapParagraphs in clean.go: " " for a space-separated language
// (English), "" for one that isn't (Japanese doesn't put spaces between
// words, so a caption wrapped across multiple lines must be rejoined
// without inserting one).
func condenseIllustrations(body string, wordJoin string) (string, error) {
	var b strings.Builder
	rest := body
	for {
		idx := strings.Index(rest, illustrationOpenTag)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx])

		block, remainder, err := extractBracketedBlock(rest[idx:])
		if err != nil {
			return "", fmt.Errorf("condense illustration block: %w", err)
		}
		b.WriteString(condenseWhitespace(block, wordJoin))
		rest = remainder
	}
	return b.String(), nil
}

// extractBracketedBlock takes s starting with illustrationOpenTag (which
// itself starts with '[') and returns the full bracket-matched "[...]"
// block, tracking nested brackets so an inner "[...]" (e.g. a copyright
// notice) doesn't end the match early, plus everything in s after it.
func extractBracketedBlock(s string) (block, remainder string, err error) {
	depth := 0
	for i, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[:i+1], s[i+1:], nil
			}
		}
	}
	return "", "", fmt.Errorf("unmatched '[' starting at %q", truncate(s, 60))
}

// condenseWhitespace collapses every run of whitespace (spaces, tabs,
// newlines) in s down to a single wordJoin, and trims the ends — same
// wordJoin convention as dewrapParagraphs (clean.go).
func condenseWhitespace(s string, wordJoin string) string {
	return strings.Join(strings.Fields(s), wordJoin)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
