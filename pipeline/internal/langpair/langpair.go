// Package langpair defines the learner's language pair — the language
// they're studying (their L2, "Target") and their own language (their
// L1, "Native", the language explanations are written in). Pulled out
// of question/breakdown's prompts instead of being hardcoded English/
// Japanese prose in each one, per AIDOKU_DESIGN.md §7's "i18n
// architecture" open question.
//
// There is no default pair — every pipeline run must pick one
// explicitly (see cmd/process's required -pair flag); question.Generator
// and breakdown.Generator both fail loud rather than silently guessing
// if LanguagePair is left unset. See ByCode for the pairs a run can
// choose from.
//
// Deliberately one package-wide config per run, not a per-book field —
// §7 frames a second pair as the whole pipeline run being reconfigured
// (see cmd/process -pair and pipeline/catalogs/, one catalog file per
// pair), not mixed-language content within a single catalog run.
package langpair

import (
	"fmt"
	"regexp"
	"unicode"
)

// LanguagePair bundles everything question/breakdown's prompts need to
// address a learner in a specific native language while teaching a
// specific target language.
type LanguagePair struct {
	// Target is the ISO 639-1 code of the language the learner is
	// studying (L2) — what the source text, and the passages questions
	// are asked about, are written in. This is what's stored/compared
	// everywhere (book.target_language, etc.) — see DisplayName for
	// turning it into the human-readable name prompts actually use.
	Target string

	// Native is the ISO 639-1 code of the learner's own language (L1) —
	// what question prompts/options/explanations and the breakdown are
	// written in.
	Native string

	// BreakdownSectionLabels names breakdown's four standard sections
	// (sentence structure, vocabulary, grammar, meaning), in Native,
	// each already wrapped in whatever heading convention this pair's
	// house style uses (EN_JP's brackets, e.g. 【文構造】, aren't a
	// universal convention — a pair using a different one supplies its
	// own here). breakdown's prompt is built around exactly these four
	// existing, in this order.
	BreakdownSectionLabels [4]string

	// BreakdownExample is a short worked excerpt of the breakdown
	// format, in Native, included in the prompt purely as a formatting
	// example (not content to reuse — see breakdown's prompt). Optional;
	// an empty string just means the prompt's written instructions carry
	// the format alone, without a worked example to sharpen it — better
	// than a fabricated example nobody's actually reviewed for quality.
	BreakdownExample string

	// ValidateNativeText sanity-checks that generated text actually
	// looks like it's written in Native — e.g. a script check, for a
	// language with a distinct Unicode range. Nil skips this check
	// entirely: some Native languages have no cheap way to tell "written
	// in Native" apart from "written in Target" by inspecting the text
	// alone, and there's nothing wrong with relying on the human QA pass
	// (AIDOKU_DESIGN.md §3 stage 5) for those instead.
	ValidateNativeText func(text string) error

	// TargetChunkChars/ChunkCharTolerance are Stage B's (internal/chunk)
	// soft target chunk length and tolerance, in Target characters — see
	// AIDOKU_DESIGN.md §3a. Plain character counts, chosen by feel per
	// pair rather than derived from anything (same as EN_JP's original
	// 240±60 — see internal/chunk's own history): different languages
	// pack a different amount of reading content into the same number of
	// characters (Japanese has no spaces and dense kanji compounds), so
	// a count calibrated for one language's prose doesn't transfer
	// directly to another's.
	TargetChunkChars   int
	ChunkCharTolerance int
}

// ByCode is every language pair the pipeline currently supports, keyed
// by the code cmd/process's -pair flag takes — also the catalog
// filename each maps to (pipeline/catalogs/<code>.txt).
var ByCode = map[string]LanguagePair{
	"EN_JP": EN_JP,
	"JP_EN": JP_EN,
}

