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
	"aidoku/pipeline/internal/langpair"
	"aidoku/pipeline/internal/types"
)

var testChunk = types.Chunk{
	Index:     0,
	Text:      "It is a truth universally acknowledged, that a single man in possession of a good fortune, must be in want of a wife.",
	CharCount: 118,
}

const validBreakdown = "【文構造】\"It is a truth universally acknowledged, that ...\" は形式主語構文です。\n\n【語彙】\n・acknowledged「認められている」\n\n【意味】当時の結婚観への皮肉です。"

func TestValidateBreakdown_Valid(t *testing.T) {
	if err := validateBreakdown(langpair.EN_JP, validBreakdown); err != nil {
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
			err := validateBreakdown(langpair.EN_JP, tc.content)
			if err == nil {
				t.Fatal("validateBreakdown() = nil, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateBreakdown_JPEN_RejectsJapaneseLeftover(t *testing.T) {
	// JP_EN's ValidateNativeText checks the opposite direction from
	// EN_JP's — it should reject a "breakdown" that's still in Japanese
	// (the Target language here, not Native).
	if err := validateBreakdown(langpair.JP_EN, validBreakdown); err == nil {
		t.Fatal("validateBreakdown(JP_EN, <Japanese text>) = nil, want an error")
	}
	if err := validateBreakdown(langpair.JP_EN, "This is a fully English explanation."); err != nil {
		t.Errorf("validateBreakdown(JP_EN, <English text>) = %v, want nil", err)
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
	g := &Generator{Client: caller, Logger: log.New(&logBuf, "", 0), LanguagePair: langpair.EN_JP}
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

func TestGenerateBreakdown_FailsIfLanguagePairUnset(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{{text: validBreakdown}}}
	g := &Generator{Client: caller} // LanguagePair left zero-valued

	_, err := g.GenerateBreakdown(context.Background(), testChunk)
	if err == nil {
		t.Fatal("GenerateBreakdown() with no LanguagePair = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "LanguagePair") {
		t.Errorf("error = %q, want it to mention LanguagePair", err.Error())
	}
	if caller.calls != 0 {
		t.Errorf("expected no LLM call when LanguagePair is unset, got %d", caller.calls)
	}
}

func TestBuildSystemPrompt_ReflectsThePairPassedIn(t *testing.T) {
	enJP := buildSystemPrompt(langpair.EN_JP)
	if !strings.Contains(enJP, "Japanese speaker (L1) learning English (L2)") {
		t.Errorf("EN_JP prompt doesn't mention the expected learner description: %s", enJP)
	}
	if !strings.Contains(enJP, "【文構造】") {
		t.Errorf("EN_JP prompt doesn't mention its section labels: %s", enJP)
	}

	jpEN := buildSystemPrompt(langpair.JP_EN)
	if !strings.Contains(jpEN, "English speaker (L1) learning Japanese (L2)") {
		t.Errorf("JP_EN prompt doesn't mention the expected learner description: %s", jpEN)
	}
	if !strings.Contains(jpEN, "[Sentence Structure]") {
		t.Errorf("JP_EN prompt doesn't mention its section labels: %s", jpEN)
	}
	if enJP == jpEN {
		t.Error("EN_JP and JP_EN produced identical prompts, want them to differ")
	}

	for _, p := range []string{enJP, jpEN} {
		if !strings.Contains(p, "never translated") {
			t.Errorf("prompt doesn't warn the model against translating quoted spans: %s", p)
		}
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
