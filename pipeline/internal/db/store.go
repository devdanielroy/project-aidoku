package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

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

	// Genres/Summary are carried straight through from catalog.Entry —
	// both hand-curated per book, never derived by the pipeline, same
	// as Title/Author. Genres stays the catalog's already-validated,
	// ", "-joined TEXT form end to end (see catalog.Entry.Genres's own
	// doc comment for why a raw comma-separated string rather than a
	// Postgres array); Summary is verbatim.
	Genres  string
	Summary string

	// Image/ImageContentType are the book's cover, downloaded from the
	// catalog entry's ImageURL (see cmd/process, internal/ingest.Client.
	// FetchImage) — both empty/nil when no cover was downloaded, whether
	// because the catalog entry had no ImageURL or the download
	// soft-failed (see catalog.Entry.ImageURL's own doc comment). Not
	// set by NewBookFromEntry itself, which is a pure conversion with no
	// I/O — the caller downloads the image separately and attaches it
	// here before calling UpsertBook.
	Image            []byte
	ImageContentType string
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
		Genres:         e.Genres,
		Summary:        e.Summary,
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
	if b.Genres == "" {
		return 0, fmt.Errorf("db: UpsertBook: Genres is required (see catalog.Entry.Genres)")
	}
	if b.Summary == "" {
		return 0, fmt.Errorf("db: UpsertBook: Summary is required (see catalog.Entry.Summary)")
	}

	// nil/nil (not "", which pgx would happily write as an empty, non-
	// NULL bytea) when there's no image to store, so the COALESCE below
	// can tell "this run has no image" apart from "this run has an
	// image that happens to be empty" (which should never occur, but a
	// pointer distinguishing absence is the correct type either way).
	var image []byte
	var imageContentType *string
	if len(b.Image) > 0 && b.ImageContentType != "" {
		image = b.Image
		imageContentType = &b.ImageContentType
	}

	const q = `
		INSERT INTO book (gutenberg_id, title, author, source_url, level, target_language, native_language, status, book_image, book_image_content_type, genres, summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (gutenberg_id) DO UPDATE SET
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			source_url = EXCLUDED.source_url,
			level = EXCLUDED.level,
			target_language = EXCLUDED.target_language,
			native_language = EXCLUDED.native_language,
			status = EXCLUDED.status,
			-- A soft-failed download on a re-run (EXCLUDED.book_image
			-- NULL) must not wipe out a cover a previous run already
			-- stored successfully — keep the existing row's image
			-- unless this run actually has a new one.
			book_image = COALESCE(EXCLUDED.book_image, book.book_image),
			book_image_content_type = COALESCE(EXCLUDED.book_image_content_type, book.book_image_content_type),
			-- Genres/summary are always supplied (validated above), so
			-- unlike the cover these overwrite unconditionally, same as
			-- title/author.
			genres = EXCLUDED.genres,
			summary = EXCLUDED.summary
		RETURNING id`

	var id int
	err := s.db.QueryRow(ctx, q, b.GutenbergID, b.Title, b.Author, b.SourceURL, int(b.Level), b.TargetLanguage, b.NativeLanguage, status, image, imageContentType, b.Genres, b.Summary).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: UpsertBook: %w", err)
	}
	return id, nil
}

// GetBookIDByGutenbergID looks up an already-persisted book's id by its
// Gutenberg ID — used to resume a previously interrupted/failed
// cmd/process run against a book that's already partway through the
// pipeline, without re-fetching, re-cleaning, or re-grouping it from
// scratch. found is false, with a nil error, when no such book exists
// yet — that's the ordinary "processing this book for the first time"
// case, not a failure.
func (s *Store) GetBookIDByGutenbergID(ctx context.Context, gutenbergID int) (id int, found bool, err error) {
	const q = `SELECT id FROM book WHERE gutenberg_id = $1`

	err = s.db.QueryRow(ctx, q, gutenbergID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("db: GetBookIDByGutenbergID: %w", err)
	}
	return id, true, nil
}

// ChunkProgress is one already-persisted chunk plus how much of the
// pipeline has already run for it — see LoadChunkProgress.
type ChunkProgress struct {
	ID           int
	Chunk        types.Chunk
	HasQuestions bool
	HasBreakdown bool
}

// LoadChunkProgress returns every chunk already persisted for bookID,
// in index order, alongside whether each already has its questions and
// breakdown saved. Used to resume a previously interrupted/failed
// cmd/process run against chunks Stage B already grouped (a paid API
// call, already spent) — skipping straight to whichever of
// questions/breakdown a given chunk is still missing, rather than
// re-running every stage and re-spending API calls on work already
// paid for. See the pipeline-incremental-persistence design this
// builds on.
//
// HasQuestions/HasBreakdown are existence checks, not counts: SaveChunk
// writes a chunk's full question set (all three types) in one
// transaction, so a chunk either has none or all three — there's no
// partial state to distinguish between "has at least one" and "has all
// three" here.
func (s *Store) LoadChunkProgress(ctx context.Context, bookID int) ([]ChunkProgress, error) {
	const q = `
		SELECT
			c.id, c.index, c.text, c.char_count,
			EXISTS (SELECT 1 FROM question q WHERE q.chunk_id = c.id) AS has_questions,
			EXISTS (SELECT 1 FROM breakdown b WHERE b.chunk_id = c.id) AS has_breakdown
		FROM chunk c
		WHERE c.book_id = $1
		ORDER BY c.index`

	rows, err := s.db.Query(ctx, q, bookID)
	if err != nil {
		return nil, fmt.Errorf("db: LoadChunkProgress: %w", err)
	}
	defer rows.Close()

	var progress []ChunkProgress
	for rows.Next() {
		var p ChunkProgress
		if err := rows.Scan(&p.ID, &p.Chunk.Index, &p.Chunk.Text, &p.Chunk.CharCount, &p.HasQuestions, &p.HasBreakdown); err != nil {
			return nil, fmt.Errorf("db: LoadChunkProgress: scan: %w", err)
		}
		progress = append(progress, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: LoadChunkProgress: %w", err)
	}
	return progress, nil
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

// SaveBreakdown upserts chunkID's breakdown content (see db/schema.sql
// — one-to-one with chunk).
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
