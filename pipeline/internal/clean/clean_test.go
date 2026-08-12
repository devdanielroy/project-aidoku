package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aidoku/pipeline/internal/catalog"
)

func TestClean_StripsHeaderAndFooter(t *testing.T) {
	raw := "The Project Gutenberg eBook of Some Book\n" +
		"\n" +
		"This eBook is for the use of anyone anywhere...\n" +
		"\n" +
		"*** START OF THE PROJECT GUTENBERG EBOOK SOME BOOK ***\n" +
		"\n" +
		"Chapter I.\n" +
		"\n" +
		"It was a dark and stormy night.\n" +
		"\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK SOME BOOK ***\n" +
		"\n" +
		"This and all associated files of various formats will be found in...\n"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}

	want := "Chapter I.\n\nIt was a dark and stormy night."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestClean_HandlesThisVariant(t *testing.T) {
	// Some Gutenberg editions say "OF THIS PROJECT GUTENBERG EBOOK" rather
	// than "OF THE PROJECT GUTENBERG EBOOK".
	raw := "*** START OF THIS PROJECT GUTENBERG EBOOK SOME BOOK ***\nBody text.\n*** END OF THIS PROJECT GUTENBERG EBOOK SOME BOOK ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if got != "Body text." {
		t.Errorf("Clean() = %q, want %q", got, "Body text.")
	}
}

func TestClean_NormalizesCRLF(t *testing.T) {
	// Two blank-line-separated paragraphs, each wrapped across two lines,
	// all using CRLF - checks CRLF normalization survives both the
	// paragraph-break detection and the line-dewrapping.
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\r\n" +
		"Line one\r\nstill paragraph one.\r\n" +
		"\r\n" +
		"Line two\r\nstill paragraph two.\r\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if strings.Contains(got, "\r") {
		t.Errorf("Clean() output still contains CR bytes: %q", got)
	}
	want := "Line one still paragraph one.\n\nLine two still paragraph two."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestClean_StripsBOM(t *testing.T) {
	raw := utf8BOM + "*** START OF THE PROJECT GUTENBERG EBOOK X ***\nBody.\n*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if strings.Contains(got, utf8BOM) {
		t.Errorf("Clean() output still contains a BOM: %q", got)
	}
	if got != "Body." {
		t.Errorf("Clean() = %q, want %q", got, "Body.")
	}
}

func TestClean_CollapsesExcessBlankLines(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"Paragraph one.\n\n\n\n\nParagraph two.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("Clean() output still has a run of 3+ newlines: %q", got)
	}
	if got != "Paragraph one.\n\nParagraph two." {
		t.Errorf("Clean() = %q, want %q", got, "Paragraph one.\n\nParagraph two.")
	}
}

func TestClean_DewrapsHardWrappedLines(t *testing.T) {
	// Mimics Gutenberg's fixed-width wrapping: a sentence broken across
	// lines mid-word-boundary, with indentation (as centered title-page
	// text commonly has) that shouldn't leave a gap when joined.
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"That is an\n" +
		"uncommon advantage, and uncommon I hope it will\n" +
		"                    continue.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	want := "That is an uncommon advantage, and uncommon I hope it will continue."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestClean_PreservesParagraphBreaks(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"First\nparagraph, wrapped.\n" +
		"\n" +
		"Second\nparagraph, also wrapped.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	want := "First paragraph, wrapped.\n\nSecond paragraph, also wrapped."
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestClean_LeavesEmphasisUnderscoresUntouched(t *testing.T) {
	// Clean previously converted Gutenberg's "_word_" emphasis convention
	// into special markers; reverted (see git history / design doc §7d)
	// since the output was noisier than useful at this stage. Underscores
	// should just pass through as ordinary characters, same as any other
	// punctuation Clean doesn't otherwise touch.
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"“_You_ want to tell me, and I have no objection to hearing it.”\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := Clean(raw)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	want := "“_You_ want to tell me, and I have no objection to hearing it.”"
	if got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestTrim_Basic(t *testing.T) {
	text := "Front matter, junk, preface, whatever.\n\n" +
		"Chapter I.\n\nIt was a dark and stormy night. The story went on for a while.\n\n" +
		"Back matter: colophon, THE END, printer's imprint."

	got, err := Trim(text, "Chapter I.", "The story went on for a while.")
	if err != nil {
		t.Fatalf("Trim: %v", err)
	}
	want := "Chapter I.\n\nIt was a dark and stormy night. The story went on for a while."
	if got != want {
		t.Errorf("Trim() = %q, want %q", got, want)
	}
}

func TestTrim_FirstLineNotFound(t *testing.T) {
	_, err := Trim("Some text here.", "Not present anywhere.", "text here.")
	if err == nil {
		t.Fatal("Trim() = nil error, want an error when the first-line anchor isn't found")
	}
}

func TestTrim_LastLineNotFound(t *testing.T) {
	_, err := Trim("Some text here.", "Some text", "Not present anywhere.")
	if err == nil {
		t.Fatal("Trim() = nil error, want an error when the last-line anchor isn't found")
	}
}

func TestTrim_AmbiguousAnchor(t *testing.T) {
	// "the cat" appears twice - not specific enough to be a safe cut point.
	text := "the cat sat down. Later, the cat sat down again."
	_, err := Trim(text, "the cat sat down", "again.")
	if err == nil {
		t.Fatal("Trim() = nil error, want an error when the first-line anchor matches more than once")
	}
}

func TestTrim_LastLineBeforeFirstLine(t *testing.T) {
	text := "one two three four five"
	_, err := Trim(text, "four", "two")
	if err == nil {
		t.Fatal("Trim() = nil error, want an error when the last-line anchor appears before the first-line anchor")
	}
}

func TestClean_MissingMarkers(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{
			name: "no start marker",
			raw:  "Just some text.\n*** END OF THE PROJECT GUTENBERG EBOOK X ***",
		},
		{
			name: "no end marker",
			raw:  "*** START OF THE PROJECT GUTENBERG EBOOK X ***\nJust some text.",
		},
		{
			name: "no markers at all",
			raw:  "Just some plain text with no Gutenberg wrapper.",
		},
		{
			name: "end appears before start",
			raw:  "*** END OF THE PROJECT GUTENBERG EBOOK X ***\nbackwards\n*** START OF THE PROJECT GUTENBERG EBOOK X ***",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Clean(tc.raw); err == nil {
				t.Fatal("Clean() = nil error, want an error")
			}
		})
	}
}

