package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRecentFiles_Add(t *testing.T) {
	r := &RecentFiles{}
	r.Add("a.md")
	r.Add("b.md")
	r.Add("c.md")

	if len(r.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(r.Files))
	}
	if r.Files[0] != "c.md" {
		t.Errorf("most recent should be c.md, got %q", r.Files[0])
	}
}

func TestRecentFiles_AddDuplicate(t *testing.T) {
	r := &RecentFiles{}
	r.Add("a.md")
	r.Add("b.md")
	r.Add("a.md") // re-add moves to front

	if len(r.Files) != 2 {
		t.Fatalf("expected 2 files after dedup, got %d", len(r.Files))
	}
	if r.Files[0] != "a.md" {
		t.Errorf("most recent should be a.md, got %q", r.Files[0])
	}
	if r.Files[1] != "b.md" {
		t.Errorf("second should be b.md, got %q", r.Files[1])
	}
}

func TestRecentFiles_MaxLimit(t *testing.T) {
	r := &RecentFiles{}
	for i := 0; i < 60; i++ {
		r.Add(filepath.Join("dir", string(rune('a'+i%26))+".md"))
	}
	if len(r.Files) > maxRecentFiles {
		t.Errorf("expected max %d files, got %d", maxRecentFiles, len(r.Files))
	}
}

func TestRecentFiles_SaveAndLoad(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create config dir
	configDir := filepath.Join(tmpHome, ".config", "fur")
	os.MkdirAll(configDir, 0o755)

	r := &RecentFiles{path: filepath.Join(configDir, "recent.json")}
	r.Add("first.md")
	r.Add("second.md")
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load from same path
	loaded := LoadRecentFiles()
	if len(loaded.Files) != 2 {
		t.Fatalf("expected 2 files after load, got %d", len(loaded.Files))
	}
	if loaded.Files[0] != "second.md" {
		t.Errorf("first loaded should be second.md, got %q", loaded.Files[0])
	}
}

// TestRecentFiles_SavePerms is the Chain F regression guard for the recent
// list: the file records what the user has browsed and must not be
// world-readable on a shared box. The write is also atomic (temp + rename).
func TestRecentFiles_SavePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission model")
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "fur")
	os.MkdirAll(configDir, 0o700)

	r := &RecentFiles{path: filepath.Join(configDir, "recent.json")}
	r.Add("private-notes.md")
	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(r.path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("recent.json mode = %o, want owner-only (Chain F)", perm)
	}

	// No leftover temp files from the atomic write.
	entries, _ := os.ReadDir(configDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".recent-") {
			t.Errorf("leftover temp file %s from atomic write", e.Name())
		}
	}
}

func TestRecentFiles_LoadMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	r := LoadRecentFiles()
	if r == nil {
		t.Fatal("LoadRecentFiles should never return nil")
	}
	if len(r.Files) != 0 {
		t.Errorf("expected 0 files from missing file, got %d", len(r.Files))
	}
}

func TestRecentFiles_SaveNoPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	r := &RecentFiles{}
	r.Add("test.md")
	if err := r.Save(); err != nil {
		t.Fatalf("Save without path: %v", err)
	}

	// Verify file was created
	configDir := filepath.Join(tmpHome, ".config", "fur")
	if _, err := os.Stat(filepath.Join(configDir, "recent.json")); err != nil {
		t.Errorf("recent.json not created: %v", err)
	}
}

func TestMergeProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a .fur.yaml in the temp dir
	os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), []byte(`
theme: dark
keymap: vim
`), 0o644)

	// Change to that dir
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if cfg.Theme != "dark" {
		t.Errorf("expected project config theme=dark, got %q", cfg.Theme)
	}
	if cfg.Keymap != "vim" {
		t.Errorf("expected project config keymap=vim, got %q", cfg.Keymap)
	}
}

func TestMergeProjectConfig_TomlFormat(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, ".fur.toml"), []byte(`
theme = "light"
`), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if cfg.Theme != "light" {
		t.Errorf("expected toml theme=light, got %q", cfg.Theme)
	}
}

