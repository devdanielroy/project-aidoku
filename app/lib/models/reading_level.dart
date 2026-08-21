/// The ten reading-comprehension levels a book (or a reader's own
/// self-reported level) can be — 1 (easiest) through 10 (hardest). See
/// README's Reading Levels table for the full CEFR mapping. Single
/// source of truth, shared between Book.readingLevelName (a book's
/// assigned level) and SettingsScreen's reading-level dropdown (a
/// reader's own self-reported level — see UserSettings.readingLevel).
const readingLevelNames = {
  1: 'Initiate',
  2: 'Novice',
  3: 'Apprentice',
  4: 'Reader',
  5: 'Bookworm',
  6: 'Erudite',
  7: 'Virtuoso',
  8: 'Luminary',
  9: 'Academic',
  10: 'Scholar',
};
