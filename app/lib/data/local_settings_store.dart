import 'package:shared_preferences/shared_preferences.dart';

import '../models/user_settings.dart';
import 'settings_store.dart';

/// SettingsStore backed by shared_preferences — this device's local
/// storage, not synced anywhere. See SettingsStore's own doc comment
/// for the intended migration path once there's a real user account.
///
/// Stored as individual typed keys, not one JSON blob — UserSettings
/// has a small, fixed shape (unlike ScoreStore's dynamic per-question
/// map, which does need JSON), so shared_preferences' own typed
/// getters/setters are simpler and avoid a serialization layer that
/// would only ever hold one object.
class LocalSettingsStore implements SettingsStore {
  static const _keyUsername = 'settings.username';
  static const _keyNativeLanguage = 'settings.native_language';
  static const _keyStudyLanguages = 'settings.study_languages';
  static const _keyActiveStudyLanguage = 'settings.active_study_language';

  @override
  Future<UserSettings> getSettings() async {
    final prefs = await SharedPreferences.getInstance();
    return UserSettings(
      username: prefs.getString(_keyUsername) ?? '',
      nativeLanguage: prefs.getString(_keyNativeLanguage),
      studyLanguages: prefs.getStringList(_keyStudyLanguages) ?? const [],
      activeStudyLanguage: prefs.getString(_keyActiveStudyLanguage),
    );
  }

  @override
  Future<void> saveSettings(UserSettings settings) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyUsername, settings.username);
    await _setOrRemove(prefs, _keyNativeLanguage, settings.nativeLanguage);
    await prefs.setStringList(_keyStudyLanguages, settings.studyLanguages);
    await _setOrRemove(
      prefs,
      _keyActiveStudyLanguage,
      settings.activeStudyLanguage,
    );
  }

  Future<void> _setOrRemove(
    SharedPreferences prefs,
    String key,
    String? value,
  ) {
    return value == null ? prefs.remove(key) : prefs.setString(key, value);
  }
}
