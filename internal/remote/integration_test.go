//go:build integration

package remote

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests require SSH access to a test server.
// Run with: go test -tags=integration ./internal/remote/ -v
//
// These tests expect:
// - SSH config for "command" host in ~/.ssh/config
// - ~/test_md directory on the remote with markdown files

const testHost = "command"
const testPath = "~/test_md"

func TestIntegration_Connect(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	if conn.State() != ConnConnected {
		t.Errorf("state = %v, want Connected", conn.State())
	}

	// Path should be resolved (~ expanded)
	resolved := conn.Target()
	if resolved.Path == testPath {
		t.Error("path should be expanded from ~")
	}
	if resolved.Path == "" {
		t.Error("resolved path is empty")
	}
	t.Logf("Resolved path: %s", resolved.Path)

	// SFTP should be available
	if conn.SFTP() == nil {
		t.Error("SFTP client should not be nil after connect")
	}
}

func TestIntegration_ConnectAndList(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	resolved := conn.Target()
	entries, err := conn.SFTP().ReadDir(resolved.Path)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", resolved.Path, err)
	}

	if len(entries) == 0 {
		t.Error("expected files in remote test_md directory")
	}

	for _, e := range entries {
		t.Logf("  %s (%d bytes)", e.Name(), e.Size())
	}
}

func TestIntegration_InitialSync(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	cacheDir := t.TempDir()
	syncer := NewSyncer(conn, cacheDir)

	if err := syncer.InitialSync(); err != nil {
		t.Fatalf("InitialSync: %v", err)
	}

	status := syncer.Status()
	if status.State != SyncIdle {
		t.Errorf("state after sync = %v, want SyncIdle", status.State)
	}
	if status.FilesTotal == 0 {
		t.Error("expected files synced")
	}
	if status.LastSync.IsZero() {
		t.Error("LastSync should be set after sync")
	}

	t.Logf("Synced %d files", status.FilesTotal)

	// Verify local cache has files
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("ReadDir cache: %v", err)
	}
	if len(entries) == 0 {
		t.Error("cache directory is empty after sync")
	}

	// Verify at least one .md file
	hasMD := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			hasMD = true
			data, err := os.ReadFile(filepath.Join(cacheDir, e.Name()))
			if err != nil {
				t.Errorf("reading cached %s: %v", e.Name(), err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("cached %s is empty", e.Name())
			}
		}
	}
	if !hasMD {
		t.Error("no .md files found in cache")
	}
}

func TestIntegration_SyncIdempotent(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	cacheDir := t.TempDir()
	syncer := NewSyncer(conn, cacheDir)

	// First sync
	if err := syncer.InitialSync(); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	status1 := syncer.Status()

	// Second sync should not error or change file count
	if err := syncer.InitialSync(); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	status2 := syncer.Status()

	if status1.FilesTotal != status2.FilesTotal {
		t.Errorf("file count changed: %d -> %d", status1.FilesTotal, status2.FilesTotal)
	}
}

func TestIntegration_Polling(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	cacheDir := t.TempDir()
	syncer := NewSyncer(conn, cacheDir)

	if err := syncer.InitialSync(); err != nil {
		t.Fatalf("InitialSync: %v", err)
	}

	// Start polling briefly
	syncer.StartPolling()
	time.Sleep(2 * time.Second)
	syncer.Stop()

	// Should still be in a valid state after stop
	status := syncer.Status()
	if status.State == SyncError {
		t.Errorf("unexpected error state after polling: %v", syncer.conn.LastError())
	}
}

func TestIntegration_Reconnect(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Close and reconnect
	conn.Close()
	if conn.State() != ConnDisconnected {
		t.Errorf("state after close = %v, want Disconnected", conn.State())
	}

	if err := conn.Reconnect(); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}
	defer conn.Close()

	if conn.State() != ConnConnected {
		t.Errorf("state after reconnect = %v, want Connected", conn.State())
	}

	// SFTP should work after reconnect
	resolved := conn.Target()
	entries, err := conn.SFTP().ReadDir(resolved.Path)
	if err != nil {
		t.Fatalf("ReadDir after reconnect: %v", err)
	}
	if len(entries) == 0 {
		t.Error("no files after reconnect")
	}
}

func TestIntegration_CachePath(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	if err := conn.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	resolved := conn.Target()
	cachePath, err := CachePath(resolved)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}

	if cachePath == "" {
		t.Error("CachePath returned empty string")
	}
	t.Logf("Cache path: %s", cachePath)

	// Verify it's deterministic
	cachePath2, _ := CachePath(resolved)
	if cachePath != cachePath2 {
		t.Error("CachePath not deterministic")
	}
}

func TestIntegration_BuildSSHConfig(t *testing.T) {
	target := Target{Host: testHost, Path: testPath}
	conn := NewConn(target)

	sshCfg, err := conn.buildSSHConfig()
	if err != nil {
		t.Fatalf("buildSSHConfig: %v", err)
	}

	if sshCfg.User == "" {
		t.Error("SSH config user should not be empty")
	}
	if len(sshCfg.Auth) == 0 {
		t.Error("SSH config should have at least one auth method")
	}
	if sshCfg.HostKeyCallback == nil {
		t.Error("SSH config should have host key callback")
	}
	t.Logf("SSH user: %s, auth methods: %d", sshCfg.User, len(sshCfg.Auth))
}
