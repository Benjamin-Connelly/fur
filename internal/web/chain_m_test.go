package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/render"
)

// TestHandleAPIDocumentSlugsCentralized is the Chain M web-boundary guard.
//
// The /__api/document endpoint must emit exactly the slugs produced by the
// centralized render.AnchorSlugs, so a fragment link the client builds from
// the API resolves to the same heading the TUI and server-side TOC resolve.
// Duplicate headings must get unique, deterministic suffixes.
func TestHandleAPIDocumentSlugsCentralized(t *testing.T) {
	dir := t.TempDir()
	content := "# Setup\n\ntext\n\n## Setup\n\nmore\n\n## Setup\n\nend\n"
	if err := os.WriteFile(filepath.Join(dir, "dup.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=dup.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result struct {
		Headings []struct {
			Slug string `json:"slug"`
		} `json:"headings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := render.AnchorSlugs(content)
	if len(result.Headings) != len(want) {
		t.Fatalf("got %d headings, want %d", len(result.Headings), len(want))
	}
	seen := map[string]bool{}
	for i, h := range result.Headings {
		if h.Slug != want[i] {
			t.Errorf("heading %d slug = %q, want %q (API diverged from render.AnchorSlugs)", i, h.Slug, want[i])
		}
		if seen[h.Slug] {
			t.Errorf("duplicate slug %q in API output (Chain M anchor collision)", h.Slug)
		}
		seen[h.Slug] = true
	}
}
