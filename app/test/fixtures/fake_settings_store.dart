import 'package:aidoku/data/settings_store.dart';
import 'package:aidoku/models/user_settings.dart';

/// An in-memory SettingsStore for widget tests — same rationale as
/// FakeProgressStore/FakeScoreStore. Optionally seeded, to test screens
/// that read settings without driving SettingsScreen's own save flow.
class FakeSettingsStore implements SettingsStore {
  UserSettings _settings;

  FakeSettingsStore({UserSettings? seed})
    : _settings = seed ?? const UserSettings();

  @override
  Future<UserSettings> getSettings() async => _settings;

  @override
  Future<void> saveSettings(UserSettings settings) async {
    _settings = settings;
  }
}
