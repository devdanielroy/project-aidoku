import '../models/user_settings.dart';

/// Where the reader's top-level settings (see UserSettings) live — same
/// consumer-defined-interface pattern as ProgressStore/ScoreStore (see
/// ProgressStore's own doc comment for the rationale): LocalSettingsStore
/// is the only implementation for now, but the shape is chosen so an
/// account-backed implementation (real sign-in, settings synced across
/// devices) can swap in later without LibraryScreen/SettingsScreen
/// changing at all.
///
/// Unlike ProgressStore/ScoreStore, this isn't keyed per book — there's
/// only ever one UserSettings per device (pre-accounts), so get/save
/// take and return the whole thing rather than individual fields.
abstract class SettingsStore {
  /// The reader's current settings — UserSettings' all-default,
  /// nothing-chosen-yet value if nothing has been saved.
  Future<UserSettings> getSettings();

  /// Overwrites the reader's settings with settings in full — not a
  /// partial update. Callers (SettingsScreen) always have the complete,
  /// already-loaded UserSettings to hand, edited in place, so there's
  /// no need for field-level setters here.
  Future<void> saveSettings(UserSettings settings);
}
