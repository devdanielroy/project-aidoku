// Hides supabase_flutter's own AuthUser - this file's AuthUser (from
// auth_repository.dart) is the one meant to be visible here.
import 'package:supabase_flutter/supabase_flutter.dart' hide AuthUser;

import 'auth_repository.dart';

/// Where Supabase sends a reader back to after they tap the link in a
/// sign-up confirmation email — a custom URL scheme (registered in
/// macos/Runner/Info.plist's CFBundleURLTypes, matching the app's own
/// bundle id) so the OS hands control back to this app instead of
/// trying to open Supabase's default (a dead localhost:3000 meant for
/// a web app). Must also be added to the Supabase project's own
/// Authentication → URL Configuration → Redirect URLs allowlist —
/// Supabase rejects any redirect target not on that list.
const _emailConfirmationRedirect = 'com.aidoku.aidoku://login-callback';

/// AuthRepository backed by Supabase Auth — the real implementation,
/// used everywhere widget tests don't inject a fake (see
/// test/fixtures/fake_auth_repository.dart). Talks to
/// Supabase.instance.client directly rather than taking a client in the
/// constructor the way BookContentRepository takes an http.Client:
/// Supabase's own SDK is already the thing being faked out in tests
/// (via AuthRepository itself), so there's no need for a second layer
/// of client injection here.
class SupabaseAuthRepository implements AuthRepository {
  const SupabaseAuthRepository();

  AuthUser? _toAuthUser(User? user) =>
      user == null ? null : (id: user.id, email: user.email ?? '');

  @override
  AuthUser? get currentUser {
    try {
      return _toAuthUser(Supabase.instance.client.auth.currentUser);
    } catch (_) {
      // Supabase.initialize() hasn't run - true in every widget test,
      // which never calls main() (see AidokuApp's own entry point).
      // Passively reading "who's signed in" should degrade to "nobody"
      // rather than crash every screen that happens to show this, same
      // as any other screen's optional dependency defaulting quietly
      // when unset. Only currentUser gets this treatment — signUp/
      // signIn/signOut are only ever reached by an explicit tap on
      // AccountScreen, so a real misconfiguration there should still
      // fail loud, not silently no-op.
      return null;
    }
  }

  @override
  Future<bool> signUp({required String email, required String password}) async {
    try {
      final response = await Supabase.instance.client.auth.signUp(
        email: email,
        password: password,
        emailRedirectTo: _emailConfirmationRedirect,
      );
      return response.session != null;
    } on AuthException catch (e) {
      throw AuthRepositoryException(e.message);
    }
  }

  @override
  Future<void> signIn({required String email, required String password}) async {
    try {
      await Supabase.instance.client.auth.signInWithPassword(
        email: email,
        password: password,
      );
    } on AuthException catch (e) {
      throw AuthRepositoryException(e.message);
    }
  }

  @override
  Future<void> signOut() => Supabase.instance.client.auth.signOut();
}
