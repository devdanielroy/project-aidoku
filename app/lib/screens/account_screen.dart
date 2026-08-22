import 'package:flutter/material.dart';

import '../data/auth_repository.dart';
import '../data/supabase_auth_repository.dart';

/// "My Account" — reachable from SettingsScreen. Shows a sign-up/log-in
/// form when signed out, or the signed-in email + a sign-out button
/// when signed in. Auth itself happens directly against Supabase (see
/// AuthRepository's own doc comment) — nothing here talks to
/// book-content or any other service this app owns.
///
/// Deliberately doesn't gate anything yet: no feature in the app
/// actually checks whether a reader is signed in (see README's
/// user-data service milestone for what will — purchase entitlements,
/// account-backed progress sync). This screen is just the identity
/// piece on its own for now.
class AccountScreen extends StatefulWidget {
  /// Overridable for tests — defaults to the real thing, same pattern
  /// as every other screen with a real backing service.
  final AuthRepository? authRepository;

  const AccountScreen({super.key, this.authRepository});

  @override
  State<AccountScreen> createState() => _AccountScreenState();
}

class _AccountScreenState extends State<AccountScreen> {
  late final AuthRepository _authRepository =
      widget.authRepository ?? const SupabaseAuthRepository();

  @override
  Widget build(BuildContext context) {
    final user = _authRepository.currentUser;
    return Scaffold(
      appBar: AppBar(title: const Text('My Account')),
      body: user == null
          ? _SignedOutView(
              authRepository: _authRepository,
              onSignedIn: () => setState(() {}),
            )
          : _SignedInView(
              user: user,
              authRepository: _authRepository,
              onSignedOut: () => setState(() {}),
            ),
    );
  }
}

class _SignedInView extends StatefulWidget {
  final AuthUser user;
  final AuthRepository authRepository;
  final VoidCallback onSignedOut;

  const _SignedInView({
    required this.user,
    required this.authRepository,
    required this.onSignedOut,
  });

  @override
  State<_SignedInView> createState() => _SignedInViewState();
}

class _SignedInViewState extends State<_SignedInView> {
  bool _signingOut = false;

  Future<void> _signOut() async {
    setState(() => _signingOut = true);
    await widget.authRepository.signOut();
    if (mounted) widget.onSignedOut();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.account_circle,
              size: 64,
              color: theme.colorScheme.onSurfaceVariant,
            ),
            const SizedBox(height: 16),
            Text('Signed in as', style: theme.textTheme.bodyMedium),
            const SizedBox(height: 4),
            Text(widget.user.email, style: theme.textTheme.titleMedium),
            const SizedBox(height: 24),
            OutlinedButton(
              onPressed: _signingOut ? null : _signOut,
              child: const Text('Sign Out'),
            ),
          ],
        ),
      ),
    );
  }
}

class _SignedOutView extends StatefulWidget {
  final AuthRepository authRepository;
  final VoidCallback onSignedIn;

  const _SignedOutView({required this.authRepository, required this.onSignedIn});

  @override
  State<_SignedOutView> createState() => _SignedOutViewState();
}

class _SignedOutViewState extends State<_SignedOutView> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _submitting = false;
  String? _error;

  // Set once a sign-up succeeds without immediately signing in - the
  // Supabase project requires the reader to confirm their email first
  // (a project-level setting, not something this app controls; see
  // AuthRepository.signUp's own doc comment).
  bool _awaitingEmailConfirmation = false;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _signIn() async {
    await _submit(() async {
      await widget.authRepository.signIn(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      );
      return true;
    });
  }

  Future<void> _signUp() async {
    await _submit(
      () => widget.authRepository.signUp(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
    );
  }

  Future<void> _submit(Future<bool> Function() action) async {
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final signedIn = await action();
      if (!mounted) return;
      if (signedIn) {
        widget.onSignedIn();
        return; // AccountScreen rebuilds into _SignedInView - nothing
        // left here to reset _submitting on.
      }
      setState(() => _awaitingEmailConfirmation = true);
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_awaitingEmailConfirmation) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Check your email to confirm your account, then log in.',
            textAlign: TextAlign.center,
            style: theme.textTheme.bodyLarge,
          ),
        ),
      );
    }
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _emailController,
            decoration: const InputDecoration(
              labelText: 'Email',
              border: OutlineInputBorder(),
            ),
            keyboardType: TextInputType.emailAddress,
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _passwordController,
            decoration: const InputDecoration(
              labelText: 'Password',
              border: OutlineInputBorder(),
            ),
            obscureText: true,
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
          ],
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _submitting ? null : _signIn,
            child: const Text('Log In'),
          ),
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: _submitting ? null : _signUp,
            child: const Text('Create Account'),
          ),
        ],
      ),
    );
  }
}
