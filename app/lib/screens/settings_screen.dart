import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/local_settings_store.dart';
import '../data/settings_store.dart';
import '../models/book.dart';
import '../models/language_pair.dart';
import '../models/user_settings.dart';

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

  const SettingsScreen({super.key, this.repository, this.settingsStore});

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
  final Future<void> Function(UserSettings) onSave;

  const _SettingsForm({
    required this.initial,
    required this.pairs,
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
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
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
        const SizedBox(height: 32),
        FilledButton(
          onPressed: _saving ? null : _save,
          child: const Text('Save'),
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
