package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/config"
)

func TestIsMarkdownExt(t *testing.T) {
	cases := map[string]bool{
		".md": true, ".markdown": true, ".mdown": true,
		".MD": true, ".Markdown": true, // case-insensitive
		".txt": false, ".go": false, "": false, "md": false,
	}
	for ext, want := range cases {
		if got := isMarkdownExt(ext); got != want {
			t.Errorf("isMarkdownExt(%q) = %v, want %v", ext, got, want)
		}
	}
}

func TestIsImageExt(t *testing.T) {
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg", ".ico"} {
		if !isImageExt(ext) {
			t.Errorf("isImageExt(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{".md", ".txt", ".PNG", "", ".go"} {
		if isImageExt(ext) {
			t.Errorf("isImageExt(%q) = true, want false", ext)
		}
	}
}

func TestRenderWidth(t *testing.T) {
	// In tests stdout is not a wide TTY; renderWidth must return the 80 default
	// (or a real terminal width >= 20). Either way it must be sane.
	if w := renderWidth(); w < 20 {
		t.Errorf("renderWidth() = %d, want >= 20", w)
	}
}

func TestDetectShell(t *testing.T) {
	t.Setenv("PSModulePath", "")
	for _, sh := range []string{"bash", "zsh", "fish"} {
		t.Setenv("SHELL", "/usr/bin/"+sh)
		if got := detectShell(); got != sh {
			t.Errorf("detectShell() with SHELL=%s = %q, want %q", sh, got, sh)
		}
	}
	t.Setenv("SHELL", "/usr/bin/unknownsh")
	if got := detectShell(); got != "" {
		t.Errorf("detectShell() for unknown shell = %q, want empty", got)
	}
	t.Setenv("SHELL", "")
	t.Setenv("PSModulePath", "C:\\x")
	if got := detectShell(); got != "powershell" {
		t.Errorf("detectShell() with PSModulePath = %q, want powershell", got)
	}
}

func TestCompletionPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cases := []struct {
		shell      string
		destSuffix string
		wantInstr  bool
	}{
		{"bash", filepath.Join("completions", "fur"), false},
		{"zsh", filepath.Join(".zfunc", "_fur"), true},
		{"fish", filepath.Join("completions", "fur.fish"), false},
		{"powershell", "", true},
	}
	for _, tc := range cases {
		dest, instr := completionPath(tc.shell)
		if tc.destSuffix != "" && !strings.HasSuffix(dest, tc.destSuffix) {
			t.Errorf("completionPath(%q) dest=%q, want suffix %q", tc.shell, dest, tc.destSuffix)
		}
		if (instr != "") != tc.wantInstr {
			t.Errorf("completionPath(%q) instr=%q, wantInstr=%v", tc.shell, instr, tc.wantInstr)
		}
	}
	if dest, instr := completionPath("nonsense"); dest != "" || instr != "" {
		t.Errorf("completionPath(unknown) = (%q,%q), want empty", dest, instr)
	}
}

func TestResolveRemoteTarget(t *testing.T) {
	saved := cfg
	t.Cleanup(func() { cfg = saved })
	cfg = config.DefaultConfig()

	// SCP-style remote.
	if tgt := resolveRemoteTarget("host:/srv/docs"); tgt == nil || tgt.Host != "host" {
		t.Errorf("resolveRemoteTarget(host:/srv/docs) = %+v, want host=host", tgt)
	}
	// Local path is not a remote target.
	if tgt := resolveRemoteTarget("./local/path"); tgt != nil {
		t.Errorf("resolveRemoteTarget(local) = %+v, want nil", tgt)
	}
	// Named remote that is NOT configured → nil.
	if tgt := resolveRemoteTarget("@missing"); tgt != nil {
		t.Errorf("resolveRemoteTarget(@missing) = %+v, want nil", tgt)
	}
}

// TestResolveRemoteTarget_NamedRemote exercises the @name lookup against a
// populated config.
func TestResolveRemoteTarget_NamedRemote(t *testing.T) {
	saved := cfg
	t.Cleanup(func() { cfg = saved })
	cfg = config.DefaultConfig()
	cfg.Remotes = map[string]config.RemoteConfig{
		"docs": {Host: "h", User: "u", Port: 2222, Path: "/d"},
	}

	tgt := resolveRemoteTarget("@docs")
	if tgt == nil {
		t.Fatal("resolveRemoteTarget(@docs) = nil, want a target")
	}
	if tgt.Host != "h" || tgt.User != "u" || tgt.Port != 2222 || tgt.Path != "/d" {
		t.Errorf("resolveRemoteTarget(@docs) = %+v, want h/u/2222//d", tgt)
	}
}

// versionOutput captures stdout while running the version command.
func TestVersionCommand(t *testing.T) {
	out := captureStdout(t, func() {
		versionCmd.Run(versionCmd, nil)
	})
	for _, want := range []string{"fur ", "commit:", "built:", "go:", "os/arch:"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q; got:\n%s", want, out)
		}
	}
	// The ASCII banner is printed above the version info.
	if !strings.Contains(out, "Further Reading") {
		t.Errorf("version output missing the banner; got:\n%s", out)
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns what was
// written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 1024)
		for {
			n, err := r.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if err != nil {
				break
			}
		}
		done <- string(buf)
	}()
	fn()
	w.Close()
	os.Stdout = saved
	return <-done
}
