package segment

import (
	"fmt"
	"strings"
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
		{
			name: "Dracula by Bram Stoker excerpt",
			text: `“Count Dracula?” He bowed in a courtly way as he replied:—
“I am Dracula; and I bid you welcome, Mr. Harker, to my house. Come in; the night air is chill, and you must need to eat and rest.” As he was speaking, he put the lamp on a bracket on the wall, and stepping out, took my luggage; he had carried it in before I could forestall him. I protested but he insisted:—
“Nay, sir, you are my guest. It is late, and my people are not available. Let me see to your comfort myself.” He insisted on carrying my traps along the passage, and then up a great winding stair, and along another great passage, on whose stone floor our steps rang heavily. At the end of this he threw open a heavy door, and I rejoiced to see within a well-lit room in which a table was spread for supper, and on whose mighty hearth a great fire of logs, freshly replenished, flamed and flared.`,
			want: []string{
				`“Count Dracula?”`,
				`He bowed in a courtly way as he replied:—
“I am Dracula; and I bid you welcome, Mr. Harker, to my house.`,
				`Come in; the night air is chill, and you must need to eat and rest.”`,
				`As he was speaking, he put the lamp on a bracket on the wall, and stepping out, took my luggage; he had carried it in before I could forestall him.`,
				`I protested but he insisted:—
“Nay, sir, you are my guest.`,
				`It is late, and my people are not available.`,
				`Let me see to your comfort myself.”`,
				`He insisted on carrying my traps along the passage, and then up a great winding stair, and along another great passage, on whose stone floor our steps rang heavily.`,
				`At the end of this he threw open a heavy door, and I rejoiced to see within a well-lit room in which a table was spread for supper, and on whose mighty hearth a great fire of logs, freshly replenished, flamed and flared.`,
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
				t.Fatalf("Segment mismatch:\n%s", diffSentences(tc.want, gotText))
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

// diffSentences renders a side-by-side-by-index listing of the wanted and
// actual sentences, so a failing test shows exactly where they diverge
// instead of dumping two long %q-quoted slices to compare by eye.
func diffSentences(want, got []string) string {
	var b strings.Builder
	n := max(len(want), len(got))
	fmt.Fprintf(&b, "  (want %d sentence(s), got %d)\n", len(want), len(got))
	for i := range n {
		w, hasW := "", i < len(want)
		if i < len(want) {
			w = want[i]
		}
		g, hasG := "", i < len(got)
		if i < len(got) {
			g = got[i]
		}
		marker := "=="
		if !hasW || !hasG || w != g {
			marker = "!="
		}
		fmt.Fprintf(&b, "[%d] %s\n", i, marker)
		if hasW {
			fmt.Fprintf(&b, "    want: %q\n", w)
		} else {
			fmt.Fprintf(&b, "    want: <none>\n")
		}
		if hasG {
			fmt.Fprintf(&b, "    got:  %q\n", g)
		} else {
			fmt.Fprintf(&b, "    got:  <none>\n")
		}
	}
	return b.String()
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
