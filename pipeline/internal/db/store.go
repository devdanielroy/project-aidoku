package db

import (
	"context"
	"fmt"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/types"
)

// Book is the row-level shape UpsertBook writes. Its own type rather
// than reusing catalog.Entry directly, since not every field lines up
// one-to-one (TargetLanguage/NativeLanguage/Status have no catalog
// equivalent — a catalog file's entries are all one pair, supplied by
// whoever's calling NewBookFromEntry, not per-entry) — but every
// catalog.Entry field a Book needs now has a home in it.
type Book struct {
	GutenbergID int
	Title       string
	Author      string
	SourceURL   string
	Level       catalog.ReadingLevel
	// TargetLanguage/NativeLanguage are ISO 639-1 codes (langpair.
	// LanguagePair.Target/Native, e.g. "en"/"ja") and required — see
	// db/schema.sql's comment on why there's no default. Status
	// defaults to "processing" if empty (see db/schema.sql's CHECK
	// constraint for the only other allowed value, "published").
	TargetLanguage string
	NativeLanguage string
	Status         string
}

// NewBookFromEntry builds a Book from a catalog.Entry and the pair that
// catalog file is for (every entry in one catalog file shares the same
// pair — see cmd/process's -pair flag) — the normal way to get one, now
// that a catalog file's format carries title and author (see the
// README's "Adding a Book to the Catalog" guide). Status is left at its
// zero value; UpsertBook defaults it to "processing".
func NewBookFromEntry(e catalog.Entry, pair langpair.LanguagePair) Book {
	return Book{
		GutenbergID:    e.GutenbergID,
		Title:          e.Title,
		Author:         e.Author,
		SourceURL:      e.SourceURL,
		Level:          e.Level,
		TargetLanguage: pair.Target,
		NativeLanguage: pair.Native,
	}
}

// UpsertBook inserts b, or updates the existing row if a book with the
// same GutenbergID already exists (re-running the pipeline against the
// same book is expected during development, not an error — see
// db/schema.sql's comment on the unique constraint). Returns the row's
// id either way.
func (s *Store) UpsertBook(ctx context.Context, b Book) (int, error) {
	status := orDefault(b.Status, "processing")

	if !b.Level.Valid() {
		return 0, fmt.Errorf("db: UpsertBook: level %d out of range, want 1-10 inclusive", b.Level)
	}
	if b.TargetLanguage == "" || b.NativeLanguage == "" {
		return 0, fmt.Errorf("db: UpsertBook: TargetLanguage and NativeLanguage are both required (see langpair.ByCode)")
	}

	const q = `
		INSERT INTO book (gutenberg_id, title, author, source_url, level, target_language, native_language, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (gutenberg_id) DO UPDATE SET
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			source_url = EXCLUDED.source_url,
			level = EXCLUDED.level,
			target_language = EXCLUDED.target_language,
			native_language = EXCLUDED.native_language,
			status = EXCLUDED.status
		RETURNING id`

	var id int
	err := s.db.QueryRow(ctx, q, b.GutenbergID, b.Title, b.Author, b.SourceURL, int(b.Level), b.TargetLanguage, b.NativeLanguage, status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: UpsertBook: %w", err)
	}
	return id, nil
}

// SaveChunk upserts chunk under bookID and replaces its questions with
// questions, atomically (both the chunk row and all question rows
// commit together, or neither does — a chunk with only some of its
// questions written isn't a state anything downstream should ever see).
// questions is expected to already be question.ValidateQuestionSet's
// output — exactly one vocab, one grammar, one comprehension question —
// SaveChunk relies on db/schema.sql's constraints (not its own
// re-validation) to reject anything that isn't.
//
// Upserts on (book_id, index) for the chunk and (chunk_id, type) for
// each question, same reasoning as UpsertBook: re-running Stage B or
// question generation against the same chunk during development should
// overwrite, not fail.
func (s *Store) SaveChunk(ctx context.Context, bookID int, chunk types.Chunk, questions []types.Question) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("db: SaveChunk: begin: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	const chunkQ = `
		INSERT INTO chunk (book_id, index, text, char_count)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (book_id, index) DO UPDATE SET
			text = EXCLUDED.text,
			char_count = EXCLUDED.char_count
		RETURNING id`

	var chunkID int
	if err := tx.QueryRow(ctx, chunkQ, bookID, chunk.Index, chunk.Text, chunk.CharCount).Scan(&chunkID); err != nil {
		return 0, fmt.Errorf("db: SaveChunk: upsert chunk: %w", err)
	}

	const questionQ = `
		INSERT INTO question (chunk_id, type, prompt, options, answer_index, explanation, highlight)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chunk_id, type) DO UPDATE SET
			prompt = EXCLUDED.prompt,
			options = EXCLUDED.options,
			answer_index = EXCLUDED.answer_index,
			explanation = EXCLUDED.explanation,
			highlight = EXCLUDED.highlight`

	for _, q := range questions {
		// Comprehension questions carry no highlight (types.Question
		// leaves it "" via omitempty) — the column is nullable and
		// db/schema.sql's CHECK requires NULL there, so an empty string
		// must become a real NULL, not an empty-string value.
		var highlight *string
		if q.Highlight != "" {
			highlight = &q.Highlight
		}
		if _, err := tx.Exec(ctx, questionQ, chunkID, string(q.Type), q.Prompt, q.Options, q.AnswerIndex, q.Explanation, highlight); err != nil {
			return 0, fmt.Errorf("db: SaveChunk: upsert %s question: %w", q.Type, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("db: SaveChunk: commit: %w", err)
	}
	return chunkID, nil
}

// SaveBreakdown upserts chunkID's breakdown content (see
// db/schema.sql — one-to-one with chunk). Not called by anything yet:
// breakdown generation (pipeline stage 4.3) isn't built. Included now so
// the db package's write surface matches the schema in full.
func (s *Store) SaveBreakdown(ctx context.Context, chunkID int, content string) error {
	const q = `
		INSERT INTO breakdown (chunk_id, content)
		VALUES ($1, $2)
		ON CONFLICT (chunk_id) DO UPDATE SET content = EXCLUDED.content`

	if _, err := s.db.Exec(ctx, q, chunkID, content); err != nil {
		return fmt.Errorf("db: SaveBreakdown: %w", err)
	}
	return nil
}

func orDefault(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}
