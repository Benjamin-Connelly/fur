package remote

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- ParseTarget tests ---

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *Target
	}{
		// Valid remote paths
		{"simple host:path", "myhost:/home/user/docs", &Target{Host: "myhost", Path: "/home/user/docs"}},
		{"user@host:path", "deploy@myhost:/var/www", &Target{User: "deploy", Host: "myhost", Path: "/var/www"}},
		{"user@host:port:path", "deploy@myhost:2222:/var/www", &Target{User: "deploy", Host: "myhost", Port: 2222, Path: "/var/www"}},
		{"host with relative path", "server:docs/readme", &Target{Host: "server", Path: "docs/readme"}},
		{"host with home-relative path", "server:~/projects", &Target{Host: "server", Path: "~/projects"}},
		{"ip address host", "192.168.1.50:/data", &Target{Host: "192.168.1.50", Path: "/data"}},
		{"user@ip:path", "root@10.0.0.1:/etc/nginx", &Target{User: "root", Host: "10.0.0.1", Path: "/etc/nginx"}},
		{"host with tilde only", "server:~", &Target{Host: "server", Path: "~"}},
		{"hyphenated hostname", "my-server:/data", &Target{Host: "my-server", Path: "/data"}},
		{"underscore hostname", "my_server:/data", &Target{Host: "my_server", Path: "/data"}},
		{"fqdn", "host.example.com:/data", &Target{Host: "host.example.com", Path: "/data"}},
		{"user with dots", "first.last@host:/data", &Target{User: "first.last", Host: "host", Path: "/data"}},
		{"deep path", "host:/a/b/c/d/e/f", &Target{Host: "host", Path: "/a/b/c/d/e/f"}},
		{"path with spaces", "host:/path with spaces/dir", &Target{Host: "host", Path: "/path with spaces/dir"}},
		{"port 22", "host:22:/data", &Target{Host: "host", Port: 22, Path: "/data"}},
		{"high port", "host:65535:/data", &Target{Host: "host", Port: 65535, Path: "/data"}},

		// Not remote paths
		{"local absolute path", "/home/user/docs", nil},
		{"relative local path", "./docs", nil},
		{"current dir", ".", nil},
		{"parent dir", "..", nil},
		{"windows drive backslash", "C:\\Users\\docs", nil},
		{"windows drive forward slash", "C:/Users/docs", nil},
		{"empty string", "", nil},
		{"just a word", "hostname", nil},
		{"double colon no match", "host::", nil}, // edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTarget(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseTarget(%q) = %+v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseTarget(%q) = nil, want %+v", tt.input, tt.want)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
		})
	}
}

// --- IsRemotePath tests ---

func TestIsRemotePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"host:/path", true},
		{"user@host:/path", true},
		{"host:~/docs", true},
		{"/local/path", false},
		{"./relative", false},
		{".", false},
		{"..", false},
		{"C:\\windows", false},
		{"", false},
		{"hostname", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsRemotePath(tt.input); got != tt.want {
				t.Errorf("IsRemotePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Target.String() tests ---

func TestTargetString(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{"simple", Target{Host: "myhost", Path: "/docs"}, "myhost:/docs"},
		{"with user", Target{User: "deploy", Host: "myhost", Path: "/docs"}, "deploy@myhost:/docs"},
		{"with port", Target{User: "deploy", Host: "myhost", Port: 2222, Path: "/docs"}, "deploy@myhost:2222:/docs"},
		{"default port omitted", Target{Host: "myhost", Port: 22, Path: "/docs"}, "myhost:/docs"},
		{"no user with port", Target{Host: "myhost", Port: 2222, Path: "/docs"}, "myhost:2222:/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Target.Display() tests ---

func TestTargetDisplay(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{"simple", Target{Host: "h", Path: "/p"}, "h:/p"},
		{"with user", Target{User: "u", Host: "h", Path: "/p"}, "u@h:/p"},
		{"port not shown in display", Target{Host: "h", Port: 2222, Path: "/p"}, "h:/p"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.Display(); got != tt.want {
				t.Errorf("Display() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- CachePath tests ---

func TestCachePath(t *testing.T) {
	target := Target{Host: "myhost", Path: "/docs"}
	path, err := CachePath(target)
	if err != nil {
		t.Fatalf("CachePath() error: %v", err)
	}
	if path == "" {
		t.Fatal("CachePath() returned empty string")
	}

	// Deterministic
	path2, _ := CachePath(target)
	if path != path2 {
		t.Errorf("CachePath() not deterministic: %q != %q", path, path2)
	}

	// Different targets produce different paths
	target2 := Target{Host: "other", Path: "/docs"}
	path3, _ := CachePath(target2)
	if path == path3 {
		t.Error("CachePath() same for different targets")
	}

	// User affects cache path
	target3 := Target{User: "alice", Host: "myhost", Path: "/docs"}
	path4, _ := CachePath(target3)
	if path == path4 {
		t.Error("CachePath() ignores user field")
	}

	// Port affects cache path
	target4 := Target{Host: "myhost", Port: 2222, Path: "/docs"}
	path5, _ := CachePath(target4)
	if path == path5 {
		t.Error("CachePath() ignores port field")
	}

	// Path is under cache dir
	cacheDir, _ := os.UserCacheDir()
	if !strings.HasPrefix(path, filepath.Join(cacheDir, "fur", "remote")) {
		t.Errorf("CachePath() not under expected cache dir: %q", path)
	}
}

// --- ConnState tests ---

func TestConnState_String(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{ConnDisconnected, "Disconnected"},
		{ConnConnecting, "Connecting"},
		{ConnConnected, "Connected"},
		{ConnReconnecting, "Reconnecting"},
		{ConnState(99), "Disconnected"}, // unknown state
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// --- Conn unit tests (no network) ---

func TestNewConn_InitialState(t *testing.T) {
	target := Target{Host: "myhost", Path: "/docs"}
	conn := NewConn(target)

	if conn.State() != ConnDisconnected {
		t.Errorf("initial state = %v, want Disconnected", conn.State())
	}
	if conn.LastError() != nil {
		t.Errorf("initial error = %v, want nil", conn.LastError())
	}
	if conn.SFTP() != nil {
		t.Error("initial SFTP() should be nil")
	}
	if conn.Target().Host != "myhost" {
		t.Errorf("Target().Host = %q, want %q", conn.Target().Host, "myhost")
	}
}

func TestConn_CloseIdempotent(t *testing.T) {
	conn := NewConn(Target{Host: "myhost", Path: "/docs"})

	// Close without connecting should not panic
	if err := conn.Close(); err != nil {
		t.Errorf("Close() on unconnected = %v", err)
	}

	// Double close should not panic
	if err := conn.Close(); err != nil {
		t.Errorf("double Close() = %v", err)
	}

	if conn.State() != ConnDisconnected {
		t.Errorf("state after close = %v, want Disconnected", conn.State())
	}
}

// --- loadSSHConfig tests ---

func TestLoadSSHConfig_MissingFile(t *testing.T) {
	// Temporarily override HOME to a dir without .ssh/config
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := loadSSHConfig()
	if cfg != nil {
		t.Error("loadSSHConfig() should return nil when ~/.ssh/config missing")
	}
}

func TestLoadSSHConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	os.MkdirAll(sshDir, 0o700)

	configContent := `Host testhost
    Hostname 10.0.0.1
    User testuser
    Port 2222
    IdentityFile ~/.ssh/test_key

Host *.example.com
    User webadmin
`
	os.WriteFile(filepath.Join(sshDir, "config"), []byte(configContent), 0o600)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := loadSSHConfig()
	if cfg == nil {
		t.Fatal("loadSSHConfig() returned nil for valid config")
	}

	// Verify resolution
	hostname, err := cfg.Get("testhost", "Hostname")
	if err != nil {
		t.Fatalf("Get Hostname: %v", err)
	}
	if hostname != "10.0.0.1" {
		t.Errorf("Hostname = %q, want %q", hostname, "10.0.0.1")
	}

	user, _ := cfg.Get("testhost", "User")
	if user != "testuser" {
		t.Errorf("User = %q, want %q", user, "testuser")
	}

	port, _ := cfg.Get("testhost", "Port")
	if port != "2222" {
		t.Errorf("Port = %q, want %q", port, "2222")
	}
}

func TestLoadSSHConfig_MalformedFile(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	// Write garbage that the parser will reject
	os.WriteFile(filepath.Join(sshDir, "config"), []byte("\x00\x01\x02"), 0o600)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := loadSSHConfig()
	// Should return nil on parse error, not panic
	_ = cfg
}

// --- configGet tests ---

func TestConfigGet_NilConfig(t *testing.T) {
	conn := &Conn{target: Target{Host: "testhost"}, sshCfg: nil}
	if got := conn.configGet("Hostname"); got != "" {
		t.Errorf("configGet on nil config = %q, want empty", got)
	}
}

func TestConfigGet_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	os.WriteFile(filepath.Join(sshDir, "config"), []byte(`Host myalias
    Hostname 172.16.0.1
    User admin
    Port 8022
`), 0o600)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	conn := NewConn(Target{Host: "myalias", Path: "/data"})
	if conn.sshCfg == nil {
		t.Fatal("sshCfg should be loaded")
	}

	if got := conn.configGet("Hostname"); got != "172.16.0.1" {
		t.Errorf("configGet(Hostname) = %q, want %q", got, "172.16.0.1")
	}
	if got := conn.configGet("User"); got != "admin" {
		t.Errorf("configGet(User) = %q, want %q", got, "admin")
	}
	if got := conn.configGet("Port"); got != "8022" {
		t.Errorf("configGet(Port) = %q, want %q", got, "8022")
	}
	if got := conn.configGet("NonExistent"); got != "" {
		t.Errorf("configGet(NonExistent) = %q, want empty", got)
	}
}

// --- resolveUser tests ---

func TestResolveUser(t *testing.T) {
	// With explicit user in target
	conn := &Conn{target: Target{User: "explicit", Host: "h"}}
	if got := conn.resolveUser(); got != "explicit" {
		t.Errorf("resolveUser with explicit = %q, want %q", got, "explicit")
	}

	// Without explicit user, no config → falls back to OS user
	conn2 := &Conn{target: Target{Host: "h"}, sshCfg: nil}
	got := conn2.resolveUser()
	if got == "" {
		t.Error("resolveUser without config should return OS user, got empty")
	}
}

// --- resolveHost tests ---

func TestResolveHost(t *testing.T) {
	// No config → returns target host as-is
	conn := &Conn{target: Target{Host: "myalias"}, sshCfg: nil}
	if got := conn.resolveHost(); got != "myalias" {
		t.Errorf("resolveHost no config = %q, want %q", got, "myalias")
	}
}

// --- resolvePort tests ---

func TestResolvePort(t *testing.T) {
	// Explicit port in target
	conn := &Conn{target: Target{Host: "h", Port: 2222}, sshCfg: nil}
	if got := conn.resolvePort(); got != 2222 {
		t.Errorf("resolvePort explicit = %d, want %d", got, 2222)
	}

	// No port, no config → default 22
	conn2 := &Conn{target: Target{Host: "h"}, sshCfg: nil}
	if got := conn2.resolvePort(); got != 22 {
		t.Errorf("resolvePort default = %d, want %d", got, 22)
	}
}

// --- SyncStatus tests ---

func TestSyncStatus_Initial(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/p"})
	syncer := NewSyncer(conn, t.TempDir())

	status := syncer.Status()
	if status.State != SyncIdle {
		t.Errorf("initial state = %v, want SyncIdle", status.State)
	}
	if !status.LastSync.IsZero() {
		t.Error("initial LastSync should be zero")
	}
	if status.FilesTotal != 0 {
		t.Errorf("initial FilesTotal = %d, want 0", status.FilesTotal)
	}
}

func TestSyncer_CacheDir(t *testing.T) {
	dir := t.TempDir()
	conn := NewConn(Target{Host: "h", Path: "/p"})
	syncer := NewSyncer(conn, dir)

	if syncer.CacheDir() != dir {
		t.Errorf("CacheDir() = %q, want %q", syncer.CacheDir(), dir)
	}
}

func TestSyncer_StopIdempotent(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/p"})
	syncer := NewSyncer(conn, t.TempDir())

	// Stop without starting should not panic
	syncer.Stop()
	syncer.Stop() // double stop
}

func TestSyncer_InitialSync_NoConnection(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/p"})
	syncer := NewSyncer(conn, t.TempDir())

	err := syncer.InitialSync()
	if err == nil {
		t.Error("InitialSync without connection should fail")
	}

	status := syncer.Status()
	if status.State != SyncError {
		t.Errorf("state after failed sync = %v, want SyncError", status.State)
	}
}

func TestSyncer_SetOnChange(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/p"})
	syncer := NewSyncer(conn, t.TempDir())

	called := false
	syncer.SetOnChange(func() { called = true })

	// onChange shouldn't be called without polling
	if called {
		t.Error("onChange called before polling started")
	}
}

func TestSyncer_SingleFileField(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/path/to/file.md"})
	syncer := NewSyncer(conn, t.TempDir())

	// singleFile should be false initially (set during sync based on Stat)
	if syncer.singleFile {
		t.Error("singleFile should be false before sync")
	}
}

// --- Roundtrip parse → string tests ---

func TestParseTarget_Roundtrip(t *testing.T) {
	inputs := []string{
		"host:/path",
		"user@host:/path",
		"user@host:2222:/path",
	}
	for _, input := range inputs {
		target := ParseTarget(input)
		if target == nil {
			t.Fatalf("ParseTarget(%q) = nil", input)
		}
		str := target.String()
		reparsed := ParseTarget(str)
		if reparsed == nil {
			t.Fatalf("ParseTarget(%q) roundtrip failed: nil", str)
		}
		if reparsed.User != target.User || reparsed.Host != target.Host ||
			reparsed.Port != target.Port || reparsed.Path != target.Path {
			t.Errorf("roundtrip mismatch: %+v -> %q -> %+v", target, str, reparsed)
		}
	}
}

// --- Edge cases ---

func TestParseTarget_SpecialCharacters(t *testing.T) {
	// Path with unicode
	got := ParseTarget("host:/données/résumé.md")
	if got == nil {
		t.Fatal("ParseTarget with unicode path = nil")
	}
	if got.Path != "/données/résumé.md" {
		t.Errorf("Path = %q, want %q", got.Path, "/données/résumé.md")
	}
}

func TestCachePath_Stability(t *testing.T) {
	// Verify the hash is stable across calls (no randomness)
	target := Target{User: "u", Host: "h", Port: 22, Path: "/p"}
	paths := make([]string, 10)
	for i := range paths {
		p, err := CachePath(target)
		if err != nil {
			t.Fatalf("CachePath() error: %v", err)
		}
		paths[i] = p
	}
	for i := 1; i < len(paths); i++ {
		if paths[i] != paths[0] {
			t.Errorf("CachePath() not stable: %q != %q", paths[i], paths[0])
		}
	}
}

// --- Concurrency tests ---

func TestConn_ConcurrentStateAccess(t *testing.T) {
	conn := NewConn(Target{Host: "h", Path: "/p"})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			_ = conn.State()
			_ = conn.LastError()
			_ = conn.SFTP()
			_ = conn.Target()
		}
	}()

	for i := 0; i < 1000; i++ {
		_ = conn.State()
		_ = conn.LastError()
		_ = conn.SFTP()
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent access test timed out (possible deadlock)")
	}
}
