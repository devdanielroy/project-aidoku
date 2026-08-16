import 'book.dart';
import 'user_settings.dart';

/// A (target, native) language pair — ISO 639-1 codes, same shape as
/// Book.targetLanguage/nativeLanguage. A record rather than a class:
/// pure data, structurally equal (two pairs with the same codes compare
/// equal, which availableLanguagePairs' Set relies on), no behavior.
typedef LanguagePair = ({String target, String native});

/// Every distinct language pair actually present among [books] — the
/// source of truth SettingsScreen builds its pickers from, instead of a
/// hardcoded language list. Means a new pair shows up in the app the
/// moment book-content starts serving books for it, with no Flutter
/// code change required — same "derive it from data, don't hardcode a
/// language" instinct the pipeline side (internal/langpair) is built
/// around.
Set<LanguagePair> availableLanguagePairs(Iterable<Book> books) {
  return {
    for (final b in books) (target: b.targetLanguage, native: b.nativeLanguage),
  };
}

/// Human-readable name for an ISO 639-1 code, for display only — every
/// stored/compared value stays a code (mirrors pipeline/internal/
/// langpair.DisplayName's split between the code and the name shown in
/// prose). Falls back to the code itself if unmapped, same as the Go
/// side.
String languageDisplayName(String code) => _languageDisplayNames[code] ?? code;

const _languageDisplayNames = {'en': 'English', 'ja': 'Japanese'};

/// settings, unless nothing has been explicitly chosen yet (no native
/// language and no active study language saved) and [books]
/// collectively offer exactly one language pair — in that case, adopts
/// that pair rather than making a first-time reader visit Settings just
/// to confirm the one option that exists. Never overrides a real,
/// explicit choice, even one that currently matches zero published
/// books (see LibraryScreen's own empty state for that case) — this
/// only ever fills in a genuinely blank slate, and the moment a second
/// pair exists it stops applying, so the picker becomes a real choice
/// again automatically. Doesn't persist anything — SettingsStore is
/// only ever written to by an explicit save in SettingsScreen.
UserSettings resolveActiveSettings(UserSettings settings, List<Book> books) {
  if (settings.nativeLanguage != null || settings.activeStudyLanguage != null) {
    return settings;
  }
  final pairs = availableLanguagePairs(books);
  if (pairs.length != 1) return settings;
  final only = pairs.single;
  return settings.copyWith(
    nativeLanguage: only.native,
    studyLanguages: [only.target],
    activeStudyLanguage: only.target,
  );
}
