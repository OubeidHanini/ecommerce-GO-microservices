package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretPrefersFileOverVariable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_PASSWORD_FILE", path)
	t.Setenv("DB_PASSWORD", "from-env")

	env := New()
	if got := env.Secret("DB_PASSWORD"); got != "from-file" {
		t.Errorf("Secret = %q, want %q (trailing newline trimmed)", got, "from-file")
	}
	if err := env.Err(); err != nil {
		t.Errorf("Err = %v, want nil", err)
	}
}

func TestErrListsEveryMissingSetting(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_PASSWORD_FILE", "")

	env := New()
	env.Required("DB_HOST")
	env.Secret("DB_PASSWORD")

	err := env.Err()
	if err == nil {
		t.Fatal("Err = nil, want an error")
	}
	for _, want := range []string{"DB_HOST is required", "DB_PASSWORD_FILE or DB_PASSWORD is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestSecretReportsUnreadableFile(t *testing.T) {
	t.Setenv("DB_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing"))

	env := New()
	env.Secret("DB_PASSWORD")
	if env.Err() == nil {
		t.Error("Err = nil, want an error for a missing secret file")
	}
}
