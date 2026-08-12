package question

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

func validRawSet() rawQuestionSet {
	return rawQuestionSet{Questions: []rawQuestion{
		{
			Type:        "vocab",
			Prompt:      "下線を引いた語の意味に最も近いものはどれですか?",
			Options:     []string{"広く認められている", "忘れられている", "疑われている", "秘密にされている"},
			AnswerIndex: 0,
			Explanation: "acknowledged は「認められている」という意味です。",
			Highlight:   "acknowledged",
		},
		{
			Type:        "grammar",
			Prompt:      "下線を引いた \"must\" は何を表していますか?",
			Options:     []string{"論理的な推測", "義務", "許可", "過去の習慣"},
			AnswerIndex: 0,
			Explanation: "この must は論理的な推測を表します。",
			Highlight:   "must",
		},
		{
			Type:        "comprehension",
			Prompt:      "この文が伝えている考えは何ですか?",
			Options:     []string{"裕福な独身男性は妻を必要としているに違いないと考えられている", "裕福な男性は結婚に興味がない", "独身女性より独身男性の方が多い", "妻を持つことは真実ではない"},
			AnswerIndex: 0,
			Explanation: "当時の結婚観への皮肉を述べています。",
		},
	}}
}

func TestValidateQuestionSet_Valid(t *testing.T) {
	questions, err := ValidateQuestionSet(testChunk, validRawSet())
	if err != nil {
		t.Fatalf("ValidateQuestionSet() = %v, want nil", err)
	}
	if len(questions) != 3 {
		t.Fatalf("got %d questions, want 3", len(questions))
	}

	wantOrder := []types.QuestionType{types.QuestionTypeVocab, types.QuestionTypeGrammar, types.QuestionTypeComprehension}
	for i, wantType := range wantOrder {
		if questions[i].Type != wantType {
			t.Errorf("questions[%d].Type = %q, want %q", i, questions[i].Type, wantType)
		}
	}
	if questions[0].Highlight != "acknowledged" {
		t.Errorf("vocab Highlight = %q, want %q", questions[0].Highlight, "acknowledged")
	}
	if questions[2].Highlight != "" {
		t.Errorf("comprehension Highlight = %q, want empty", questions[2].Highlight)
	}
}

func TestValidateQuestionSet_ValidRegardlessOfInputOrder(t *testing.T) {
	set := validRawSet()
	set.Questions[0], set.Questions[2] = set.Questions[2], set.Questions[0] // comprehension, grammar, vocab

	questions, err := ValidateQuestionSet(testChunk, set)
	if err != nil {
		t.Fatalf("ValidateQuestionSet() = %v, want nil", err)
	}
	if questions[0].Type != types.QuestionTypeVocab || questions[2].Type != types.QuestionTypeComprehension {
		t.Errorf("questions not reordered to canonical order: %+v", questions)
	}
}

