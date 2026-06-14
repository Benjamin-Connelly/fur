package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigDirNew returns ~/.config/fur when no legacy path exists.
func TestConfigDirNew(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "") // ConfigDir uses ~/.config/fur directly

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config", "fur")
	if got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
}

// TestConfigDirMigratesLegacy renames ~/.config/lookit to ~/.config/fur.
func TestConfigDirMigratesLegacy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	legacy := filepath.Join(home, ".config", "lookit")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.yaml"), []byte("theme: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config", "fur")
	if got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
	// The migrated file should now live under the new path.
	if _, err := os.Stat(filepath.Join(want, "config.yaml")); err != nil {
		t.Errorf("legacy config not migrated: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy dir should be gone after migration")
	}
}
