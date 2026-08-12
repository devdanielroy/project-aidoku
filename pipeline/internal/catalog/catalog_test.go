package catalog

import (
	"strings"
	"testing"
)

func TestParse_SingleEntry(t *testing.T) {
	src := "https://www.gutenberg.org/cache/epub/1342/pg1342.txt\n" +
		"It is a truth universally acknowledged.\n" +
		"The end of the book.\n"

	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	want := Entry{
		GutenbergID: 1342,
		SourceURL:   "https://www.gutenberg.org/cache/epub/1342/pg1342.txt",
		FirstLine:   "It is a truth universally acknowledged.",
		LastLine:    "The end of the book.",
	}
	if entries[0] != want {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want)
	}
}

func TestParse_MultipleEntriesAndComments(t *testing.T) {
	src := "# Catalog header comment, spanning\n" +
		"# several lines.\n" +
		"\n" +
		"# Book One\n" +
		"https://www.gutenberg.org/cache/epub/1/pg1.txt\n" +
		"First line of book one.\n" +
		"Last line of book one.\n" +
		"\n" +
		"# Book Two\n" +
		"https://www.gutenberg.org/files/2/2-0.txt\n" +
		"First line of book two.\n" +
		"Last line of book two.\n"

	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].GutenbergID != 1 || entries[1].GutenbergID != 2 {
		t.Errorf("GutenbergIDs = %d, %d, want 1, 2", entries[0].GutenbergID, entries[1].GutenbergID)
	}
}

func TestParse_URLShapes(t *testing.T) {
	cases := []struct {
		url    string
		wantID int
	}{
		{"https://www.gutenberg.org/cache/epub/1342/pg1342.txt", 1342},
		{"https://www.gutenberg.org/files/84/84-0.txt", 84},
		{"https://www.gutenberg.org/ebooks/345", 345},
		{"http://gutenberg.org/cache/epub/11/pg11.txt", 11}, // no "www.", http not https
	}

	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			src := tc.url + "\nfirst\nlast\n"
			entries, err := Parse(strings.NewReader(src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if entries[0].GutenbergID != tc.wantID {
				t.Errorf("GutenbergID = %d, want %d", entries[0].GutenbergID, tc.wantID)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "entry with only 2 lines",
			src:  "https://www.gutenberg.org/cache/epub/1/pg1.txt\nonly one content line\n",
		},
		{
			name: "entry with 4 lines",
			src:  "https://www.gutenberg.org/cache/epub/1/pg1.txt\nfirst\nlast\nextra\n",
		},
		{
			name: "first line is not a URL",
			src:  "not a url\nfirst\nlast\n",
		},
		{
			name: "URL with no recognizable Gutenberg ID",
			src:  "https://www.gutenberg.org/some/other/path\nfirst\nlast\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(tc.src)); err == nil {
				t.Fatal("Parse() = nil error, want an error")
			}
		})
	}
}

func TestParse_Empty(t *testing.T) {
	entries, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestParse_CommentOnlyFile(t *testing.T) {
	entries, err := Parse(strings.NewReader("# just a comment\n# another\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

// TestParseFile_RealBooksTxt parses the actual pipeline/books.txt this
// project uses, to keep this package's understanding of the format in
// sync with the real file (and its own header comment).
func TestParseFile_RealBooksTxt(t *testing.T) {
	entries, err := ParseFile("../../books.txt")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry in the real books.txt")
	}

	found := false
	for _, e := range entries {
		if e.GutenbergID == 1342 {
			found = true
			if !strings.Contains(e.FirstLine, "truth universally acknowledged") {
				t.Errorf("Pride and Prejudice entry FirstLine looks wrong: %q", e.FirstLine)
			}
		}
	}
	if !found {
		t.Error("expected a Gutenberg ID 1342 (Pride and Prejudice) entry")
	}
}
