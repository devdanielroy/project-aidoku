package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aidoku/book-content/internal/db"
)

// fakeStore lets each test set only the method(s) it actually exercises;
// calling an unset one panics (a nil func call), which fails the test
// loudly rather than silently returning a zero value.
type fakeStore struct {
	listBooks          func(ctx context.Context) ([]db.Book, error)
	getBook            func(ctx context.Context, bookID int) (db.Book, error)
	getBookImage       func(ctx context.Context, bookID int) ([]byte, string, error)
	listChunkIDs       func(ctx context.Context, bookID int) ([]int, error)
	listChunkSummaries func(ctx context.Context, bookID int) ([]db.ChunkSummary, error)
	getChunk           func(ctx context.Context, chunkID int) (db.Chunk, error)
	listQuestionIDs    func(ctx context.Context, chunkID int) ([]int, error)
	getQuestion        func(ctx context.Context, questionID int) (db.Question, error)
	getBreakdown       func(ctx context.Context, chunkID int) (db.Breakdown, error)
}

func (f *fakeStore) ListBooks(ctx context.Context) ([]db.Book, error) { return f.listBooks(ctx) }
func (f *fakeStore) GetBook(ctx context.Context, bookID int) (db.Book, error) {
	return f.getBook(ctx, bookID)
}
func (f *fakeStore) GetBookImage(ctx context.Context, bookID int) ([]byte, string, error) {
	return f.getBookImage(ctx, bookID)
}
func (f *fakeStore) ListChunkIDs(ctx context.Context, bookID int) ([]int, error) {
	return f.listChunkIDs(ctx, bookID)
}
func (f *fakeStore) ListChunkSummaries(ctx context.Context, bookID int) ([]db.ChunkSummary, error) {
	return f.listChunkSummaries(ctx, bookID)
}
func (f *fakeStore) GetChunk(ctx context.Context, chunkID int) (db.Chunk, error) {
	return f.getChunk(ctx, chunkID)
}
func (f *fakeStore) ListQuestionIDs(ctx context.Context, chunkID int) ([]int, error) {
	return f.listQuestionIDs(ctx, chunkID)
}
func (f *fakeStore) GetQuestion(ctx context.Context, questionID int) (db.Question, error) {
	return f.getQuestion(ctx, questionID)
}
func (f *fakeStore) GetBreakdown(ctx context.Context, chunkID int) (db.Breakdown, error) {
	return f.getBreakdown(ctx, chunkID)
}

func doRequest(t *testing.T, mux http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
}

