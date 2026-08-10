package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateMessage_Success(t *testing.T) {
	var gotReq MessagesRequest
	var gotHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MessagesResponse{
			ID:         "msg_123",
			Model:      "claude-sonnet-5",
			StopReason: "end_turn",
			Content:    []ContentBlock{{Type: "text", Text: `{"ok":true}`}},
		})
	}))
	defer server.Close()

	c := &Client{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()}
	resp, err := c.CreateMessage(context.Background(), MessagesRequest{
		Model:     "claude-sonnet-5",
		MaxTokens: 1024,
		System:    "system prompt",
		Messages:  []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	if got := gotHeaders.Get("x-api-key"); got != "test-key" {
		t.Errorf("x-api-key header = %q, want %q", got, "test-key")
	}
	if got := gotHeaders.Get("anthropic-version"); got != apiVersion {
		t.Errorf("anthropic-version header = %q, want %q", got, apiVersion)
	}
	if gotReq.Model != "claude-sonnet-5" || gotReq.System != "system prompt" || len(gotReq.Messages) != 1 {
		t.Errorf("unexpected request body: %+v", gotReq)
	}

	text, err := resp.Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if text != `{"ok":true}` {
		t.Errorf("Text() = %q, want %q", text, `{"ok":true}`)
	}
}

func TestCreateMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	c := &Client{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()}
	_, err := c.CreateMessage(context.Background(), MessagesRequest{
		Model:     "m",
		MaxTokens: 1,
		Messages:  []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusTooManyRequests)
	}
}

func TestCreateMessage_MissingAPIKey(t *testing.T) {
	c := &Client{}
	_, err := c.CreateMessage(context.Background(), MessagesRequest{})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestMessagesResponse_Text_NoTextBlocks(t *testing.T) {
	resp := &MessagesResponse{StopReason: "max_tokens"}
	if _, err := resp.Text(); err == nil {
		t.Fatal("expected error for response with no text blocks")
	}
}
