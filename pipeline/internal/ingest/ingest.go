// Package ingest implements the first pipeline stage: pulling a raw book
// text from a public domain source. See AIDOKU_DESIGN.md §3 step 1.
//
// v1 sources English texts from Project Gutenberg exclusively, for
// unambiguous copyright status (e.g.
// https://www.gutenberg.org/cache/epub/1342/pg1342.txt for Pride and
// Prejudice). FetchText returns the raw text completely unmodified —
// stripping Gutenberg's license header/footer is the clean package's job,
// not this one's.
package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client fetches raw book text over HTTP.
type Client struct {
	HTTPClient *http.Client // defaults to a 30s-timeout client if nil
}

// NewClient returns a Client with production defaults.
func NewClient() *Client {
	return &Client{HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

// FetchText downloads and returns the raw text at url as-is.
func (c *Client) FetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("ingest: build request: %w", err)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
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
