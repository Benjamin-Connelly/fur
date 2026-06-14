package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execFur runs the root command with the given args in an isolated HOME,
// capturing stdout. It exercises the full path including the loadConfig
// PersistentPreRunE. Returns captured stdout and any execute error.
func execFur(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	var execErr error
	out := captureStdout(t, func() {
		rootCmd.SetArgs(args)
		execErr = rootCmd.Execute()
	})
	return out, execErr
}

// fixtureTree builds a small tree (markdown + code + csv + a TODO) under a
// temp dir and returns (home, treeDir).
func fixtureTree(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "tree")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"README.md": "# Title\n\nSee [guide](guide.md).\n\n- [ ] TODO: wire it up !high #infra\n",
		"guide.md":  "# Guide\n\nback to [README](README.md)\n",
		"data.csv":  "name,age\nalice,30\nbob,25\n",
		"main.go":   "package main\n\nfunc main() {}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return home, dir
}

func TestExec_Cat(t *testing.T) {
	home, dir := fixtureTree(t)
	out, err := execFur(t, home, "cat", filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("cat: %v", err)
	}
	if !strings.Contains(out, "Title") {
		t.Errorf("cat output missing heading:\n%s", out)
	}
}

func TestExec_GraphDOTAndJSON(t *testing.T) {
	home, dir := fixtureTree(t)

	dot, err := execFur(t, home, "graph", dir)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	if !strings.Contains(dot, "digraph") {
		t.Errorf("graph DOT output missing 'digraph':\n%s", dot)
	}

	js, err := execFur(t, home, "graph", "--json", dir)
	if err != nil {
		t.Fatalf("graph --json: %v", err)
	}
	if !strings.Contains(js, "nodes") {
		t.Errorf("graph --json missing 'nodes':\n%s", js)
	}
}

func TestExec_Tasks(t *testing.T) {
	home, dir := fixtureTree(t)
	out, err := execFur(t, home, "tasks", dir)
	if err != nil {
		t.Fatalf("tasks: %v", err)
	}
	if !strings.Contains(out, "wire it up") {
		t.Errorf("tasks output missing the TODO:\n%s", out)
	}
}

func TestExec_Doctor(t *testing.T) {
	home, _ := fixtureTree(t)
	out, err := execFur(t, home, "doctor")
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if out == "" {
		t.Error("doctor produced no output")
	}
}

func TestExec_ConfigPathInitShow(t *testing.T) {
	home, _ := fixtureTree(t)

	if _, err := execFur(t, home, "config", "init"); err != nil {
		t.Fatalf("config init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "fur", "config.yaml")); err != nil {
		t.Errorf("config init did not write config.yaml: %v", err)
	}

	pathOut, err := execFur(t, home, "config", "path")
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if !strings.Contains(pathOut, "config.yaml") {
		t.Errorf("config path output: %q", pathOut)
	}

	showOut, err := execFur(t, home, "config", "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(showOut, "theme") {
		t.Errorf("config show missing keys:\n%s", showOut)
	}
}

func TestExec_Completion(t *testing.T) {
	home, _ := fixtureTree(t)
	for _, shell := range []string{"bash", "zsh", "fish"} {
		out, err := execFur(t, home, "completion", shell)
		if err != nil {
			t.Fatalf("completion %s: %v", shell, err)
		}
		if out == "" {
			t.Errorf("completion %s produced no script", shell)
		}
	}
}

func TestExec_GenMan(t *testing.T) {
	home, _ := fixtureTree(t)
	// gen-man copies pages to a "internal/manpages/pages" path relative to the
	// working directory; chdir into the temp HOME so it doesn't pollute the
	// source tree when the suite runs from cmd/fur.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(home); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	manDir := filepath.Join(home, "man")
	if _, err := execFur(t, home, "gen-man", manDir); err != nil {
		t.Fatalf("gen-man: %v", err)
	}
	entries, _ := filepath.Glob(filepath.Join(manDir, "**", "*.1"))
	if len(entries) == 0 {
		// Fall back to a recursive check (layout may nest under man1/).
		var found bool
		_ = filepath.Walk(manDir, func(p string, _ os.FileInfo, _ error) error {
			if strings.HasSuffix(p, ".1") {
				found = true
			}
			return nil
		})
		if !found {
			t.Error("gen-man wrote no .1 pages")
		}
	}
}

func TestExec_CompletionInstall(t *testing.T) {
	home, _ := fixtureTree(t)
	if _, err := execFur(t, home, "completion", "bash", "--install"); err != nil {
		t.Fatalf("completion --install: %v", err)
	}
	// The bash completion lands under the isolated HOME.
	want := filepath.Join(home, ".local", "share", "bash-completion", "completions", "fur")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("completion --install did not write %s: %v", want, err)
	}
}

