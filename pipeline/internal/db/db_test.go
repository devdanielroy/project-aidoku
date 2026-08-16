package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/types"
)

// openTestStore connects to the local dev Postgres (docker compose up -d)
// and wraps a fresh transaction in a Store, rolled back when the test
// ends so nothing a test writes persists in the real dev database. The
// raw tx is also returned so tests can verify what Store wrote via
// direct queries — Store's own public surface is deliberately just the
// Save/Upsert operations pipeline stages need, not a general query API.
//
// Skips (not fails) if Postgres isn't reachable — these are integration
// tests against a real database, not something every `go test ./...`
// run should require Docker for.
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

func testBook(gutenbergID int) Book {
	return Book{
		GutenbergID:    gutenbergID,
		Title:          "Test Book",
		Author:         "Test Author",
		SourceURL:      "https://example.com/test.txt",
		Level:          catalog.LevelBookworm,
		TargetLanguage: langpair.EN_JP.Target,
		NativeLanguage: langpair.EN_JP.Native,
	}
}

// TestNewBookFromEntry doesn't need Postgres — it's a pure conversion —
// so it isn't behind openTestStore's skip-if-unreachable guard.
func TestNewBookFromEntry(t *testing.T) {
	entry := catalog.Entry{
		Title:       "The Vampyre",
		Author:      "John William Polidori",
		GutenbergID: 6087,
		SourceURL:   "https://www.gutenberg.org/cache/epub/6087/pg6087.txt",
		FirstLine:   "IT happened that in the midst of the dissipations...",
		LastLine:    "...glutted the thirst of a VAMPYRE!",
		Level:       catalog.LevelScholar,
	}

	got := NewBookFromEntry(entry, langpair.JP_EN)
	want := Book{
		Title:          "The Vampyre",
		Author:         "John William Polidori",
		GutenbergID:    6087,
		SourceURL:      "https://www.gutenberg.org/cache/epub/6087/pg6087.txt",
		Level:          catalog.LevelScholar,
		TargetLanguage: langpair.JP_EN.Target,
		NativeLanguage: langpair.JP_EN.Native,
	}
	if got != want {
		t.Errorf("NewBookFromEntry(%+v) = %+v, want %+v", entry, got, want)
	}
}

func TestUpsertBook_RejectsMissingLanguagePair(t *testing.T) {
	store, _, ctx := openTestStore(t)

	b := testBook(900008)
	b.TargetLanguage = ""
	if _, err := store.UpsertBook(ctx, b); err == nil {
		t.Fatal("UpsertBook with empty TargetLanguage = nil error, want an error")
	}

	b = testBook(900009)
	b.NativeLanguage = ""
	if _, err := store.UpsertBook(ctx, b); err == nil {
		t.Fatal("UpsertBook with empty NativeLanguage = nil error, want an error")
	}
}

func testQuestions() []types.Question {
	return []types.Question{
		{Type: types.QuestionTypeVocab, Prompt: "vocab prompt", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 0, Explanation: "vocab explanation", Highlight: "want"},
		{Type: types.QuestionTypeGrammar, Prompt: "grammar prompt", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 1, Explanation: "grammar explanation", Highlight: "It is a truth"},
		{Type: types.QuestionTypeComprehension, Prompt: "comprehension prompt", Options: []string{"a", "b", "c", "d"}, AnswerIndex: 2, Explanation: "comprehension explanation"},
	}
}

func TestUpsertBook_InsertAndUpdate(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	id, err := store.UpsertBook(ctx, testBook(900001))
	if err != nil {
		t.Fatalf("UpsertBook (insert): %v", err)
	}
	if id == 0 {
		t.Fatal("UpsertBook returned id 0")
	}

	updated := testBook(900001)
	updated.Title = "Updated Title"
	updated.Level = catalog.LevelScholar
	gotID, err := store.UpsertBook(ctx, updated)
	if err != nil {
		t.Fatalf("UpsertBook (update): %v", err)
	}
	if gotID != id {
		t.Errorf("UpsertBook (update) returned id %d, want the same id %d as the insert", gotID, id)
	}

	var title string
	var level int
	if err := tx.QueryRow(ctx, "SELECT title, level FROM book WHERE id = $1", id).Scan(&title, &level); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if title != "Updated Title" || level != int(catalog.LevelScholar) {
		t.Errorf("after update: title=%q level=%d, want %q %d", title, level, "Updated Title", int(catalog.LevelScholar))
	}
}

