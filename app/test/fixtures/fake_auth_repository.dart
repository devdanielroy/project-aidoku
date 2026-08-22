import 'package:aidoku/data/auth_repository.dart';

/// An in-memory AuthRepository for widget tests — no real Supabase
/// call, same rationale as FakeProgressStore/FakeScoreStore/
/// FakeSettingsStore. Optionally seeded as already-signed-in.
class FakeAuthRepository implements AuthRepository {
  AuthUser? _currentUser;

  /// Emails that fail signUp/signIn with the given message - lets a
  /// test exercise the error path without a real Supabase call.
  final Map<String, String> failSignUpFor;
  final Map<String, String> failSignInFor;

  /// When true, signUp "succeeds" without signing in - same as a real
  /// Supabase project with "Confirm email" turned on (see
  /// AuthRepository.signUp's own doc comment).
  bool requireEmailConfirmation;

  FakeAuthRepository({
    AuthUser? initialUser,
    this.failSignUpFor = const {},
    this.failSignInFor = const {},
    this.requireEmailConfirmation = false,
  }) : _currentUser = initialUser;

  @override
  AuthUser? get currentUser => _currentUser;

  @override
  Future<bool> signUp({required String email, required String password}) async {
    if (failSignUpFor.containsKey(email)) {
      throw AuthRepositoryException(failSignUpFor[email]!);
    }
    if (requireEmailConfirmation) return false;
    _currentUser = (id: 'fake-$email', email: email);
    return true;
  }

  @override
  Future<void> signIn({required String email, required String password}) async {
    if (failSignInFor.containsKey(email)) {
      throw AuthRepositoryException(failSignInFor[email]!);
    }
    _currentUser = (id: 'fake-$email', email: email);
  }

  @override
  Future<void> signOut() async {
    _currentUser = null;
  }
}
