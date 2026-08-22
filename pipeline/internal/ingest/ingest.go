// Package ingest implements the first pipeline stage: pulling a raw book
// text from a public domain source. See AIDOKU_DESIGN.md §3 step 1.
//
// v1 sources English texts from Project Gutenberg exclusively, for
// unambiguous copyright status (e.g.
// https://www.gutenberg.org/cache/epub/1342/pg1342.txt for Pride and
// Prejudice). FetchText returns the raw text completely unmodified —
// stripping Gutenberg's license header/footer is the clean package's job,
// not this one's.
//
// FetchBytes is the same idea for binary content (currently just book
// cover images, see catalog.Entry.ImageURL) — arbitrary source, size-
// capped, content-type sniffed from the actual bytes rather than trusted
// from the URL or response header.
package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxImageBytes caps how large a downloaded cover image can be —
// generous for a book cover thumbnail, but enough to reject something
// wildly oversized (a misconfigured URL serving a full HTML page, a
// multi-megabyte poster-sized image) rather than silently accepting it.
const maxImageBytes = 5 * 1024 * 1024 // 5 MiB

// Client fetches raw book text over HTTP.
type Client struct {
	HTTPClient *http.Client // defaults to a 30s-timeout client if nil
}

// NewClient returns a Client with production defaults.
func NewClient() *Client {
	return &Client{HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// FetchText downloads and returns the raw text at url as-is.
func (c *Client) FetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("ingest: build request: %w", err)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("ingest: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ingest: %s returned status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ingest: read response body: %w", err)
	}
	return string(body), nil
}

// FetchImage downloads url and returns its raw bytes plus a content
// type sniffed from those bytes (http.DetectContentType) — never
// trusted from the URL or the server's own Content-Type header, which
// could be wrong, missing, or (worst case) misleading. Fails if the
// response isn't recognizably image content, or exceeds maxImageBytes;
// see catalog.Entry.ImageURL and cmd/process, which treats either as a
// soft failure — a missing cover shouldn't block the rest of the book.
func (c *Client) FetchImage(ctx context.Context, url string) (data []byte, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("ingest: build request: %w", err)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("ingest: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("ingest: %s returned status %d", url, resp.StatusCode)
	}

	// Read one byte past the limit so an oversized response is detected
	// (len(body) > maxImageBytes) rather than silently truncated to
	// exactly the limit and treated as a complete, valid image.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("ingest: read response body: %w", err)
	}
	if len(body) > maxImageBytes {
		return nil, "", fmt.Errorf("ingest: %s exceeds the %d byte image size limit", url, maxImageBytes)
	}

	sniffed := http.DetectContentType(body)
	if !strings.HasPrefix(sniffed, "image/") {
		return nil, "", fmt.Errorf("ingest: %s does not look like an image (detected %q)", url, sniffed)
	}
	return body, sniffed, nil
}
