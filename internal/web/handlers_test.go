package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Benjamin-Connelly/fur/internal/config"
	gitpkg "github.com/Benjamin-Connelly/fur/internal/git"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/render"
)

// --- handleRoot tests ---

func TestHandleRootServesDirectory(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestHandleRootPathTraversalBlocked(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	paths := []string{
		"/../etc/passwd",
		"/../../etc/shadow",
		"/docs/../../etc/passwd",
	}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		s.handleRoot(rec, req)

		// filepath.Clean resolves ".." so these become clean paths that
		// either don't exist (404) or are blocked (403). Either is safe.
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
			t.Errorf("path %q: status = %d, want 403 or 404", path, rec.Code)
		}
	}
}

func TestHandleRoot404ForMissing(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/nonexistent.txt", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleRootServesMarkdown(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/README.md", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Error("markdown page should contain rendered heading text")
	}
}

func TestHandleRootServesCodeFile(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/main.go", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "main") {
		t.Error("code page should contain source content")
	}
}

func TestHandleRootServesSubdirectory(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/docs", nil)
	rec := httptest.NewRecorder()
	s.handleRoot(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "guide.md") {
		t.Error("directory listing should contain guide.md")
	}
}

// --- handleDirectory tests ---

func TestHandleDirectoryListsChildren(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	s.handleDirectory(rec, req, ".")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// Should list files at root level
	if !strings.Contains(body, "README.md") {
		t.Error("should list README.md")
	}
	if !strings.Contains(body, "main.go") {
		t.Error("should list main.go")
	}
	if !strings.Contains(body, "docs") {
		t.Error("should list docs directory")
	}
}

func TestHandleDirectorySubdir(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/docs", nil)
	rec := httptest.NewRecorder()
	s.handleDirectory(rec, req, "docs")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "guide.md") {
		t.Error("should list guide.md in docs")
	}
}

// --- handleMarkdown tests ---

func TestHandleMarkdownRendersHTML(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/README.md", nil)
	rec := httptest.NewRecorder()
	s.handleMarkdown(rec, req, "README.md")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	// Goldmark should render the heading
	if !strings.Contains(body, "Hello") {
		t.Error("should contain rendered heading")
	}
	// Content paragraph
	if !strings.Contains(body, "World") {
		t.Error("should contain paragraph text")
	}
}

func TestHandleMarkdownExtractsTOC(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/README.md", nil)
	rec := httptest.NewRecorder()
	s.handleMarkdown(rec, req, "README.md")

	body := rec.Body.String()
	// The TOC slug for "Hello" should appear as an anchor
	if !strings.Contains(body, "hello") {
		t.Error("should contain TOC slug for heading")
	}
}

func TestHandleMarkdownMermaidPostProcessed(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/README.md", nil)
	rec := httptest.NewRecorder()
	s.handleMarkdown(rec, req, "README.md")

	body := rec.Body.String()
	// Mermaid code blocks should be converted to <pre class="mermaid">
	if !strings.Contains(body, `class="mermaid"`) {
		t.Error("mermaid blocks should be post-processed")
	}
	// Should NOT contain language-mermaid class (goldmark's default)
	if strings.Contains(body, `language-mermaid`) {
		t.Error("language-mermaid class should be replaced by mermaid class")
	}
}

func TestHandleMarkdownIncludesBacklinks(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	// guide.md has a backlink from README.md via the link graph
	req := httptest.NewRequest("GET", "/docs/guide.md", nil)
	rec := httptest.NewRecorder()
	s.handleMarkdown(rec, req, "docs/guide.md")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestHandleMarkdownMissingFile(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/ghost.md", nil)
	rec := httptest.NewRecorder()
	s.handleMarkdown(rec, req, "ghost.md")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// --- handleFile tests ---

func TestHandleFileHighlightsCode(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/main.go", nil)
	rec := httptest.NewRecorder()
	s.handleFile(rec, req, "main.go")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Chroma should produce HTML with spans for syntax highlighting
	if !strings.Contains(body, "chroma") {
		t.Error("should contain Chroma CSS classes for syntax highlighting")
	}
}

func TestHandleFileDetectsLanguage(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/main.go", nil)
	rec := httptest.NewRecorder()
	s.handleFile(rec, req, "main.go")

	body := rec.Body.String()
	if !strings.Contains(body, "Go") {
		t.Error("should detect Go language")
	}
}

func TestHandleFileMissing(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/ghost.go", nil)
	rec := httptest.NewRecorder()
	s.handleFile(rec, req, "ghost.go")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// --- handleAPIFiles tests ---

func TestHandleAPIFilesReturnsJSON(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/files", nil)
	rec := httptest.NewRecorder()
	s.handleAPIFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var entries []index.FileEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry")
	}
}

func TestHandleAPIFilesFuzzySearch(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/files?q=readme", nil)
	rec := httptest.NewRecorder()
	s.handleAPIFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var entries []index.FileEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.RelPath), "readme") {
			found = true
			break
		}
	}
	if !found {
		t.Error("fuzzy search for 'readme' should match README.md")
	}
}

