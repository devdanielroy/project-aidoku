/// A signed-in reader's account identity — just enough for the UI to
/// show who's signed in ("My Account"'s subtitle, AccountScreen's
/// "Signed in as ..."). Deliberately not Supabase's own User type: a
/// small first-party record keeps this interface's abstraction clean,
/// same as every other repository here (BookContentException rather
/// than a raw http exception; Book rather than a raw JSON map) — a
/// future non-Supabase auth backend wouldn't need to fake Supabase's
/// types just to implement AuthRepository.
typedef AuthUser = ({String id, String email});

/// Thrown by signUp/signIn on failure (bad password, email already
/// registered, wrong credentials, etc.) — message is written to be
/// reasonable to show a user directly, same as BookContentException.
class AuthRepositoryException implements Exception {
  final String message;

  const AuthRepositoryException(this.message);

  @override
  String toString() => message;
}

/// Where account authentication happens — sign up, log in, log out, and
/// who (if anyone) is currently signed in. Same consumer-defined-
/// interface pattern as ProgressStore/ScoreStore/SettingsStore (see
/// ProgressStore's own doc comment for the rationale):
/// SupabaseAuthRepository is the only real implementation, talking to
/// Supabase Auth directly — sign-up/login/session all happen client-
/// side against Supabase, not through book-content or any other
/// service this app owns. See the README's user-data service milestone
/// for what *does* eventually route through a real backend
/// (entitlements, account-backed progress sync) — this interface only
/// covers identity, not that.
abstract class AuthRepository {
  /// Null if nobody's signed in.
  AuthUser? get currentUser;

  /// Returns true if the new account is signed in immediately, false if
  /// the Supabase project requires email confirmation first (a project-
  /// level setting, not something this app controls) — callers show a
  /// "check your email" state in the false case rather than treating it
  /// as already signed in.
  Future<bool> signUp({required String email, required String password});

  Future<void> signIn({required String email, required String password});

  Future<void> signOut();
}
