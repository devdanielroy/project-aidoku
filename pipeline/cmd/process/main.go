// Command process runs the full pipeline — every stage, including the
// paid Claude API calls — against one or more books from the catalog,
// and persists the result to Postgres via internal/db. This is the "run
// it for real" counterpart to cmd/ingest (which deliberately stops
// before Stage 4 — see its own doc comment) and cmd/livetest (a
// throwaway smoke test on hand-typed sentences, not a real book).
//
// Runs, per book: pipeline.PrepareBook (fetch/clean/trim) -> Stage A
// (segment.Segment or segment.SegmentJapanese, dispatched on -pair) ->
// windowing (chunk.SplitIntoWindows) -> Stage B
// (chunk grouping, one real API call per window) -> question generation
// and breakdown generation (one real API call each, per chunk) ->
// persisted via internal/db. Every book is saved with Status left at its
// default ("processing") — this command does not flip anything to
// "published"; that's stage 5 (QA pass) and stage 6 (Publish), both
// still manual/not built.
//
// Persistence is incremental, not batched: each window's chunks are
// saved to Postgres the moment that window's Stage B call succeeds
// (groupAllChunks), and each chunk's questions/breakdown are saved the
// moment they're generated (the loop below) — both upsert on the same
// row, so a chunk saved grouping-only is simply filled in once its
// questions/breakdown arrive. A crash, interruption, or an
// intentionally-stopped run partway through a book loses no
// already-paid-for work; see the pipeline-incremental-persistence
// memory this was built from.
//
// A chunk that fails question or breakdown generation after retrying is
// logged to stderr and skipped, not treated as fatal — matching
// cmd/ingest's "continue past a single failure" stance, extended to
// per-chunk granularity here. The rest of the book still gets saved;
// staying in "processing" status means nothing half-generated can reach
// a real user regardless.
//
// Re-running against a book that already has grouped chunks in
// Postgres resumes rather than starting over: fetch/clean/segment/
// Stage B grouping are skipped entirely (see chunksForBook), and each
// individual chunk only regenerates whichever of questions/breakdown it
// doesn't already have (see the loop in processBook). Nothing already
// paid for is ever paid for twice by re-running this command.
//
// Use -dry-run first: it runs every free stage (fetch through
// windowing) for real, prints exactly how many real API calls the full
// run would make, and a rough cost estimate — with zero Claude API
// calls and zero Postgres writes. Requires ANTHROPIC_API_KEY and the
// local dev Postgres (`docker compose up -d`) only when not in
// -dry-run.
//
// -pair is required — there is no default language pair (see
// internal/langpair). It picks both the LanguagePair every generated
// question/breakdown is written in and which catalog file
// (pipeline/catalogs/<pair>.txt) supplies the books, so the two can't
// drift out of sync.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/breakdown"
	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/chunk"
	"aidoku/pipeline/internal/clean"
	"aidoku/pipeline/internal/db"
	"aidoku/pipeline/internal/ingest"
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/pipeline"
	"aidoku/pipeline/internal/question"
	"aidoku/pipeline/internal/segment"
	"aidoku/pipeline/internal/types"
	"aidoku/shared/dotenv"
)