func TestHealth(t *testing.T) {
	mux := NewRouter(&fakeStore{})
	rec := doRequest(t, mux, "GET", "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	decodeJSON(t, rec, &body)
	if body["status"] != "ok" {
		t.Errorf("body = %v, want status=ok", body)
	}
}

func TestListBooks_WrapsInNamedField(t *testing.T) {
	store := &fakeStore{
		listBooks: func(ctx context.Context) ([]db.Book, error) {
			return []db.Book{{ID: 1, Title: "The Vampyre"}}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/books")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Books []db.Book `json:"books"`
	}
	decodeJSON(t, rec, &body)
	if len(body.Books) != 1 || body.Books[0].Title != "The Vampyre" {
		t.Errorf("body.books = %+v, want one book titled The Vampyre", body.Books)
	}
}

func TestListBooks_EmptyIsAnEmptyArrayNotNull(t *testing.T) {
	store := &fakeStore{
		listBooks: func(ctx context.Context) ([]db.Book, error) { return nil, nil },
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/books")

	// A nil Go slice marshals to JSON null, not []  — a client
	// expecting an always-iterable array shouldn't have to special-case
	// this. listBooks (the handler) is responsible for the nil->[]
	// substitution; this test is what actually pins that behavior.
	if got := rec.Body.String(); !jsonHasEmptyArray(got, "books") {
		t.Errorf("body = %s, want books: [] (not null)", got)
	}
}

func jsonHasEmptyArray(body, field string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return false
	}
	raw, ok := m[field]
	if !ok {
		return false
	}
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return false
	}
	return arr != nil && len(arr) == 0
}

func TestListBooks_ServerErrorHidesUnderlyingDetail(t *testing.T) {
	store := &fakeStore{
		listBooks: func(ctx context.Context) ([]db.Book, error) {
			return nil, errors.New("pq: connection reset by peer, password=hunter2")
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/books")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Body.String(); jsonContains(got, "hunter2") {
		t.Errorf("response body leaked the underlying error: %s", got)
	}
}

func jsonContains(body, substr string) bool {
	for i := 0; i+len(substr) <= len(body); i++ {
		if body[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetBook_NotFoundMapsTo404(t *testing.T) {
	store := &fakeStore{
		getBook: func(ctx context.Context, bookID int) (db.Book, error) {
			return db.Book{}, db.ErrNotFound
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/999")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetBook_NonIntegerIDIs400(t *testing.T) {
	rec := doRequest(t, NewRouter(&fakeStore{}), "GET", "/aidoku/book/not-a-number")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetBook_OK(t *testing.T) {
	store := &fakeStore{
		getBook: func(ctx context.Context, bookID int) (db.Book, error) {
			if bookID != 42 {
				t.Errorf("GetBook called with bookID=%d, want 42", bookID)
			}
			return db.Book{ID: 42, Title: "The Vampyre", Status: "published"}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/42")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got db.Book
	decodeJSON(t, rec, &got)
	if got.ID != 42 || got.Title != "The Vampyre" {
		t.Errorf("body = %+v, want id=42 title=The Vampyre", got)
	}
}

func TestGetBookImage_NotFoundMapsTo404(t *testing.T) {
	store := &fakeStore{
		getBookImage: func(ctx context.Context, bookID int) ([]byte, string, error) {
			return nil, "", db.ErrNotFound
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/999/image")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetBookImage_NonIntegerIDIs400(t *testing.T) {
	rec := doRequest(t, NewRouter(&fakeStore{}), "GET", "/aidoku/book/not-a-number/image")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetBookImage_OK(t *testing.T) {
	imageBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	store := &fakeStore{
		getBookImage: func(ctx context.Context, bookID int) ([]byte, string, error) {
			if bookID != 42 {
				t.Errorf("GetBookImage called with bookID=%d, want 42", bookID)
			}
			return imageBytes, "image/jpeg", nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/42/image")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q, want %q", got, "image/jpeg")
	}
	if !bytes.Equal(rec.Body.Bytes(), imageBytes) {
		t.Errorf("body = %v, want %v", rec.Body.Bytes(), imageBytes)
	}
}

func TestListChunkIDs_WrapsInNamedField(t *testing.T) {
	store := &fakeStore{
		listChunkIDs: func(ctx context.Context, bookID int) ([]int, error) {
			if bookID != 7 {
				t.Errorf("ListChunkIDs called with bookID=%d, want 7", bookID)
			}
			return []int{10, 11, 12}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/7/chunks")

	var body struct {
		ChunkIDs []int `json:"chunk_ids"`
	}
	decodeJSON(t, rec, &body)
	if len(body.ChunkIDs) != 3 {
		t.Errorf("body.chunk_ids = %v, want [10 11 12]", body.ChunkIDs)
	}
}

func TestListChunkSummaries_WrapsInNamedField(t *testing.T) {
	store := &fakeStore{
		listChunkSummaries: func(ctx context.Context, bookID int) ([]db.ChunkSummary, error) {
			if bookID != 7 {
				t.Errorf("ListChunkSummaries called with bookID=%d, want 7", bookID)
			}
			return []db.ChunkSummary{
				{ID: 10, Index: 0, Preview: "It was a dark and stormy night."},
			}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/7/chunks/summary")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Chunks []db.ChunkSummary `json:"chunks"`
	}
	decodeJSON(t, rec, &body)
	if len(body.Chunks) != 1 || body.Chunks[0].Preview != "It was a dark and stormy night." {
		t.Errorf("body.chunks = %+v, want one summary with that preview", body.Chunks)
	}
}

func TestListChunkSummaries_EmptyIsAnEmptyArrayNotNull(t *testing.T) {
	store := &fakeStore{
		listChunkSummaries: func(ctx context.Context, bookID int) ([]db.ChunkSummary, error) {
			return nil, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/book/7/chunks/summary")

	if got := rec.Body.String(); !jsonHasEmptyArray(got, "chunks") {
		t.Errorf("body = %s, want chunks: [] (not null)", got)
	}
}

func TestGetChunk_ScopesToChunkFromPath(t *testing.T) {
	store := &fakeStore{
		getChunk: func(ctx context.Context, chunkID int) (db.Chunk, error) {
			if chunkID != 12 {
				t.Errorf("GetChunk called with chunkID=%d, want 12", chunkID)
			}
			return db.Chunk{ID: 12, BookID: 7, Text: "chunk text"}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/chunk/12")

	var got db.Chunk
	decodeJSON(t, rec, &got)
	if got.Text != "chunk text" {
		t.Errorf("body.text = %q, want %q", got.Text, "chunk text")
	}
}

func TestListQuestionIDs_WrapsInNamedField(t *testing.T) {
	store := &fakeStore{
		listQuestionIDs: func(ctx context.Context, chunkID int) ([]int, error) {
			return []int{1, 2, 3}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/chunk/12/questions")

	var body struct {
		QuestionIDs []int `json:"question_ids"`
	}
	decodeJSON(t, rec, &body)
	if len(body.QuestionIDs) != 3 {
		t.Errorf("body.question_ids = %v, want [1 2 3]", body.QuestionIDs)
	}
}

func TestGetQuestion_ScopesToQuestionFromPath(t *testing.T) {
	store := &fakeStore{
		getQuestion: func(ctx context.Context, questionID int) (db.Question, error) {
			if questionID != 3 {
				t.Errorf("GetQuestion called with questionID=%d, want 3", questionID)
			}
			return db.Question{ID: 3, ChunkID: 12, Type: "vocab"}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/question/3")

	var got db.Question
	decodeJSON(t, rec, &got)
	if got.Type != "vocab" {
		t.Errorf("body.type = %q, want %q", got.Type, "vocab")
	}
}

func TestGetQuestion_NotFoundMapsTo404(t *testing.T) {
	store := &fakeStore{
		getQuestion: func(ctx context.Context, questionID int) (db.Question, error) {
			return db.Question{}, db.ErrNotFound
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/question/999")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetBreakdown_OK(t *testing.T) {
	store := &fakeStore{
		getBreakdown: func(ctx context.Context, chunkID int) (db.Breakdown, error) {
			if chunkID != 12 {
				t.Errorf("GetBreakdown called with chunkID=%d, want 12", chunkID)
			}
			return db.Breakdown{ChunkID: 12, Content: "breakdown content"}, nil
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/chunk/12/breakdown")

	var got db.Breakdown
	decodeJSON(t, rec, &got)
	if got.Content != "breakdown content" {
		t.Errorf("body.content = %q, want %q", got.Content, "breakdown content")
	}
}

func TestGetBreakdown_NotFoundMapsTo404(t *testing.T) {
	store := &fakeStore{
		getBreakdown: func(ctx context.Context, chunkID int) (db.Breakdown, error) {
			return db.Breakdown{}, db.ErrNotFound
		},
	}
	rec := doRequest(t, NewRouter(store), "GET", "/aidoku/chunk/12/breakdown")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
