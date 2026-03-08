package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"os/exec"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/lookit/internal/git"
)

// Check represents a single diagnostic check.
type Check struct {
	Name    string
	Status  CheckStatus
	Message string
}

// CheckStatus is the result of a diagnostic check.
type CheckStatus int

const (
	CheckOK CheckStatus = iota
	CheckWarn
	CheckFail
)

// Run executes all diagnostic checks and returns the results.
func Run() []Check {
	var checks []Check

	checks = append(checks, checkGo())
	checks = append(checks, checkGit())
	checks = append(checks, checkGitRepo())
	checks = append(checks, checkGitignore())
	checks = append(checks, checkTerminal())
	checks = append(checks, checkConfig())
	checks = append(checks, checkMarkdownFiles())
	checks = append(checks, checkLargeFiles())
	checks = append(checks, checkPDFTool())

	return checks
}

// Print displays diagnostic results to stdout with color.
func Print(checks []Check) {
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	nameStyle := lipgloss.NewStyle().Bold(true)

	for _, c := range checks {
		var icon string
		switch c.Status {
		case CheckOK:
			icon = okStyle.Render("[OK]")
		case CheckWarn:
			icon = warnStyle.Render("[WARN]")
		case CheckFail:
			icon = failStyle.Render("[FAIL]")
		}
		fmt.Printf("%s %s: %s\n", icon, nameStyle.Render(c.Name), c.Message)
	}

	// Summary
	var ok, warn, fail int
	for _, c := range checks {
		switch c.Status {
		case CheckOK:
			ok++
		case CheckWarn:
			warn++
		case CheckFail:
			fail++
		}
	}

	fmt.Println()
	summary := fmt.Sprintf("%d passed, %d warnings, %d failed", ok, warn, fail)
	if fail > 0 {
		fmt.Println(failStyle.Render(summary))
	} else if warn > 0 {
		fmt.Println(warnStyle.Render(summary))
	} else {
		fmt.Println(okStyle.Render(summary))
	}
}

func checkGo() Check {
	return Check{
		Name:    "Go runtime",
		Status:  CheckOK,
		Message: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

func checkGit() Check {
	cmd := exec.Command("git", "--version")
	out, err := cmd.Output()
	if err != nil {
		return Check{
			Name:    "Git",
			Status:  CheckFail,
			Message: "git not found in PATH",
		}
	}
	return Check{
		Name:    "Git",
		Status:  CheckOK,
		Message: strings.TrimSpace(string(out)),
	}
}

func checkGitRepo() Check {
	cwd, err := os.Getwd()
	if err != nil {
		return Check{
			Name:    "Git repository",
			Status:  CheckWarn,
			Message: "could not determine working directory",
		}
	}

	if git.IsRepo(cwd) {
		repo, err := git.Open(cwd)
		if err != nil {
			return Check{
				Name:    "Git repository",
				Status:  CheckOK,
				Message: "in a git repository",
			}
		}
		branch, err := repo.Branch()
		if err != nil {
			branch = "unknown"
		}
		return Check{
			Name:    "Git repository",
			Status:  CheckOK,
			Message: fmt.Sprintf("on branch %s", branch),
		}
	}

	return Check{
		Name:    "Git repository",
		Status:  CheckWarn,
		Message: "not inside a git repository",
	}
}

func checkGitignore() Check {
	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, ".gitignore")
	if _, err := os.Stat(path); err != nil {
		return Check{
			Name:    ".gitignore",
			Status:  CheckWarn,
			Message: "no .gitignore found",
		}
	}
	return Check{
		Name:    ".gitignore",
		Status:  CheckOK,
		Message: ".gitignore present",
	}
}

func checkTerminal() Check {
	term := os.Getenv("TERM")
	if term == "" {
		term = "(not set)"
	}

	colorterm := os.Getenv("COLORTERM")
	var colorSupport string
	switch colorterm {
	case "truecolor", "24bit":
		colorSupport = "truecolor"
	case "":
		if strings.Contains(term, "256color") {
			colorSupport = "256 colors"
		} else {
			colorSupport = "basic"
		}
	default:
		colorSupport = colorterm
	}

	cols := os.Getenv("COLUMNS")
	lines := os.Getenv("LINES")
	size := "unknown"
	if cols != "" && lines != "" {
		size = fmt.Sprintf("%sx%s", cols, lines)
	}

	return Check{
		Name:    "Terminal",
		Status:  CheckOK,
		Message: fmt.Sprintf("TERM=%s, color=%s, size=%s", term, colorSupport, size),
	}
}

func checkConfig() Check {
	home, err := os.UserHomeDir()
	if err != nil {
		return Check{
			Name:    "Config",
			Status:  CheckWarn,
			Message: "could not determine home directory",
		}
	}

	configPath := filepath.Join(home, ".config", "lookit", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return Check{
			Name:    "Config",
			Status:  CheckWarn,
			Message: fmt.Sprintf("no config file at %s (using defaults)", configPath),
		}
	}

	return Check{
		Name:    "Config",
		Status:  CheckOK,
		Message: fmt.Sprintf("config file at %s", configPath),
	}
}

func checkMarkdownFiles() Check {
	cwd, _ := os.Getwd()
	var count int

	_ = filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".markdown" || ext == ".mdown" {
			count++
		}
		return nil
	})

	if count == 0 {
		return Check{
			Name:    "Markdown files",
			Status:  CheckWarn,
			Message: "no markdown files found in current directory",
		}
	}

	return Check{
		Name:    "Markdown files",
		Status:  CheckOK,
		Message: fmt.Sprintf("%d markdown files found", count),
	}
}

func checkLargeFiles() Check {
	cwd, _ := os.Getwd()
	const threshold = 10 * 1024 * 1024 // 10MB
	var largeFiles []string

	_ = filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > threshold {
			rel, _ := filepath.Rel(cwd, path)
			largeFiles = append(largeFiles, fmt.Sprintf("%s (%.1fMB)", rel, float64(info.Size())/1024/1024))
		}
		return nil
	})

	if len(largeFiles) > 0 {
		msg := fmt.Sprintf("%d large files (>10MB): %s", len(largeFiles), strings.Join(largeFiles, ", "))
		if len(msg) > 200 {
			msg = fmt.Sprintf("%d large files (>10MB)", len(largeFiles))
		}
		return Check{
			Name:    "Large files",
			Status:  CheckWarn,
			Message: msg,
		}
	}

	return Check{
		Name:    "Large files",
		Status:  CheckOK,
		Message: "no files over 10MB",
	}
}

func checkPDFTool() Check {
	// Check for headless Chrome/Chromium first (best fidelity)
	for _, name := range []string{"chromium-browser", "chromium", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(name); err == nil {
			return Check{
				Name:    "PDF tool",
				Status:  CheckOK,
				Message: fmt.Sprintf("found %s", path),
			}
		}
	}
	// Fall back to wkhtmltopdf
	if path, err := exec.LookPath("wkhtmltopdf"); err == nil {
		return Check{
			Name:    "PDF tool",
			Status:  CheckOK,
			Message: fmt.Sprintf("found %s", path),
		}
	}
	return Check{
		Name:    "PDF tool",
		Status:  CheckWarn,
		Message: "no PDF tool found (install chromium, google-chrome, or wkhtmltopdf for PDF export)",
	}
}
