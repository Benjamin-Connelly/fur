package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withTempDir changes to a temp directory for the duration of fn, then restores
// the original working directory.
func withTempDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	fn(dir)
}

func TestCheckGo(t *testing.T) {
	c := checkGo()
	if c.Status != CheckOK {
		t.Errorf("expected CheckOK, got %d", c.Status)
	}
	if c.Name != "Go runtime" {
		t.Errorf("expected name 'Go runtime', got %q", c.Name)
	}
	goVer := runtime.Version()
	if !strings.Contains(c.Message, goVer) {
		t.Errorf("expected message to contain %q, got %q", goVer, c.Message)
	}
	goos := runtime.GOOS
	if !strings.Contains(c.Message, goos) {
		t.Errorf("expected message to contain %q, got %q", goos, c.Message)
	}
}

func TestCheckGit(t *testing.T) {
	c := checkGit()
	if c.Name != "Git" {
		t.Errorf("expected name 'Git', got %q", c.Name)
	}
	// Git may or may not be in PATH depending on environment
	switch c.Status {
	case CheckOK:
		if !strings.Contains(c.Message, "git version") {
			t.Errorf("expected message to contain 'git version', got %q", c.Message)
		}
	case CheckFail:
		if !strings.Contains(c.Message, "not found") {
			t.Errorf("expected 'not found' message, got %q", c.Message)
		}
	default:
		t.Errorf("unexpected status %d", c.Status)
	}
}

func TestCheckGitignore_Present(t *testing.T) {
	withTempDir(t, func(dir string) {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
			t.Fatal(err)
		}
		c := checkGitignore()
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK, got %d", c.Status)
		}
		if c.Message != ".gitignore present" {
			t.Errorf("unexpected message: %q", c.Message)
		}
	})
}

func TestCheckGitignore_Missing(t *testing.T) {
	withTempDir(t, func(dir string) {
		c := checkGitignore()
		if c.Status != CheckWarn {
			t.Errorf("expected CheckWarn, got %d", c.Status)
		}
		if c.Message != "no .gitignore found" {
			t.Errorf("unexpected message: %q", c.Message)
		}
	})
}

func TestCheckMarkdownFiles_Found(t *testing.T) {
	withTempDir(t, func(dir string) {
		for _, name := range []string{"README.md", "docs.markdown", "notes.mdown"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("# Hello\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		c := checkMarkdownFiles()
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK, got %d", c.Status)
		}
		if !strings.Contains(c.Message, "3 markdown files") {
			t.Errorf("expected 3 files counted, got %q", c.Message)
		}
	})
}

func TestCheckMarkdownFiles_None(t *testing.T) {
	withTempDir(t, func(dir string) {
		// Create a non-markdown file so the directory isn't empty.
		if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi\n"), 0644); err != nil {
			t.Fatal(err)
		}
		c := checkMarkdownFiles()
		if c.Status != CheckWarn {
			t.Errorf("expected CheckWarn, got %d", c.Status)
		}
		if !strings.Contains(c.Message, "no markdown files") {
			t.Errorf("unexpected message: %q", c.Message)
		}
	})
}

