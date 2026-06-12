package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes a global config.yaml under an isolated HOME and returns
// the home dir. Keys present in the file are the ones viper will let an env
// var override (AutomaticEnv only overrides registered keys).
func writeConfig(t *testing.T, yaml string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".config", "fur")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestEnv_ServerKeysNotPivotable is the Chain L regression guard.
//
// The threat model includes an adversary who can set environment variables on
// a victim's shell (sourced rc files, container exec). fur documents FUR_*
// overrides, so the worry is FUR_SERVER_HOST=0.0.0.0 (exposing the web UI) or
// FUR_SERVER_CUSTOM_CSS=<attacker file>. In practice viper's AutomaticEnv
// does not map nested keys (server.host) to FUR_SERVER_HOST without an env
// key replacer, so these runtime-sensitive nested keys are NOT overridable
// via the environment. This test pins that: a future change that adds a
// blanket SetEnvKeyReplacer (to make nested env work) must not silently open
// the server pivot — it would have to keep server.* off the env surface or
// rely on the bind/custom-css containment, and this test forces that to be a
// conscious decision.
func TestEnv_ServerKeysNotPivotable(t *testing.T) {
	writeConfig(t, "theme: light\nserver:\n  host: localhost\n  port: 7777\n  custom_css: \"\"\n")
	t.Setenv("FUR_SERVER_HOST", "0.0.0.0")
	t.Setenv("FUR_SERVER_CUSTOM_CSS", "/tmp/attacker.css")
	t.Setenv("FUR_SERVER_LISTEN_PUBLIC", "true")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("FUR_SERVER_HOST pivoted server.host to %q (Chain L bind exposure)", cfg.Server.Host)
	}
	if cfg.Server.CustomCSS != "" {
		t.Errorf("FUR_SERVER_CUSTOM_CSS pivoted server.custom_css to %q (Chain L)", cfg.Server.CustomCSS)
	}
	if cfg.Server.ListenPublic {
		t.Error("FUR_SERVER_LISTEN_PUBLIC pivoted server.listen_public (Chain L)")
	}
}

// TestEnv_TopLevelKeyApplies documents the surface that env overrides DO
// reach: top-level UX keys registered by the config file. This is benign
// (theme/keymap/show_hidden) and is the accurate scope of FUR_* overrides.
func TestEnv_TopLevelKeyApplies(t *testing.T) {
	writeConfig(t, "theme: light\n")
	t.Setenv("FUR_THEME", "dark")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("FUR_THEME did not override theme: got %q, want dark", cfg.Theme)
	}
}