func TestValidateQuestionSet_Invalid(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(set *rawQuestionSet)
		wantErr string
	}{
		{
			name:    "wrong number of questions",
			mutate:  func(set *rawQuestionSet) { set.Questions = set.Questions[:2] },
			wantErr: "expected 3 questions",
		},
		{
			name:    "unknown type",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Type = "spelling" },
			wantErr: "unknown type",
		},
		{
			name: "duplicate type",
			mutate: func(set *rawQuestionSet) {
				set.Questions[2].Type = "vocab" // now two vocab, no comprehension
			},
			wantErr: "duplicate type",
		},
		{
			name:    "empty prompt",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Prompt = "  " },
			wantErr: "empty prompt",
		},
		{
			name:    "wrong option count",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Options = set.Questions[0].Options[:3] },
			wantErr: "expected 4 options",
		},
		{
			name:    "empty option",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Options[1] = "" },
			wantErr: "is empty",
		},
		{
			name: "duplicate option",
			mutate: func(set *rawQuestionSet) {
				set.Questions[0].Options[1] = set.Questions[0].Options[0]
			},
			wantErr: "duplicates another option",
		},
		{
			name:    "answer_index too high",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].AnswerIndex = 4 },
			wantErr: "out of range",
		},
		{
			name:    "answer_index negative",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].AnswerIndex = -1 },
			wantErr: "out of range",
		},
		{
			name:    "empty explanation",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Explanation = "" },
			wantErr: "empty explanation",
		},
		{
			name:    "vocab missing highlight",
			mutate:  func(set *rawQuestionSet) { set.Questions[0].Highlight = "" },
			wantErr: "missing highlight",
		},
		{
			name:    "grammar highlight not verbatim in chunk text",
			mutate:  func(set *rawQuestionSet) { set.Questions[1].Highlight = "shall" },
			wantErr: "does not appear verbatim",
		},
		{
			name:    "comprehension must not have a highlight",
			mutate:  func(set *rawQuestionSet) { set.Questions[2].Highlight = "wife" },
			wantErr: "must not have a highlight",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set := validRawSet()
			tc.mutate(&set)
			_, err := ValidateQuestionSet(testChunk, set)
			if err == nil {
				t.Fatal("ValidateQuestionSet() = nil, want an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// fakeCaller drives Generator.GenerateQuestions through the
// call/validate/retry/fail path without any network dependency: each
// call returns the next configured response in order.
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

const validSetJSON = `{"questions":[
	{"type":"vocab","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e","highlight":"acknowledged"},
	{"type":"grammar","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e","highlight":"must"},
	{"type":"comprehension","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e"}
]}`

// missingHighlightJSON is valid JSON that still fails semantic
// validation: the grammar question has no "highlight" field at all.
const missingHighlightJSON = `{"questions":[
	{"type":"vocab","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e","highlight":"acknowledged"},
	{"type":"grammar","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e"},
	{"type":"comprehension","prompt":"p","options":["a","b","c","d"],"answer_index":0,"explanation":"e"}
]}`

func testGenerator(caller llmCaller) (*Generator, *bytes.Buffer) {
	var logBuf bytes.Buffer
	g := &Generator{Client: caller, Logger: log.New(&logBuf, "", 0)}
	return g, &logBuf
}

func TestGenerateQuestions_SuccessFirstTry(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{{text: validSetJSON}}}
	g, logBuf := testGenerator(caller)

	questions, err := g.GenerateQuestions(context.Background(), testChunk)
	if err != nil {
		t.Fatalf("GenerateQuestions: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("got %d questions, want 3", len(questions))
	}
	if caller.calls != 1 {
		t.Errorf("expected exactly 1 LLM call, got %d", caller.calls)
	}
	if logBuf.Len() != 0 {
		t.Errorf("expected no log output on success, got %q", logBuf.String())
	}
}

func TestGenerateQuestions_RetryThenSuccess(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{
		{text: `not valid json`},
		{text: validSetJSON},
	}}
	g, logBuf := testGenerator(caller)

	questions, err := g.GenerateQuestions(context.Background(), testChunk)
	if err != nil {
		t.Fatalf("GenerateQuestions: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("got %d questions, want 3", len(questions))
	}
	if caller.calls != 2 {
		t.Errorf("expected exactly 2 LLM calls, got %d", caller.calls)
	}
	if !strings.Contains(logBuf.String(), "chunk 0") {
		t.Errorf("expected the failed first attempt to be logged, got %q", logBuf.String())
	}
}

func TestGenerateQuestions_FailsAfterExhaustingRetries(t *testing.T) {
	caller := &fakeCaller{responses: []fakeResponse{
		{err: errors.New("network unreachable")},
		{text: `{"questions":[]}`}, // wrong count
		{text: missingHighlightJSON},
	}}
	g, _ := testGenerator(caller)

	_, err := g.GenerateQuestions(context.Background(), testChunk)
	if err == nil {
		t.Fatal("GenerateQuestions() = nil error, want an error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "chunk 0") {
		t.Errorf("error should reference the chunk, got: %v", err)
	}
	if caller.calls != len(caller.responses) {
		t.Errorf("expected all %d configured attempts to be used, got %d calls", len(caller.responses), caller.calls)
	}
}
