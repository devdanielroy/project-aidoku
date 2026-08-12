package breakdown

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"aidoku/pipeline/internal/anthropic"
	"aidoku/pipeline/internal/types"
)

var testChunk = types.Chunk{
	Index:     0,
	Text:      "It is a truth universally acknowledged, that a single man in possession of a good fortune, must be in want of a wife.",
	CharCount: 118,
}

const validBreakdown = "【文構造】\"It is a truth universally acknowledged, that ...\" は形式主語構文です。\n\n【語彙】\n・acknowledged「認められている」\n\n【意味】当時の結婚観への皮肉です。"

func TestValidateBreakdown_Valid(t *testing.T) {
	if err := validateBreakdown(validBreakdown); err != nil {
		t.Errorf("validateBreakdown(%q) = %v, want nil", validBreakdown, err)
	}
}

func TestValidateBreakdown_Invalid(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "empty", content: "", wantErr: "empty breakdown"},
		{name: "whitespace only", content: "   \n\t  ", wantErr: "empty breakdown"},
		{name: "English only, no Japanese", content: "This is a fully English explanation with no Japanese at all.", wantErr: "does not appear to contain any Japanese"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBreakdown(tc.content)
			if err == nil {
				t.Fatal("validateBreakdown() = nil, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// fakeCaller drives Generator.GenerateBreakdown through the
// call/validate/retry/fail path without any network dependency: each
// call returns the next configured response in order. Same pattern as
// question.fakeCaller.
type fakeCaller struct {
	responses []fakeResponse
	calls     int
}

type fakeResponse struct {
	text string
	err  error
}

func (f *fakeCaller) CreateMessage(ctx context.Context, req anthropic.MessagesRequest) (*anthropic.MessagesResponse, error) {
	if f.calls >= len(f.responses) {
		return nil, fmt.Errorf("fakeCaller: unexpected call #%d (only %d response(s) configured)", f.calls+1, len(f.responses))
	}
	r := f.responses[f.calls]
	f.calls++
	if r.err != nil {
		return nil, r.err
	}
	return &anthropic.MessagesResponse{
		StopReason: "end_turn",
		Content:    []anthropic.ContentBlock{{Type: "text", Text: r.text}},
	}, nil
}

func testGenerator(caller llmCaller) (*Generator, *bytes.Buffer) {
	var logBuf bytes.Buffer
	g := &Generator{Client: caller, Logger: log.New(&logBuf, "", 0)}
	return g, &logBuf
}

func TestGenerateBreakdown_SuccessFirstTry(t *testing.T) {
	// Leading/trailing whitespace around an otherwise valid response
	// should be trimmed off the returned content.
	caller := &fakeCaller{responses: []fakeResponse{{text: "\n  " + validBreakdown + "  \n"}}}
	g, logBuf := testGenerator(caller)

	got, err := g.GenerateBreakdown(context.Background(), testChunk)
	if err != nil {
		t.Fatalf("GenerateBreakdown: %v", err)
	}
	if got != validBreakdown {
		t.Errorf("GenerateBreakdown() = %q, want %q (trimmed)", got, validBreakdown)
	}
	if caller.calls != 1 {
		t.Errorf("expected exactly 1 LLM call, got %d", caller.calls)
	}
	if logBuf.Len() != 0 {
		t.Errorf("expected no log output on success, got %q", logBuf.String())
	}
}

func TestGenerateBreakdown_RetryThenSuccess(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{
		{text: "This response has no Japanese in it at all."},
		{text: validBreakdown},
	}}
	g, logBuf := testGenerator(caller)

	got, err := g.GenerateBreakdown(context.Background(), testChunk)
	if err != nil {
		t.Fatalf("GenerateBreakdown: %v", err)
	}
	if got != validBreakdown {
		t.Errorf("GenerateBreakdown() = %q, want %q", got, validBreakdown)
	}
	if caller.calls != 2 {
		t.Errorf("expected exactly 2 LLM calls, got %d", caller.calls)
	}
	if !strings.Contains(logBuf.String(), "chunk 0") {
		t.Errorf("expected the failed first attempt to be logged, got %q", logBuf.String())
	}
}

func TestGenerateBreakdown_FailsAfterExhaustingRetries(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{
		{err: errors.New("network unreachable")},
		{text: "   "},                                   // empty after trim
		{text: "English only, still no Japanese here."}, // wrong language
	}}
	g, _ := testGenerator(caller)

	_, err := g.GenerateBreakdown(context.Background(), testChunk)
	if err == nil {
		t.Fatal("GenerateBreakdown() = nil error, want an error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "chunk 0") {
		t.Errorf("error should reference the chunk, got: %v", err)
	}
	if caller.calls != len(caller.responses) {
		t.Errorf("expected all %d configured attempts to be used, got %d calls", len(caller.responses), caller.calls)
	}
}
