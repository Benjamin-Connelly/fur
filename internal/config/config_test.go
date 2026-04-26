package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Theme != "auto" {
		t.Errorf("default theme should be auto, got %q", cfg.Theme)
	}
	if cfg.Keymap != "default" {
		t.Errorf("default keymap should be default, got %q", cfg.Keymap)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("default port should be 7777, got %d", cfg.Server.Port)
	}
	if !cfg.Git.Enabled {
		t.Error("git should be enabled by default")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		theme   string
		keymap  string
		wantErr bool
	}{
		{"valid defaults", "auto", "default", false},
		{"dark theme", "dark", "vim", false},
		{"light emacs", "light", "emacs", false},
		{"bad theme", "neon", "default", true},
		{"bad keymap", "auto", "dvorak", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Theme = tt.theme
			cfg.Keymap = tt.keymap
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadMissing(t *testing.T) {
	// Loading from a nonexistent dir should return defaults
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		// It's acceptable to get defaults or an error for bad paths
		return
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Theme != "auto" {
		t.Errorf("expected default theme, got %q", cfg.Theme)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
theme: dark
keymap: vim
server:
  port: 8080
`), 0o644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("expected dark theme, got %q", cfg.Theme)
	}
	if cfg.Keymap != "vim" {
		t.Errorf("expected vim keymap, got %q", cfg.Keymap)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
theme: neon
keymap: default
`), 0o644)

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected validation error for invalid theme")
	}
}

func TestString(t *testing.T) {
	cfg := DefaultConfig()
	s := cfg.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
	if !strings.HasSuffix(dir, filepath.Join(".config", "fur")) {
		t.Errorf("expected path ending in .config/fur, got %q", dir)
	}
}

func TestCreateDefault_New(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	path, err := CreateDefault()
	if err != nil {
		t.Fatalf("CreateDefault() error: %v", err)
	}
	if path == "" {
		t.Fatal("expected path to created config, got empty string")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading created config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "theme: auto") {
		t.Error("created config should contain 'theme: auto'")
	}
	if !strings.Contains(content, "port: 7777") {
		t.Error("created config should contain 'port: 7777'")
	}
}

func TestCreateDefault_Exists(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create config first
	configDir := filepath.Join(tmpHome, ".config", "fur")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("theme: dark\n"), 0o644)

	path, err := CreateDefault()
	if err != nil {
		t.Fatalf("CreateDefault() error: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty string for existing config, got %q", path)
	}

	// Verify original content preserved (no overwrite)
	data, _ := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if string(data) != "theme: dark\n" {
		t.Error("existing config was overwritten")
	}
}

func TestCreateDefaultForce_New(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	path, backup, err := CreateDefaultForce()
	if err != nil {
		t.Fatalf("CreateDefaultForce() error: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if backup != "" {
		t.Errorf("expected no backup for fresh install, got %q", backup)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written config: %v", err)
	}
	if !strings.Contains(string(data), "show_hidden") {
		t.Error("default template should mention show_hidden")
	}
}

func TestCreateDefaultForce_Overwrites(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	configDir := filepath.Join(tmpHome, ".config", "fur")
	os.MkdirAll(configDir, 0o755)
	original := []byte("theme: dark\n")
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, original, 0o644)

	path, backup, err := CreateDefaultForce()
	if err != nil {
		t.Fatalf("CreateDefaultForce() error: %v", err)
	}
	if path != configPath {
		t.Errorf("expected path %q, got %q", configPath, path)
	}
	if backup != configPath+".bak" {
		t.Errorf("expected backup at %q, got %q", configPath+".bak", backup)
	}
	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(backupData) != string(original) {
		t.Errorf("backup should preserve original content, got %q", backupData)
	}
	newData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading new config: %v", err)
	}
	if string(newData) == string(original) {
		t.Error("new config should differ from original")
	}
	if !strings.Contains(string(newData), "show_hidden") {
		t.Error("new config should contain show_hidden")
	}
}

func TestValidate_AsciiTheme(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "ascii"
	if err := cfg.Validate(); err != nil {
		t.Errorf("ascii theme should be valid, got error: %v", err)
	}
}

func TestValidate_EmptyStrings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty theme should be invalid")
	}

	cfg = DefaultConfig()
	cfg.Keymap = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty keymap should be invalid")
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("theme: light\nkeymap: default\n"), 0o644)

	os.Setenv("FUR_THEME", "dark")
	defer os.Unsetenv("FUR_THEME")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("expected env override theme=dark, got %q", cfg.Theme)
	}
}
