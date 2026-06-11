package remote

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kevinburke/ssh_config"
)

// sshConfigKeyAllowlist is the complete set of ~/.ssh/config keys fur is
// permitted to consult. Reading anything else — especially ProxyCommand,
// ProxyJump, LocalCommand, or Match exec — would let a planted ~/.ssh/config
// turn a remote-browse into arbitrary command execution on the victim's box
// (audit Chain E). See bd memory
// "fur-deliberately-reads-only-user-hostname-port-identityfile".
var sshConfigKeyAllowlist = map[string]bool{
	"User":         true,
	"Hostname":     true,
	"Port":         true,
	"IdentityFile": true,
}

var configGetRe = regexp.MustCompile(`configGet\("([^"]+)"\)`)

// TestSSHConfigKeyAllowlist is the Chain E regression guard. It scans the
// connection source and fails if any configGet call reads a key outside the
// allowlist. This catches a future change that adds ProxyCommand/Match
// support before it can ship.
func TestSSHConfigKeyAllowlist(t *testing.T) {
	src, err := os.ReadFile("conn.go")
	if err != nil {
		t.Fatalf("read conn.go: %v", err)
	}
	matches := configGetRe.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("no configGet calls found — test or source out of sync")
	}
	for _, m := range matches {
		key := m[1]
		if !sshConfigKeyAllowlist[key] {
			t.Errorf("conn.go reads SSH config key %q, which is not in the "+
				"allowlist (%v). Reading exec-bearing keys (ProxyCommand, "+
				"ProxyJump, LocalCommand, Match exec) enables Chain E RCE via a "+
				"planted ~/.ssh/config.", key, keysOf(sshConfigKeyAllowlist))
		}
	}
}

// TestNoExecOrProxyInRemote asserts the remote package never shells out and
// never references the exec-bearing SSH config directives, and that the dial
// path is a direct TCP dial (no proxy indirection).
func TestNoExecOrProxyInRemote(t *testing.T) {
	src, err := os.ReadFile("conn.go")
	if err != nil {
		t.Fatalf("read conn.go: %v", err)
	}
	text := string(src)
	for _, banned := range []string{"os/exec", "exec.Command", "exec.CommandContext", "ProxyCommand", "ProxyJump", "LocalCommand"} {
		if strings.Contains(text, banned) {
			t.Errorf("conn.go references %q — fur must never exec or proxy SSH connections (Chain E)", banned)
		}
	}
	if !strings.Contains(text, `ssh.Dial("tcp"`) {
		t.Error("conn.go no longer uses a direct ssh.Dial(\"tcp\", ...) — verify no proxy indirection was introduced")
	}
}

// TestProxyCommandParsedButIgnored confirms the behavior end to end: the
// ssh_config parser CAN see a ProxyCommand directive, but fur's connection
// logic resolves only the allowlisted keys from it and never executes the
// proxy command (no canary side effect).
func TestProxyCommandParsedButIgnored(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SSH_AUTH_SOCK", "") // no agent

	canary := filepath.Join(tmpHome, "pwned")
	cfgText := "Host evil\n" +
		"  Hostname 10.0.0.9\n" +
		"  User realuser\n" +
		"  Port 2222\n" +
		"  IdentityFile ~/.ssh/id_ed25519\n" +
		"  ProxyCommand /bin/sh -c \"touch " + canary + "\"\n" +
		"  LocalCommand touch " + canary + "\n"

	parsed, err := ssh_config.Decode(strings.NewReader(cfgText))
	if err != nil {
		t.Fatalf("decode ssh config: %v", err)
	}

	c := &Conn{target: Target{Host: "evil"}, sshCfg: parsed}

	// Parser sees ProxyCommand (proves the directive is present, not silently
	// dropped by the decoder) ...
	if pc := c.configGet("ProxyCommand"); pc == "" {
		t.Fatal("precondition: ssh_config parser should expose ProxyCommand")
	}
	// ... but the allowlisted resolvers reflect only safe keys.
	if got := c.resolveHost(); got != "10.0.0.9" {
		t.Errorf("resolveHost = %q, want 10.0.0.9", got)
	}
	if got := c.resolveUser(); got != "realuser" {
		t.Errorf("resolveUser = %q, want realuser", got)
	}
	if got := c.resolvePort(); got != 2222 {
		t.Errorf("resolvePort = %d, want 2222", got)
	}

	// buildSSHConfig exercises the auth/host-key path; with an isolated empty
	// HOME and no agent it returns an error, but it must never execute the
	// proxy/local command.
	_, _ = c.buildSSHConfig()

	if _, err := os.Stat(canary); err == nil {
		t.Fatalf("ProxyCommand/LocalCommand was executed — canary %s exists (Chain E RCE)", canary)
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
