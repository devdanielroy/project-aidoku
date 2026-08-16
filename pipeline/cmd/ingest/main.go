// Command ingest reads a book catalog (see pipeline/catalogs/, one file
// per language pair) and, for each entry, runs it through every
// pipeline stage that doesn't require the (paid) Claude API: fetch,
// clean, trim (internal/pipeline.PrepareBook — stages 1-2), then
// sentence segmentation (internal/segment — stage 3). Writes two files
// per book: the prepared novel text, and its sentences one per line.
// Stage 4 (LLM chunk grouping) and beyond are not run here.
//
// -pair is required, same as cmd/process and for the same reason (see
// its own doc comment) — this command doesn't call the language
// pair's LanguagePair itself (it never touches question/breakdown
// generation), but it does need to know which Clean/Segment variant to
// run: Clean's dewrapping and Segment's sentence-boundary rules both
// differ by source language (see internal/clean/clean_japanese.go,
// internal/segment/segment_japanese.go).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/clean"
	"aidoku/pipeline/internal/ingest"
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/pipeline"
	"aidoku/pipeline/internal/segment"
	"aidoku/pipeline/internal/types"
)

func main() {
	pairCode := flag.String("pair", "", "language pair to process — required, one of: "+strings.Join(sortedPairCodes(), ", "))
	outDir := flag.String("out", "books", "directory to write prepared book text into, one file per book")
	flag.Parse()

	if err := run(*pairCode, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "ingest:", err)
		os.Exit(1)
	}
}

func sortedPairCodes() []string {
	codes := make([]string, 0, len(langpair.ByCode))
	for code := range langpair.ByCode {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// resolveCatalogPath finds pairCode's catalog file — pipeline/catalogs/
// when run from the repo root, or catalogs/ when run from pipeline/
// itself (same dual-cwd support as cmd/process).
func resolveCatalogPath(pairCode string) (string, error) {
	candidates := []string{
		filepath.Join("catalogs", pairCode+".txt"),
		filepath.Join("pipeline", "catalogs", pairCode+".txt"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no catalog file found for -pair %s (tried %s)", pairCode, strings.Join(candidates, ", "))
}

func run(pairCode, outDir string) error {
	if pairCode == "" {
		return fmt.Errorf("-pair is required, one of: %s", strings.Join(sortedPairCodes(), ", "))
	}
	pair, ok := langpair.ByCode[pairCode]
	if !ok {
		return fmt.Errorf("unknown -pair %q, want one of: %s", pairCode, strings.Join(sortedPairCodes(), ", "))
	}

	catalogPath, err := resolveCatalogPath(pairCode)
	if err != nil {
		return err
	}

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

	// Both Clean and Segment have a Japanese-aware sibling (see
	// internal/clean/clean_japanese.go, internal/segment/
	// segment_japanese.go) — dispatch once per run rather than
	// re-branching per book.
	cleanFn := clean.Clean
	segmentFn := segment.Segment
	if pair.Target == "ja" {
		cleanFn = clean.CleanJapanese
		segmentFn = segment.SegmentJapanese
	}

	failed := 0
	for _, entry := range entries {
		fmt.Printf("[%d] fetching %s ... ", entry.GutenbergID, entry.SourceURL)

		text, err := pipeline.PrepareBook(ctx, client, entry, cleanFn)
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

		sentences := segmentFn(text)
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
