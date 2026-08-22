import 'package:flutter/material.dart';

import '../data/auth_repository.dart';
import '../data/book_content_repository.dart';
import '../data/local_settings_store.dart';
import '../data/settings_store.dart';
import '../data/supabase_auth_repository.dart';
import '../models/book.dart';
import '../models/language_pair.dart';
import '../models/reading_level.dart';
import '../models/user_settings.dart';
import 'account_screen.dart';

/// Lets the reader configure their top-level settings: display name,
/// native language, and which language(s) they're studying — see
/// UserSettings. Reachable from LibraryScreen's app bar at any time,
/// not just on first launch — there's no forced onboarding; a reader
/// with nothing configured yet just sees an empty-state prompt on
/// Library instead, with a button back here.
///
/// Native/study language options aren't hardcoded: they're the distinct
/// language pairs actually present among book-content's published
/// books (see availableLanguagePairs), fetched fresh every time this
/// screen opens — a new pair shows up here the moment it has real
/// published content, no app update required.
class SettingsScreen extends StatefulWidget {
  /// Overridable for tests — defaults to the real thing, same pattern
  /// as every other screen that talks to book-content.
  final BookContentRepository? repository;

  /// Overridable for tests — defaults to this device's local storage.
  /// See SettingsStore's own doc comment for why this is an interface.
  final SettingsStore? settingsStore;

  /// Overridable for tests — defaults to the real thing. See
  /// AuthRepository's own doc comment for why this is an interface;
  /// forwarded to the "My Account" row's AccountScreen.
  final AuthRepository? authRepository;

  /// Called with the saved settings' theme mode on every successful
  /// save (whether or not it actually changed — a redundant call is
  /// harmless). AidokuApp (main.dart) is the one thing that needs to
  /// react live: it owns MaterialApp's themeMode, several navigation
  /// levels above this screen, so there's no ancestor state it can just
  /// setState() on directly — this callback is threaded down through
  /// LibraryScreen instead. Optional because tests that don't care
  /// about live theme switching shouldn't have to supply a no-op.
  final ValueChanged<ThemeModeSetting>? onThemeModeChanged;

  const SettingsScreen({
    super.key,
    this.repository,
    this.settingsStore,
    this.authRepository,
    this.onThemeModeChanged,
  });

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

typedef _SettingsData = ({UserSettings settings, Set<LanguagePair> pairs});

class _SettingsScreenState extends State<SettingsScreen> {
  late final BookContentRepository _repository =
      widget.repository ?? BookContentRepository();
  late final SettingsStore _settingsStore =
      widget.settingsStore ?? LocalSettingsStore();
  late final Future<_SettingsData> _dataFuture = _load();

  Future<_SettingsData> _load() async {
    final results = await Future.wait([
      _settingsStore.getSettings(),
      _repository.getBooks(),
    ]);
    return (
      settings: results[0] as UserSettings,
      pairs: availableLanguagePairs(results[1] as List<Book>),
    );
  }

  Future<void> _save(UserSettings settings) async {
    await _settingsStore.saveSettings(settings);
    widget.onThemeModeChanged?.call(settings.themeMode);
    if (!mounted) return;
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: FutureBuilder<_SettingsData>(
        future: _dataFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(
              child: Text('Failed to load settings: ${snapshot.error}'),
            );
          }
          final data = snapshot.data!;
          return _SettingsForm(
            initial: data.settings,
            pairs: data.pairs,
            authRepository: widget.authRepository,
            onSave: _save,
          );
        },
      ),
    );
  }
}

class _SettingsForm extends StatefulWidget {
  final UserSettings initial;
  final Set<LanguagePair> pairs;
  final AuthRepository? authRepository;
  final Future<void> Function(UserSettings) onSave;

  const _SettingsForm({
    required this.initial,
    required this.pairs,
    required this.authRepository,
    required this.onSave,
  });

  @override
  State<_SettingsForm> createState() => _SettingsFormState();
}

