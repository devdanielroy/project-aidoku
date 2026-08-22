# CLAUDE.md

Session-handoff notes for future Claude Code sessions on this repo. See
[README.md](./README.md) for the actual architecture and milestone
list — this file is just for picking a thread back up, not permanent
documentation.

## Next steps (as of 2026-08-23)

Account auth (Supabase) is working end to end on macOS: sign up, email
confirmation (via a `com.aidoku.aidoku://login-callback` deep link),
log in, log out, all reachable from a "My Account" row at the top of
Settings. Email/password only for now — Google/Apple aren't wired up.

Still open, roughly in the order they'd naturally come up next:

- **iOS/Android deep-link config** — the `com.aidoku.aidoku://login-callback`
  URL scheme is only registered on macOS so far
  (`app/macos/Runner/Info.plist`'s `CFBundleURLTypes`). iOS needs the
  same in `app/ios/Runner/Info.plist`; Android needs an intent filter
  in `AndroidManifest.xml`. Deferred until those platforms are actually
  targeted — no need to do it preemptively.
- **Google Sign-In / Sign in with Apple** — deliberately deferred past
  this first pass (email/password only, see README's "User accounts"
  milestone). Each needs its own native OAuth setup; Supabase supports
  both once that's done.
- **`user-data` Go service** — doesn't exist yet. Needed once there's
  real server-side data to attach to a signed-in user: purchase
  entitlements and account-backed progress/score/settings sync (see
  README's `user-data service` milestone). Would live in its own
  tables in the app's *own* Postgres, keyed by Supabase's user id as a
  plain UUID column (no cross-database foreign key — Supabase's own
  Postgres, where `auth.users` actually lives, stays a separate,
  untouched database; see AuthRepository's own doc comment for why).
- **Content complaint/report button** — same README milestone, not
  started.