func TestUpsertBook_RejectsInvalidLevel(t *testing.T) {
	store, _, ctx := openTestStore(t)

	b := testBook(900002)
	b.Level = 0
	if _, err := store.UpsertBook(ctx, b); err == nil {
		t.Fatal("UpsertBook with level 0 = nil error, want an error")
	}

	b.Level = 11
	if _, err := store.UpsertBook(ctx, b); err == nil {
		t.Fatal("UpsertBook with level 11 = nil error, want an error")
	}
}

func TestSaveChunk_InsertsChunkAndQuestions(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID, err := store.UpsertBook(ctx, testBook(900003))
	if err != nil {
		t.Fatalf("UpsertBook: %v", err)
	}

	chunk := types.Chunk{Index: 0, Text: "It is a truth universally acknowledged, that a single man in possession of a good fortune must be in want of a wife.", CharCount: 116}
	chunkID, err := store.SaveChunk(ctx, bookID, chunk, testQuestions())
	if err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if chunkID == 0 {
		t.Fatal("SaveChunk returned chunk id 0")
	}

	var text string
	if err := tx.QueryRow(ctx, "SELECT text FROM chunk WHERE id = $1", chunkID).Scan(&text); err != nil {
		t.Fatalf("verify chunk: %v", err)
	}
	if text != chunk.Text {
		t.Errorf("chunk text = %q, want %q", text, chunk.Text)
	}

	var questionCount int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM question WHERE chunk_id = $1", chunkID).Scan(&questionCount); err != nil {
		t.Fatalf("verify questions: %v", err)
	}
	if questionCount != 3 {
		t.Errorf("question count = %d, want 3", questionCount)
	}

	var comprehensionHighlight *string
	if err := tx.QueryRow(ctx, "SELECT highlight FROM question WHERE chunk_id = $1 AND type = 'comprehension'", chunkID).Scan(&comprehensionHighlight); err != nil {
		t.Fatalf("verify comprehension highlight: %v", err)
	}
	if comprehensionHighlight != nil {
		t.Errorf("comprehension question highlight = %q, want NULL", *comprehensionHighlight)
	}
}

func TestSaveChunk_UpsertReplacesQuestionsWithoutDuplicating(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID, err := store.UpsertBook(ctx, testBook(900004))
	if err != nil {
		t.Fatalf("UpsertBook: %v", err)
	}
	chunk := types.Chunk{Index: 0, Text: "Some chunk text.", CharCount: 17}

	chunkID, err := store.SaveChunk(ctx, bookID, chunk, testQuestions())
	if err != nil {
		t.Fatalf("SaveChunk (first): %v", err)
	}

	revised := testQuestions()
	revised[0].Prompt = "revised vocab prompt"
	gotChunkID, err := store.SaveChunk(ctx, bookID, chunk, revised)
	if err != nil {
		t.Fatalf("SaveChunk (second): %v", err)
	}
	if gotChunkID != chunkID {
		t.Errorf("SaveChunk (second) returned chunk id %d, want the same id %d", gotChunkID, chunkID)
	}

	var questionCount int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM question WHERE chunk_id = $1", chunkID).Scan(&questionCount); err != nil {
		t.Fatalf("verify question count: %v", err)
	}
	if questionCount != 3 {
		t.Errorf("question count after re-save = %d, want 3 (re-saving should replace, not duplicate)", questionCount)
	}

	var prompt string
	if err := tx.QueryRow(ctx, "SELECT prompt FROM question WHERE chunk_id = $1 AND type = 'vocab'", chunkID).Scan(&prompt); err != nil {
		t.Fatalf("verify vocab prompt: %v", err)
	}
	if prompt != "revised vocab prompt" {
		t.Errorf("vocab prompt = %q, want the revised prompt", prompt)
	}
}

