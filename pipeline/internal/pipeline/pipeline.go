// Package pipeline wires the ingest, clean, and catalog packages together
// into one step: turning a catalog.Entry into the final, novel-only text
// ready for Stage A segmentation. See AIDOKU_DESIGN.md §3 steps 1-2.
package pipeline

import (
	"context"
	"fmt"

	"aidoku/pipeline/internal/catalog"
	"aidoku/pipeline/internal/clean"
)

// textFetcher is the subset of *ingest.Client that PrepareBook needs.
// Defined on the consumer side so tests can supply a fake with no network
// dependency — same pattern as the LLM-calling packages' llmCaller.
type textFetcher interface {
	FetchText(ctx context.Context, url string) (string, error)
}

// CleanFunc is the shape of clean.Clean and its per-language siblings
// (clean.CleanJapanese, etc.) — see internal/clean. PrepareBook takes
// one explicitly rather than hardcoding clean.Clean, so the caller
// picks the language-appropriate variant for whichever catalog/pair
// it's running (see cmd/process and cmd/ingest, both of which dispatch
// on their -pair flag) instead of this package needing to know
// anything about language pairs itself.
type CleanFunc = func(raw string) (string, error)

// PrepareBook fetches entry's source text, strips Project Gutenberg's
// wrapper and normalizes it (cleanFn), then trims it down to just the
// novel content using entry's anchors (clean.Trim). The result is the
// final text for this book, ready to hand to Stage A (segment.Segment
// or segment.SegmentJapanese — same per-caller dispatch as cleanFn).
func PrepareBook(ctx context.Context, fetcher textFetcher, entry catalog.Entry, cleanFn CleanFunc) (string, error) {
	raw, err := fetcher.FetchText(ctx, entry.SourceURL)
	if err != nil {
		return "", fmt.Errorf("pipeline: book %d: fetch: %w", entry.GutenbergID, err)
	}

	cleaned, err := cleanFn(raw)
	if err != nil {
		return "", fmt.Errorf("pipeline: book %d: clean: %w", entry.GutenbergID, err)
	}

	trimmed, err := clean.Trim(cleaned, entry.FirstLine, entry.LastLine)
	if err != nil {
		return "", fmt.Errorf("pipeline: book %d: trim: %w", entry.GutenbergID, err)
	}

	return trimmed, nil
}
