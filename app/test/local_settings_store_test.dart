// LocalSettingsStore in isolation, against real shared_preferences (its
// platform channel mocked via setMockInitialValues — same approach as
// local_score_store_test.dart/progress_store_test.dart). See
// settings_screen_test.dart / library_screen_test.dart for
// SettingsStore actually wired into a screen.

import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:aidoku/data/local_settings_store.dart';
import 'package:aidoku/models/user_settings.dart';

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  test('getSettings is all-default when nothing has been saved', () async {
    final store = LocalSettingsStore();
    final settings = await store.getSettings();
    expect(settings.username, '');
    expect(settings.nativeLanguage, isNull);
    expect(settings.studyLanguages, isEmpty);
    expect(settings.activeStudyLanguage, isNull);
    expect(settings.themeMode, ThemeModeSetting.system);
  });

  test('themeMode round-trips for both light and dark', () async {
    final store = LocalSettingsStore();

    await store.saveSettings(
      const UserSettings(themeMode: ThemeModeSetting.dark),
    );
    expect((await store.getSettings()).themeMode, ThemeModeSetting.dark);

    await store.saveSettings(
      const UserSettings(themeMode: ThemeModeSetting.light),
    );
    expect((await store.getSettings()).themeMode, ThemeModeSetting.light);
  });

  test(
    'a stale/unrecognized stored theme_mode value falls back to system',
    () async {
      SharedPreferences.setMockInitialValues({
        'settings.theme_mode': 'sepia', // not a real ThemeModeSetting
      });
      final store = LocalSettingsStore();
      expect((await store.getSettings()).themeMode, ThemeModeSetting.system);
    },
  );

  test('saveSettings then getSettings round-trips every field', () async {
    final store = LocalSettingsStore();
    await store.saveSettings(
      const UserSettings(
        username: 'Roi',
        nativeLanguage: 'en',
        studyLanguages: ['ja'],
        activeStudyLanguage: 'ja',
      ),
    );

    final settings = await store.getSettings();
    expect(settings.username, 'Roi');
    expect(settings.nativeLanguage, 'en');
    expect(settings.studyLanguages, ['ja']);
    expect(settings.activeStudyLanguage, 'ja');
  });

  test('saveSettings overwrites a previously saved value', () async {
    final store = LocalSettingsStore();
    await store.saveSettings(
      const UserSettings(nativeLanguage: 'en', activeStudyLanguage: 'ja'),
    );
    await store.saveSettings(
      const UserSettings(nativeLanguage: 'ja', activeStudyLanguage: 'en'),
    );

    final settings = await store.getSettings();
    expect(settings.nativeLanguage, 'ja');
    expect(settings.activeStudyLanguage, 'en');
  });

  test(
    'saveSettings can clear a previously saved language back to null',
    () async {
      final store = LocalSettingsStore();
      await store.saveSettings(
        const UserSettings(nativeLanguage: 'en', activeStudyLanguage: 'ja'),
      );
      await store.saveSettings(const UserSettings());

      final settings = await store.getSettings();
      expect(settings.nativeLanguage, isNull);
      expect(settings.activeStudyLanguage, isNull);
      expect(settings.studyLanguages, isEmpty);
    },
  );

  test('multiple enrolled study languages round-trip in order', () async {
    final store = LocalSettingsStore();
    await store.saveSettings(
      const UserSettings(
        nativeLanguage: 'en',
        studyLanguages: ['ja', 'es'],
        activeStudyLanguage: 'ja',
      ),
    );

    final settings = await store.getSettings();
    expect(settings.studyLanguages, ['ja', 'es']);
  });
}
