package catalog

import "testing"

func TestReadingLevel_Valid(t *testing.T) {
	cases := []struct {
		level ReadingLevel
		want  bool
	}{
		{0, false},
		{LevelInitiate, true},
		{5, true},
		{LevelScholar, true},
		{11, false},
		{-1, false},
	}
	for _, tc := range cases {
		if got := tc.level.Valid(); got != tc.want {
			t.Errorf("ReadingLevel(%d).Valid() = %v, want %v", tc.level, got, tc.want)
		}
	}
}

func TestReadingLevel_String(t *testing.T) {
	cases := []struct {
		level ReadingLevel
		want  string
	}{
		{LevelInitiate, "Initiate"},
		{LevelNovice, "Novice"},
		{LevelApprentice, "Apprentice"},
		{LevelReader, "Reader"},
		{LevelBookworm, "Bookworm"},
		{LevelErudite, "Erudite"},
		{LevelVirtuoso, "Virtuoso"},
		{LevelLuminary, "Luminary"},
		{LevelAcademic, "Academic"},
		{LevelScholar, "Scholar"},
		{0, "ReadingLevel(0)"},
		{11, "ReadingLevel(11)"},
	}
	for _, tc := range cases {
		if got := tc.level.String(); got != tc.want {
			t.Errorf("ReadingLevel(%d).String() = %q, want %q", tc.level, got, tc.want)
		}
	}
}
