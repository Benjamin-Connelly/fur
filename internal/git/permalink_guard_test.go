package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitPackageNoExec is the Chain I regression guard.
//
// Chain I asks whether a hostile git remote URL (e.g.
// "ssh://-oProxyCommand=evil@host/x" or "git@$(touch pwned):repo") can reach
// a command execution via the permalink builder. fur uses go-git (pure Go,
// no `git` subprocess) and the permalink builder is pure string manipulation,
// so there is no exec to inject into. This guard fails if any non-test source
// in internal/git introduces os/exec — which would reopen the vector.
func TestGitPackageNoExec(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(src)
		for _, banned := range []string{`"os/exec"`, "exec.Command", "exec.CommandContext"} {
			if strings.Contains(text, banned) {
				t.Errorf("%s references %q — the git/permalink path must never shell "+
					"out; a hostile remote URL would become a command-injection vector "+
					"(Chain I)", name, banned)
			}
		}
	}
}

// TestNormalizeRemoteURLHostile feeds adversarial origin URLs through the
// permalink normalizer and asserts it never panics, never executes a shell
// command (canary), and always yields a plain string. The exact output is not
// the contract here — the contract is "no exec, no panic".
func TestNormalizeRemoteURLHostile(t *testing.T) {
	canary := filepath.Join(t.TempDir(), "pwned")
	hostile := []string{
		"ssh://-oProxyCommand=touch " + canary + "@github.com/o/r.git",
		"git@$(touch " + canary + "):o/r.git",
		"git@`touch " + canary + "`:o/r",
		"https://github.com/o/r$(touch " + canary + ").git",
		"ssh://git@github.com:22/o/r;touch " + canary,
		"git://github.com/o/r\n touch " + canary,
		"https://github.com/" + strings.Repeat("a", 100000) + "/r",
		"",
		"::::",
		"git@:",
	}
	for _, u := range hostile {
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					t.Errorf("normalizeRemoteURL panicked on %q: %v", u, rec)
				}
			}()
			out := normalizeRemoteURL(u)
			_ = detectStyle(u)
			_ = buildFileLink(out, detectStyle(u), "main", "README.md", 1, 0)
		}()
	}

	if _, err := os.Stat(canary); err == nil {
		t.Fatalf("a hostile remote URL executed a command — canary %s exists (Chain I)", canary)
	}
}
