package db

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// openTestStore connects to the local dev Postgres (docker compose up -d)
// and wraps a fresh transaction in a Store, rolled back when the test
// ends so nothing a test writes persists in the real dev database. The
// raw tx is also returned so tests can seed rows directly via SQL — this
// package's Store is read-only, so there's no Save/Upsert to seed
// through. Same pattern as pipeline/internal/db's openTestStore.
//
// Skips (not fails) if Postgres isn't reachable.
func openTestStore(t *testing.T) (*Store, pgx.Tx, context.Context) {
	t.Helper()
	ctx := context.Background()

	pool, err := Open(ctx, ConnStringFromEnv())
	if err != nil {
		t.Skipf("local Postgres not reachable (is `docker compose up -d` running?): %v", err)
	}
	t.Cleanup(pool.Close)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	return New(tx), tx, ctx
}

// insertBook seeds a minimal book row directly (bypassing this package's
// read-only Store), returning its id. gutenbergID must be unique per
// test — callers pass a distinct constant so seeded rows never collide
// within a single rolled-back transaction.
func insertBook(t *testing.T, tx pgx.Tx, ctx context.Context, gutenbergID int, status string) int {
	t.Helper()
	const q = `
		INSERT INTO book (gutenberg_id, title, author, source_url, level, target_language, native_language, status)
		VALUES ($1, 'Test Book', 'Test Author', 'https://example.com/test.txt', 5, 'en', 'ja', $2)
		RETURNING id`
	var id int
	if err := tx.QueryRow(ctx, q, gutenbergID, status).Scan(&id); err != nil {
		t.Fatalf("seed book: %v", err)
	}
	return id
}

