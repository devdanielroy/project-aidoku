package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCondenseIllustrations_CaptionAndCopyright(t *testing.T) {
	raw := "[Illustration:\n\n" +
		"“He came down to see the place”\n\n" +
		"[_Copyright 1894 by George Allen._]]"

	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	want := "[Illustration: “He came down to see the place” [_Copyright 1894 by George Allen._]]"
	if got != want {
		t.Errorf("condenseIllustrations() = %q, want %q", got, want)
	}
}

func TestCondenseIllustrations_CaptionOnly(t *testing.T) {
	raw := "[Illustration:\n\n" +
		"“He rode a black horse”\n\n" +
		"]"

	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	want := "[Illustration: “He rode a black horse” ]"
	if got != want {
		t.Errorf("condenseIllustrations() = %q, want %q", got, want)
	}
}

func TestCondenseIllustrations_DecorativeTextNoCaptionStructure(t *testing.T) {
	// Not every illustration block follows the caption+copyright shape -
	// e.g. a title-page decoration transcribed as plain lines. The
	// bracket-matching approach should condense it regardless of what's
	// actually inside.
	raw := "[Illustration:\n\n" +
		"                             GEORGE ALLEN\n" +
		"                               PUBLISHER\n\n" +
		"                        156 CHARING CROSS ROAD\n" +
		"                                LONDON\n" +
		"                                   ]"

	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	want := "[Illustration: GEORGE ALLEN PUBLISHER 156 CHARING CROSS ROAD LONDON ]"
	if got != want {
		t.Errorf("condenseIllustrations() = %q, want %q", got, want)
	}
}

func TestCondenseIllustrations_SurroundingTextPreserved(t *testing.T) {
	raw := "Some prose before.\n\n" +
		"[Illustration:\n\n“A caption”\n\n]" +
		"\n\nSome prose after."

	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	want := "Some prose before.\n\n[Illustration: “A caption” ]\n\nSome prose after."
	if got != want {
		t.Errorf("condenseIllustrations() = %q, want %q", got, want)
	}
}

func TestCondenseIllustrations_MultipleBlocks(t *testing.T) {
	raw := "[Illustration:\n\nFirst\n\n] and later [Illustration:\n\nSecond\n\n]"

	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	want := "[Illustration: First ] and later [Illustration: Second ]"
	if got != want {
		t.Errorf("condenseIllustrations() = %q, want %q", got, want)
	}
}

func TestCondenseIllustrations_NoIllustrations(t *testing.T) {
	raw := "Just ordinary prose with no illustration blocks at all."
	got, err := condenseIllustrations(raw)
	if err != nil {
		t.Fatalf("condenseIllustrations: %v", err)
	}
	if got != raw {
		t.Errorf("condenseIllustrations() = %q, want unchanged %q", got, raw)
	}
}

func TestCondenseIllustrations_UnmatchedBracket(t *testing.T) {
	raw := "[Illustration: this never closes"
	if _, err := condenseIllustrations(raw); err == nil {
		t.Fatal("condenseIllustrations() = nil error, want an error for an unmatched '['")
	}
}

func TestRemoveBareIllustrations_Basic(t *testing.T) {
	raw := "CHAPTER II.\n\n[Illustration]\n\nFirst real paragraph."
	got := removeBareIllustrations(raw)
	if strings.Contains(got, "[Illustration]") {
		t.Errorf("removeBareIllustrations() = %q, still contains the tag", got)
	}
}

func TestRemoveBareIllustrations_MultipleOccurrences(t *testing.T) {
	raw := "[Illustration]one[Illustration]two[Illustration]"
	want := "onetwo"
	if got := removeBareIllustrations(raw); got != want {
		t.Errorf("removeBareIllustrations() = %q, want %q", got, want)
	}
}

func TestRemoveBareIllustrations_LeavesCaptionedBlocksAlone(t *testing.T) {
	raw := `[Illustration: “A caption” [_Copyright 1894 by George Allen._]]`
	if got := removeBareIllustrations(raw); got != raw {
		t.Errorf("removeBareIllustrations() = %q, want unchanged %q", got, raw)
	}
}

func TestRemoveBareIllustrations_NoBareTags(t *testing.T) {
	raw := "Just ordinary prose."
	if got := removeBareIllustrations(raw); got != raw {
		t.Errorf("removeBareIllustrations() = %q, want unchanged %q", got, raw)
	}
}

func TestClean_RemovesBareIllustrationTag(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"CHAPTER II.\n\n[Illustration]\n\nFirst real paragraph.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	want := "CHAPTER II.\n\nFirst real paragraph."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestClean_CondensesIllustrationBlock(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"Chapter I.\n\n" +
		"[Illustration:\n\n“He came down to see the place”\n\n[_Copyright 1894 by George Allen._]]\n\n" +
		"This was invitation enough.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	want := "Chapter I.\n\n" +
		"[Illustration: “He came down to see the place” [_Copyright 1894 by George Allen._]]\n\n" +
		"This was invitation enough."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

// TestClean_RealPrideAndPrejudice_CondensesIllustrations checks the real
// book: every illustration block should now be a single line, and the
// specific "He came down to see the place" example should read exactly
// as condensed.
func TestClean_RealPrideAndPrejudice_CondensesIllustrations(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pg1342.txt"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	got, err := Clean(string(raw))
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}

	want := `[Illustration: “He came down to see the place” [_Copyright 1894 by George Allen._]]`
	if !strings.Contains(got, want) {
		t.Errorf("expected the condensed illustration line %q to be present", want)
	}

	// No illustration block should still be spread across multiple lines
	// (i.e. no "[Illustration:" immediately followed by a blank line).
	if strings.Contains(got, "[Illustration:\n") || strings.Contains(got, "[Illustration:\n\n") {
		t.Error("found an uncondensed, multi-line illustration block")
	}
}

// TestClean_RealPrideAndPrejudice_RemovesBareIllustrationTags checks that
// none of this edition's many caption-less "[Illustration]" markers
// survive Clean, while a real captioned block still does.
func TestClean_RealPrideAndPrejudice_RemovesBareIllustrationTags(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pg1342.txt"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	got, err := Clean(string(raw))
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}

	if strings.Contains(got, "[Illustration]") {
		t.Error("Clean() output still contains a bare [Illustration] tag")
	}
	if !strings.Contains(got, "[Illustration:") {
		t.Error("expected at least one captioned [Illustration: ...] block to remain")
	}
}
