// Command livetest is a manual, throwaway smoke test against the real
// Anthropic API — NOT part of the pipeline proper and not wired into any
// other command. It runs Stage B (chunk grouping) once on a handful of
// real sentences from the opening of Pride and Prejudice, then runs
// question generation and breakdown generation once each on the
// resulting first chunk. Total: exactly three real API calls, all
// against claude-sonnet-5 with tiny inputs.
//
// Results are persisted to Postgres afterward (see internal/db) so
// they're still there on the next run instead of only ever living in
// this process's stdout — that's a local write, not an API call, so it
// doesn't count against the "only I invoke the Claude API" rule and
// needs no separate go-ahead. Requires the local dev Postgres to be up
// (`docker compose up -d` from the repo root); this fails loudly if it
// isn't, rather than silently skipping the save.
//
// Requires ANTHROPIC_API_KEY, either already exported or present in a
// .env file at the repo root (see shared/dotenv).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/breakdown"
	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/chunk"
	"aidoku/pipeline/internal/db"
	"aidoku/pipeline/internal/question"
	"aidoku/pipeline/internal/types"
	"aidoku/shared/dotenv"
)

// book identifies Pride and Prejudice (a real book, Gutenberg #1342) —
// hardcoded rather than loaded from the catalog, matching this command's
// existing "no dependency on testdata file layout or the
// ingest/clean/segment stages" design, and independent of whatever
// pipeline/books.txt's live catalog currently lists (currently The
// Vampyre — see books.txt). Every field here is a true fact about the
// real book; it's just not sourced from the catalog file.
var book = db.Book{
	GutenbergID: 1342,
	Title:       "Pride and Prejudice",
	Author:      "Jane Austen",
	SourceURL:   "https://www.gutenberg.org/cache/epub/1342/pg1342.txt",
	Level:       catalog.LevelScholar,
}

func main() {
	dotenv.Load(".env")
	dotenv.Load("../.env") // in case run from pipeline/

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("livetest: ANTHROPIC_API_KEY is not set (checked real env, .env, and ../.env)")
	}

	// Six real sentences from the opening of Pride and Prejudice (public
	// domain), hand-entered so this script has no dependency on testdata
	// file layout or the ingest/clean/segment stages.
	rawSentences := []string{
		`It is a truth universally acknowledged, that a single man in possession of a good fortune must be in want of a wife.`,
		`However little known the feelings or views of such a man may be on his first entering a neighbourhood, this truth is so well fixed in the minds of the surrounding families, that he is considered as the rightful property of some one or other of their daughters.`,
		`“My dear Mr. Bennet,” said his lady to him one day, “have you heard that Netherfield Park is let at last?”`,
		`Mr. Bennet replied that he had not.`,
		`“But it is,” returned she; “for Mrs. Long has just been here, and she told me all about it.”`,
		`Mr. Bennet made no answer.`,
	}
	sentences := make([]types.SentenceInput, len(rawSentences))
	for i, text := range rawSentences {
		sentences[i] = types.SentenceInput{Index: i, Text: text, CharCount: len([]rune(text))}
	}

	client := anthropic.NewClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Println("=== Stage B: chunk grouping (1 real API call, claude-sonnet-5) ===")
	grouper := chunk.NewGrouper(client)
	groupingResp, err := grouper.GroupSentencesIntoChunks(ctx, sentences)
	if err != nil {
		log.Fatalf("livetest: chunk grouping: %v", err)
	}
	chunks := chunk.BuildChunks(sentences, groupingResp)
	for _, c := range chunks {
		fmt.Printf("\nchunk %d (%d chars):\n%s\n", c.Index, c.CharCount, c.Text)
	}
	if len(chunks) == 0 {
		log.Fatal("livetest: grouping produced zero chunks — nothing to generate questions for")
	}

	fmt.Println("\n=== Question generation: chunk 0 only (1 real API call, claude-sonnet-5) ===")
	generator := question.NewGenerator(client)
	questions, err := generator.GenerateQuestions(ctx, chunks[0])
	if err != nil {
		log.Fatalf("livetest: question generation: %v", err)
	}
	for _, q := range questions {
		fmt.Printf("\n[%s]\nprompt: %s\noptions: %v\nanswer_index: %d\nexplanation: %s\n",
			q.Type, q.Prompt, q.Options, q.AnswerIndex, q.Explanation)
		if q.Highlight != "" {
			fmt.Printf("highlight: %q\n", q.Highlight)
		}
	}

	fmt.Println("\n=== Breakdown generation: chunk 0 only (1 real API call, claude-sonnet-5) ===")
	breakdownGenerator := breakdown.NewGenerator(client)
	breakdownContent, err := breakdownGenerator.GenerateBreakdown(ctx, chunks[0])
	if err != nil {
		log.Fatalf("livetest: breakdown generation: %v", err)
	}
	fmt.Printf("\n%s\n", breakdownContent)

	fmt.Println("\n=== persisting to Postgres (local write, not an API call) ===")
	pool, err := db.Open(ctx, db.ConnStringFromEnv())
	if err != nil {
		log.Fatalf("livetest: connect to Postgres (is `docker compose up -d` running?): %v", err)
	}
	defer pool.Close()
	store := db.New(pool)

	bookID, err := store.UpsertBook(ctx, book)
	if err != nil {
		log.Fatalf("livetest: save book: %v", err)
	}
	for i, c := range chunks {
		// Only chunk 0 got real questions and a breakdown generated
		// above (see the comment at the top of main) — the rest still
		// get saved, just with no questions or breakdown yet.
		var qs []types.Question
		if i == 0 {
			qs = questions
		}
		chunkID, err := store.SaveChunk(ctx, bookID, c, qs)
		if err != nil {
			log.Fatalf("livetest: save chunk %d: %v", c.Index, err)
		}
		if i == 0 {
			if err := store.SaveBreakdown(ctx, chunkID, breakdownContent); err != nil {
				log.Fatalf("livetest: save breakdown for chunk %d: %v", c.Index, err)
			}
		}
	}
	fmt.Printf("saved book %q (id %d) with %d chunk(s), chunk 0 including its 3 questions and breakdown\n", book.Title, bookID, len(chunks))

	fmt.Println("\n=== done: 3 real API calls made ===")
}