func insertChunk(t *testing.T, tx pgx.Tx, ctx context.Context, bookID, index int, text string) int {
	t.Helper()
	const q = `
		INSERT INTO chunk (book_id, index, text, char_count)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	var id int
	if err := tx.QueryRow(ctx, q, bookID, index, text, len(text)).Scan(&id); err != nil {
		t.Fatalf("seed chunk: %v", err)
	}
	return id
}

func insertQuestion(t *testing.T, tx pgx.Tx, ctx context.Context, chunkID int, qType, highlight string) int {
	t.Helper()
	var h any
	if highlight != "" {
		h = highlight
	}
	const q = `
		INSERT INTO question (chunk_id, type, prompt, options, answer_index, explanation, highlight)
		VALUES ($1, $2, 'prompt', ARRAY['a','b','c','d'], 0, 'explanation', $3)
		RETURNING id`
	var id int
	if err := tx.QueryRow(ctx, q, chunkID, qType, h).Scan(&id); err != nil {
		t.Fatalf("seed question: %v", err)
	}
	return id
}

func insertBreakdown(t *testing.T, tx pgx.Tx, ctx context.Context, chunkID int, content string) int {
	t.Helper()
	const q = `INSERT INTO breakdown (chunk_id, content) VALUES ($1, $2) RETURNING id`
	var id int
	if err := tx.QueryRow(ctx, q, chunkID, content).Scan(&id); err != nil {
		t.Fatalf("seed breakdown: %v", err)
	}
	return id
}

func TestListBooks_OnlyReturnsPublished(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	published := insertBook(t, tx, ctx, 810001, "published")
	insertBook(t, tx, ctx, 810002, "processing")

	books, err := store.ListBooks(ctx)
	if err != nil {
		t.Fatalf("ListBooks: %v", err)
	}
	for _, b := range books {
		if b.Status != "published" {
			t.Errorf("ListBooks returned a book with status %q, want only \"published\"", b.Status)
		}
	}
	var found bool
	for _, b := range books {
		if b.ID == published {
			found = true
		}
	}
	if !found {
		t.Error("ListBooks didn't include the published book that was seeded")
	}
}

func TestGetBook_NotFoundForUnpublished(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	id := insertBook(t, tx, ctx, 810003, "processing")

	if _, err := store.GetBook(ctx, id); err != ErrNotFound {
		t.Errorf("GetBook on a processing book: got %v, want ErrNotFound", err)
	}
}

func TestGetBook_NotFoundForMissingID(t *testing.T) {
	store, _, ctx := openTestStore(t)

	if _, err := store.GetBook(ctx, 99999999); err != ErrNotFound {
		t.Errorf("GetBook on a nonexistent id: got %v, want ErrNotFound", err)
	}
}

func TestGetBook_ReturnsPublishedBook(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	id := insertBook(t, tx, ctx, 810004, "published")

	got, err := store.GetBook(ctx, id)
	if err != nil {
		t.Fatalf("GetBook: %v", err)
	}
	if got.ID != id || got.GutenbergID != 810004 || got.Status != "published" {
		t.Errorf("GetBook = %+v, want id=%d gutenberg_id=810004 status=published", got, id)
	}
}

func TestListChunkIDs_OrderedByIndex(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810005, "published")
	// Seed out of index order on purpose — the query must sort by index,
	// not by insertion/id order.
	third := insertChunk(t, tx, ctx, bookID, 2, "third")
	first := insertChunk(t, tx, ctx, bookID, 0, "first")
	second := insertChunk(t, tx, ctx, bookID, 1, "second")

	ids, err := store.ListChunkIDs(ctx, bookID)
	if err != nil {
		t.Fatalf("ListChunkIDs: %v", err)
	}
	want := []int{first, second, third}
	if len(ids) != len(want) {
		t.Fatalf("ListChunkIDs = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("ListChunkIDs[%d] = %d, want %d (chunk index order)", i, ids[i], want[i])
		}
	}
}

func TestListChunkIDs_EmptyForUnpublishedBook(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810006, "processing")
	insertChunk(t, tx, ctx, bookID, 0, "text")

	ids, err := store.ListChunkIDs(ctx, bookID)
	if err != nil {
		t.Fatalf("ListChunkIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ListChunkIDs on an unpublished book = %v, want empty", ids)
	}
}

func TestListChunkSummaries_OrderedByIndexWithPreviews(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810016, "published")
	second := insertChunk(t, tx, ctx, bookID, 1, "Second chunk. More text after.")
	first := insertChunk(t, tx, ctx, bookID, 0, "First chunk. More text after.")

	summaries, err := store.ListChunkSummaries(ctx, bookID)
	if err != nil {
		t.Fatalf("ListChunkSummaries: %v", err)
	}
	want := []int{first, second}
	if len(summaries) != len(want) {
		t.Fatalf("ListChunkSummaries = %+v, want %d summaries", summaries, len(want))
	}
	for i, id := range want {
		if summaries[i].ID != id {
			t.Errorf("ListChunkSummaries[%d].ID = %d, want %d (chunk index order)", i, summaries[i].ID, id)
		}
	}
	if summaries[0].Preview != "First chunk." {
		t.Errorf("ListChunkSummaries[0].Preview = %q, want %q", summaries[0].Preview, "First chunk.")
	}
}

func TestListChunkSummaries_EmptyForUnpublishedBook(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810017, "processing")
	insertChunk(t, tx, ctx, bookID, 0, "text")

	summaries, err := store.ListChunkSummaries(ctx, bookID)
	if err != nil {
		t.Fatalf("ListChunkSummaries: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("ListChunkSummaries on an unpublished book = %v, want empty", summaries)
	}
}

func TestGetChunk_NotFoundForUnpublishedBook(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810009, "processing")
	chunkID := insertChunk(t, tx, ctx, bookID, 0, "text")

	if _, err := store.GetChunk(ctx, chunkID); err != ErrNotFound {
		t.Errorf("GetChunk under an unpublished book: got %v, want ErrNotFound", err)
	}
}

func TestGetChunk_ReturnsFullText(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810010, "published")
	chunkID := insertChunk(t, tx, ctx, bookID, 3, "the chunk's full text")

	got, err := store.GetChunk(ctx, chunkID)
	if err != nil {
		t.Fatalf("GetChunk: %v", err)
	}
	if got.Text != "the chunk's full text" || got.Index != 3 || got.BookID != bookID {
		t.Errorf("GetChunk = %+v, want text/index/book_id to match what was seeded", got)
	}
}

func TestListQuestionIDs_ScopedToChunk(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810011, "published")
	chunk1 := insertChunk(t, tx, ctx, bookID, 0, "chunk one")
	chunk2 := insertChunk(t, tx, ctx, bookID, 1, "chunk two")

	q1 := insertQuestion(t, tx, ctx, chunk1, "vocab", "the")
	insertQuestion(t, tx, ctx, chunk1, "grammar", "chunk")
	insertQuestion(t, tx, ctx, chunk2, "comprehension", "")

	ids, err := store.ListQuestionIDs(ctx, chunk1)
	if err != nil {
		t.Fatalf("ListQuestionIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("ListQuestionIDs(chunk1) = %v, want 2 ids", ids)
	}
	var foundQ1 bool
	for _, id := range ids {
		if id == q1 {
			foundQ1 = true
		}
	}
	if !foundQ1 {
		t.Error("ListQuestionIDs(chunk1) didn't include a question actually seeded under chunk1")
	}
}

func TestGetQuestion_NotFoundForUnpublishedBook(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810012, "processing")
	chunkID := insertChunk(t, tx, ctx, bookID, 0, "chunk")
	qID := insertQuestion(t, tx, ctx, chunkID, "vocab", "chunk")

	if _, err := store.GetQuestion(ctx, qID); err != ErrNotFound {
		t.Errorf("GetQuestion under an unpublished book: got %v, want ErrNotFound", err)
	}
}

func TestGetQuestion_ComprehensionHasNilHighlight(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810013, "published")
	chunkID := insertChunk(t, tx, ctx, bookID, 0, "chunk")
	qID := insertQuestion(t, tx, ctx, chunkID, "comprehension", "")

	got, err := store.GetQuestion(ctx, qID)
	if err != nil {
		t.Fatalf("GetQuestion: %v", err)
	}
	if got.Highlight != nil {
		t.Errorf("comprehension question Highlight = %q, want nil", *got.Highlight)
	}
}

func TestGetBreakdown_ReturnsContent(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810014, "published")
	chunkID := insertChunk(t, tx, ctx, bookID, 0, "chunk")
	insertBreakdown(t, tx, ctx, chunkID, "breakdown content")

	got, err := store.GetBreakdown(ctx, chunkID)
	if err != nil {
		t.Fatalf("GetBreakdown: %v", err)
	}
	if got.Content != "breakdown content" || got.ChunkID != chunkID {
		t.Errorf("GetBreakdown = %+v, want content/chunk_id to match what was seeded", got)
	}
}

func TestGetBreakdown_NotFoundWhenMissing(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID := insertBook(t, tx, ctx, 810015, "published")
	chunkID := insertChunk(t, tx, ctx, bookID, 0, "chunk")
	// No breakdown inserted.

	if _, err := store.GetBreakdown(ctx, chunkID); err != ErrNotFound {
		t.Errorf("GetBreakdown with no breakdown row: got %v, want ErrNotFound", err)
	}
}

// firstSentencePreview doesn't touch Postgres — plain unit tests, no
// openTestStore/skip needed.
func TestFirstSentencePreview(t *testing.T) {
	longWordRun := strings.Repeat("a", 200) // no spaces, no punctuation

	cases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "short first sentence returned whole, including punctuation",
			text: "It was a dark night. Something else happened next.",
			want: "It was a dark night.",
		},
		{
			name: "no punctuation at all falls back to a word-boundary truncation",
			// 100 a's + a space + 49 b's = 150 chars, no . ! ? anywhere.
			// Truncating to 140 chars lands inside the run of b's; the
			// word-boundary rule pulls it back to the space at index 100.
			text: strings.Repeat("a", 100) + " " + strings.Repeat("b", 49),
			want: strings.Repeat("a", 100) + "…",
		},
		{
			name: "first sentence longer than maxLen falls back to truncation, not a huge preview",
			text: longWordRun + ".",
			want: longWordRun[:140] + "…",
		},
		{
			name: "text shorter than maxLen with no punctuation is returned whole",
			text: "short text",
			want: "short text",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := firstSentencePreview(tc.text, 140)
			if got != tc.want {
				t.Errorf("firstSentencePreview(%.20q..., 140) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}
