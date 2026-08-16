import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/models/book.dart';
import 'package:aidoku/models/language_pair.dart';
import 'package:aidoku/models/user_settings.dart';

Book _book({required String target, required String native}) {
  return Book(
    id: 1,
    gutenbergId: 1,
    title: 'Test',
    author: 'Test',
    sourceUrl: 'https://example.com',
    level: 1,
    targetLanguage: target,
    nativeLanguage: native,
    status: 'published',
  );
}

void main() {
  group('availableLanguagePairs', () {
    test('is empty for no books', () {
      expect(availableLanguagePairs([]), isEmpty);
    });

    test('collapses multiple books sharing the same pair into one entry', () {
      final books = [
        _book(target: 'en', native: 'ja'),
        _book(target: 'en', native: 'ja'),
      ];
      expect(availableLanguagePairs(books), {(target: 'en', native: 'ja')});
    });

    test('returns every distinct pair present', () {
      final books = [
        _book(target: 'en', native: 'ja'),
        _book(target: 'ja', native: 'en'),
      ];
      expect(availableLanguagePairs(books), {
        (target: 'en', native: 'ja'),
        (target: 'ja', native: 'en'),
      });
    });
  });

  group('languageDisplayName', () {
    test('resolves known ISO codes', () {
      expect(languageDisplayName('en'), 'English');
      expect(languageDisplayName('ja'), 'Japanese');
    });

    test('falls back to the code itself when unmapped', () {
      expect(languageDisplayName('es'), 'es');
    });
  });

  group('resolveActiveSettings', () {
    test('auto-adopts the only available pair when nothing is set', () {
      final books = [_book(target: 'en', native: 'ja')];
      final resolved = resolveActiveSettings(const UserSettings(), books);
      expect(resolved.nativeLanguage, 'ja');
      expect(resolved.activeStudyLanguage, 'en');
      expect(resolved.studyLanguages, ['en']);
    });

    test('leaves settings untouched when 2+ pairs are available', () {
      final books = [
        _book(target: 'en', native: 'ja'),
        _book(target: 'ja', native: 'en'),
      ];
      const settings = UserSettings();
      expect(resolveActiveSettings(settings, books), same(settings));
    });

    test('leaves settings untouched when no books exist at all', () {
      const settings = UserSettings();
      expect(resolveActiveSettings(settings, []), same(settings));
    });

    test('never overrides an explicit nativeLanguage choice', () {
      final books = [_book(target: 'en', native: 'ja')];
      const settings = UserSettings(nativeLanguage: 'en');
      expect(resolveActiveSettings(settings, books), same(settings));
    });

    test(
      'never overrides an explicit choice even if it matches zero books',
      () {
        final books = [_book(target: 'en', native: 'ja')];
        const settings = UserSettings(
          nativeLanguage: 'ja',
          activeStudyLanguage: 'es',
        );
        expect(resolveActiveSettings(settings, books), same(settings));
      },
    );

    test('preserves username when auto-adopting a pair', () {
      final books = [_book(target: 'en', native: 'ja')];
      final resolved = resolveActiveSettings(
        const UserSettings(username: 'Roi'),
        books,
      );
      expect(resolved.username, 'Roi');
    });
  });
}