// EN_JP: the learner's native language is Japanese, they're studying
// English. v1's shipped pair, and the only one with real catalog
// entries/processed books so far.
var EN_JP = LanguagePair{
	Target: "en",
	Native: "ja",
	BreakdownSectionLabels: [4]string{
		"【文構造】", // sentence structure
		"【語彙】",  // vocabulary
		"【文法】",  // grammar
		"【意味】",  // meaning
	},
	BreakdownExample: `【文構造】"It is a truth universally acknowledged, that ..." は "It is + 過去分詞 + that節" という形式主語構文で、「〜ということは広く認められた真実である」という意味です。

【語彙】
・acknowledged「認められている」
・in want of a wife「妻を欲している」

【文法】"must" は義務ではなく論理的な推測(「〜にちがいない」)を表します。

【意味】表面上は一般論を装っていますが、実際には当時の結婚観への皮肉です。`,
	ValidateNativeText: validateContainsJapanese,
	TargetChunkChars:   240,
	ChunkCharTolerance: 60,
}

// JP_EN: the reverse of EN_JP — the learner's native language is
// English, they're studying Japanese.
var JP_EN = LanguagePair{
	Target: "ja",
	Native: "en",
	BreakdownSectionLabels: [4]string{
		"[Sentence Structure]",
		"[Vocabulary]",
		"[Grammar]",
		"[Meaning]",
	},
	// A real breakdown from processing 羅生門 (Rashomon) for real, not a
	// fabricated one — see EN_JP's own BreakdownExample doc comment for
	// why that matters. One stray zero-width space (U+200B) the model
	// emitted mid-word was stripped; otherwise verbatim.
	BreakdownExample: `[Sentence Structure]
This chunk has three connected sentences building a cause-and-effect chain of decay.

The first, "洛中がその始末であるから、羅生門の修理などは、元より誰も捨てて顧みる者がなかった," is a causal construction: the から-clause ("洛中がその始末であるから") gives the reason, and the main clause ("羅生門の修理などは...なかった") states the consequence — nobody looked after the gate's repairs.

The second part uses a "taking advantage of" construction, "その荒れ果てたのをよい事にして," followed by two short, blunt declarative sentences: "狐狸（こり）が棲む。盗人が棲む。" Notice how these two clauses are given as separate sentences rather than joined with "や" or "し" — this staccato rhythm mimics a list piling up, emphasizing how thoroughly the gate has been abandoned to lawlessness.

The third sentence, "とうとうしまいには、引取り手のない死人を、この門へ持って来て、捨てて行くと云う習慣さえ出来た," builds a long noun phrase ("引取り手のない死人") as the object of a chain of te-form verbs ("持って来て、捨てて行く"), all modifying "習慣" — literally "the custom of bringing here and discarding unclaimed corpses."

[Vocabulary]
・洛中 (rakuchū) – "the capital city" (i.e., Kyoto); used here to refer to the general state of the city as a whole.
・始末 (shimatsu) – here means "such a state of affairs," typically implying a bad or regrettable outcome.
・顧みる (kaerimiru) – to look after, to give thought/care to something.
・荒れ果てる (arehateru) – to fall into utter ruin/disrepair; the てた form here is past/perfective, "had become utterly ruined."
・狐狸 (kori, glossed in the text) – foxes and badgers/raccoon dogs, often used together idiomatically to mean "wild animals" or, figuratively, "sly creatures."
・棲む (sumu) – to inhabit, live (used for animals inhabiting a place, as opposed to 住む for humans).
・引取り手 (hikitorite) – someone who comes to claim or retrieve (a person, an item, a body).
・習慣 (shūkan) – custom, habitual practice.

[Grammar]
・から – causal conjunction ("because"), linking the state of the capital to the neglect of the gate.
・のをよい事にして – a fixed expression meaning "taking advantage of the fact that...," here used to show how the gate's ruin is exploited (opportunistically) by animals and criminals.
・と云う習慣さえ出来た – "even the custom of ~ came to exist." The さえ ("even") intensifies the sense of how far things have deteriorated — it's not just decay, but a full-blown social practice of dumping corpses.

[Meaning]
This passage builds a picture of total societal and physical decay, using the neglected Rashōmon gate as a symbol for a Kyoto in decline. Because the capital itself is in a wretched state, no one bothers to maintain the gate; this neglect invites wild animals and criminals to take up residence; and finally, the abandonment becomes so complete that people begin using the gate as a dumping ground for unclaimed corpses. The escalation — from disrepair, to animals, to thieves, to corpses — mirrors the moral and social breakdown that runs throughout the story, setting up the amoral, desperate world the protagonist inhabits. The terse, list-like phrasing ("狐狸が棲む。盗人が棲む。") reinforces the matter-of-fact tone with which such horrors are now treated as ordinary.`,
	// ValidateNativeText checks for the *absence* of Japanese script,
	// not the presence of English — there's no cheap, reliable way to
	// positively confirm "this is English" by inspecting text alone
	// (unlike Japanese, English shares its script with most of the
	// pipeline's other prompt/instruction text). Checking that the
	// model didn't just ignore instructions and answer in Japanese
	// (the Target language) anyway still catches the real failure mode.
	ValidateNativeText: validateNotJapanese,
	TargetChunkChars:   120,
	ChunkCharTolerance: 30,
}

