package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/clean"
)

type fakeFetcher struct {
	text string
	err  error
	// gotURL records the URL the last FetchText call was made with, so
	// tests can confirm PrepareBook actually used entry.SourceURL.
	gotURL string
}

func (f *fakeFetcher) FetchText(ctx context.Context, url string) (string, error) {
	f.gotURL = url
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

func testEntry() catalog.Entry {
	return catalog.Entry{
		GutenbergID: 1,
		SourceURL:   "https://www.gutenberg.org/cache/epub/1/pg1.txt",
		FirstLine:   "Chapter I.",
		LastLine:    "The end of the story.",
	}
}

func TestPrepareBook_Success(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"Some preface junk before the real content.\n" +
		"\n" +
		"Chapter I.\n" +
		"\n" +
		"It was a dark and stormy night. The end of the story.\n" +
		"\n" +
		"Colophon and license junk after.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	fetcher := &fakeFetcher{text: raw}
	entry := testEntry()

	got, err := PrepareBook(context.Background(), fetcher, entry, clean.Clean)
	if err != nil {
		t.Fatalf("PrepareBook: %v", err)
	}

	if fetcher.gotURL != entry.SourceURL {
		t.Errorf("fetched URL = %q, want %q", fetcher.gotURL, entry.SourceURL)
	}
	want := "Chapter I.\n\nIt was a dark and stormy night. The end of the story."
	if got != want {
		t.Errorf("PrepareBook() = %q, want %q", got, want)
	}
	if strings.Contains(got, "preface junk") || strings.Contains(got, "Colophon") {
		t.Errorf("PrepareBook() output still contains front/back matter: %q", got)
	}
}

func TestPrepareBook_FetchError(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("network down")}
	_, err := PrepareBook(context.Background(), fetcher, testEntry(), clean.Clean)
	if err == nil {
		t.Fatal("PrepareBook() = nil error, want an error when fetching fails")
	}
}

func TestPrepareBook_CleanError(t *testing.T) {
	// No Gutenberg START/END markers at all - Clean should reject it.
	fetcher := &fakeFetcher{text: "Just some text with no Gutenberg wrapper."}
	_, err := PrepareBook(context.Background(), fetcher, testEntry(), clean.Clean)
	if err == nil {
		t.Fatal("PrepareBook() = nil error, want an error when Clean fails")
	}
}

func TestPrepareBook_TrimError(t *testing.T) {
	// Well-formed Gutenberg wrapper, but the entry's anchors don't
	// actually appear in the content - Trim should reject it.
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"This text does not contain either anchor line.\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	fetcher := &fakeFetcher{text: raw}
	_, err := PrepareBook(context.Background(), fetcher, testEntry(), clean.Clean)
	if err == nil {
		t.Fatal("PrepareBook() = nil error, want an error when Trim's anchors aren't found")
	}
}

// TestPrepareBook_UsesTheProvidedCleanFunc confirms PrepareBook actually
// calls whichever CleanFunc it's given rather than hardcoding clean.Clean
// — the whole point of taking one as a parameter (see cmd/process and
// cmd/ingest, which pick clean.Clean vs clean.CleanJapanese based on
// their -pair flag).
func TestPrepareBook_UsesTheProvidedCleanFunc(t *testing.T) {
	fetcher := &fakeFetcher{text: "irrelevant, cleanFn below ignores its input"}
	entry := testEntry()
	entry.FirstLine = "custom clean output"
	entry.LastLine = "custom clean output"

	var gotRaw string
	customClean := func(raw string) (string, error) {
		gotRaw = raw
		return "custom clean output", nil
	}

	got, err := PrepareBook(context.Background(), fetcher, entry, customClean)
	if err != nil {
		t.Fatalf("PrepareBook: %v", err)
	}
	if gotRaw != fetcher.text {
		t.Errorf("custom CleanFunc was called with %q, want the fetched text %q", gotRaw, fetcher.text)
	}
	if got != "custom clean output" {
		t.Errorf("PrepareBook() = %q, want the custom CleanFunc's output to have been used", got)
	}
}
