// AccountScreen in isolation - sign up, log in, sign out, and the
// email-confirmation-pending state, all against a FakeAuthRepository
// rather than real Supabase (see its own doc comment for why).

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/screens/account_screen.dart';

import 'fixtures/fake_auth_repository.dart';

void main() {
  Future<void> pump(WidgetTester tester, FakeAuthRepository authRepository) =>
      tester.pumpWidget(
        MaterialApp(home: AccountScreen(authRepository: authRepository)),
      );

  testWidgets('signed out: shows the sign-up/log-in form', (
    WidgetTester tester,
  ) async {
    await pump(tester, FakeAuthRepository());
    await tester.pumpAndSettle();

    expect(find.widgetWithText(TextField, 'Email'), findsOneWidget);
    expect(find.widgetWithText(TextField, 'Password'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Log In'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Create Account'), findsOneWidget);
  });

  testWidgets('creating an account signs in immediately when no email '
      'confirmation is required', (WidgetTester tester) async {
    final auth = FakeAuthRepository();
    await pump(tester, auth);
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Email'),
      'new@example.com',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Password'),
      'hunter2',
    );
    await tester.tap(find.widgetWithText(OutlinedButton, 'Create Account'));
    await tester.pumpAndSettle();

    expect(find.text('Signed in as'), findsOneWidget);
    expect(find.text('new@example.com'), findsOneWidget);
  });

  testWidgets(
    'creating an account shows a confirmation-pending message when the '
    'project requires it',
    (WidgetTester tester) async {
      final auth = FakeAuthRepository(requireEmailConfirmation: true);
      await pump(tester, auth);
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextField, 'Email'),
        'new@example.com',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Password'),
        'hunter2',
      );
      await tester.tap(find.widgetWithText(OutlinedButton, 'Create Account'));
      await tester.pumpAndSettle();

      expect(find.textContaining('Check your email'), findsOneWidget);
      // Not signed in yet - still pending confirmation.
      expect(auth.currentUser, isNull);
    },
  );

  testWidgets('sign-up failure shows the error inline, not a crash', (
    WidgetTester tester,
  ) async {
    final auth = FakeAuthRepository(
      failSignUpFor: {'taken@example.com': 'Email already registered'},
    );
    await pump(tester, auth);
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Email'),
      'taken@example.com',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Password'),
      'hunter2',
    );
    await tester.tap(find.widgetWithText(OutlinedButton, 'Create Account'));
    await tester.pumpAndSettle();

    expect(find.text('Email already registered'), findsOneWidget);
  });

  testWidgets('logging in signs in with the entered credentials', (
    WidgetTester tester,
  ) async {
    final auth = FakeAuthRepository();
    await pump(tester, auth);
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Email'),
      'existing@example.com',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Password'),
      'hunter2',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Log In'));
    await tester.pumpAndSettle();

    expect(find.text('existing@example.com'), findsOneWidget);
  });

  testWidgets('log-in failure shows the error inline', (
    WidgetTester tester,
  ) async {
    final auth = FakeAuthRepository(
      failSignInFor: {'wrong@example.com': 'Invalid login credentials'},
    );
    await pump(tester, auth);
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Email'),
      'wrong@example.com',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Password'),
      'bad',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Log In'));
    await tester.pumpAndSettle();

    expect(find.text('Invalid login credentials'), findsOneWidget);
  });

  testWidgets('signed in: shows the email and a Sign Out button that '
      'signs out', (WidgetTester tester) async {
    await pump(
      tester,
      FakeAuthRepository(initialUser: (id: '1', email: 'me@example.com')),
    );
    await tester.pumpAndSettle();

    expect(find.text('me@example.com'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Sign Out'), findsOneWidget);

    await tester.tap(find.widgetWithText(OutlinedButton, 'Sign Out'));
    await tester.pumpAndSettle();

    // Back to the signed-out form.
    expect(find.widgetWithText(TextField, 'Email'), findsOneWidget);
  });
}
