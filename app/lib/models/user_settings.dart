/// A reader's top-level, device-local settings: which language they're
/// studying and what language they study it in, plus a display name.
/// Analogous to Duolingo's course picker — nativeLanguage/
/// activeStudyLanguage together name one of book-content's published
/// language pairs (see pipeline/internal/langpair — Target/Native are
/// ISO 639-1 codes there too, and this deliberately reuses that
/// terminology), and LibraryScreen filters its book list to whichever
/// pair is active. See SettingsStore for where this is persisted.
class UserSettings {
  final String username;

  /// ISO 639-1 code (e.g. "en", "ja"), or null before the reader has
  /// picked one — see LibraryScreen's empty state for what that looks
  /// like.
  final String? nativeLanguage;

  /// ISO 639-1 codes the reader is enrolled in studying. Can hold more
  /// than one, Duolingo-course-style — but only activeStudyLanguage
  /// actually drives what Library shows at any given time. Always
  /// expected to contain activeStudyLanguage when the latter is
  /// non-null; SettingsScreen is what enforces that invariant (see its
  /// own doc comment), not this type.
  final List<String> studyLanguages;

  /// Which of studyLanguages is currently active, or null if none is
  /// (including whenever studyLanguages is empty).
  final String? activeStudyLanguage;

  const UserSettings({
    this.username = '',
    this.nativeLanguage,
    this.studyLanguages = const [],
    this.activeStudyLanguage,
  });

  UserSettings copyWith({
    String? username,
    String? nativeLanguage,
    List<String>? studyLanguages,
    String? activeStudyLanguage,
  }) {
    return UserSettings(
      username: username ?? this.username,
      nativeLanguage: nativeLanguage ?? this.nativeLanguage,
      studyLanguages: studyLanguages ?? this.studyLanguages,
      activeStudyLanguage: activeStudyLanguage ?? this.activeStudyLanguage,
    );
  }
}
