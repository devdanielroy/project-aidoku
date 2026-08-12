package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_SetsUnsetVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("# a comment\nFOO=bar\n\nBAZ=qux\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	t.Setenv("FOO", "")
	os.Unsetenv("FOO")
	t.Setenv("BAZ", "")
	os.Unsetenv("BAZ")

	Load(path)
	defer os.Unsetenv("FOO")
	defer os.Unsetenv("BAZ")

	if got := os.Getenv("FOO"); got != "bar" {
		t.Errorf("FOO = %q, want %q", got, "bar")
	}
	if got := os.Getenv("BAZ"); got != "qux" {
		t.Errorf("BAZ = %q, want %q", got, "qux")
	}
}

func TestLoad_DoesNotOverrideAlreadySetVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("FOO=from-file\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	t.Setenv("FOO", "from-real-env")
	Load(path)

	if got := os.Getenv("FOO"); got != "from-real-env" {
		t.Errorf("FOO = %q, want the real env value to win: %q", got, "from-real-env")
	}
}

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	// Just confirming this doesn't panic — there's nothing else to
	// assert when the file doesn't exist.
	Load(filepath.Join(t.TempDir(), "does-not-exist.env"))
}
