// Package audit holds repository-wide, source-scanning regression guards that
// enforce cross-cutting security invariants no single package owns.
package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// moduleRoot returns the repository root (two levels up from internal/audit).
func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// walkGoSources calls fn for every non-test, non-vendored .go file under root.
func walkGoSources(t *testing.T, root string, fn func(path, text string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "demo", "delete-me":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fn(path, string(b))
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// shellExecPatterns match the argv-unsafe exec forms fur must never use: a
// shell interpreter invoked with -c lets any interpolated string become a
// command. fur always uses exec.Command(name, args...) with separate args.
var shellExecPatterns = []*regexp.Regexp{
	regexp.MustCompile(`exec\.Command(Context)?\(\s*"(/bin/)?sh"`),
	regexp.MustCompile(`exec\.Command(Context)?\(\s*"(/bin/)?bash"`),
	regexp.MustCompile(`exec\.Command(Context)?\(\s*"(/usr/bin/)?env"`),
	regexp.MustCompile(`"sh"\s*,\s*"-c"`),
	regexp.MustCompile(`"bash"\s*,\s*"-c"`),
}

// TestNoShellExec is the hardening 4.2 guard: no shell-out-with-interpolation
// anywhere in the codebase. A regression here would reintroduce the command
// injection class the audit closed in Chain G.
func TestNoShellExec(t *testing.T) {
	root := moduleRoot(t)
	walkGoSources(t, root, func(path, text string) {
		for _, re := range shellExecPatterns {
			if re.MatchString(text) {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s matches argv-unsafe shell-exec pattern %q — use "+
					"exec.Command(name, args...) with separate args (hardening 4.2)", rel, re.String())
			}
		}
	})
}

// TestExecSitesAreKnown pins the inventory of exec.Command call sites so a new
// one cannot be added without a conscious update to this guard (and a review
// of its argv safety). The map value documents why each site is safe.
func TestExecSitesAreKnown(t *testing.T) {
	known := map[string]string{
		"internal/tui/opener.go":    "EditorCmd/OpenSystemCmd route filenames through safeFilenameArg (Chain G)",
		"internal/export/export.go": "fixed PDF tool + argv-separated file paths",
		"internal/web/server.go":    "xdg-open with an http:// URL (own server), argv form",
		"internal/web/handlers.go":  "git grep / grep with -- separator before the query, argv form",
		"internal/doctor/doctor.go": "git --version, no user input",
	}
	root := moduleRoot(t)
	execRe := regexp.MustCompile(`exec\.Command(Context)?\(`)
	found := map[string]bool{}
	walkGoSources(t, root, func(path, text string) {
		if execRe.MatchString(text) {
			rel, _ := filepath.Rel(root, path)
			found[filepath.ToSlash(rel)] = true
		}
	})
	for site := range found {
		if _, ok := known[site]; !ok {
			t.Errorf("new exec.Command site %s is not in the reviewed inventory; "+
				"verify it uses argv-separated args (no shell, no user-controlled "+
				"command name, leading-dash filenames neutralized) and add it to "+
				"TestExecSitesAreKnown (hardening 4.2)", site)
		}
	}
	for site := range known {
		if !found[site] {
			t.Errorf("known exec site %s no longer present — remove it from the "+
				"inventory to keep this guard honest", site)
		}
	}
}
