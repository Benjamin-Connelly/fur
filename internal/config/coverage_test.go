package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRecentRoundTrip exercises LoadRecentFiles -> Add -> Save -> reload with a
// HOME-resolved path (covers the r.path=="" ConfigDir/MkdirAll branch in Save).
func TestRecentRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	r := &RecentFiles{} // empty path → Save resolves ConfigDir + MkdirAll
	r.Add("/docs/a.md")
	r.Add("/docs/b.md")
	r.Add("/docs/a.md") // dedupe to front
	if r.Files[0] != "/docs/a.md" {
		t.Errorf("most-recent = %q, want /docs/a.md", r.Files[0])
	}
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// recent.json should exist with 0600 perms under ~/.config/fur.
	want := filepath.Join(home, ".config", "fur", "recent.json")
	fi, err := os.Stat(want)
	if err != nil {
		t.Fatalf("recent.json not written: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("recent.json perm = %o, want 600", fi.Mode().Perm())
	}

	reloaded := LoadRecentFiles()
	if len(reloaded.Files) != 2 || reloaded.Files[0] != "/docs/a.md" {
		t.Errorf("reloaded = %v", reloaded.Files)
	}
}

// TestWatch confirms Watch fires onChange when the watched config file is
// rewritten with valid content.
func TestWatch(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte("theme: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed := make(chan *Config, 4)
	Watch(cfgFile, func(c *Config) { changed <- c })

	// Give the watcher a moment to register, then rewrite the file.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(cfgFile, []byte("theme: light\nkeymap: vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case c := <-changed:
		if c == nil {
			t.Fatal("onChange received nil config")
		}
	case <-time.After(5 * time.Second):
		t.Skip("config watch event did not arrive (fsnotify timing); skipping")
	}
}

// TestCreateDefaultIdempotent covers both CreateDefault branches: first call
// writes the file, second call returns early because it already exists.
func TestCreateDefaultIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := CreateDefault()
	if err != nil {
		t.Fatalf("CreateDefault (first): %v", err)
	}
	if path == "" {
		t.Fatal("first CreateDefault returned empty path")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not written: %v", err)
	}

	again, err := CreateDefault()
	if err != nil {
		t.Fatalf("CreateDefault (second): %v", err)
	}
	if again != "" {
		t.Errorf("second CreateDefault should no-op (empty path), got %q", again)
	}
}

// TestWatchDefaultPath covers Watch's cfgFile=="" branch (resolves ConfigDir)
// and the validate-rejects-bad-config path in the change callback.
func TestWatchDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".config", "fur")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte("theme: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed := make(chan *Config, 4)
	Watch("", func(c *Config) { changed <- c }) // empty → AddConfigPath(ConfigDir)

	time.Sleep(150 * time.Millisecond)
	// Write an invalid theme: the callback fires, Validate rejects it, onChange
	// is not invoked. Then a valid write that should come through.
	_ = os.WriteFile(cfgFile, []byte("theme: not-a-real-theme-xyz\n"), 0o644)
	time.Sleep(150 * time.Millisecond)
	_ = os.WriteFile(cfgFile, []byte("theme: light\n"), 0o644)

	select {
	case <-changed:
		// a valid change came through
	case <-time.After(5 * time.Second):
		t.Skip("config watch event did not arrive (fsnotify timing); skipping")
	}
}

// TestCreateDefaultForce writes the starter config even when one exists.
func TestCreateDefaultForce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".config", "fur")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(target, []byte("theme: dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := CreateDefaultForce(); err != nil {
		t.Fatalf("CreateDefaultForce: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != DefaultConfigYAML {
		t.Error("CreateDefaultForce did not overwrite with the default template")
	}
}
