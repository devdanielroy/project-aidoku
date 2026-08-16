package chunk

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

func sentences(n, startIndex, charsEach int) []types.SentenceInput {
	out := make([]types.SentenceInput, n)
	for i := 0; i < n; i++ {
		out[i] = types.SentenceInput{
			Index:     startIndex + i,
			Text:      fmt.Sprintf("Sentence %d.", startIndex+i),
			CharCount: charsEach,
		}
	}
	return out
}

func TestValidatePartition_Valid(t *testing.T) {
	cases := []struct {
		name  string
		sents []types.SentenceInput
		resp  types.ChunkGroupingResponse
	}{
		{
			name:  "single chunk covering everything",
			sents: sentences(3, 0, 10),
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{0, 1, 2}},
			}},
		},
		{
			name:  "multiple chunks",
			sents: sentences(5, 0, 10),
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{0, 1}},
				{ChunkIndex: 1, SentenceIndices: []int{2, 3, 4}},
			}},
		},
		{
			name:  "window not starting at zero",
			sents: sentences(3, 50, 10),
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{50}},
				{ChunkIndex: 1, SentenceIndices: []int{51, 52}},
			}},
		},
		{
			name:  "empty input, empty response",
			sents: nil,
			resp:  types.ChunkGroupingResponse{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePartition(tc.sents, tc.resp); err != nil {
				t.Fatalf("ValidatePartition() = %v, want nil", err)
			}
		})
	}
}

func TestValidatePartition_Invalid(t *testing.T) {
	base := sentences(5, 0, 10) // indices 0..4

	cases := []struct {
		name string
		resp types.ChunkGroupingResponse
	}{
		{
			name: "gap between chunks",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{0, 1}},
				{ChunkIndex: 1, SentenceIndices: []int{3, 4}}, // skips 2
			}},
		},
		{
			name: "overlap between chunks",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{0, 1, 2}},
				{ChunkIndex: 1, SentenceIndices: []int{2, 3, 4}}, // 2 repeated
			}},
		},
		{
			name: "reordered indices within a chunk",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{1, 0}},
				{ChunkIndex: 1, SentenceIndices: []int{2, 3, 4}},
			}},
		},
		{
			name: "chunk_index out of order",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 1, SentenceIndices: []int{0, 1, 2, 3, 4}},
			}},
		},
		{
			name: "missing trailing sentences",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{0, 1, 2}},
			}},
		},
		{
			name: "empty chunk list for non-empty input",
			resp: types.ChunkGroupingResponse{},
		},
		{
			name: "chunk with no sentence indices",
			resp: types.ChunkGroupingResponse{Chunks: []types.ChunkGroup{
				{ChunkIndex: 0, SentenceIndices: []int{}},
				{ChunkIndex: 1, SentenceIndices: []int{0, 1, 2, 3, 4}},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePartition(base, tc.resp); err == nil {
				t.Fatal("ValidatePartition() = nil, want an error")
			}
		})
	}
}

