package config

import (
	"path/filepath"
	"testing"
)

// TestConfigDirNew returns ~/.config/fur.
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
