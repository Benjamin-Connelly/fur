package remote

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     *Target
	}{
		{
			name:  "simple host:path",
			input: "myhost:/home/user/docs",
			want:  &Target{Host: "myhost", Path: "/home/user/docs"},
		},
		{
			name:  "user@host:path",
			input: "deploy@myhost:/var/www",
			want:  &Target{User: "deploy", Host: "myhost", Path: "/var/www"},
		},
		{
			name:  "user@host:port:path",
			input: "deploy@myhost:2222:/var/www",
			want:  &Target{User: "deploy", Host: "myhost", Port: 2222, Path: "/var/www"},
		},
		{
			name:  "host with relative path",
			input: "server:docs/readme",
			want:  &Target{Host: "server", Path: "docs/readme"},
		},
		{
			name:  "host with home-relative path",
			input: "server:~/projects",
			want:  &Target{Host: "server", Path: "~/projects"},
		},
		{
			name:  "local path no colon",
			input: "/home/user/docs",
			want:  nil,
		},
		{
			name:  "relative local path",
			input: "./docs",
			want:  nil,
		},
		{
			name:  "current dir",
			input: ".",
			want:  nil,
		},
		{
			name:  "windows drive letter",
			input: "C:\\Users\\docs",
			want:  nil,
		},
		{
			name:  "windows drive letter forward slash",
			input: "C:/Users/docs",
			want:  nil,
		},
		{
			name:  "ip address host",
			input: "192.168.1.50:/data",
			want:  &Target{Host: "192.168.1.50", Path: "/data"},
		},
		{
			name:  "user@ip:path",
			input: "root@10.0.0.1:/etc/nginx",
			want:  &Target{User: "root", Host: "10.0.0.1", Path: "/etc/nginx"},
		},
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

func TestIsRemotePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"host:/path", true},
		{"user@host:/path", true},
		{"/local/path", false},
		{"./relative", false},
		{".", false},
		{"C:\\windows", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsRemotePath(tt.input); got != tt.want {
				t.Errorf("IsRemotePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTargetString(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{
			name:   "simple",
			target: Target{Host: "myhost", Path: "/docs"},
			want:   "myhost:/docs",
		},
		{
			name:   "with user",
			target: Target{User: "deploy", Host: "myhost", Path: "/docs"},
			want:   "deploy@myhost:/docs",
		},
		{
			name:   "with port",
			target: Target{User: "deploy", Host: "myhost", Port: 2222, Path: "/docs"},
			want:   "deploy@myhost:2222:/docs",
		},
		{
			name:   "default port omitted",
			target: Target{Host: "myhost", Port: 22, Path: "/docs"},
			want:   "myhost:/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTargetDisplay(t *testing.T) {
	target := Target{User: "deploy", Host: "myhost", Path: "/docs"}
	want := "deploy@myhost:/docs"
	if got := target.Display(); got != want {
		t.Errorf("Display() = %q, want %q", got, want)
	}
}

func TestCachePath(t *testing.T) {
	target := Target{Host: "myhost", Path: "/docs"}
	path, err := CachePath(target)
	if err != nil {
		t.Fatalf("CachePath() error: %v", err)
	}
	if path == "" {
		t.Error("CachePath() returned empty string")
	}
	// Same target should produce same cache path
	path2, _ := CachePath(target)
	if path != path2 {
		t.Errorf("CachePath() not deterministic: %q != %q", path, path2)
	}

	// Different target should produce different cache path
	target2 := Target{Host: "other", Path: "/docs"}
	path3, _ := CachePath(target2)
	if path == path3 {
		t.Error("CachePath() returned same path for different targets")
	}
}

func TestConnState_String(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{ConnDisconnected, "Disconnected"},
		{ConnConnecting, "Connecting"},
		{ConnConnected, "Connected"},
		{ConnReconnecting, "Reconnecting"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
