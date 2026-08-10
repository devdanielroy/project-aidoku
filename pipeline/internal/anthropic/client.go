// Package anthropic is a minimal client for Anthropic's Messages API
// (https://api.anthropic.com/v1/messages), used by the pipeline's offline,
// batch-mode AI generation stages — chunk grouping first, question and
// breakdown generation and book grading later. See AIDOKU_DESIGN.md
// §7a/§7b.
//
// Deliberately stdlib-only (net/http, no SDK dependency): there was no
// first-party Anthropic Go SDK confirmed available when this was written,
// and a plain HTTP client is easy to test and reason about for a batch
// pipeline that makes one call at a time.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
)

// Client calls the Anthropic Messages API using APIKey.
type Client struct {
	APIKey     string
	BaseURL    string       // defaults to defaultBaseURL if empty
	HTTPClient *http.Client // defaults to a 60s-timeout client if nil
}

// NewClient returns a Client for apiKey with production defaults.
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Message is one turn in a Messages API conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessagesRequest is the request body for POST /v1/messages.
type MessagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

// ContentBlock is one block of a Messages API response's content array.
// Only Type "text" is populated by the text-only prompts this client
// sends; other block types are left as zero values.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MessagesResponse is the response body from POST /v1/messages.
type MessagesResponse struct {
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Content    []ContentBlock `json:"content"`
}

// Text concatenates all "text" content blocks in the response. It errors
// if the response has no text content at all (e.g. an unexpected
// stop_reason or content type), so callers don't silently proceed with an
// empty string.
func (r *MessagesResponse) Text() (string, error) {
	var out string
	for _, b := range r.Content {
		if b.Type == "text" {
			out += b.Text
		}
	}
	if out == "" {
		return "", fmt.Errorf("anthropic: response has no text content (stop_reason=%q)", r.StopReason)
	}
	return out, nil
}

// APIError is returned when the API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic: API returned status %d: %s", e.StatusCode, e.Body)
}

// CreateMessage sends req to POST /v1/messages and returns the parsed
// response.
func (c *Client) CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("anthropic: APIKey is empty")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", apiVersion)

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: httpResp.StatusCode, Body: string(respBody)}
	}

	var parsed MessagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
	}
	return &parsed, nil
}