func TestGreedyGroup(t *testing.T) {
	t.Run("accumulates under the target+tolerance limit", func(t *testing.T) {
		sents := sentences(5, 0, 60) // running totals: 60,120,180,240,300; limit is 300
		resp := GreedyGroup(sents, langpair.EN_JP)
		if err := ValidatePartition(sents, resp); err != nil {
			t.Fatalf("GreedyGroup produced an invalid partition: %v", err)
		}
		// The 5th sentence brings the running total to exactly the limit
		// (300), which should not trigger a cut - all 5 land in one chunk.
		if len(resp.Chunks) != 1 {
			t.Fatalf("got %d chunks, want 1: %+v", len(resp.Chunks), resp.Chunks)
		}
	})

	t.Run("cuts once the limit would be exceeded", func(t *testing.T) {
		sents := sentences(6, 0, 60) // 6th sentence would push the total to 360 > 300
		resp := GreedyGroup(sents, langpair.EN_JP)
		if err := ValidatePartition(sents, resp); err != nil {
			t.Fatalf("GreedyGroup produced an invalid partition: %v", err)
		}
		if len(resp.Chunks) != 2 {
			t.Fatalf("got %d chunks, want 2: %+v", len(resp.Chunks), resp.Chunks)
		}
		if len(resp.Chunks[0].SentenceIndices) != 5 {
			t.Errorf("first chunk has %d sentence(s), want 5", len(resp.Chunks[0].SentenceIndices))
		}
	})

	t.Run("a single oversized sentence still becomes its own chunk", func(t *testing.T) {
		sents := []types.SentenceInput{
			{Index: 0, Text: "short.", CharCount: 50},
			{Index: 1, Text: "long.", CharCount: 500}, // alone, way over the limit
			{Index: 2, Text: "short.", CharCount: 50},
		}
		resp := GreedyGroup(sents, langpair.EN_JP)
		if err := ValidatePartition(sents, resp); err != nil {
			t.Fatalf("GreedyGroup produced an invalid partition: %v", err)
		}
		if len(resp.Chunks) != 3 {
			t.Fatalf("got %d chunks, want 3 (the oversized sentence forces a cut on both sides): %+v", len(resp.Chunks), resp.Chunks)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		resp := GreedyGroup(nil, langpair.EN_JP)
		if len(resp.Chunks) != 0 {
			t.Fatalf("got %d chunks, want 0", len(resp.Chunks))
		}
	})
}

// fakeCaller drives Grouper.GroupSentencesIntoChunks through the
// LLM-call/validate/retry/fallback path without any network dependency:
// each call returns the next configured response in order.
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

func testGrouper(caller llmCaller) (*Grouper, *bytes.Buffer) {
	var logBuf bytes.Buffer
	g := &Grouper{
		Client:       caller,
		Logger:       log.New(&logBuf, "", 0),
		LanguagePair: langpair.EN_JP,
	}
	return g, &logBuf
}

func TestGroupSentencesIntoChunks_SuccessFirstTry(t *testing.T) {
	sents := sentences(3, 0, 10)
	caller := &fakeCaller{responses: []fakeResponse{
		{text: `{"chunks":[{"chunk_index":0,"sentence_indices":[0,1,2]}]}`},
	}}
	g, logBuf := testGrouper(caller)

	resp, err := g.GroupSentencesIntoChunks(context.Background(), sents)
	if err != nil {
		t.Fatalf("GroupSentencesIntoChunks: %v", err)
	}
	if len(resp.Chunks) != 1 || len(resp.Chunks[0].SentenceIndices) != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if caller.calls != 1 {
		t.Errorf("expected exactly 1 LLM call, got %d", caller.calls)
	}
	if logBuf.Len() != 0 {
		t.Errorf("expected no log output on success, got %q", logBuf.String())
	}
}

func TestGroupSentencesIntoChunks_RetryThenSuccess(t *testing.T) {
	sents := sentences(3, 0, 10)
	caller := &fakeCaller{responses: []fakeResponse{
		{text: `not valid json`},
		{text: `{"chunks":[{"chunk_index":0,"sentence_indices":[0,1,2]}]}`},
	}}
	g, _ := testGrouper(caller)

	resp, err := g.GroupSentencesIntoChunks(context.Background(), sents)
	if err != nil {
		t.Fatalf("GroupSentencesIntoChunks: %v", err)
	}
	if caller.calls != 2 {
		t.Errorf("expected exactly 2 LLM calls (1 retry), got %d", caller.calls)
	}
	if err := ValidatePartition(sents, resp); err != nil {
		t.Fatalf("returned an invalid partition: %v", err)
	}
}

func TestGroupSentencesIntoChunks_FallsBackAfterRepeatedFailures(t *testing.T) {
	sents := sentences(6, 0, 60)
	caller := &fakeCaller{responses: []fakeResponse{
		{text: `{"chunks":[{"chunk_index":0,"sentence_indices":[0,1]}]}`}, // gap - invalid partition
		{err: errors.New("network unreachable")},                          // retry fails outright too
	}}
	g, logBuf := testGrouper(caller)

	resp, err := g.GroupSentencesIntoChunks(context.Background(), sents)
	if err != nil {
		t.Fatalf("GroupSentencesIntoChunks should not return an error (fallback should always succeed), got: %v", err)
	}
	if caller.calls != 2 {
		t.Errorf("expected exactly 2 LLM attempts before falling back, got %d", caller.calls)
	}
	want := GreedyGroup(sents, langpair.EN_JP)
	if fmt.Sprint(resp) != fmt.Sprint(want) {
		t.Errorf("fallback response = %+v, want GreedyGroup result %+v", resp, want)
	}
	if !strings.Contains(logBuf.String(), "falling back") {
		t.Errorf("expected a fallback event to be logged, got %q", logBuf.String())
	}
}

func TestGroupSentencesIntoChunks_EmptyInput(t *testing.T) {
	caller := &fakeCaller{} // no responses configured - must not be called
	g, _ := testGrouper(caller)

	resp, err := g.GroupSentencesIntoChunks(context.Background(), nil)
	if err != nil {
		t.Fatalf("GroupSentencesIntoChunks: %v", err)
	}
	if len(resp.Chunks) != 0 {
		t.Errorf("got %d chunks for empty input, want 0", len(resp.Chunks))
	}
	if caller.calls != 0 {
		t.Errorf("expected no LLM calls for empty input, got %d", caller.calls)
	}
}

func TestGroupSentencesIntoChunks_FailsIfLanguagePairUnset(t *testing.T) {
	sents := sentences(3, 0, 10)
	caller := &fakeCaller{responses: []fakeResponse{
		{text: `{"chunks":[{"chunk_index":0,"sentence_indices":[0,1,2]}]}`},
	}}
	g := &Grouper{Client: caller} // LanguagePair left zero-valued

	_, err := g.GroupSentencesIntoChunks(context.Background(), sents)
	if err == nil {
		t.Fatal("GroupSentencesIntoChunks() with no LanguagePair = nil error, want an error")
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
	if !strings.Contains(enJP, "targeting approximately 240 characters per chunk") {
		t.Errorf("EN_JP prompt doesn't mention its target chunk length: %s", enJP)
	}

	jpEN := buildSystemPrompt(langpair.JP_EN)
	if !strings.Contains(jpEN, "targeting approximately 140 characters per chunk") {
		t.Errorf("JP_EN prompt doesn't mention its target chunk length: %s", jpEN)
	}
	if enJP == jpEN {
		t.Error("EN_JP and JP_EN produced identical prompts, want them to differ")
	}

	for _, p := range []string{enJP, jpEN} {
		if !strings.Contains(p, "Never translate") {
			t.Errorf("prompt doesn't warn the model against translating: %s", p)
		}
	}
}
