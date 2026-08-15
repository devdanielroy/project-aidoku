// Package api is the HTTP layer: routes, request parsing, and response
// encoding for the endpoints the Flutter app calls. All read-only for
// now — no endpoint here writes anything. Answer submission and
// UserProgress read/write endpoints are a separate, not-yet-scoped
// piece of work (see README.md's Milestones "Planned features").
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"aidoku/book-content/internal/db"
)

// Store is the subset of *db.Store the HTTP handlers need — a
// consumer-defined interface (same pattern as pipeline's llmCaller/conn)
// so handlers can be tested against a fake without a real Postgres
// connection.
//
// Chunk/question lookups take only their own id, not a book_id too —
// chunk.id and question.id are global surrogate keys (see
// db/schema.sql), so a client that already has one (from ListChunkIDs /
// ListQuestionIDs) doesn't need to carry its parent book_id around as
// well. See store.go's GetChunk/GetQuestion/GetBreakdown doc comments.
type Store interface {
	ListBooks(ctx context.Context) ([]db.Book, error)
	GetBook(ctx context.Context, bookID int) (db.Book, error)
	ListChunkIDs(ctx context.Context, bookID int) ([]int, error)
	ListChunkSummaries(ctx context.Context, bookID int) ([]db.ChunkSummary, error)
	GetChunk(ctx context.Context, chunkID int) (db.Chunk, error)
	ListQuestionIDs(ctx context.Context, chunkID int) ([]int, error)
	GetQuestion(ctx context.Context, questionID int) (db.Question, error)
	GetBreakdown(ctx context.Context, chunkID int) (db.Breakdown, error)
}

// NewRouter builds the full set of routes against store. Uses the
// stdlib net/http ServeMux's method+wildcard routing (Go 1.22+, this
// module's on 1.26) rather than a third-party router — no dependency
// pulls its weight for a handful of GET routes this simple.
func NewRouter(store Store) *http.ServeMux {
	h := &handler{store: store}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.health)

	mux.HandleFunc("GET /aidoku/books", h.listBooks)
	mux.HandleFunc("GET /aidoku/book/{book_id}", h.getBook)
	mux.HandleFunc("GET /aidoku/book/{book_id}/chunks", h.listChunkIDs)
	mux.HandleFunc("GET /aidoku/book/{book_id}/chunks/summary", h.listChunkSummaries)
	mux.HandleFunc("GET /aidoku/chunk/{chunk_id}", h.getChunk)
	mux.HandleFunc("GET /aidoku/chunk/{chunk_id}/questions", h.listQuestionIDs)
	mux.HandleFunc("GET /aidoku/question/{question_id}", h.getQuestion)
	mux.HandleFunc("GET /aidoku/chunk/{chunk_id}/breakdown", h.getBreakdown)

	return mux
}

type handler struct {
	store Store
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) listBooks(w http.ResponseWriter, r *http.Request) {
	books, err := h.store.ListBooks(r.Context())
	if err != nil {
		writeServerError(w, "listBooks", err)
		return
	}
	if books == nil {
		books = []db.Book{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"books": books})
}

func (h *handler) getBook(w http.ResponseWriter, r *http.Request) {
	bookID, ok := pathInt(w, r, "book_id")
	if !ok {
		return
	}
	book, err := h.store.GetBook(r.Context(), bookID)
	if !handleLookupErr(w, "getBook", err) {
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (h *handler) listChunkIDs(w http.ResponseWriter, r *http.Request) {
	bookID, ok := pathInt(w, r, "book_id")
	if !ok {
		return
	}
	ids, err := h.store.ListChunkIDs(r.Context(), bookID)
	if err != nil {
		writeServerError(w, "listChunkIDs", err)
		return
	}
	if ids == nil {
		ids = []int{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"chunk_ids": ids})
}

func (h *handler) listChunkSummaries(w http.ResponseWriter, r *http.Request) {
	bookID, ok := pathInt(w, r, "book_id")
	if !ok {
		return
	}
	summaries, err := h.store.ListChunkSummaries(r.Context(), bookID)
	if err != nil {
		writeServerError(w, "listChunkSummaries", err)
		return
	}
	if summaries == nil {
		summaries = []db.ChunkSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"chunks": summaries})
}

func (h *handler) getChunk(w http.ResponseWriter, r *http.Request) {
	chunkID, ok := pathInt(w, r, "chunk_id")
	if !ok {
		return
	}
	chunk, err := h.store.GetChunk(r.Context(), chunkID)
	if !handleLookupErr(w, "getChunk", err) {
		return
	}
	writeJSON(w, http.StatusOK, chunk)
}

func (h *handler) listQuestionIDs(w http.ResponseWriter, r *http.Request) {
	chunkID, ok := pathInt(w, r, "chunk_id")
	if !ok {
		return
	}
	ids, err := h.store.ListQuestionIDs(r.Context(), chunkID)
	if err != nil {
		writeServerError(w, "listQuestionIDs", err)
		return
	}
	if ids == nil {
		ids = []int{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"question_ids": ids})
}

func (h *handler) getQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, ok := pathInt(w, r, "question_id")
	if !ok {
		return
	}
	question, err := h.store.GetQuestion(r.Context(), questionID)
	if !handleLookupErr(w, "getQuestion", err) {
		return
	}
	writeJSON(w, http.StatusOK, question)
}

func (h *handler) getBreakdown(w http.ResponseWriter, r *http.Request) {
	chunkID, ok := pathInt(w, r, "chunk_id")
	if !ok {
		return
	}
	breakdown, err := h.store.GetBreakdown(r.Context(), chunkID)
	if !handleLookupErr(w, "getBreakdown", err) {
		return
	}
	writeJSON(w, http.StatusOK, breakdown)
}

// pathInt reads and parses an integer path parameter, writing a 400 and
// returning ok=false if it's missing or not a valid integer — callers
// return immediately when ok is false.
func pathInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	v, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, name+" must be an integer")
		return 0, false
	}
	return v, true
}

// handleLookupErr writes the appropriate response for a single-resource
// lookup's error (db.ErrNotFound -> 404, anything else -> 500) and
// reports whether the caller should go on to write its own success
// response.
func handleLookupErr(w http.ResponseWriter, op string, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return false
	}
	writeServerError(w, op, err)
	return false
}

// writeServerError logs the real error server-side (never sent to the
// client — could leak query/schema details) and writes a generic 500.
func writeServerError(w http.ResponseWriter, op string, err error) {
	log.Printf("api: %s: %v", op, err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: write response: %v", err)
	}
}
