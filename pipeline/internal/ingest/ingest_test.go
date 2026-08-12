package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		_, _ = w.Write([]byte("raw book text\nwith multiple lines\n"))
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	text, err := c.FetchText(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchText: %v", err)
	}
	if text != "raw book text\nwith multiple lines\n" {
		t.Errorf("FetchText() = %q, unexpected content", text)
	}
}

func TestFetchText_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	_, err := c.FetchText(context.Background(), server.URL)
	if err == nil {
		t.Fatal("FetchText() = nil error, want an error for a 404 response")
	}
}

func TestFetchText_RequestFailure(t *testing.T) {
	c := &Client{HTTPClient: http.DefaultClient}
	// Nothing listening on this port (hopefully) - request should fail to
	// even connect, exercising the transport-error path rather than a
	// non-2xx status.
	_, err := c.FetchText(context.Background(), "http://127.0.0.1:1/unreachable")
	if err == nil {
		t.Fatal("FetchText() = nil error, want an error for an unreachable server")
	}
}

func TestFetchText_InvalidURL(t *testing.T) {
	c := NewClient()
	_, err := c.FetchText(context.Background(), "://not-a-valid-url")
	if err == nil {
		t.Fatal("FetchText() = nil error, want an error for a malformed URL")
	}
}
