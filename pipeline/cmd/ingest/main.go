// Command ingest reads a book catalog (see pipeline/catalogs/, one file
// per language pair — -catalog defaults to the EN_JP one) and, for each
// entry, runs it through every pipeline stage that doesn't require the
// (paid) Claude API: fetch, clean, trim (internal/pipeline.PrepareBook —
// stages 1-2), then sentence segmentation (internal/segment — stage 3).
// Writes two files per book: the prepared novel text, and its sentences
// one per line. Stage 4 (LLM chunk grouping) and beyond are not run
// here — this command doesn't need to know which language pair it's
// for at all, since it never touches question/breakdown generation.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/ingest"
	"aidoku/pipeline/internal/pipeline"
	"aidoku/pipeline/internal/segment"
	"aidoku/pipeline/internal/types"
)

func main() {
	catalogPath := flag.String("catalog", "catalogs/EN_JP.txt", "path to the book catalog file")
	outDir := flag.String("out", "books", "directory to write prepared book text into, one file per book")
	flag.Parse()

	if err := run(*catalogPath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "ingest:", err)
		os.Exit(1)
	}
}

func run(catalogPath, outDir string) error {
	entries, err := catalog.ParseFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("catalog %s has no entries", catalogPath)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	client := ingest.NewClient()
	ctx := context.Background()

	failed := 0
	for _, entry := range entries {
		fmt.Printf("[%d] fetching %s ... ", entry.GutenbergID, entry.SourceURL)

		text, err := pipeline.PrepareBook(ctx, client, entry)
		if err != nil {
			fmt.Println("FAILED")
			fmt.Fprintf(os.Stderr, "  [%d] %v\n", entry.GutenbergID, err)
			failed++
			continue
		}

		outPath := filepath.Join(outDir, fmt.Sprintf("%d.txt", entry.GutenbergID))
		if err := os.WriteFile(outPath, []byte(text), 0644); err != nil {
			fmt.Println("FAILED")
			fmt.Fprintf(os.Stderr, "  [%d] write %s: %v\n", entry.GutenbergID, outPath, err)
			failed++
			continue
		}

		sentences := segment.Segment(text)
		sentencesPath := filepath.Join(outDir, fmt.Sprintf("%d.sentences.txt", entry.GutenbergID))
		if err := os.WriteFile(sentencesPath, []byte(formatSentences(sentences)), 0644); err != nil {
			fmt.Println("FAILED")
			fmt.Fprintf(os.Stderr, "  [%d] write %s: %v\n", entry.GutenbergID, sentencesPath, err)
			failed++
			continue
		}

		fmt.Printf("ok (%d chars, %d sentences) -> %s, %s\n", len([]rune(text)), len(sentences), outPath, sentencesPath)
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d book(s) failed", failed, len(entries))
	}
	return nil
}

// formatSentences renders sentences one per line as "[index] (char_count)
// text", for human inspection - not a format anything else in the
// pipeline parses (Stage B takes []types.SentenceInput directly, in
// memory; this file is a debugging/inspection artifact, not a handoff
// format).
func formatSentences(sentences []types.SentenceInput) string {
	var b strings.Builder
	for _, s := range sentences {
		fmt.Fprintf(&b, "[%d] (%d) %s\n", s.Index, s.CharCount, s.Text)
	}
	return b.String()
}