// displayNames maps the ISO 639-1 codes LanguagePair.Target/Native use
// to the human-readable English name question/breakdown's prompts
// substitute into their (always English) instructional prose — e.g.
// "The learner is a Japanese speaker (L1) learning English (L2)", not
// "...a ja speaker...".
var displayNames = map[string]string{
	"en": "English",
	"ja": "Japanese",
}

// DisplayName returns code's human-readable English name, or code
// itself if unmapped — a missing entry (a new pair whose code wasn't
// added here) should still produce a readable-if-imperfect prompt
// rather than an empty string.
func DisplayName(code string) string {
	if name, ok := displayNames[code]; ok {
		return name
	}
	return code
}

// containsJapanese matches any Hiragana, Katakana, or CJK Unified
// Ideograph rune.
var containsJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`)

// validateContainsJapanese is EN_JP's ValidateNativeText — a cheap
// sanity check that generated text is actually written in Japanese, per
// AIDOKU_DESIGN.md §2 step 5's hard requirement.
func validateContainsJapanese(text string) error {
	if !containsJapanese.MatchString(text) {
		return fmt.Errorf("text does not appear to contain any Japanese")
	}
	return nil
}

// japaneseRatioThreshold is validateNotJapanese's failure threshold —
// the fraction of Japanese-script-vs-Latin letters that must be
// Japanese before it treats text as still being in Japanese rather than
// English. Not zero-tolerance: breakdown's own prompt requires quoting
// exact Japanese (Target) spans verbatim inside the English (Native)
// explanation (see breakdown.go's "Quote exact spans" rule), so a
// handful of legitimately-quoted Japanese characters is expected and
// must not trip this — only a response that's substantially still in
// Japanese, the real failure mode this guards against (the model
// ignoring instructions and answering in Target instead of Native),
// should. Only counting letters (not punctuation, brackets, or digits)
// in the denominator matters here: this pair's own house style uses
// Japanese-style brackets in quoted spans (e.g. 「」), which would
// otherwise dilute a genuine "still all in Japanese" response's ratio
// as much as a compliant one's, defeating the point. 0.3 comfortably
// separates a compliant breakdown (one or two short quoted words/
// sentences against several sentences of English prose — well under
// 10% in practice) from one that's substantially still in Japanese
// (typically at or near 100%, since a non-compliant response has no
// reason to include English words at all).
const japaneseRatioThreshold = 0.3

// validateNotJapanese is JP_EN's ValidateNativeText — see its own doc
// comment for why this checks the opposite direction from
// validateContainsJapanese, and japaneseRatioThreshold's for why it's a
// ratio rather than an absolute "any Japanese at all" check.
func validateNotJapanese(text string) error {
	var japanese, latin int
	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han):
			japanese++
		case unicode.Is(unicode.Latin, r):
			latin++
		}
	}
	if japanese+latin == 0 {
		return nil // no letters at all to judge; empty content is validateBreakdown's job
	}
	if ratio := float64(japanese) / float64(japanese+latin); ratio > japaneseRatioThreshold {
		return fmt.Errorf("text appears to still be substantially in Japanese, not English (%.0f%% Japanese-script letters)", ratio*100)
	}
	return nil
}
