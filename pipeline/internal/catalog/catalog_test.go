package catalog

import (
	"strings"
	"testing"
)

func TestParse_SingleEntry(t *testing.T) {
	src := "Pride and Prejudice\n" +
		"Jane Austen\n" +
		"https://www.gutenberg.org/cache/epub/1342/pg1342.txt\n" +
		"https://example.com/covers/1342.jpg\n" +
		"It is a truth universally acknowledged.\n" +
		"The end of the book.\n" +
		"Level=10\n"

	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	want := Entry{
		Title:       "Pride and Prejudice",
		Author:      "Jane Austen",
		GutenbergID: 1342,
		SourceURL:   "https://www.gutenberg.org/cache/epub/1342/pg1342.txt",
		ImageURL:    "https://example.com/covers/1342.jpg",
		FirstLine:   "It is a truth universally acknowledged.",
		LastLine:    "The end of the book.",
		Level:       LevelScholar,
	}
	if entries[0] != want {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want)
	}
}

func TestParse_MultipleEntriesAndComments(t *testing.T) {
	src := "# Catalog header comment, spanning\n" +
		"# several lines.\n" +
		"\n" +
		"Book One\n" +
		"Author One\n" +
		"https://www.gutenberg.org/cache/epub/1/pg1.txt\n" +
		"https://example.com/covers/1.jpg\n" +
		"First line of book one.\n" +
		"Last line of book one.\n" +
		"Level=1\n" +
		"\n" +
		"Book Two\n" +
		"Author Two\n" +
		"https://www.gutenberg.org/files/2/2-0.txt\n" +
		"https://example.com/covers/2.jpg\n" +
		"First line of book two.\n" +
		"Last line of book two.\n" +
		"Level=10\n"

	entries, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Title != "Book One" || entries[1].Title != "Book Two" {
		t.Errorf("Titles = %q, %q, want %q, %q", entries[0].Title, entries[1].Title, "Book One", "Book Two")
	}
	if entries[0].GutenbergID != 1 || entries[1].GutenbergID != 2 {
		t.Errorf("GutenbergIDs = %d, %d, want 1, 2", entries[0].GutenbergID, entries[1].GutenbergID)
	}
	if entries[0].ImageURL != "https://example.com/covers/1.jpg" || entries[1].ImageURL != "https://example.com/covers/2.jpg" {
		t.Errorf("ImageURLs = %q, %q, unexpected", entries[0].ImageURL, entries[1].ImageURL)
	}
	if entries[0].Level != LevelInitiate || entries[1].Level != LevelScholar {
		t.Errorf("Levels = %v, %v, want %v, %v", entries[0].Level, entries[1].Level, LevelInitiate, LevelScholar)
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
			src := "Title\nAuthor\n" + tc.url + "\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=5\n"
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
			src:  "Title\nAuthor\n",
		},
		{
			name: "entry with only 6 lines (missing Level=)",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\n",
		},
		{
			name: "entry with 8 lines",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=1\nextra\n",
		},
		{
			name: "third line is not a URL",
			src:  "Title\nAuthor\nnot a url\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=1\n",
		},
		{
			name: "fourth line (image URL) is not a URL",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nnot a url\nfirst\nlast\nLevel=1\n",
		},
		{
			name: "URL with no recognizable Gutenberg ID",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/some/other/path\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=1\n",
		},
		{
			name: "level line missing the Level= prefix",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\n10\n",
		},
		{
			name: "level is not a number",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=ten\n",
		},
		{
			name: "level is zero",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=0\n",
		},
		{
			name: "level is above 10",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=11\n",
		},
		{
			name: "level is negative",
			src:  "Title\nAuthor\nhttps://www.gutenberg.org/cache/epub/1/pg1.txt\nhttps://example.com/cover.jpg\nfirst\nlast\nLevel=-1\n",
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

// TestParseFile_RealBooksTxt parses the actual pipeline/catalogs/EN_JP.txt
// this project uses, to keep this package's understanding of the format
// in sync with the real file (and its own header comment).
func TestParseFile_RealBooksTxt(t *testing.T) {
	entries, err := ParseFile("../../catalogs/EN_JP.txt")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry in the real EN_JP.txt")
	}

	found := false
	for _, e := range entries {
		if e.GutenbergID == 6087 {
			found = true
			if e.Title != "The Vampyre" {
				t.Errorf("The Vampyre entry Title = %q, want %q", e.Title, "The Vampyre")
			}
			if !strings.Contains(e.Author, "Polidori") {
				t.Errorf("The Vampyre entry Author looks wrong: %q", e.Author)
			}
			if e.ImageURL == "" {
				t.Error("The Vampyre entry has no ImageURL")
			}
			if !strings.Contains(e.FirstLine, "dissipations attendant upon a London winter") {
				t.Errorf("The Vampyre entry FirstLine looks wrong: %q", e.FirstLine)
			}
			if e.Level != LevelScholar {
				t.Errorf("The Vampyre entry Level = %v, want %v", e.Level, LevelScholar)
			}
		}
	}
	if !found {
		t.Error("expected a Gutenberg ID 6087 (The Vampyre) entry")
	}
}