func TestCheckMarkdownFiles_SkipsDirs(t *testing.T) {
	withTempDir(t, func(dir string) {
		// One markdown file at root should be counted.
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hi\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Files inside skipped directories should not be counted.
		for _, skipDir := range []string{".git", "node_modules", "vendor"} {
			subdir := filepath.Join(dir, skipDir)
			if err := os.MkdirAll(subdir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(subdir, "notes.md"), []byte("# Skip me\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		c := checkMarkdownFiles()
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK, got %d", c.Status)
		}
		if !strings.Contains(c.Message, "1 markdown file") {
			t.Errorf("expected exactly 1 file counted, got %q", c.Message)
		}
	})
}

func TestCheckLargeFiles_None(t *testing.T) {
	withTempDir(t, func(dir string) {
		// Create a small file.
		if err := os.WriteFile(filepath.Join(dir, "small.txt"), []byte("small\n"), 0644); err != nil {
			t.Fatal(err)
		}
		c := checkLargeFiles()
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK, got %d", c.Status)
		}
		if c.Message != "no files over 10MB" {
			t.Errorf("unexpected message: %q", c.Message)
		}
	})
}

func TestCheckLargeFiles_SkipsDirs(t *testing.T) {
	withTempDir(t, func(dir string) {
		// Even if a skipped directory had large files, they shouldn't be reported.
		// We can't easily create a 10MB+ file in .git without slowing the test,
		// so just verify no false positives with small files in skipped dirs.
		for _, skipDir := range []string{".git", "node_modules", "vendor"} {
			subdir := filepath.Join(dir, skipDir)
			if err := os.MkdirAll(subdir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(subdir, "data.bin"), []byte("data"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		c := checkLargeFiles()
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK, got %d", c.Status)
		}
	})
}

func TestCheckTerminal(t *testing.T) {
	c := checkTerminal()
	if c.Status != CheckOK {
		t.Errorf("expected CheckOK, got %d", c.Status)
	}
	if c.Name != "Terminal" {
		t.Errorf("expected name 'Terminal', got %q", c.Name)
	}
	if !strings.Contains(c.Message, "TERM=") {
		t.Errorf("expected message to contain 'TERM=', got %q", c.Message)
	}
	if !strings.Contains(c.Message, "color=") {
		t.Errorf("expected message to contain 'color=', got %q", c.Message)
	}
	if !strings.Contains(c.Message, "size=") {
		t.Errorf("expected message to contain 'size=', got %q", c.Message)
	}
}

func TestCheckTerminal_Truecolor(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	c := checkTerminal()
	if !strings.Contains(c.Message, "color=truecolor") {
		t.Errorf("expected truecolor detection, got %q", c.Message)
	}
}

func TestCheckTerminal_24bit(t *testing.T) {
	t.Setenv("COLORTERM", "24bit")
	c := checkTerminal()
	if !strings.Contains(c.Message, "color=truecolor") {
		t.Errorf("expected truecolor detection for 24bit, got %q", c.Message)
	}
}

func TestCheckTerminal_256color(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	c := checkTerminal()
	if !strings.Contains(c.Message, "color=256 colors") {
		t.Errorf("expected 256 color detection, got %q", c.Message)
	}
}

func TestCheckTerminal_Basic(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "dumb")
	c := checkTerminal()
	if !strings.Contains(c.Message, "color=basic") {
		t.Errorf("expected basic color detection, got %q", c.Message)
	}
}

func TestCheckTerminal_WithSize(t *testing.T) {
	t.Setenv("COLUMNS", "120")
	t.Setenv("LINES", "40")
	c := checkTerminal()
	if !strings.Contains(c.Message, "size=120x40") {
		t.Errorf("expected size=120x40 in message, got %q", c.Message)
	}
}

func TestCheckTerminal_NoSize(t *testing.T) {
	t.Setenv("COLUMNS", "")
	t.Setenv("LINES", "")
	c := checkTerminal()
	if !strings.Contains(c.Message, "size=unknown") {
		t.Errorf("expected size=unknown in message, got %q", c.Message)
	}
}

func TestCheckConfig(t *testing.T) {
	c := checkConfig()
	// We don't control whether the config file exists, but we can verify the
	// check returns a valid status and references the config path.
	if c.Status != CheckOK && c.Status != CheckWarn {
		t.Errorf("expected CheckOK or CheckWarn, got %d", c.Status)
	}
	if c.Name != "Config" {
		t.Errorf("expected name 'Config', got %q", c.Name)
	}
	if !strings.Contains(c.Message, "config") {
		t.Errorf("expected message to reference config, got %q", c.Message)
	}
}

func TestCheckPDFTool(t *testing.T) {
	c := checkPDFTool()
	if c.Name != "PDF tool" {
		t.Errorf("expected name 'PDF tool', got %q", c.Name)
	}

	// Determine expected status based on what's actually in PATH.
	hasTool := false
	for _, name := range []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable", "wkhtmltopdf"} {
		if _, err := exec.LookPath(name); err == nil {
			hasTool = true
			break
		}
	}

	if hasTool {
		if c.Status != CheckOK {
			t.Errorf("expected CheckOK (tool found in PATH), got %d", c.Status)
		}
		if !strings.Contains(c.Message, "found") {
			t.Errorf("expected message to contain 'found', got %q", c.Message)
		}
	} else {
		if c.Status != CheckWarn {
			t.Errorf("expected CheckWarn (no tool in PATH), got %d", c.Status)
		}
		if !strings.Contains(c.Message, "no PDF tool found") {
			t.Errorf("unexpected message: %q", c.Message)
		}
	}
}

func TestCheckGitRepo_InRepo(t *testing.T) {
	// The test is running inside the fur repo, so this should return OK.
	c := checkGitRepo()
	if c.Name != "Git repository" {
		t.Errorf("expected name 'Git repository', got %q", c.Name)
	}
	if c.Status != CheckOK {
		t.Errorf("expected CheckOK (running inside repo), got %d", c.Status)
	}
	if !strings.Contains(c.Message, "branch") {
		t.Errorf("expected message to mention branch, got %q", c.Message)
	}
}

func TestCheckGitRepo_NotInRepo(t *testing.T) {
	withTempDir(t, func(dir string) {
		c := checkGitRepo()
		if c.Status != CheckWarn {
			t.Errorf("expected CheckWarn (not a repo), got %d", c.Status)
		}
		if !strings.Contains(c.Message, "not inside") {
			t.Errorf("unexpected message: %q", c.Message)
		}
	})
}

func TestRun(t *testing.T) {
	checks := Run()
	if len(checks) != 9 {
		t.Errorf("expected 9 checks, got %d", len(checks))
	}

	// Verify each check has a non-empty name and message.
	for i, c := range checks {
		if c.Name == "" {
			t.Errorf("check %d has empty name", i)
		}
		if c.Message == "" {
			t.Errorf("check %d (%s) has empty message", i, c.Name)
		}
		if c.Status < CheckOK || c.Status > CheckFail {
			t.Errorf("check %d (%s) has invalid status %d", i, c.Name, c.Status)
		}
	}

	// Verify expected check names in order.
	expectedNames := []string{
		"Go runtime", "Git", "Git repository", ".gitignore",
		"Terminal", "Config", "Markdown files", "Large files", "PDF tool",
	}
	for i, name := range expectedNames {
		if checks[i].Name != name {
			t.Errorf("check %d: expected name %q, got %q", i, name, checks[i].Name)
		}
	}
}

func TestPrint_NoPanic(t *testing.T) {
	// Verify Print handles all status types without panicking.
	checks := []Check{
		{Name: "ok check", Status: CheckOK, Message: "all good"},
		{Name: "warn check", Status: CheckWarn, Message: "watch out"},
		{Name: "fail check", Status: CheckFail, Message: "broken"},
	}
	// Print writes to stdout; just ensure no panic.
	Print(checks)
}

func TestPrint_Empty(t *testing.T) {
	// Verify Print handles empty slice without panicking.
	Print(nil)
	Print([]Check{})
}

func TestPrint_AllOK(t *testing.T) {
	checks := []Check{
		{Name: "a", Status: CheckOK, Message: "fine"},
		{Name: "b", Status: CheckOK, Message: "fine"},
	}
	Print(checks)
}

func TestPrint_AllFail(t *testing.T) {
	checks := []Check{
		{Name: "a", Status: CheckFail, Message: "bad"},
		{Name: "b", Status: CheckFail, Message: "bad"},
	}
	Print(checks)
}

func TestCheckStatus_Values(t *testing.T) {
	// Verify iota ordering is correct.
	if CheckOK != 0 {
		t.Errorf("CheckOK should be 0, got %d", CheckOK)
	}
	if CheckWarn != 1 {
		t.Errorf("CheckWarn should be 1, got %d", CheckWarn)
	}
	if CheckFail != 2 {
		t.Errorf("CheckFail should be 2, got %d", CheckFail)
	}
}
