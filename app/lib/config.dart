/// App-wide configuration, resolved at build time via `--dart-define` —
/// not a runtime-editable settings screen. That's the right shape for
/// what's here so far (which backend to talk to): it's decided once,
/// at release time, not by the end user, and dart-define values are
/// baked into the compiled binary with no bundled config file or
/// secret to ship.
///
/// Local dev needs nothing extra — the default already points at the
/// book-content service's local Docker port (see docker-compose.yml).
/// A real deployment (book-content on AWS Fargate, per the plan) will
/// pass its own value:
///
///   flutter run --dart-define=BOOK_CONTENT_BASE_URL=https://api.example.com
///   flutter build macos --dart-define=BOOK_CONTENT_BASE_URL=https://api.example.com
class AppConfig {
  AppConfig._(); // static-only, never instantiated

  /// Base URL for the book-content service (see /book-content at the
  /// repo root) — no trailing slash. Every route it serves starts with
  /// /aidoku/..., not included here so callers write the full path
  /// themselves (matches the route list in book-content/internal/api).
  static const String bookContentBaseUrl = String.fromEnvironment(
    'BOOK_CONTENT_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );
}
