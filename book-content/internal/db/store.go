package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned by any lookup that finds nothing — either the
// row genuinely doesn't exist, or its book isn't published yet.
// Handlers map this straight to an HTTP 404, deliberately not
// distinguishing "doesn't exist" from "exists but isn't published" —
// nothing outside this module should be able to tell those apart (see
// the package doc comment).
var ErrNotFound = errors.New("db: not found")

// Book is the read-shaped response for a book. A deliberately separate
// type from pipeline's db.Book (that one's write-shaped, keyed by
// GutenbergID for upserts) — this one is what the API actually returns.
type Book struct {
	ID          int    `json:"id"`
	GutenbergID int    `json:"gutenberg_id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	SourceURL   string `json:"source_url"`
	Level       int    `json:"level"`
	Language    string `json:"language"`
	Status      string `json:"status"`
}

// Chunk is the read-shaped response for one chunk, full text included —
// only returned by the single-chunk endpoint, not the list-of-chunks
// endpoint (which returns bare IDs; see ListChunkIDs).
type Chunk struct {
	ID        int    `json:"id"`
	BookID    int    `json:"book_id"`
	Index     int    `json:"index"`
	Text      string `json:"text"`
	CharCount int    `json:"char_count"`
}

// Question is the read-shaped response for one question. Highlight is a
// pointer because it's genuinely NULL for comprehension questions (see
// db/schema.sql's CHECK constraint) — omitted from the JSON entirely
// rather than serialized as "".
type Question struct {
	ID          int      `json:"id"`
	ChunkID     int      `json:"chunk_id"`
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	AnswerIndex int      `json:"answer_index"`
	Explanation string   `json:"explanation"`
	Highlight   *string  `json:"highlight,omitempty"`
}

// Breakdown is the read-shaped response for a chunk's breakdown.
type Breakdown struct {
	ID      int    `json:"id"`
	ChunkID int    `json:"chunk_id"`
	Content string `json:"content"`
}

// ListBooks returns every published book, ordered by id. Books still in
// "processing" status (the default — see db/schema.sql) never appear
// here; nothing reaches this API before the still-manual QA/publish
// steps flip that status (AIDOKU_DESIGN.md §3, stage 5/6).
func (s *Store) ListBooks(ctx context.Context) ([]Book, error) {
	const q = `
		SELECT id, gutenberg_id, title, author, source_url, level, language, status
		FROM book
		WHERE status = 'published'
		ORDER BY id`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("db: ListBooks: %w", err)
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.GutenbergID, &b.Title, &b.Author, &b.SourceURL, &b.Level, &b.Language, &b.Status); err != nil {
			return nil, fmt.Errorf("db: ListBooks: scan: %w", err)
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: ListBooks: %w", err)
	}
	return books, nil
}

// GetBook returns the published book with the given id, or ErrNotFound
// if it doesn't exist or isn't published.
func (s *Store) GetBook(ctx context.Context, bookID int) (Book, error) {
	const q = `
		SELECT id, gutenberg_id, title, author, source_url, level, language, status
		FROM book
		WHERE id = $1 AND status = 'published'`

	var b Book
	err := s.db.QueryRow(ctx, q, bookID).Scan(&b.ID, &b.GutenbergID, &b.Title, &b.Author, &b.SourceURL, &b.Level, &b.Language, &b.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Book{}, ErrNotFound
	}
	if err != nil {
		return Book{}, fmt.Errorf("db: GetBook: %w", err)
	}
	return b, nil
}

// ListChunkIDs returns the ids of every chunk in bookID, in reading
// order — bare ids, not full chunk objects (see the Chunk doc comment
// for why: fetching one book's worth of chunk text in a single response
// isn't the shape the reading flow needs). Empty (not ErrNotFound) if
// bookID exists but isn't published or has no chunks yet — callers that
// need to distinguish those should GetBook first.
func (s *Store) ListChunkIDs(ctx context.Context, bookID int) ([]int, error) {
	const q = `
		SELECT c.id
		FROM chunk c
		JOIN book b ON b.id = c.book_id
		WHERE c.book_id = $1 AND b.status = 'published'
		ORDER BY c.index`

	rows, err := s.db.Query(ctx, q, bookID)
	if err != nil {
		return nil, fmt.Errorf("db: ListChunkIDs: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("db: ListChunkIDs: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: ListChunkIDs: %w", err)
	}
	return ids, nil
}

// GetChunk returns chunkID's full content, as long as its book is
// published. chunk.id is a global surrogate key (SERIAL PRIMARY KEY,
// not scoped per book), so no book id is needed to address it — a
// client that already has chunkID (from ListChunkIDs) doesn't need to
// carry its book_id around too. See AIDOKU_DESIGN.md's backend handoff
// notes for why the routes above this method dropped book_id from the
// URL entirely.
func (s *Store) GetChunk(ctx context.Context, chunkID int) (Chunk, error) {
	const q = `
		SELECT c.id, c.book_id, c.index, c.text, c.char_count
		FROM chunk c
		JOIN book b ON b.id = c.book_id
		WHERE c.id = $1 AND b.status = 'published'`

	var c Chunk
	err := s.db.QueryRow(ctx, q, chunkID).Scan(&c.ID, &c.BookID, &c.Index, &c.Text, &c.CharCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Chunk{}, ErrNotFound
	}
	if err != nil {
		return Chunk{}, fmt.Errorf("db: GetChunk: %w", err)
	}
	return c, nil
}

// ListQuestionIDs returns the ids of chunkID's questions (up to 3 — see
// db/schema.sql's UNIQUE(chunk_id, type)), same ids-only shape as
// ListChunkIDs and for the same reason: the client fetches each
// question's full content only once the user is actually answering it.
func (s *Store) ListQuestionIDs(ctx context.Context, chunkID int) ([]int, error) {
	const q = `
		SELECT q.id
		FROM question q
		JOIN chunk c ON c.id = q.chunk_id
		JOIN book b ON b.id = c.book_id
		WHERE q.chunk_id = $1 AND b.status = 'published'
		ORDER BY q.id`

	rows, err := s.db.Query(ctx, q, chunkID)
	if err != nil {
		return nil, fmt.Errorf("db: ListQuestionIDs: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("db: ListQuestionIDs: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: ListQuestionIDs: %w", err)
	}
	return ids, nil
}

// GetQuestion returns questionID's full content, as long as its book is
// published — question.id is a global surrogate key too, same reasoning
// as GetChunk.
func (s *Store) GetQuestion(ctx context.Context, questionID int) (Question, error) {
	const q = `
		SELECT q.id, q.chunk_id, q.type, q.prompt, q.options, q.answer_index, q.explanation, q.highlight
		FROM question q
		JOIN chunk c ON c.id = q.chunk_id
		JOIN book b ON b.id = c.book_id
		WHERE q.id = $1 AND b.status = 'published'`

	var question Question
	err := s.db.QueryRow(ctx, q, questionID).Scan(
		&question.ID, &question.ChunkID, &question.Type, &question.Prompt,
		&question.Options, &question.AnswerIndex, &question.Explanation, &question.Highlight)
	if errors.Is(err, pgx.ErrNoRows) {
		return Question{}, ErrNotFound
	}
	if err != nil {
		return Question{}, fmt.Errorf("db: GetQuestion: %w", err)
	}
	return question, nil
}

// GetBreakdown returns chunkID's breakdown, as long as its book is
// published. Breakdown is 1:1 with chunk (db/schema.sql's UNIQUE
// chunk_id), so there's no list variant.
func (s *Store) GetBreakdown(ctx context.Context, chunkID int) (Breakdown, error) {
	const q = `
		SELECT br.id, br.chunk_id, br.content
		FROM breakdown br
		JOIN chunk c ON c.id = br.chunk_id
		JOIN book b ON b.id = c.book_id
		WHERE br.chunk_id = $1 AND b.status = 'published'`

	var br Breakdown
	err := s.db.QueryRow(ctx, q, chunkID).Scan(&br.ID, &br.ChunkID, &br.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return Breakdown{}, ErrNotFound
	}
	if err != nil {
		return Breakdown{}, fmt.Errorf("db: GetBreakdown: %w", err)
	}
	return br, nil
}

// compile-time check that pgx.Tx satisfies conn.
var _ conn = (pgx.Tx)(nil)
