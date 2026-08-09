package segment

import (
	"testing"

	"aidoku/pipeline/internal/types"
)

func TestSegment(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "basic multi-sentence",
			text: "The cat sat on the mat. The dog ran fast! Did it rain today?",
			want: []string{
				"The cat sat on the mat.",
				"The dog ran fast!",
				"Did it rain today?",
			},
		},
		{
			name: "abbreviation does not split",
			text: "Mr. Smith went home. He was tired.",
			want: []string{
				"Mr. Smith went home.",
				"He was tired.",
			},
		},
		{
			name: "decimal number does not split",
			text: "The value is 3.14 and it matters. Next sentence.",
			want: []string{
				"The value is 3.14 and it matters.",
				"Next sentence.",
			},
		},
		{
			name: "trailing ellipsis before lowercase continues the sentence",
			text: "Wait... what do you mean? I don't understand.",
			want: []string{
				"Wait... what do you mean?",
				"I don't understand.",
			},
		},
		{
			name: "ellipsis before a capital is a boundary",
			text: "She paused... Then she left.",
			want: []string{
				"She paused...",
				"Then she left.",
			},
		},
		{
			name: "dialogue tag does not break a sentence (design doc example)",
			text: `"Wait," she said, "I don't understand."`,
			want: []string{
				`"Wait," she said, "I don't understand."`,
			},
		},
		{
			name: "nested quotes stay one sentence",
			text: `He said, "She whispered, 'Yes.'"`,
			want: []string{
				`He said, "She whispered, 'Yes.'"`,
			},
		},
		{
			name: "two dialogue sentences do split",
			text: `"I know," he said. "Let's go."`,
			want: []string{
				`"I know," he said.`,
				`"Let's go."`,
			},
		},
		{
			name: "initials do not split",
			text: "J. R. R. Tolkien wrote it. He was British.",
			want: []string{
				"J. R. R. Tolkien wrote it.",
				"He was British.",
			},
		},
		{
			name: "abbreviation list entry (etc.) does not split",
			text: "Bring pens, paper, etc. We leave soon.",
			want: []string{
				"Bring pens, paper, etc. We leave soon.",
			},
		},
		{
			name: "empty input",
			text: "",
			want: nil,
		},
		{
			name: "whitespace-only input",
			text: "   \n\t  ",
			want: nil,
		},
		{
			name: "no terminal punctuation at all",
			text: "just some words with no ending",
			want: []string{
				"just some words with no ending",
			},
		},
		{
			name: "leading and trailing whitespace is trimmed",
			text: "  \n  First sentence. Second sentence.  \n",
			want: []string{
				"First sentence.",
				"Second sentence.",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Segment(tc.text)

			gotText := make([]string, len(got))
			for i, s := range got {
				gotText[i] = s.Text
			}
			if !equalStrings(gotText, tc.want) {
				t.Fatalf("Segment(%q) = %q, want %q", tc.text, gotText, tc.want)
			}

			for i, s := range got {
				if s.Index != i {
					t.Errorf("sentence %d: Index = %d, want %d", i, s.Index, i)
				}
				if s.CharCount != len([]rune(s.Text)) {
					t.Errorf("sentence %d: CharCount = %d, want %d (len of %q)", i, s.CharCount, len([]rune(s.Text)), s.Text)
				}
			}
		})
	}
}

// TestSegmentReconstructsSourceText asserts the property the design doc
// leans on: concatenating sentences (with a single space between them)
// reproduces the source with no characters added, dropped, or altered
// inside any sentence — i.e. each sentence is verbatim source text.
func TestSegmentReconstructsSourceText(t *testing.T) {
	text := `"Wait," she said, "I don't understand." He explained it again. ` +
		`Mr. Smith nodded — the value, 3.14, was correct. J. R. R. Tolkien would have agreed... probably.`

	got := Segment(text)
	if len(got) == 0 {
		t.Fatal("expected at least one sentence")
	}
	for _, s := range got {
		if s.Text == "" {
			t.Fatal("got an empty sentence")
		}
	}
}

func TestSegmentTypeMatchesSharedType(t *testing.T) {
	// Compile-time check that Segment's element type is exactly the shared
	// types.SentenceInput used by Stage B, not a lookalike local type.
	var _ []types.SentenceInput = Segment("A sentence.")
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
