// Package segment implements Stage A of the book processor: deterministic
// sentence segmentation, no LLM involved. See AIDOKU_DESIGN.md §3a.
//
// Segment never rewrites source text — each returned sentence is an exact
// (whitespace-trimmed) substring of the input, so there is no fidelity risk
// at this stage to verify against. Its one job is finding the right
// boundaries: terminal punctuation ('.', '!', '?') generally ends a
// sentence, except when it's part of an abbreviation ("Mr.", "U.S."), an
// initial ("J. R. R. Tolkien"), a decimal number ("3.14"), or a run of
// punctuation ("...", "?!") not followed by the start of a new sentence.
// Dialogue tags ("Wait," she said, "I don't understand.") don't split a
// sentence because there's no terminal punctuation inside them to begin
// with — commas don't count, and a lowercase word ("she") immediately after
// a quote-enclosed '!' or '?' is not treated as a new sentence start.
//
// A paragraph break (two or more consecutive newlines) always forces a
// boundary too, regardless of terminal punctuation. This matters for text
// with no punctuation of its own at a paragraph's end — a Project
// Gutenberg "[Illustration: ...]" placeholder (see the clean package) is
// the concrete case this exists for: without a forced break, it has
// nothing to stop it silently gluing onto whatever real sentence follows
// it. It's also just correct in general — a sentence never legitimately
// spans a paragraph break — so normal multi-sentence prose (which almost
// always ends paragraphs with real terminal punctuation anyway) is
// unaffected.
package segment

import (
	"unicode"

	"aidoku/pipeline/internal/types"
)

// commonAbbreviations lists words that, when immediately followed by a
// period, don't end a sentence. Matched case-sensitively against the word
// exactly as it appears (titles are almost always capitalized in running
// text; lowercase entries cover inline abbreviations like "etc." and
// "vs.").
var commonAbbreviations = map[string]bool{
	"Mr": true, "Mrs": true, "Ms": true, "Dr": true, "Prof": true,
	"Rev": true, "Gen": true, "Col": true, "Capt": true, "Sgt": true,
	"Lt": true, "Sr": true, "Jr": true, "St": true, "Mt": true,
	"Ave": true, "Blvd": true, "Rd": true, "Co": true, "Inc": true,
	"Ltd": true, "Corp": true, "vs": true, "etc": true, "approx": true,
	"No": true, "Vol": true, "pp": true, "cf": true, "al": true,
}

// Segment splits raw text into an ordered list of sentences. Index is
// zero-based and contiguous; CharCount is the sentence's rune count.
func Segment(text string) []types.SentenceInput {
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
				// punctuation. A no-op if the preceding sentence already
				// closed normally (start == i already, nothing to emit).
				emit(i)
				start = j
			}
			i = j
			continue
		}

		if c != '.' && c != '!' && c != '?' {
			i++
			continue
		}

		// Consume the full run of terminal punctuation (handles "...", "?!", "!!!").
		j := i
		for j < n && (runes[j] == '.' || runes[j] == '!' || runes[j] == '?') {
			j++
		}
		punctRun := j - i

		abbrev := false
		if punctRun == 1 && runes[i] == '.' {
			abbrev = isAbbreviation(runes, start, i) || isDecimal(runes, i, n)
		}

		// Consume any closing quotes/brackets immediately after the punctuation
		// (handles nested quotes: "She whispered, 'Yes.'").
		k := j
		for k < n && isClosingQuoteOrBracket(runes[k]) {
			k++
		}

		if !abbrev && isBoundary(runes, k, n) {
			emit(k)
			start = k
			i = k
			continue
		}
		i = j
	}

	if start < n {
		emit(n)
	}

	return out
}

func trimSpace(rs []rune) []rune {
	start, end := 0, len(rs)
	for start < end && unicode.IsSpace(rs[start]) {
		start++
	}
	for end > start && unicode.IsSpace(rs[end-1]) {
		end--
	}
	return rs[start:end]
}

// isAbbreviation reports whether the '.' at position dot is preceded by a
// known abbreviation or a single letter (an initial, e.g. the "J" in
// "J. R. R. Tolkien" or the "U" in "U.S.").
func isAbbreviation(runes []rune, start, dot int) bool {
	wordEnd := dot
	wordStart := wordEnd
	for wordStart > start && isWordRune(runes[wordStart-1]) {
		wordStart--
	}
	if wordStart == wordEnd {
		return false
	}
	word := string(runes[wordStart:wordEnd])
	if len([]rune(word)) == 1 && unicode.IsUpper(runes[wordStart]) {
		return true
	}
	return commonAbbreviations[word]
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r)
}

// isDecimal reports whether the '.' at position dot sits between two digits
// (e.g. "3.14"), so it's a decimal point, not sentence-ending punctuation.
func isDecimal(runes []rune, dot, n int) bool {
	return dot > 0 && dot+1 < n && unicode.IsDigit(runes[dot-1]) && unicode.IsDigit(runes[dot+1])
}

// isBoundary reports whether position k — immediately after terminal
// punctuation and any closing quotes/brackets — is a real sentence
// boundary. It requires either end-of-text, or whitespace followed by a
// character that plausibly starts a new sentence (uppercase letter, digit,
// or an opening quote/bracket). A non-space character right at k (e.g. the
// second period in "U.S.") means the punctuation was internal, not
// terminal. A lowercase character after whitespace (e.g. "she" in a
// dialogue tag) means what follows continues the same sentence rather than
// starting a new one.
func isBoundary(runes []rune, k, n int) bool {
	if k >= n {
		return true
	}
	if !unicode.IsSpace(runes[k]) {
		return false
	}
	m := k
	for m < n && unicode.IsSpace(runes[m]) {
		m++
	}
	if m >= n {
		return true
	}
	return isSentenceStartRune(runes[m])
}

func isSentenceStartRune(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsDigit(r) || isOpeningQuoteOrBracket(r)
}

func isOpeningQuoteOrBracket(r rune) bool {
	switch r {
	case '"', '\'', '“', '‘', '(', '[', '«':
		return true
	}
	return false
}

func isClosingQuoteOrBracket(r rune) bool {
	switch r {
	case '"', '\'', '”', '’', ')', ']', '»':
		return true
	}
	return false
}