func TestMergeProjectConfig_WalksUp(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "deep")
	os.MkdirAll(subDir, 0o755)

	// Config in parent
	os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), []byte("theme: dark\n"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(subDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if cfg.Theme != "dark" {
		t.Errorf("expected parent config theme=dark, got %q", cfg.Theme)
	}
}

func TestMergeProjectConfig_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	origTheme := cfg.Theme
	mergeProjectConfig(cfg)

	if cfg.Theme != origTheme {
		t.Errorf("config changed without project config present")
	}
}

// TestMergeProjectConfig_CustomCSSPivot is the Chain A regression guard.
//
// The audit threat model treats a checked-out hostile repository as an
// adversary class. mergeProjectConfig walks up from CWD looking for
// .fur.{toml,yaml,yml} and merges found settings into the active config.
// Before the allowlist fix it merged *every* key — including
// server.custom_css — so a hostile repo could ship a .fur.yaml pivoting
// custom_css onto an attacker-controlled stylesheet; `fur serve` then
// served that CSS into the victim's browser on every rendered page.
//
// The fix restricts per-project sources to a display/UX allowlist
// (projectConfigAllowlist). server.* must never be honored from a
// per-project file. References: lookit-9py.3.5 / .3.5.2; SECURITY-INVENTORY
// §15; bd memory "chain-a-s-plugin-hook-variant-is-moot".
func TestMergeProjectConfig_CustomCSSPivot(t *testing.T) {
	tmpDir := t.TempDir()

	hostile := []byte("server:\n  custom_css: evil.css\n  host: 0.0.0.0\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), hostile, 0o644); err != nil {
		t.Fatalf("write hostile .fur.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	origHost := cfg.Server.Host
	mergeProjectConfig(cfg)

	if cfg.Server.CustomCSS != "" {
		t.Errorf("per-project .fur.yaml set server.custom_css=%q (Chain A pivot); "+
			"per-project sources must not override server-runtime keys", cfg.Server.CustomCSS)
	}
	if cfg.Server.Host != origHost {
		t.Errorf("per-project .fur.yaml set server.host=%q (Chain A bind pivot); "+
			"want unchanged %q", cfg.Server.Host, origHost)
	}
}

// TestMergeProjectConfig_RemotesPivot guards the SSH-remote pivot: a hostile
// repo must not be able to inject a named remote (which could carry an
// attacker host/user) via per-project config.
func TestMergeProjectConfig_RemotesPivot(t *testing.T) {
	tmpDir := t.TempDir()

	hostile := []byte("remotes:\n  evil:\n    host: attacker.example\n    user: root\n    path: /\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), hostile, 0o644); err != nil {
		t.Fatalf("write hostile .fur.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if len(cfg.Remotes) != 0 {
		t.Errorf("per-project .fur.yaml injected remotes %v (Chain A pivot); "+
			"per-project sources must not define remotes", cfg.Remotes)
	}
}

// TestMergeProjectConfig_AllowlistedKeysStillApply confirms the allowlist
// fix did not break legitimate display/UX overrides.
func TestMergeProjectConfig_AllowlistedKeysStillApply(t *testing.T) {
	tmpDir := t.TempDir()

	yaml := []byte("theme: dark\nkeymap: vim\nshow_hidden: true\nscrolloff: 9\nreading_guide: true\nmouse: true\nignore:\n  - \"*.tmp\"\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), yaml, 0o644); err != nil {
		t.Fatalf("write .fur.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if cfg.Theme != "dark" {
		t.Errorf("theme: got %q want dark", cfg.Theme)
	}
	if cfg.Keymap != "vim" {
		t.Errorf("keymap: got %q want vim", cfg.Keymap)
	}
	if !cfg.ShowHidden {
		t.Error("show_hidden: got false want true")
	}
	if cfg.ScrollOff != 9 {
		t.Errorf("scrolloff: got %d want 9", cfg.ScrollOff)
	}
	if !cfg.ReadingGuide {
		t.Error("reading_guide: got false want true")
	}
	if !cfg.Mouse {
		t.Error("mouse: got false want true")
	}
	if len(cfg.Ignore) != 1 || cfg.Ignore[0] != "*.tmp" {
		t.Errorf("ignore: got %v want [*.tmp]", cfg.Ignore)
	}
}
