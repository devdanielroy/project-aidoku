package langpair

import "testing"

func TestValidateContainsJapanese(t *testing.T) {
	if err := validateContainsJapanese("これは日本語の文章です。"); err != nil {
		t.Errorf("validateContainsJapanese(<Japanese text>) = %v, want nil", err)
	}
	if err := validateContainsJapanese("This is entirely English."); err == nil {
		t.Error("validateContainsJapanese(<English text>) = nil, want an error")
	}
}

func TestValidateNotJapanese_PureEnglish(t *testing.T) {
	if err := validateNotJapanese("This is a fully English explanation with no Japanese at all."); err != nil {
		t.Errorf("validateNotJapanese(<English text>) = %v, want nil", err)
	}
}

func TestValidateNotJapanese_SubstantiallyJapanese(t *testing.T) {
	// The real failure mode this guards against: the model ignored
	// instructions entirely and answered in the Target language.
	if err := validateNotJapanese("これは日本語の説明です。英語ではありません。まったく日本語のままです。"); err == nil {
		t.Error("validateNotJapanese(<mostly Japanese text>) = nil, want an error")
	}
}

// TestValidateNotJapanese_ToleratesQuotedTargetSpans is the regression
// test for the real bug this function had: breakdown.go's own prompt
// requires quoting exact Japanese (Target) spans verbatim inside the
// English (Native) explanation, so a compliant breakdown legitimately
// contains some Japanese characters. The old zero-tolerance
// implementation rejected every single compliant JP_EN breakdown for
// this — see the "text appears to still be in Japanese" failures this
// caused in a real cmd/process run.
func TestValidateNotJapanese_ToleratesQuotedTargetSpans(t *testing.T) {
	breakdown := `[Sentence Structure]
The sentence "或日の暮方の事である。" uses a nominalizing construction to set the scene.

[Vocabulary]
- 下人 (げにん) "servant" - a low-ranking manservant.
- 羅生門 "Rashomon" - the name of the gate, the story's title.

[Meaning]
This opening line establishes the story's setting: a certain evening, at the Rashomon gate.`

	if err := validateNotJapanese(breakdown); err != nil {
		t.Errorf("validateNotJapanese(<compliant breakdown with quoted spans>) = %v, want nil", err)
	}
}

func TestValidateNotJapanese_Empty(t *testing.T) {
	if err := validateNotJapanese(""); err != nil {
		t.Errorf("validateNotJapanese(\"\") = %v, want nil (validateBreakdown's job to catch empty content)", err)
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("en"); got != "English" {
		t.Errorf("DisplayName(%q) = %q, want %q", "en", got, "English")
	}
	if got := DisplayName("ja"); got != "Japanese" {
		t.Errorf("DisplayName(%q) = %q, want %q", "ja", got, "Japanese")
	}
	if got := DisplayName("xx"); got != "xx" {
		t.Errorf("DisplayName(%q) = %q, want the code itself as a fallback", "xx", got)
	}
}
