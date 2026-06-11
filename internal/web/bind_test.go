package web

import "testing"

// TestValidateBind is the Chain C regression guard.
//
// Before this hardening, Server.Start bound s.cfg.Server.Host:Port
// unconditionally. A user (or an env/config pivot — Chain L) setting
// server.host to 0.0.0.0 exposed the file/search/document APIs to every
// host on the network: a co-located or remote adversary could enumerate and
// read the entire browsed tree (same-origin exfil, audit Chain C).
//
// ValidateBind refuses a non-loopback bind unless --listen-public is set.
// References lookit-9py.3.7 / .4.5.
func TestValidateBind(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		listenPublic bool
		wantErr      bool
	}{
		{"localhost loopback", "localhost", false, false},
		{"127.0.0.1 loopback", "127.0.0.1", false, false},
		{"ipv6 loopback", "::1", false, false},
		{"empty treated as loopback", "", false, false},
		{"0.0.0.0 refused", "0.0.0.0", false, true},
		{"ipv6 wildcard refused", "::", false, true},
		{"lan ip refused", "192.168.1.50", false, true},
		{"hostname refused", "fileserver.local", false, true},
		{"0.0.0.0 allowed with opt-in", "0.0.0.0", true, false},
		{"lan ip allowed with opt-in", "192.168.1.50", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBind(tt.host, tt.listenPublic)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBind(%q, %v) err=%v, wantErr=%v", tt.host, tt.listenPublic, err, tt.wantErr)
			}
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	loopback := []string{"", "localhost", "127.0.0.1", "127.0.0.5", "::1"}
	for _, h := range loopback {
		if !isLoopbackHost(h) {
			t.Errorf("isLoopbackHost(%q) = false, want true", h)
		}
	}
	public := []string{"0.0.0.0", "::", "10.0.0.1", "8.8.8.8", "example.com"}
	for _, h := range public {
		if isLoopbackHost(h) {
			t.Errorf("isLoopbackHost(%q) = true, want false", h)
		}
	}
}