// --- handleAPISearch tests ---

func TestHandleAPISearchEmptyQuery(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/search?q=", nil)
	rec := httptest.NewRecorder()
	s.handleAPISearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var results []searchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("empty query should return empty results, got %d", len(results))
	}
}

func TestHandleAPISearchLongQuery(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	longQuery := strings.Repeat("a", 201)
	req := httptest.NewRequest("GET", "/__api/search?q="+longQuery, nil)
	rec := httptest.NewRecorder()
	s.handleAPISearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var results []searchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("oversized query should return empty results, got %d", len(results))
	}
}

func TestHandleAPISearchReturnsJSON(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/search?q=World", nil)
	rec := httptest.NewRecorder()
	s.handleAPISearch(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// --- handleAPIGraph tests ---

func TestHandleAPIGraphReturnsNodesAndLinks(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/graph", nil)
	rec := httptest.NewRecorder()
	s.handleAPIGraph(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var data struct {
		Nodes []struct {
			ID         string `json:"id"`
			Label      string `json:"label"`
			IsMarkdown bool   `json:"isMarkdown"`
			Links      int    `json:"links"`
		} `json:"nodes"`
		Links []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"links"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(data.Nodes) == 0 {
		t.Error("expected at least one node")
	}
	if len(data.Links) == 0 {
		t.Error("expected at least one link")
	}

	// Verify the link from README.md -> docs/guide.md
	foundLink := false
	for _, l := range data.Links {
		if l.Source == "README.md" && l.Target == "docs/guide.md" {
			foundLink = true
			break
		}
	}
	if !foundLink {
		t.Error("expected link from README.md to docs/guide.md")
	}
}

func TestHandleAPIGraphEmptyGraph(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	// Replace with empty link graph
	s.links = index.NewLinkGraph()

	req := httptest.NewRequest("GET", "/__api/graph", nil)
	rec := httptest.NewRecorder()
	s.handleAPIGraph(rec, req)

	var data struct {
		Nodes []interface{} `json:"nodes"`
		Links []interface{} `json:"links"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	// Empty graph should still return valid JSON with null/empty arrays
}

// --- handleSSE tests ---

func TestHandleSSESetsCorrectHeaders(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__events", nil)
	rec := httptest.NewRecorder()

	// Run in goroutine since handleSSE blocks; cancel via context
	ctx, cancel := newCancelContext(req)
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		s.handleSSE(rec, req)
		close(done)
	}()

	// Give handler time to set headers
	cancel()
	<-done

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	conn := rec.Header().Get("Connection")
	if conn != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", conn)
	}
}

// --- handleGraph tests ---

func TestHandleGraphServesTemplate(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/graph", nil)
	rec := httptest.NewRecorder()
	s.handleGraph(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Link Graph") {
		t.Error("graph page should contain 'Link Graph' title")
	}
}

// --- slugify tests ---

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Getting Started", "getting-started"},
		{"foo_bar_baz", "foo_bar_baz"},
		{"Hello   World", "hello---world"},
		{"CamelCase", "camelcase"},
		{"with 123 numbers", "with-123-numbers"},
		{"  leading trailing  ", "--leading-trailing--"},
		{"special!@#$chars", "specialchars"},
		{"hyphen-case", "hyphen-case"},
		{"", ""},
	}
	for _, tt := range tests {
		got := render.Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- sortDirEntries tests ---

func TestSortDirEntries(t *testing.T) {
	entries := []dirEntry{
		{Name: "zebra.go", IsDir: false},
		{Name: "docs", IsDir: true},
		{Name: "alpha.go", IsDir: false},
		{Name: "src", IsDir: true},
		{Name: "beta.md", IsDir: false},
	}
	sortDirEntries(entries)

	// Dirs should come first
	if !entries[0].IsDir || !entries[1].IsDir {
		t.Error("directories should be sorted first")
	}
	// Dirs should be alphabetical
	if entries[0].Name != "docs" || entries[1].Name != "src" {
		t.Errorf("dirs order: got %s, %s; want docs, src", entries[0].Name, entries[1].Name)
	}
	// Files should be alphabetical
	if entries[2].Name != "alpha.go" || entries[3].Name != "beta.md" || entries[4].Name != "zebra.go" {
		t.Errorf("files order: got %s, %s, %s; want alpha.go, beta.md, zebra.go",
			entries[2].Name, entries[3].Name, entries[4].Name)
	}
}

func TestSortDirEntriesCaseInsensitive(t *testing.T) {
	entries := []dirEntry{
		{Name: "Zebra.go", IsDir: false},
		{Name: "alpha.go", IsDir: false},
	}
	sortDirEntries(entries)

	if entries[0].Name != "alpha.go" {
		t.Errorf("case-insensitive sort: got %s first, want alpha.go", entries[0].Name)
	}
}

func TestSortDirEntriesEmpty(t *testing.T) {
	var entries []dirEntry
	sortDirEntries(entries) // should not panic
}

func TestSortDirEntriesSingle(t *testing.T) {
	entries := []dirEntry{{Name: "solo.go", IsDir: false}}
	sortDirEntries(entries) // should not panic
	if entries[0].Name != "solo.go" {
		t.Error("single element should remain unchanged")
	}
}

// --- dirEntryLess tests ---

func TestDirEntryLess(t *testing.T) {
	dir := dirEntry{Name: "src", IsDir: true}
	file := dirEntry{Name: "main.go", IsDir: false}

	if !dirEntryLess(dir, file) {
		t.Error("directory should sort before file")
	}
	if dirEntryLess(file, dir) {
		t.Error("file should not sort before directory")
	}

	a := dirEntry{Name: "alpha.go", IsDir: false}
	z := dirEntry{Name: "zebra.go", IsDir: false}
	if !dirEntryLess(a, z) {
		t.Error("alpha should sort before zebra")
	}
	if dirEntryLess(z, a) {
		t.Error("zebra should not sort before alpha")
	}
}

// --- gitStatusLabel tests ---

func TestGitStatusLabel(t *testing.T) {
	tests := []struct {
		name      string
		status    gitpkg.FileStatus
		wantLabel string
		wantClass string
	}{
		{
			name:      "modified worktree",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: gitpkg.StatusCode('M')},
			wantLabel: "M",
			wantClass: "modified",
		},
		{
			name:      "added staging",
			status:    gitpkg.FileStatus{Path: "f", Staging: gitpkg.StatusCode('A'), Worktree: ' '},
			wantLabel: "A",
			wantClass: "added",
		},
		{
			name:      "deleted worktree",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: gitpkg.StatusCode('D')},
			wantLabel: "D",
			wantClass: "deleted",
		},
		{
			name:      "renamed",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: gitpkg.StatusCode('R')},
			wantLabel: "R",
			wantClass: "modified",
		},
		{
			name:      "copied",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: gitpkg.StatusCode('C')},
			wantLabel: "C",
			wantClass: "added",
		},
		{
			name:      "untracked",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: gitpkg.StatusCode('?')},
			wantLabel: "?",
			wantClass: "untracked",
		},
		{
			name:      "unmodified",
			status:    gitpkg.FileStatus{Path: "f", Staging: ' ', Worktree: ' '},
			wantLabel: "",
			wantClass: "",
		},
		{
			name:      "worktree takes precedence over staging when not space",
			status:    gitpkg.FileStatus{Path: "f", Staging: gitpkg.StatusCode('A'), Worktree: gitpkg.StatusCode('M')},
			wantLabel: "M",
			wantClass: "modified",
		},
		{
			name:      "staging used when worktree is space",
			status:    gitpkg.FileStatus{Path: "f", Staging: gitpkg.StatusCode('D'), Worktree: ' '},
			wantLabel: "D",
			wantClass: "deleted",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, class := gitStatusLabel(tt.status)
			if label != tt.wantLabel {
				t.Errorf("label = %q, want %q", label, tt.wantLabel)
			}
			if class != tt.wantClass {
				t.Errorf("class = %q, want %q", class, tt.wantClass)
			}
		})
	}
}

// --- buildPageData tests ---

func TestBuildPageDataRoot(t *testing.T) {
	s, dir := setupTestServer(t)
	defer s.sse.Stop()

	pd := s.buildPageData(".")
	expected := filepath.Base(dir)
	if pd.Title != expected {
		t.Errorf("title = %q, want %q", pd.Title, expected)
	}
	if len(pd.Breadcrumbs) != 0 {
		t.Errorf("root should have no breadcrumbs, got %d", len(pd.Breadcrumbs))
	}
	if pd.GitBranch != "" {
		t.Error("git branch should be empty when git is disabled")
	}
}

func TestBuildPageDataSubpath(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	pd := s.buildPageData("docs/guide.md")
	if pd.Title != "docs/guide.md" {
		t.Errorf("title = %q, want %q", pd.Title, "docs/guide.md")
	}
	if len(pd.Breadcrumbs) != 2 {
		t.Fatalf("breadcrumbs count = %d, want 2", len(pd.Breadcrumbs))
	}
	if pd.Breadcrumbs[0].Name != "docs" {
		t.Errorf("breadcrumb[0].Name = %q, want 'docs'", pd.Breadcrumbs[0].Name)
	}
	if pd.Breadcrumbs[0].Href != "/docs" {
		t.Errorf("breadcrumb[0].Href = %q, want '/docs'", pd.Breadcrumbs[0].Href)
	}
	if pd.Breadcrumbs[1].Name != "guide.md" {
		t.Errorf("breadcrumb[1].Name = %q, want 'guide.md'", pd.Breadcrumbs[1].Name)
	}
	if pd.Breadcrumbs[1].Href != "/docs/guide.md" {
		t.Errorf("breadcrumb[1].Href = %q, want '/docs/guide.md'", pd.Breadcrumbs[1].Href)
	}
}

func TestBuildPageDataChromaCSS(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	pd := s.buildPageData(".")
	if pd.ExtraCSS == "" {
		t.Error("ExtraCSS should contain Chroma CSS")
	}
}

func TestBuildPageDataCustomCSS(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	s.cfg.Server.CustomCSS = "custom.css"
	pd := s.buildPageData(".")
	if pd.CustomCSSPath != "/__custom.css" {
		t.Errorf("CustomCSSPath = %q, want /__custom.css", pd.CustomCSSPath)
	}

	s.cfg.Server.CustomCSS = ""
	pd = s.buildPageData(".")
	if pd.CustomCSSPath != "" {
		t.Errorf("CustomCSSPath should be empty when no custom CSS, got %q", pd.CustomCSSPath)
	}
}

// --- OnFileChange test ---

func TestOnFileChange(t *testing.T) {
	s, _ := setupTestServer(t)

	ch := make(chan string, 8)
	s.sse.register <- ch

	s.OnFileChange("test.md")

	// Verify we received the notification
	select {
	case msg := <-ch:
		if msg != "test.md" {
			t.Errorf("expected 'test.md', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for SSE notification")
	}

	s.sse.Stop()
}

// --- Integration: full request through mux ---

func TestFullMuxRouting(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/", 200},
		{"/README.md", 200},
		{"/main.go", 200},
		{"/docs", 200},
		{"/__api/files", 200},
		{"/__api/graph", 200},
	}

	handler := s.middleware(s.mux)
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != tt.wantStatus {
			t.Errorf("GET %s: status = %d, want %d", tt.path, rec.Code, tt.wantStatus)
		}
	}
}

// --- Additional test for empty directory ---

func TestHandleDirectoryEmpty(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	os.MkdirAll(emptyDir, 0o755)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false

	idx := index.New(dir)
	idx.Build()

	links := index.NewLinkGraph()
	s := New(cfg, idx, links, nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/empty", nil)
	rec := httptest.NewRecorder()
	s.handleDirectory(rec, req, "empty")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// --- handleAPIDocument tests ---

func TestHandleAPIDocumentReturnsJSON(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=README.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var result struct {
		File       string `json:"file"`
		Size       int64  `json:"size"`
		IsMarkdown bool   `json:"isMarkdown"`
		Headings   []struct {
			Level int    `json:"level"`
			Text  string `json:"text"`
			Slug  string `json:"slug"`
			Line  int    `json:"line"`
		} `json:"headings"`
		ForwardLinks []struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Text   string `json:"text"`
		} `json:"forwardLinks"`
		Backlinks []json.RawMessage `json:"backlinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result.File != "README.md" {
		t.Errorf("file = %q, want README.md", result.File)
	}
	if !result.IsMarkdown {
		t.Error("isMarkdown should be true for README.md")
	}
	if result.Size == 0 {
		t.Error("size should be > 0")
	}
}

func TestHandleAPIDocumentHeadings(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=README.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	var result struct {
		Headings []struct {
			Level int    `json:"level"`
			Text  string `json:"text"`
			Slug  string `json:"slug"`
		} `json:"headings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.Headings) == 0 {
		t.Fatal("expected at least one heading")
	}
	if result.Headings[0].Text != "Hello" {
		t.Errorf("heading text = %q, want Hello", result.Headings[0].Text)
	}
	if result.Headings[0].Slug != "hello" {
		t.Errorf("heading slug = %q, want hello", result.Headings[0].Slug)
	}
	if result.Headings[0].Level != 1 {
		t.Errorf("heading level = %d, want 1", result.Headings[0].Level)
	}
}

func TestHandleAPIDocumentForwardLinks(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=README.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	var result struct {
		ForwardLinks []struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Text   string `json:"text"`
		} `json:"forwardLinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.ForwardLinks) == 0 {
		t.Fatal("expected at least one forward link")
	}
	if result.ForwardLinks[0].Target != "docs/guide.md" {
		t.Errorf("forward link target = %q, want docs/guide.md", result.ForwardLinks[0].Target)
	}
}

func TestHandleAPIDocumentMissingFileParam(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIDocumentEmptyFileParam(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAPIDocumentPathTraversal(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	traversalPaths := []string{
		"../etc/passwd",
		"../../etc/shadow",
		"docs/../../etc/passwd",
		"..%2f..%2fetc/passwd",
		"README.md/../../../etc/passwd",
	}

	for _, path := range traversalPaths {
		req := httptest.NewRequest("GET", "/__api/document?file="+path, nil)
		rec := httptest.NewRecorder()
		s.handleAPIDocument(rec, req)

		if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
			t.Errorf("path %q: status = %d, want 403 or 404", path, rec.Code)
		}
	}
}

func TestHandleAPIDocumentNotFound(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=nonexistent.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleAPIDocumentNonMarkdownFile(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=main.go", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var result struct {
		File       string `json:"file"`
		IsMarkdown bool   `json:"isMarkdown"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.IsMarkdown {
		t.Error("isMarkdown should be false for .go file")
	}
}

func TestHandleAPIDocumentNilLinkGraph(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	s.links = nil

	req := httptest.NewRequest("GET", "/__api/document?file=README.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var result struct {
		ForwardLinks []json.RawMessage `json:"forwardLinks"`
		Backlinks    []json.RawMessage `json:"backlinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.ForwardLinks != nil {
		t.Error("forwardLinks should be null when link graph is nil")
	}
	if result.Backlinks != nil {
		t.Error("backlinks should be null when link graph is nil")
	}
}

func TestHandleAPIDocumentDuplicateHeadingSlugs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "dupes.md"), []byte("# Same\n\nText.\n\n# Same\n\nMore text.\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/document?file=dupes.md", nil)
	rec := httptest.NewRecorder()
	s.handleAPIDocument(rec, req)

	var result struct {
		Headings []struct {
			Slug string `json:"slug"`
		} `json:"headings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.Headings) < 2 {
		t.Fatal("expected at least 2 headings")
	}
	if result.Headings[0].Slug == result.Headings[1].Slug {
		t.Errorf("duplicate headings should get unique slugs, both got %q", result.Headings[0].Slug)
	}
	if result.Headings[1].Slug != "same-1" {
		t.Errorf("second slug = %q, want same-1", result.Headings[1].Slug)
	}
}

// --- handleAPITasks tests ---

func TestHandleAPITasksReturnsJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "todo.md"), []byte("# Tasks\n\n- [ ] Buy milk\n- [x] Done item\n- [ ] Write tests\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/tasks", nil)
	rec := httptest.NewRecorder()
	s.handleAPITasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var result []struct {
		File    string `json:"File"`
		Line    int    `json:"Line"`
		Text    string `json:"Text"`
		Checked bool   `json:"Checked"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result))
	}
}

func TestHandleAPITasksPendingFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "todo.md"), []byte("# Tasks\n\n- [ ] Pending one\n- [x] Done one\n- [ ] Pending two\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/tasks?pending=true", nil)
	rec := httptest.NewRecorder()
	s.handleAPITasks(rec, req)

	var result []struct {
		Text    string `json:"Text"`
		Checked bool   `json:"Checked"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d", len(result))
	}
	for _, task := range result {
		if task.Checked {
			t.Errorf("pending filter returned checked task: %q", task.Text)
		}
	}
}

func TestHandleAPITasksNoTasks(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/tasks", nil)
	rec := httptest.NewRecorder()
	s.handleAPITasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandleAPITasksWithPriority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("# Work\n\n- [ ] !high Fix production bug\n- [ ] !low Update docs\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/tasks", nil)
	rec := httptest.NewRecorder()
	s.handleAPITasks(rec, req)

	var result []struct {
		Text     string `json:"Text"`
		Priority string `json:"Priority"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result))
	}

	priorities := map[string]bool{}
	for _, r := range result {
		priorities[r.Priority] = true
	}
	if !priorities["high"] {
		t.Error("expected a high priority task")
	}
	if !priorities["low"] {
		t.Error("expected a low priority task")
	}
}

func TestHandleAPITasksMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("- [ ] Task A\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte("- [ ] Task B\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/tasks", nil)
	rec := httptest.NewRecorder()
	s.handleAPITasks(rec, req)

	var result []struct {
		File string `json:"File"`
		Text string `json:"Text"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 tasks from 2 files, got %d", len(result))
	}

	files := map[string]bool{}
	for _, r := range result {
		files[r.File] = true
	}
	if !files["a.md"] {
		t.Error("expected task from a.md")
	}
	if !files["sub/b.md"] {
		t.Error("expected task from sub/b.md")
	}
}

// --- handleAPIFiles additional tests ---

func TestHandleAPIFilesNoMatchFuzzy(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/files?q=zzzznonexistent", nil)
	rec := httptest.NewRecorder()
	s.handleAPIFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var entries []index.FileEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 results for nonsense query, got %d", len(entries))
	}
}

// --- handleAPIGraph additional tests ---

func TestHandleAPIGraphContentType(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	req := httptest.NewRequest("GET", "/__api/graph", nil)
	rec := httptest.NewRecorder()
	s.handleAPIGraph(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestHandleAPIGraphNilLinkGraph(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	s.links = nil

	req := httptest.NewRequest("GET", "/__api/graph", nil)
	rec := httptest.NewRecorder()
	s.handleAPIGraph(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var data struct {
		Nodes []interface{} `json:"nodes"`
		Links []interface{} `json:"links"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
}

// --- Full mux API routing ---

func TestFullMuxAPIRouting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes\n\n- [ ] Todo item\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	idx.Build()
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	handler := s.middleware(s.mux)

	tests := []struct {
		path       string
		wantStatus int
		wantCT     string
	}{
		{"/__api/files", 200, "application/json"},
		{"/__api/search?q=", 200, "application/json"},
		{"/__api/graph", 200, "application/json"},
		{"/__api/document?file=notes.md", 200, "application/json"},
		{"/__api/tasks", 200, "application/json"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != tt.wantStatus {
			t.Errorf("GET %s: status = %d, want %d", tt.path, rec.Code, tt.wantStatus)
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, tt.wantCT) {
			t.Errorf("GET %s: Content-Type = %q, want %s", tt.path, ct, tt.wantCT)
		}
	}
}

// newCancelContext creates a cancellable context for testing blocking handlers.
func newCancelContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithCancel(r.Context())
}