class _SettingsFormState extends State<_SettingsForm> {
  late final TextEditingController _usernameController = TextEditingController(
    text: widget.initial.username,
  );
  late String? _nativeLanguage = widget.initial.nativeLanguage;
  late Set<String> _studyLanguages = widget.initial.studyLanguages.toSet();
  late String? _activeStudyLanguage = widget.initial.activeStudyLanguage;
  late ThemeModeSetting _themeMode = widget.initial.themeMode;
  late int? _readingLevel = widget.initial.readingLevel;
  bool _saving = false;

  @override
  void dispose() {
    _usernameController.dispose();
    super.dispose();
  }

  Set<String> get _nativeLanguageOptions => {
    for (final p in widget.pairs) p.native,
  };

  Set<String> get _studyLanguageOptions => {
    for (final p in widget.pairs)
      if (p.native == _nativeLanguage) p.target,
  };

  void _setNativeLanguage(String? code) {
    setState(() {
      _nativeLanguage = code;
      // Every existing enrollment was only ever valid against the old
      // native language's pairs — changing native invalidates all of
      // them at once, so there's nothing sound to carry forward.
      _studyLanguages = {};
      _activeStudyLanguage = null;
    });
  }

  void _toggleStudyLanguage(String code, bool enrolled) {
    setState(() {
      if (enrolled) {
        _studyLanguages.add(code);
        _activeStudyLanguage ??= code; // first enrollment auto-activates
      } else {
        _studyLanguages.remove(code);
        if (_activeStudyLanguage == code) {
          _activeStudyLanguage = _studyLanguages.isEmpty
              ? null
              : _studyLanguages.first;
        }
      }
    });
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    await widget.onSave(
      UserSettings(
        username: _usernameController.text.trim(),
        nativeLanguage: _nativeLanguage,
        studyLanguages: _studyLanguages.toList(),
        activeStudyLanguage: _activeStudyLanguage,
        themeMode: _themeMode,
        readingLevel: _readingLevel,
      ),
    );
    // Not resetting _saving on success — widget.onSave pops this screen
    // (see SettingsScreen._save), so there's nothing left here to
    // rebuild; only matters if onSave throws, and mounted still holds.
    if (mounted) setState(() => _saving = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Save is pinned outside the scrollable, not the last item in it —
    // the form has grown past one screen's worth of content (display
    // name, theme, native language, study languages, reading level),
    // and a button buried at the bottom of a long scroll is both poor
    // UX and untestable: ListView's underlying sliver only inflates
    // children within the viewport + cache extent into the element
    // tree at all, so a widget-testing tap this far down can't even
    // find "Save" to scroll to it, let alone tap it.
    return Column(
      children: [
        Expanded(child: _buildScrollableFields(theme)),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
            child: FilledButton(
              onPressed: _saving ? null : _save,
              child: const Text('Save'),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildScrollableFields(ThemeData theme) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _AccountSection(authRepository: widget.authRepository),
        const SizedBox(height: 24),
        Text('Display name', style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        TextField(
          controller: _usernameController,
          decoration: const InputDecoration(
            hintText: 'What should we call you?',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 24),
        Text('Theme', style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        SegmentedButton<ThemeModeSetting>(
          segments: const [
            ButtonSegment(
              value: ThemeModeSetting.system,
              label: Text('System'),
              icon: Icon(Icons.brightness_auto),
            ),
            ButtonSegment(
              value: ThemeModeSetting.light,
              label: Text('Light'),
              icon: Icon(Icons.light_mode),
            ),
            ButtonSegment(
              value: ThemeModeSetting.dark,
              label: Text('Dark'),
              icon: Icon(Icons.dark_mode),
            ),
          ],
          selected: {_themeMode},
          onSelectionChanged: (selected) =>
              setState(() => _themeMode = selected.single),
        ),
        const SizedBox(height: 24),
        Text('Native language', style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        if (_nativeLanguageOptions.isEmpty)
          const Text('No books published yet — nothing to choose from.')
        else
          DropdownButtonFormField<String>(
            initialValue: _nativeLanguage,
            decoration: const InputDecoration(border: OutlineInputBorder()),
            hint: const Text('Select your native language'),
            items: [
              for (final code in _nativeLanguageOptions)
                DropdownMenuItem(
                  value: code,
                  child: Text(languageDisplayName(code)),
                ),
            ],
            onChanged: _setNativeLanguage,
          ),
        const SizedBox(height: 24),
        Text("Languages you're studying", style: theme.textTheme.titleSmall),
        const SizedBox(height: 4),
        Text(
          _nativeLanguage == null
              ? 'Pick a native language first.'
              : "Enroll in the ones you're learning, and mark one active — "
                    'that\'s what shows up in your Library.',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 8),
        for (final code in _studyLanguageOptions)
          _StudyLanguageTile(
            code: code,
            enrolled: _studyLanguages.contains(code),
            active: _activeStudyLanguage == code,
            onToggleEnrolled: (value) => _toggleStudyLanguage(code, value),
            onSetActive: () => setState(() => _activeStudyLanguage = code),
          ),
        const SizedBox(height: 24),
        Text('Your reading level', style: theme.textTheme.titleSmall),
        const SizedBox(height: 4),
        Text(
          "Just for our own records — doesn't change what you see.",
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 8),
        DropdownButtonFormField<int?>(
          initialValue: _readingLevel,
          decoration: const InputDecoration(border: OutlineInputBorder()),
          items: [
            // A real, selectable item — not just a hint — so a reader
            // can return to "unset" after picking a level, not just on
            // the way to picking one the first time.
            const DropdownMenuItem(
              value: null,
              child: Text('Prefer not to say'),
            ),
            for (final level in readingLevelNames.keys)
              DropdownMenuItem(
                value: level,
                child: Text('${readingLevelNames[level]} ($level)'),
              ),
          ],
          onChanged: (level) => setState(() => _readingLevel = level),
        ),
      ],
    );
  }
}

/// One study-language row: a checkbox to enroll/unenroll, plus (only
/// once enrolled) a tap target to make it the active one — the language
/// LibraryScreen actually filters to. Combined into one row rather than
/// two separate lists so "enroll" and "make active" stay visibly tied
/// to the same language.
class _StudyLanguageTile extends StatelessWidget {
  final String code;
  final bool enrolled;
  final bool active;
  final ValueChanged<bool> onToggleEnrolled;
  final VoidCallback onSetActive;

  const _StudyLanguageTile({
    required this.code,
    required this.enrolled,
    required this.active,
    required this.onToggleEnrolled,
    required this.onSetActive,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: Checkbox(
          value: enrolled,
          onChanged: (value) => onToggleEnrolled(value ?? false),
        ),
        title: Text(languageDisplayName(code)),
        trailing: enrolled
            ? TextButton(
                onPressed: active ? null : onSetActive,
                child: Text(active ? 'Active' : 'Set active'),
              )
            : null,
        onTap: () => onToggleEnrolled(!enrolled),
      ),
    );
  }
}

/// "My Account" row — opens AccountScreen, showing the signed-in
/// email as a subtitle once there is one. Its own State (rather than
/// living inline in _SettingsForm) so returning from AccountScreen can
/// refresh just this row, same "await push, then refresh what changed"
/// pattern as LibraryScreen's _BookCard after ReadingSessionScreen.
class _AccountSection extends StatefulWidget {
  final AuthRepository? authRepository;

  const _AccountSection({required this.authRepository});

  @override
  State<_AccountSection> createState() => _AccountSectionState();
}

class _AccountSectionState extends State<_AccountSection> {
  late final AuthRepository _authRepository =
      widget.authRepository ?? const SupabaseAuthRepository();

  Future<void> _openAccount() async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => AccountScreen(authRepository: _authRepository),
      ),
    );
    // Refresh regardless of how the reader left (signed in, signed out,
    // or just backed out) - see _BookCard's own doc comment for why
    // this can't just be a plain setState() call site inside onTap.
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final user = _authRepository.currentUser;
    return Card(
      child: ListTile(
        leading: const Icon(Icons.account_circle),
        title: const Text('My Account'),
        subtitle: Text(user == null ? 'Not signed in' : 'Signed in as ${user.email}'),
        trailing: const Icon(Icons.chevron_right),
        onTap: _openAccount,
      ),
    );
  }
}
