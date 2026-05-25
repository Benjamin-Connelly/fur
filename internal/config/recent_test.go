package config

import (
	"os"
	"path/filepath"
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

// TestMergeProjectConfig_CustomCSSPivot_BUG is a Chain A proof-of-concept.
//
// The audit threat model treats a checked-out hostile repository as an
// adversary class. mergeProjectConfig walks up from CWD looking for
// .fur.toml / .fur.yaml / .fur.yml and silently merges every key it finds
// into the active config — including server-runtime keys like
// server.custom_css. A hostile repo ships a .fur.yaml that pivots
// custom_css onto an attacker-controlled stylesheet inside the repo;
// `fur serve` then registers /__custom.css and every rendered markdown
// page loads attacker CSS in the victim's browser.
//
// The handler at internal/web/server.go::handleCustomCSS does block
// out-of-root paths (EvalSymlinks + prefix check) so traversal isn't the
// vector — the vector is silent in-root pivoting. The fix (lookit-9py.3.5.2)
// is a trust mechanism: refuse server-runtime keys from per-project
// sources entirely, or gate them behind a .fur.trusted allowlist /
// first-run prompt. Display preferences (theme, keymap) can stay opt-in.
//
// References: lookit-9py.3.5.1; SECURITY-INVENTORY.md §15 (Chain A
// reframe); bd memories "chain-a-s-plugin-hook-variant-is-moot" and
// "per-project-fur-toml-fur-yaml-must-not".
func TestMergeProjectConfig_CustomCSSPivot_BUG(t *testing.T) {
	tmpDir := t.TempDir()

	hostile := []byte("server:\n  custom_css: evil.css\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".fur.yaml"), hostile, 0o644); err != nil {
		t.Fatalf("write hostile .fur.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := DefaultConfig()
	mergeProjectConfig(cfg)

	if cfg.Server.CustomCSS == "evil.css" {
		t.Errorf("per-project .fur.yaml silently set server.custom_css=%q "+
			"(Chain A pivot). Per-project config sources must not override "+
			"server-runtime keys without a trust signal — refuse the key, "+
			"or gate behind a .fur.trusted allowlist / first-run prompt.",
			cfg.Server.CustomCSS)
	}
}
