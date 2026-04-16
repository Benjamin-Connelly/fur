package manpages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedPagesExist(t *testing.T) {
	entries, err := pages.ReadDir("pages")
	if err != nil {
		t.Fatalf("reading embedded pages: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no embedded man pages found")
	}

	// Verify main page exists
	found := false
	for _, e := range entries {
		if e.Name() == "fur.1" {
			found = true
		}
	}
	if !found {
		t.Error("fur.1 not found in embedded pages")
	}
}

func TestEmbeddedPagesReadable(t *testing.T) {
	entries, _ := pages.ReadDir("pages")
	for _, e := range entries {
		data, err := pages.ReadFile("pages/" + e.Name())
		if err != nil {
			t.Errorf("reading %s: %v", e.Name(), err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", e.Name())
		}
	}
}

func TestInstall(t *testing.T) {
	tmpDir := t.TempDir()

	// Override destDir by installing to a temp location
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	n, err := Install("v0.0.1-test")
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least 1 page installed")
	}

	// Verify files exist
	manDir := filepath.Join(tmpDir, ".local", "share", "man", "man1")
	entries, err := os.ReadDir(manDir)
	if err != nil {
		t.Fatalf("reading install dir: %v", err)
	}

	pageCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".1" {
			pageCount++
		}
	}
	if pageCount != n {
		t.Errorf("installed %d pages but found %d files", n, pageCount)
	}

	// Verify version stamp
	stamp, err := os.ReadFile(filepath.Join(manDir, ".fur-version"))
	if err != nil {
		t.Fatalf("reading version stamp: %v", err)
	}
	if string(stamp) != "v0.0.1-test" {
		t.Errorf("version stamp = %q, want v0.0.1-test", stamp)
	}
}

func TestInstallSkipsWhenCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// First install
	n1, err := Install("v1.0.0")
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if n1 == 0 {
		t.Fatal("first install should write pages")
	}

	// Second install with same version — should skip
	n2, err := Install("v1.0.0")
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0 pages installed on second call, got %d", n2)
	}
}

func TestInstallUpgradesOnVersionChange(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	Install("v1.0.0")

	// New version should reinstall
	n, err := Install("v2.0.0")
	if err != nil {
		t.Fatalf("upgrade install: %v", err)
	}
	if n == 0 {
		t.Error("expected pages reinstalled on version change")
	}

	// Verify stamp updated
	manDir := filepath.Join(tmpDir, ".local", "share", "man", "man1")
	stamp, _ := os.ReadFile(filepath.Join(manDir, ".fur-version"))
	if string(stamp) != "v2.0.0" {
		t.Errorf("stamp = %q, want v2.0.0", stamp)
	}
}
