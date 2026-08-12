package catalog

import "fmt"

// ReadingLevel is one of the ten reading-comprehension levels a book is
// assigned in the catalog (see books.txt's "Level=" line). Unlike every
// other field on Entry, this is never derived or detected by the
// pipeline — a human decides it per book (typically weighing vocabulary
// difficulty, sentence complexity, and how the book maps to TOEIC/CEFR,
// sometimes in discussion with Claude in a separate session) and records
// it directly in the catalog, before the pipeline is even run — grading
// is deliberately not one of the numbered pipeline stages. See the
// README's Reading Levels table for the full name/TOEIC/CEFR mapping,
// and AIDOKU_DESIGN.md §7e.
type ReadingLevel int

const (
	LevelInitiate   ReadingLevel = 1
	LevelNovice     ReadingLevel = 2
	LevelApprentice ReadingLevel = 3
	LevelReader     ReadingLevel = 4
	LevelBookworm   ReadingLevel = 5
	LevelErudite    ReadingLevel = 6
	LevelVirtuoso   ReadingLevel = 7
	LevelLuminary   ReadingLevel = 8
	LevelAcademic   ReadingLevel = 9
	LevelScholar    ReadingLevel = 10
)

// levelNames mirrors the README's Reading Levels table, in order.
var levelNames = [...]string{
	LevelInitiate:   "Initiate",
	LevelNovice:     "Novice",
	LevelApprentice: "Apprentice",
	LevelReader:     "Reader",
	LevelBookworm:   "Bookworm",
	LevelErudite:    "Erudite",
	LevelVirtuoso:   "Virtuoso",
	LevelLuminary:   "Luminary",
	LevelAcademic:   "Academic",
	LevelScholar:    "Scholar",
}

// Valid reports whether l is one of the ten defined reading levels (1
// through 10 inclusive) — the range the catalog parser and any future
// caller should enforce, rather than trusting an arbitrary int.
func (l ReadingLevel) Valid() bool {
	return l >= LevelInitiate && l <= LevelScholar
}

// String returns l's name from the README's Reading Levels table (e.g.
// "Bookworm"), or a fallback like "ReadingLevel(99)" for an out-of-range
// value — never a panic or an empty string, so an invalid level is still
// safe (if unhelpful) to log or print.
func (l ReadingLevel) String() string {
	if !l.Valid() {
		return fmt.Sprintf("ReadingLevel(%d)", int(l))
	}
	return levelNames[l]
}
