package ingest

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testPNG returns a real, valid 1x1 PNG's bytes — generated via the
// standard library rather than hand-written/guessed magic bytes, so
// http.DetectContentType has something genuine to sniff.
func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return buf.Bytes()
}

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

func TestFetchImage_Success(t *testing.T) {
	pngBytes := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(pngBytes)
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	data, contentType, err := c.FetchImage(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("FetchImage: %v", err)
	}
	if !bytes.Equal(data, pngBytes) {
		t.Error("FetchImage() data doesn't match the served bytes")
	}
	if contentType != "image/png" {
		t.Errorf("FetchImage() contentType = %q, want %q", contentType, "image/png")
	}
}

func TestFetchImage_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	if _, _, err := c.FetchImage(context.Background(), server.URL); err == nil {
		t.Fatal("FetchImage() = nil error, want an error for a 404 response")
	}
}

func TestFetchImage_RejectsNonImageContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A real HTML error page, the kind of thing a broken/expired
		// image URL might actually serve instead of 404ing cleanly.
		_, _ = w.Write([]byte("<html><body>Not Found</body></html>"))
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	_, _, err := c.FetchImage(context.Background(), server.URL)
	if err == nil {
		t.Fatal("FetchImage() = nil error, want an error for non-image content")
	}
	if !strings.Contains(err.Error(), "does not look like an image") {
		t.Errorf("error = %q, want it to mention the content doesn't look like an image", err.Error())
	}
}

func TestFetchImage_RejectsOversized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oversized := make([]byte, maxImageBytes+1)
		_, _ = w.Write(oversized)
	}))
	defer server.Close()

	c := &Client{HTTPClient: server.Client()}
	_, _, err := c.FetchImage(context.Background(), server.URL)
	if err == nil {
		t.Fatal("FetchImage() = nil error, want an error for a response over the size limit")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error = %q, want it to mention the size limit", err.Error())
	}
}

func TestFetchImage_RequestFailure(t *testing.T) {
	c := &Client{HTTPClient: http.DefaultClient}
	_, _, err := c.FetchImage(context.Background(), "http://127.0.0.1:1/unreachable")
	if err == nil {
		t.Fatal("FetchImage() = nil error, want an error for an unreachable server")
	}
}
