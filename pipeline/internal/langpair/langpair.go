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
}

// JP_EN: the reverse of EN_JP — the learner's native language is
// English, they're studying Japanese. No catalog entries yet
// (pipeline/catalogs/JP_EN.txt starts empty) and no worked
// BreakdownExample for the same reason: nothing's actually been run
// through this pair for real yet to validate one against.
var JP_EN = LanguagePair{
	Target: "ja",
	Native: "en",
	BreakdownSectionLabels: [4]string{
		"[Sentence Structure]",
		"[Vocabulary]",
		"[Grammar]",
		"[Meaning]",
	},
	// ValidateNativeText checks for the *absence* of Japanese script,
	// not the presence of English — there's no cheap, reliable way to
	// positively confirm "this is English" by inspecting text alone
	// (unlike Japanese, English shares its script with most of the
	// pipeline's other prompt/instruction text). Checking that the
	// model didn't just ignore instructions and answer in Japanese
	// (the Target language) anyway still catches the real failure mode.
	ValidateNativeText: validateNotJapanese,
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

// validateNotJapanese is JP_EN's ValidateNativeText — see its own doc
// comment for why this checks the opposite direction from
// validateContainsJapanese.
func validateNotJapanese(text string) error {
	if containsJapanese.MatchString(text) {
		return fmt.Errorf("text appears to still be in Japanese, not English")
	}
	return nil
}
