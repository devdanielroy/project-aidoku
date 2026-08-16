package segment

import "aidoku/pipeline/internal/types"

// isJapaneseTerminal reports whether r is one of the three characters
// that end a Japanese sentence: 。(ideographic full stop, U+3002),
// ！(fullwidth exclamation mark, U+FF01), or ？(fullwidth question
// mark, U+FF1F). Halfwidth ASCII '.', '!', '?' are deliberately not
// included — the pipeline's Japanese source text (Aozora Bunko, via
// Project Gutenberg — see pipeline/catalogs/JP_EN.txt) consistently
// uses the fullwidth/ideographic forms, not the ASCII ones.
func isJapaneseTerminal(r rune) bool {
	return r == '。' || r == '！' || r == '？'
}

// isJapaneseClosingBracket reports whether r is a closing quote or
// bracket that should stay attached to the sentence whose terminal
// punctuation it immediately follows — e.g. the 」 in「ごめんなさい。」,
// consumed the same way Segment consumes a closing ASCII quote right
// after '.'/'!'/'?'.
func isJapaneseClosingBracket(r rune) bool {
	switch r {
	case '」', '』', '）', '】', '〉', '》':
		return true
	}
	return false
}

// isJapaneseOpeningBracket is isJapaneseClosingBracket's counterpart —
// see isBoundaryJapanese for why it's needed.
func isJapaneseOpeningBracket(r rune) bool {
	switch r {
	case '「', '『', '（', '【', '〈', '《':
		return true
	}
	return false
}

// isBoundaryJapanese reports whether k — immediately after terminal
// punctuation and any closing quote/bracket — is a real sentence
// boundary. bracketConsumed is whether a closing bracket was actually
// found there (isJapaneseClosingBracket matched at least once).
//
// If no bracket was consumed, this is always a real boundary — 。
// reliably ends a sentence on its own, with no abbreviation-style
// exceptions the way English's '.' has.
//
// If a bracket WAS consumed, it's genuinely ambiguous: that 。might
// have ended the whole sentence (「分かりません。」on its own), or it
// might just have closed an embedded quotation that a dialogue tag
// continues past — 「ごめんなさい。」と彼女は言った。doesn't actually
// end until after 言った, not at the 。 tucked inside the quote.
// Japanese has no whitespace between sentences for English's
// "whitespace, then does the next character look like a new sentence"
// check to work with, so the signal here is instead: does another
// quote/bracket start immediately, or is this the end of the text?
// Either means the quote really was the whole sentence. Anything else
// (ordinary text, e.g. と彼女は言った) means the surrounding sentence
// just continues past it.
func isBoundaryJapanese(runes []rune, k, n int, bracketConsumed bool) bool {
	if !bracketConsumed {
		return true
	}
	if k >= n {
		return true
	}
	return isJapaneseOpeningBracket(runes[k])
}

// SegmentJapanese splits raw Japanese text into an ordered list of
// sentences — Stage A for the JP_EN language pair (see
// pipeline/internal/langpair), the Japanese-aware sibling of Segment.
//
// Simpler than Segment in one way — 。isn't reused for abbreviations
// the way '.' is, so there's no equivalent of isAbbreviation/isDecimal
// — but the lookahead problem doesn't go away, it just changes shape:
// Japanese has no whitespace between sentences for English's
// "whitespace, then does the next character look like a new sentence"
// check to lean on, so isBoundaryJapanese instead looks at whether a
// closing quote/bracket was consumed and, if so, what immediately
// follows it — see its own doc comment.
//
// A paragraph break (two or more consecutive newlines) still forces a
// boundary regardless of punctuation, same rule as Segment.
//
// Deliberately operates on raw (not yet Clean-dewrapped) text, same as
// Segment: a single line-wrap newline is not a boundary and is
// preserved verbatim inside whichever sentence it falls in. Whether
// dewrapParagraphs (internal/clean) actually hands Stage A pre-joined
// Japanese text yet is a separate, tracked gap in Stage 2 (it currently
// joins wrapped lines with an English-style space, which is wrong for
// Japanese) — not this stage's problem to solve or paper over.
func SegmentJapanese(text string) []types.SentenceInput {
	runes := []rune(text)
	n := len(runes)

	var out []types.SentenceInput
	start := 0
	i := 0

	emit := func(end int) {
		s := trimSpace(runes[start:end])
		if len(s) > 0 {
			out = append(out, types.SentenceInput{
				Index:     len(out),
				Text:      string(s),
				CharCount: len(s),
			})
		}
	}

	for i < n {
		c := runes[i]

		if c == '\n' {
			j := i
			for j < n && runes[j] == '\n' {
				j++
			}
			if j-i >= 2 {
				// Paragraph break - forced boundary regardless of
				// punctuation, same as Segment.
				emit(i)
				start = j
			}
			i = j
			continue
		}

		if !isJapaneseTerminal(c) {
			i++
			continue
		}

		// Consume the full run of terminal punctuation (handles "！？",
		// "。。。").
		j := i
		for j < n && isJapaneseTerminal(runes[j]) {
			j++
		}

		// Consume any closing quotes/brackets immediately after it
		// (handles 「ごめんなさい。」 - the 」 stays with this sentence).
		k := j
		for k < n && isJapaneseClosingBracket(runes[k]) {
			k++
		}
		bracketConsumed := k > j

		if !isBoundaryJapanese(runes, k, n, bracketConsumed) {
			// A dialogue tag continues past this quote (「ごめんなさい。」
			// と彼女は言った。) - not a real boundary, keep scanning for
			// the sentence's actual end.
			i = k
			continue
		}

		emit(k)
		start = k
		i = k
	}

	if start < n {
		emit(n)
	}

	return out
}
