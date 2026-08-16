/// A book available to read. Mirrors book-content's book response
/// exactly (GET /aidoku/books, /aidoku/book/{book_id}) — see
/// book-content/internal/db's Book struct. Deliberately doesn't carry
/// its chunk ids: that's a separate call (GET
/// /aidoku/book/{book_id}/chunks), not part of this response, so this
/// type stays a 1:1 match for either endpoint rather than two
/// different shapes depending on which one produced it. See
/// AIDOKU_DESIGN.md §4.
class Book {
  final int id;
  final int gutenbergId;
  final String title;
  final String author;
  final String sourceUrl;

  /// 1 (easiest) to 10 (hardest) — see readingLevelName for the
  /// human-facing name (README's Reading Levels table).
  final int level;

  /// ISO 639-1 codes (e.g. "en", "ja"). targetLanguage is what the
  /// reader is studying (this book's text is written in it);
  /// nativeLanguage is the reader's own language (questions/breakdowns
  /// are written in it) — see pipeline/internal/langpair, the
  /// pipeline-side source of truth for what a "language pair" means.
  /// Neither is used for anything in the app yet — just carried through
  /// so this type stays a 1:1 match for book-content's response.
  final String targetLanguage;
  final String nativeLanguage;
  final String status;

  const Book({
    required this.id,
    required this.gutenbergId,
    required this.title,
    required this.author,
    required this.sourceUrl,
    required this.level,
    required this.targetLanguage,
    required this.nativeLanguage,
    required this.status,
  });

  factory Book.fromJson(Map<String, dynamic> json) {
    return Book(
      id: json['id'] as int,
      gutenbergId: json['gutenberg_id'] as int,
      title: json['title'] as String,
      author: json['author'] as String,
      sourceUrl: json['source_url'] as String,
      level: json['level'] as int,
      targetLanguage: json['target_language'] as String,
      nativeLanguage: json['native_language'] as String,
      status: json['status'] as String,
    );
  }

  /// The human-facing name for [level] — README's Reading Levels table
  /// (Initiate..Scholar), kept in sync by hand since it's just a display
  /// label, not something book-content's API sends over the wire.
  String get readingLevelName => _readingLevelNames[level] ?? 'Level $level';
}

const _readingLevelNames = {
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
