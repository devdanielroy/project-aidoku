// Package catalog reads the book catalog file (pipeline/books.txt by
// convention) that drives ingestion: which Gutenberg books to fetch, and
// where each one's actual content starts/ends (see clean.Trim). See that
// file's own header comment for the authoritative format description;
// this package's tests intentionally parse the real file to keep the two
// in sync.
package catalog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Entry is one book in the catalog.
type Entry struct {
	// GutenbergID is the book's Project Gutenberg ebook number, extracted
	// from SourceURL — the stable identifier for this specific
	// transcription/edition, unlike the URL itself (which Gutenberg
	// serves in several different equivalent forms and mirrors). Intended
	// as the future dedup/primary key once there's a storage layer to
	// skip books already ingested.
	GutenbergID int
	SourceURL   string
	FirstLine   string
	LastLine    string
}

// gutenbergIDPattern matches the book ID out of the handful of URL shapes
// Gutenberg serves plain text under, e.g.:
//   - https://www.gutenberg.org/cache/epub/1342/pg1342.txt
//   - https://www.gutenberg.org/files/1342/1342-0.txt
//   - https://www.gutenberg.org/ebooks/1342
var gutenbergIDPattern = regexp.MustCompile(`gutenberg\.org/(?:cache/epub|files|ebooks)/(\d+)`)

// ParseFile reads and parses the catalog file at path.
func ParseFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("catalog: open %s: %w", path, err)
	}
	defer f.Close()

	entries, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("catalog: %s: %w", path, err)
	}
	return entries, nil
}

// Parse reads a catalog from r. See the package doc / books.txt's own
// header for the format: blank-line-separated blocks of exactly 3 lines
// each (URL, first line, last line), with "#"-prefixed comment lines
// ignored wherever they appear.
func Parse(r io.Reader) ([]Entry, error) {
	var block []string
	blockStartLine := 0
	var entries []Entry

	flush := func(endLine int) error {
		if len(block) == 0 {
			return nil
		}
		if len(block) != 3 {
			return fmt.Errorf("entry starting at line %d has %d line(s), want exactly 3 (URL, first line, last line)", blockStartLine, len(block))
		}
		entry, err := newEntry(block[0], block[1], block[2])
		if err != nil {
			return fmt.Errorf("entry starting at line %d: %w", blockStartLine, err)
		}
		entries = append(entries, entry)
		block = nil
		return nil
	}

	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#") {
			continue // comment - never part of a block, doesn't break one either
		}
		if line == "" {
			if err := flush(lineNum); err != nil {
				return nil, err
			}
			continue
		}
		if len(block) == 0 {
			blockStartLine = lineNum
		}
		block = append(block, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	if err := flush(lineNum); err != nil { // final block, if the file doesn't end with a blank line
		return nil, err
	}

	return entries, nil
}

func newEntry(url, firstLine, lastLine string) (Entry, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return Entry{}, fmt.Errorf("first line of entry doesn't look like a URL: %q", url)
	}
	id, err := gutenbergID(url)
	if err != nil {
		return Entry{}, err
	}
	return Entry{GutenbergID: id, SourceURL: url, FirstLine: firstLine, LastLine: lastLine}, nil
}

func gutenbergID(url string) (int, error) {
	m := gutenbergIDPattern.FindStringSubmatch(url)
	if m == nil {
		return 0, fmt.Errorf("couldn't extract a Gutenberg ebook ID from URL: %q", url)
	}
	id, err := strconv.Atoi(m[1])
	if err != nil {
		// Unreachable in practice - the pattern only captures digits -
		// but strconv.Atoi still returns an error type, so handle it
		// rather than ignore it.
		return 0, fmt.Errorf("Gutenberg ID %q in URL %q is not a valid number: %w", m[1], url, err)
	}
	return id, nil
}