func main() {
	pairCode := flag.String("pair", "", "language pair to process — required, one of: "+strings.Join(sortedPairCodes(), ", "))
	wantID := flag.Int("book", 0, "Gutenberg ID to process; 0 processes every entry in the catalog")
	dryRun := flag.Bool("dry-run", false, "compute windowing and real-API-call counts plus a rough cost estimate, without calling the Claude API or writing to Postgres")
	flag.Parse()

	if err := run(*pairCode, *wantID, *dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "process:", err)
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
// itself (same dual-cwd support as the .env loading below).
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

func run(pairCode string, wantID int, dryRun bool) error {
	dotenv.Load(".env")
	dotenv.Load("../.env") // in case run from pipeline/

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
	if wantID != 0 {
		entries = filterByGutenbergID(entries, wantID)
		if len(entries) == 0 {
			return fmt.Errorf("no catalog entry with Gutenberg ID %d", wantID)
		}
	}
	if len(entries) == 0 {
		return fmt.Errorf("catalog %s has no entries", catalogPath)
	}

	ctx := context.Background()
	ingestClient := ingest.NewClient()

	// Prerequisites for the real (non-dry-run) path, checked up front so
	// a missing key or unreachable Postgres fails immediately rather
	// than after fetching every book from Gutenberg first.
	var client *anthropic.Client
	var store *db.Store
	if !dryRun {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY is not set (checked real env, .env, and ../.env)")
		}
		client = anthropic.NewClient(apiKey)

		pool, err := db.Open(ctx, db.ConnStringFromEnv())
		if err != nil {
			return fmt.Errorf("connect to Postgres (is `docker compose up -d` running?): %w", err)
		}
		defer pool.Close()
		store = db.New(pool)
	}

	failedBooks := 0
	for _, entry := range entries {
		if err := processBook(ctx, entry, pair, ingestClient, client, store, dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "book %d (%s): %v\n", entry.GutenbergID, entry.Title, err)
			failedBooks++
		}
	}
	if failedBooks > 0 {
		return fmt.Errorf("%d of %d book(s) failed", failedBooks, len(entries))
	}
	return nil
}

