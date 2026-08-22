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

  /// Supabase project URL and publishable API key (see
  /// lib/data/supabase_auth_repository.dart) — used for account auth
  /// only; app data stays in this project's own Postgres (see
  /// db/schema.sql), not Supabase's. The publishable key is safe to
  /// bake in here despite being called a "key" — unlike a secret/
  /// service_role key, it's explicitly designed to be public in client
  /// code (see the Supabase project's own dashboard, which labels it as
  /// such); real access control happens at the Supabase project level,
  /// not by hiding this value. Defaults match this project's actual
  /// Supabase project — see .env (read by the Go backend only, not
  /// Flutter — kept here too just as the one place both sides' config
  /// is recorded) — override for a different project the same way as
  /// BOOK_CONTENT_BASE_URL.
  static const String supabaseUrl = String.fromEnvironment(
    'SUPABASE_URL',
    defaultValue: 'https://htyumvxkfavkhawvqilr.supabase.co',
  );

  static const String supabasePublishableKey = String.fromEnvironment(
    'SUPABASE_PUBLISHABLE_KEY',
    defaultValue: 'sb_publishable_TRcNS0HUwcQkgVYRQgwVIw_e_hTtKx6',
  );
}