// TestClean_RealPrideAndPrejudice runs Clean against the actual Project
// Gutenberg plain-text edition of Pride and Prejudice (ebook #1342),
// downloaded verbatim — see testdata/pg1342.txt — to catch real-world
// formatting quirks a hand-crafted fixture wouldn't.
func TestClean_RealPrideAndPrejudice(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pg1342.txt"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	got, err := Clean(string(raw))
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}

	if strings.Contains(got, "START OF") || strings.Contains(got, "END OF") {
		t.Error("Clean() output still contains a Gutenberg START/END marker")
	}
	if strings.Contains(got, "Credits: Chuck Greif") {
		t.Error("Clean() output still contains pre-START header content")
	}
	if strings.Contains(got, "Updated editions will replace the previous one") {
		t.Error("Clean() output still contains post-END license content")
	}
	if !strings.Contains(got, "It is a truth universally acknowledged") {
		t.Error("Clean() output is missing the novel's actual opening line")
	}
	// This edition's raw text hard-wraps right in the middle of this
	// phrase ("...in possession\nof a good fortune..."), so this
	// specifically checks dewrapParagraphs joined it back into one
	// flowing phrase rather than leaving the line break embedded.
	if !strings.Contains(got, "in possession of a good fortune") {
		t.Error("Clean() output has a hard-wrapped line break embedded where dewrapping should have joined it")
	}

	// Known limitation, not a bug — see the package doc: this edition's
	// illustrated-edition preface, which sits *between* the START marker
	// and "Chapter I.", is intentionally NOT stripped by Clean.
	if !strings.Contains(got, "George Saintsbury") {
		t.Error("expected the (out-of-scope-to-strip) preface content to still be present")
	}

	// Emphasis underscores pass through untouched (see
	// TestClean_LeavesEmphasisUnderscoresUntouched) - this book has
	// plenty of them (e.g. Mrs. Bennet's "_You_ want to tell me").
	if !strings.Contains(got, "“_You_ want to tell me") {
		t.Error(`expected the "_You_" emphasis underscores to survive untouched`)
	}

	// TEMPORARY - for manual inspection only, remove before committing.
	outPath := filepath.Join(os.TempDir(), "pride_and_prejudice.txt")
	if err := os.WriteFile(outPath, []byte(got), 0644); err != nil {
		t.Fatalf("write inspection file: %v", err)
	}
	t.Logf("wrote cleaned output to %s", outPath)
}

// TestTrim_RealPrideAndPrejudice runs Clean then Trim on the real Pride
// and Prejudice text, using the actual anchors from pipeline/books.txt —
// both to prove Trim works end to end against real (not hand-crafted)
// content, and as a regression check that books.txt's anchors still
// match if either the book text or Clean's normalization ever changes.
func TestTrim_RealPrideAndPrejudice(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pg1342.txt"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	cleaned, err := Clean(string(raw))
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}

	entries, err := catalog.ParseFile(filepath.Join("..", "..", "books.txt"))
	if err != nil {
		t.Fatalf("catalog.ParseFile: %v", err)
	}
	var entry *catalog.Entry
	for i := range entries {
		if entries[i].GutenbergID == 1342 {
			entry = &entries[i]
		}
	}
	if entry == nil {
		t.Fatal("books.txt has no entry for Gutenberg ID 1342 (Pride and Prejudice)")
	}

	got, err := Trim(cleaned, entry.FirstLine, entry.LastLine)
	if err != nil {
		t.Fatalf("Trim: %v", err)
	}

	if !strings.HasPrefix(got, entry.FirstLine) {
		t.Errorf("Trim() output doesn't start with the first-line anchor: %q", got[:min(80, len(got))])
	}
	if !strings.HasSuffix(got, entry.LastLine) {
		t.Errorf("Trim() output doesn't end with the last-line anchor: %q", got[max(0, len(got)-80):])
	}

	// Front matter (the Saintsbury preface) and back matter (the
	// illustration caption and printer's colophon) should now be gone -
	// this is the whole point of Trim, unlike plain Clean which
	// deliberately leaves them (see TestClean_RealPrideAndPrejudice).
	if strings.Contains(got, "George Saintsbury") {
		t.Error("Trim() output still contains the preface")
	}
	if strings.Contains(got, "CHISWICK PRESS") {
		t.Error("Trim() output still contains the printer's colophon")
	}
	if strings.Contains(got, "THE END") {
		t.Error("Trim() output still contains the illustration's \"THE END\" caption")
	}
}
