package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestTrustedPluginFile checks the owner-only trust gate (lookit-9py.4.1).
func TestTrustedPluginFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission model")
	}
	dir := t.TempDir()

	good := filepath.Join(dir, "ok.yaml")
	if err := os.WriteFile(good, []byte("name: ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !trustedPluginFile(good) {
		t.Error("0600 owner-owned plugin file should be trusted")
	}

	worldWritable := filepath.Join(dir, "evil.yaml")
	if err := os.WriteFile(worldWritable, []byte("name: evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(worldWritable, 0o666); err != nil {
		t.Fatal(err)
	}
	if trustedPluginFile(worldWritable) {
		t.Error("world-writable plugin file must not be trusted (Chain A/4.1)")
	}

	groupWritable := filepath.Join(dir, "grp.yaml")
	if err := os.WriteFile(groupWritable, []byte("name: grp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(groupWritable, 0o620); err != nil {
		t.Fatal(err)
	}
	if trustedPluginFile(groupWritable) {
		t.Error("group-writable plugin file must not be trusted")
	}

	// A symlink (even to a trusted file) is not a regular file → not trusted.
	link := filepath.Join(dir, "link.yaml")
	if err := os.Symlink(good, link); err == nil {
		if trustedPluginFile(link) {
			t.Error("symlinked plugin file must not be trusted")
		}
	}

	if trustedPluginFile(filepath.Join(dir, "missing.yaml")) {
		t.Error("missing plugin file must fail closed")
	}
}

// TestLoadPlugins_SkipsUntrusted asserts the loader registers hooks from a
// trusted plugin but skips a world-writable one in the same dir.
func TestLoadPlugins_SkipsUntrusted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission model")
	}
	configDir := t.TempDir()
	pluginDir := filepath.Join(configDir, "plugins")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatal(err)
	}

	trusted := "name: trusted\nhooks:\n  - point: beforerender\n    append: \"TRUSTED\"\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "trusted.yaml"), []byte(trusted), 0o600); err != nil {
		t.Fatal(err)
	}
	untrusted := "name: untrusted\nhooks:\n  - point: beforerender\n    append: \"PWNED\"\n"
	up := filepath.Join(pluginDir, "untrusted.yaml")
	if err := os.WriteFile(up, []byte(untrusted), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(up, 0o666); err != nil {
		t.Fatal(err)
	}

	reg, err := LoadPlugins(configDir)
	if err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}

	ctx := &HookContext{Content: ""}
	if err := reg.Run(HookBeforeRender, ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := ctx.Content; got != "TRUSTED" {
		t.Errorf("content = %q; want only the trusted hook applied (untrusted, "+
			"world-writable plugin must be skipped)", got)
	}
}