// TestSaveChunk_DBRejectsHighlightlessVocabQuestion confirms
// db/schema.sql's CHECK constraint actually does the job SaveChunk's
// doc comment says it relies on, rather than trusting the schema was
// written correctly and never verifying it.
func TestSaveChunk_DBRejectsHighlightlessVocabQuestion(t *testing.T) {
	store, _, ctx := openTestStore(t)

	bookID, err := store.UpsertBook(ctx, testBook(900005))
	if err != nil {
		t.Fatalf("UpsertBook: %v", err)
	}
	chunk := types.Chunk{Index: 0, Text: "Some chunk text.", CharCount: 17}

	badQuestions := testQuestions()
	badQuestions[0].Highlight = "" // vocab question with no highlight — invalid

	if _, err := store.SaveChunk(ctx, bookID, chunk, badQuestions); err == nil {
		t.Fatal("SaveChunk with a highlight-less vocab question = nil error, want the DB's CHECK constraint to reject it")
	}
}

func TestSaveBreakdown_InsertAndUpdate(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID, err := store.UpsertBook(ctx, testBook(900006))
	if err != nil {
		t.Fatalf("UpsertBook: %v", err)
	}
	chunk := types.Chunk{Index: 0, Text: "Some chunk text.", CharCount: 17}
	chunkID, err := store.SaveChunk(ctx, bookID, chunk, testQuestions())
	if err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}

	if err := store.SaveBreakdown(ctx, chunkID, "first breakdown content"); err != nil {
		t.Fatalf("SaveBreakdown (insert): %v", err)
	}
	if err := store.SaveBreakdown(ctx, chunkID, "revised breakdown content"); err != nil {
		t.Fatalf("SaveBreakdown (update): %v", err)
	}

	var content string
	var count int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM breakdown WHERE chunk_id = $1", chunkID).Scan(&count); err != nil {
		t.Fatalf("verify breakdown count: %v", err)
	}
	if count != 1 {
		t.Errorf("breakdown row count = %d, want 1 (re-saving should replace, not duplicate)", count)
	}
	if err := tx.QueryRow(ctx, "SELECT content FROM breakdown WHERE chunk_id = $1", chunkID).Scan(&content); err != nil {
		t.Fatalf("verify breakdown content: %v", err)
	}
	if content != "revised breakdown content" {
		t.Errorf("breakdown content = %q, want the revised content", content)
	}
}

// TestDeleteBook_CascadesToChunksQuestionsAndBreakdowns confirms
// db/schema.sql's ON DELETE CASCADE actually cascades, not just that it
// was declared.
func TestDeleteBook_CascadesToChunksQuestionsAndBreakdowns(t *testing.T) {
	store, tx, ctx := openTestStore(t)

	bookID, err := store.UpsertBook(ctx, testBook(900007))
	if err != nil {
		t.Fatalf("UpsertBook: %v", err)
	}
	chunk := types.Chunk{Index: 0, Text: "Some chunk text.", CharCount: 17}
	chunkID, err := store.SaveChunk(ctx, bookID, chunk, testQuestions())
	if err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if err := store.SaveBreakdown(ctx, chunkID, "breakdown content"); err != nil {
		t.Fatalf("SaveBreakdown: %v", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM book WHERE id = $1", bookID); err != nil {
		t.Fatalf("delete book: %v", err)
	}

	for table, query := range map[string]string{
		"chunk":     "SELECT count(*) FROM chunk WHERE id = $1",
		"question":  "SELECT count(*) FROM question WHERE chunk_id = $1",
		"breakdown": "SELECT count(*) FROM breakdown WHERE chunk_id = $1",
	} {
		var count int
		if err := tx.QueryRow(ctx, query, chunkID).Scan(&count); err != nil {
			t.Fatalf("verify %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s rows remaining after deleting the book = %d, want 0 (ON DELETE CASCADE should have removed them)", table, count)
		}
	}
}