func TestExec_FlagMerge(t *testing.T) {
	home, dir := fixtureTree(t)
	// Exercise the loadConfig flag-override branches.
	if _, err := execFur(t, home, "cat", "--theme", "dark", filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("cat --theme dark: %v", err)
	}
	if _, err := execFur(t, home, "--show-hidden", "tasks", dir); err != nil {
		t.Fatalf("--show-hidden tasks: %v", err)
	}
}

func TestExec_PipedStdin(t *testing.T) {
	home, _ := fixtureTree(t)
	t.Setenv("HOME", home)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = w.Write([]byte("# Piped Heading\n\nbody\n"))
		_ = w.Close()
	}()
	savedStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = savedStdin }()

	out := captureStdout(t, func() {
		rootCmd.SetArgs([]string{})
		_ = rootCmd.Execute()
	})
	// Glamour styles each word separately, so assert on a single token rather
	// than the full phrase.
	if !strings.Contains(out, "Piped") || !strings.Contains(out, "body") {
		t.Errorf("piped stdin not rendered:\n%s", out)
	}
}

func TestExec_CompletionAllShells(t *testing.T) {
	home, _ := fixtureTree(t)
	// stdout generation for every shell (covers genCompletion branches).
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		if _, err := execFur(t, home, "completion", sh); err != nil {
			t.Fatalf("completion %s: %v", sh, err)
		}
	}
	// --install for the file-writing shells (covers installCompletion branches).
	for _, sh := range []string{"zsh", "fish"} {
		if _, err := execFur(t, home, "completion", sh, "--install"); err != nil {
			t.Fatalf("completion %s --install: %v", sh, err)
		}
	}
}

func TestLoadConfig_ServeFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Set serve-specific flags and run loadConfig against serveCmd to cover the
	// serve-flag merge branch.
	_ = serveCmd.Flags().Set("no-https", "true")
	_ = serveCmd.Flags().Set("open", "true")
	_ = serveCmd.Flags().Set("listen-public", "true")
	_ = serveCmd.Flags().Set("css", "/tmp/x.css")
	t.Cleanup(func() {
		_ = serveCmd.Flags().Set("no-https", "false")
		_ = serveCmd.Flags().Set("open", "false")
		_ = serveCmd.Flags().Set("listen-public", "false")
		_ = serveCmd.Flags().Set("css", "")
	})

	if err := loadConfig(serveCmd, nil); err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if !cfg.Server.NoHTTPS || !cfg.Server.Open || !cfg.Server.ListenPublic {
		t.Errorf("serve flags not merged: %+v", cfg.Server)
	}
	if cfg.Server.CustomCSS != "/tmp/x.css" {
		t.Errorf("css not merged: %q", cfg.Server.CustomCSS)
	}
}

func TestExec_CatJSON(t *testing.T) {
	home, dir := fixtureTree(t)
	// --json path of the cat command.
	out, err := execFur(t, home, "cat", "--json", filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("cat --json: %v", err)
	}
	if !strings.Contains(out, "\"") {
		t.Errorf("cat --json produced no JSON:\n%s", out)
	}
}

func TestExec_Export(t *testing.T) {
	home, dir := fixtureTree(t)
	outDir := filepath.Join(home, "out")
	_, err := execFur(t, home, "export", "--format", "html", "--output", outDir, dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	// At least one .html should have been written.
	var found bool
	_ = filepath.Walk(outDir, func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".html") {
			found = true
		}
		return nil
	})
	if !found {
		t.Error("export produced no .html files")
	}
}