func filterByGutenbergID(entries []catalog.Entry, id int) []catalog.Entry {
	var filtered []catalog.Entry
	for _, e := range entries {
		if e.GutenbergID == id {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func processBook(ctx context.Context, entry catalog.Entry, pair langpair.LanguagePair, ingestClient *ingest.Client, client *anthropic.Client, store *db.Store, dryRun bool) error {
	fmt.Printf("=== %s by %s (Gutenberg #%d) ===\n", entry.Title, entry.Author, entry.GutenbergID)

	if dryRun {
		return printDryRunEstimate(ctx, entry, pair, ingestClient)
	}

	book := db.NewBookFromEntry(entry, pair)
	bookID, err := store.UpsertBook(ctx, book)
	if err != nil {
		return fmt.Errorf("save book: %w", err)
	}

	allChunks, err := chunksForBook(ctx, entry, pair, ingestClient, client, store, bookID)
	if err != nil {
		return err
	}
	fmt.Printf("  -> %d chunk(s) total\n", len(allChunks))

	questionGen := question.NewGenerator(client, pair)
	breakdownGen := breakdown.NewGenerator(client, pair)

	failedChunks, skippedChunks := 0, 0
	for i := range allChunks {
		p := allChunks[i]
		c := p.Chunk

		if p.HasQuestions && p.HasBreakdown {
			skippedChunks++
			continue
		}

		fmt.Printf("  chunk %d/%d: ", c.Index+1, len(allChunks))

		chunkID := p.ID
		if p.HasQuestions {
			// Resumed from a prior run that got this far — questions
			// are already saved (chunkID already known), only
			// breakdown is missing. No point re-spending a real API
			// call regenerating questions that already exist.
			fmt.Print("questions already saved, skipping...")
		} else {
			fmt.Print("questions...")
			questions, err := questionGen.GenerateQuestions(ctx, c)
			if err != nil {
				fmt.Println(" FAILED")
				fmt.Fprintf(os.Stderr, "    chunk %d questions: %v\n", c.Index, err)
				failedChunks++
				continue
			}
			chunkID, err = store.SaveChunk(ctx, bookID, c, questions)
			if err != nil {
				return fmt.Errorf("save chunk %d: %w", c.Index, err)
			}
		}

		fmt.Print(" breakdown...")
		content, err := breakdownGen.GenerateBreakdown(ctx, c)
		if err != nil {
			fmt.Println(" FAILED")
			fmt.Fprintf(os.Stderr, "    chunk %d breakdown: %v\n", c.Index, err)
			failedChunks++
			continue
		}

		if err := store.SaveBreakdown(ctx, chunkID, content); err != nil {
			return fmt.Errorf("save breakdown for chunk %d: %w", c.Index, err)
		}
		fmt.Println(" saved")
	}

	fmt.Printf("=== %q: %d/%d chunk(s) fully processed and saved", entry.Title, len(allChunks)-failedChunks, len(allChunks))
	if skippedChunks > 0 {
		fmt.Printf(" (%d already complete from a prior run)", skippedChunks)
	}
	fmt.Println(" ===")
	fmt.Println()
	if failedChunks > 0 {
		return fmt.Errorf("%d of %d chunk(s) failed generation (see stderr above) — the rest were saved", failedChunks, len(allChunks))
	}
	return nil
}

// cleanAndSegmentFns picks Clean/Segment's Japanese-aware sibling when
// pair's source (Target) language calls for one — see
// internal/clean/clean_japanese.go, internal/segment/segment_japanese.go.
func cleanAndSegmentFns(pair langpair.LanguagePair) (pipeline.CleanFunc, func(string) []types.SentenceInput) {
	if pair.Target == "ja" {
		return clean.CleanJapanese, segment.SegmentJapanese
	}
	return clean.Clean, segment.Segment
}

// chunksForBook returns bookID's chunks, resuming from Postgres instead
// of re-grouping from scratch whenever a prior run already got this
// book through Stage B. Stage B (chunk.Grouper) is a real, paid API
// call per window — see groupAllChunks — so a book that already has
// grouped chunks (from a run that got interrupted, hit a transient
// failure, or ran out of credits partway through question/breakdown
// generation) should never pay for that again just to pick up where it
// left off. The free stages (fetch/clean/segment/windowing) are skipped
// right along with it in that case: there's nothing left for them to
// feed into.
func chunksForBook(ctx context.Context, entry catalog.Entry, pair langpair.LanguagePair, ingestClient *ingest.Client, client *anthropic.Client, store *db.Store, bookID int) ([]db.ChunkProgress, error) {
	existing, err := store.LoadChunkProgress(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("load existing chunks: %w", err)
	}
	if len(existing) > 0 {
		fmt.Printf("  resuming: %d chunk(s) already grouped in a prior run, skipping fetch/clean/segment/grouping\n", len(existing))
		return existing, nil
	}

	cleanFn, segmentFn := cleanAndSegmentFns(pair)
	text, err := pipeline.PrepareBook(ctx, ingestClient, entry, cleanFn)
	if err != nil {
		return nil, fmt.Errorf("prepare: %w", err)
	}
	sentences := segmentFn(text)
	windows := chunk.SplitIntoWindows(sentences, chunk.WindowTargetChars)
	fmt.Printf("%d chars, %d sentences, %d window(s) for chunk grouping\n", len([]rune(text)), len(sentences), len(windows))

	grouped, err := groupAllChunks(ctx, client, store, bookID, windows, pair)
	if err != nil {
		return nil, err
	}
	progress := make([]db.ChunkProgress, len(grouped))
	for i, c := range grouped {
		progress[i] = db.ChunkProgress{Chunk: c}
	}
	return progress, nil
}

// printDryRunEstimate runs every free stage (fetch through windowing)
// for real and prints the cost estimate cmd/process's own doc comment
// promises — no Postgres involved, so it never sees (and never needs
// to skip past) a resumed book; see chunksForBook for the real-run
// resume path this mirrors free-stage-wise.
func printDryRunEstimate(ctx context.Context, entry catalog.Entry, pair langpair.LanguagePair, ingestClient *ingest.Client) error {
	cleanFn, segmentFn := cleanAndSegmentFns(pair)
	text, err := pipeline.PrepareBook(ctx, ingestClient, entry, cleanFn)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	sentences := segmentFn(text)
	windows := chunk.SplitIntoWindows(sentences, chunk.WindowTargetChars)

	charCount := len([]rune(text))
	estimatedChunks := (charCount + pair.TargetChunkChars/2) / pair.TargetChunkChars // rounded, rough
	if estimatedChunks == 0 {
		estimatedChunks = 1
	}
	fmt.Printf("%d chars, %d sentences, %d window(s) for chunk grouping, ~%d chunk(s) estimated\n",
		charCount, len(sentences), len(windows), estimatedChunks)

	printCostEstimate(len(windows), estimatedChunks)
	fmt.Println()
	return nil
}

// groupAllChunks runs Stage B once per window and concatenates the
// results into one globally-indexed chunk list for the whole book —
// each window's chunk_index starts back at 0 (see
// chunk.ValidatePartition), so indices are renumbered sequentially
// across windows here.
//
// Each window's chunks are persisted (via store.SaveChunk, no
// questions yet) immediately after that window's Stage B call
// succeeds, before moving on to the next window. That real API call is
// paid for the moment it returns — if a later window fails, or the
// process is interrupted, or the run stops for the night, every window
// already grouped stays in Postgres rather than only in allChunks'
// memory. Questions arrive later per chunk (see processBook's loop),
// which upserts the same row again via SaveChunk — see
// pipeline-incremental-persistence.
func groupAllChunks(ctx context.Context, client *anthropic.Client, store *db.Store, bookID int, windows [][]types.SentenceInput, pair langpair.LanguagePair) ([]types.Chunk, error) {
	grouper := chunk.NewGrouper(client, pair)
	var allChunks []types.Chunk
	nextIndex := 0
	for i, window := range windows {
		fmt.Printf("  chunk grouping: window %d/%d (%d sentences)...\n", i+1, len(windows), len(window))
		resp, err := grouper.GroupSentencesIntoChunks(ctx, window)
		if err != nil {
			return nil, fmt.Errorf("chunk grouping window %d: %w", i, err)
		}
		windowChunks := chunk.BuildChunks(window, resp)
		for j := range windowChunks {
			windowChunks[j].Index = nextIndex
			nextIndex++
		}
		for _, c := range windowChunks {
			if _, err := store.SaveChunk(ctx, bookID, c, nil); err != nil {
				return nil, fmt.Errorf("save chunk %d (window %d): %w", c.Index, i, err)
			}
		}
		allChunks = append(allChunks, windowChunks...)
	}
	return allChunks, nil
}

// printCostEstimate is a rough, character-count-based estimate, not a
// measurement — actual token usage varies with real content, especially
// on the output side (Japanese text tokenizes denser than English, and
// response length varies chunk to chunk). Treat this as a ballpark
// before committing spend, not a bill. Pricing is claude-sonnet-5's
// intro rate ($2/$10 per 1M input/output tokens, through 2026-08-31 per
// Anthropic's published pricing at the time this was written; $3/$15
// after) — see the claude-api skill / platform.claude.com/docs/pricing
// for current rates if this has gone stale.
func printCostEstimate(windows, estimatedChunks int) {
	const inputPerMTok, outputPerMTok = 2.00, 10.00

	// Rough per-call token budgets, informed by this pipeline's actual
	// system prompts and the response sizes seen from cmd/livetest so
	// far (see internal/chunk, internal/question, internal/breakdown).
	const (
		groupingInputTok, groupingOutputTok   = 1200.0, 200.0
		questionInputTok, questionOutputTok   = 700.0, 350.0
		breakdownInputTok, breakdownOutputTok = 700.0, 600.0
	)

	inputTok := float64(windows)*groupingInputTok + float64(estimatedChunks)*(questionInputTok+breakdownInputTok)
	outputTok := float64(windows)*groupingOutputTok + float64(estimatedChunks)*(questionOutputTok+breakdownOutputTok)
	cost := inputTok/1_000_000*inputPerMTok + outputTok/1_000_000*outputPerMTok

	totalCalls := windows + 2*estimatedChunks
	fmt.Printf("  estimated real API calls: %d chunk-grouping + %d question-gen + %d breakdown-gen = %d total\n",
		windows, estimatedChunks, estimatedChunks, totalCalls)
	fmt.Printf("  rough cost estimate: $%.3f (character-count-based, not a measurement)\n", cost)
}
