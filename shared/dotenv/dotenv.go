// Package dotenv is a minimal .env file loader for command entrypoints
// that need local credentials (ANTHROPIC_API_KEY, POSTGRES_*) without a
// third-party dependency for something this small. Shared between
// pipeline/ and book-content/ (two separate Go modules, wired together via
// go.work at the repo root) — originally duplicated between them, then
// promoted here once a third near-identical copy (book-content/'s) made the
// duplication clearly not worth it.
package dotenv

import (
	"bufio"
	"os"
	"strings"
)

// Load does a minimal KEY=VALUE parse of the file at path, setting
// environment variables that aren't already set — never overriding a
// real env var with a stale .env value. Silently does nothing if path
// doesn't exist, since real env vars may be set instead.
func Load(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			os.Setenv(key, value)
		}
	}
}
