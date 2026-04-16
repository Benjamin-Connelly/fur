package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n\nWorld\n\n- [ ] Fix bug\n- [x] Done\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "guide.md"), []byte("# Guide\n\nSee [README](README.md) and [missing](nope.md).\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "binary.bin"), []byte{0x00, 0x01, 0x02}, 0o644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub", "nested.md"), []byte("# Nested\n\n[[README]]\n"), 0o644)

	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("index build: %v", err)
	}

	links := index.NewLinkGraph()
	links.BuildFromIndex(idx)

	return New(idx, links)
}

var ctx = context.Background()
var emptyReq = mcplib.CallToolRequest{}

// --- handleSearchDocs ---

func TestSearchDocs_FuzzyFilename(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleSearchDocs(ctx, emptyReq, searchDocsArgs{Query: "readme"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "README.md") {
		t.Errorf("expected README.md in results, got: %s", text)
	}
}

func TestSearchDocs_EmptyQuery(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleSearchDocs(ctx, emptyReq, searchDocsArgs{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for empty query")
	}
}

func TestSearchDocs_QueryTooLong(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleSearchDocs(ctx, emptyReq, searchDocsArgs{Query: strings.Repeat("x", 501)})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for query > 500 chars")
	}
}

func TestSearchDocs_NoResults(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleSearchDocs(ctx, emptyReq, searchDocsArgs{Query: "zzzznonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "No results") {
		t.Errorf("expected 'No results' message, got: %s", text)
	}
}

// --- handleGetDocument ---

func TestGetDocument_ReadsFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "# Hello") {
		t.Errorf("expected markdown content, got: %s", text)
	}
}

func TestGetDocument_EmptyFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for empty file")
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: "nonexistent.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing file")
	}
}

func TestGetDocument_PathTraversal(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: "../../../etc/passwd"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
}

func TestGetDocument_BinaryFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: "binary.bin"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for binary file")
	}
}

func TestGetDocument_SizeGuard(t *testing.T) {
	s := setupTestServer(t)
	// Artificially set the entry size to >10MB
	entry := s.idx.Lookup("README.md")
	if entry == nil {
		t.Fatal("README.md not in index")
	}
	origSize := entry.Size
	entry.Size = 11 * 1024 * 1024
	defer func() { entry.Size = origSize }()

	result, err := s.handleGetDocument(ctx, emptyReq, getDocumentArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for oversized file")
	}
	text := textContent(result)
	if !strings.Contains(text, "too large") {
		t.Errorf("expected 'too large' error, got: %s", text)
	}
}

// --- handleGetRelatedDocs ---

func TestGetRelatedDocs_ForwardLinks(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: "guide.md", Direction: "forward"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "README.md") {
		t.Errorf("expected forward link to README.md, got: %s", text)
	}
}

func TestGetRelatedDocs_Backlinks(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: "README.md", Direction: "back"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "guide.md") {
		t.Errorf("expected backlink from guide.md, got: %s", text)
	}
}

func TestGetRelatedDocs_BothDirections(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: "guide.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "Forward links") {
		t.Errorf("expected forward links section, got: %s", text)
	}
}

func TestGetRelatedDocs_NotFound(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: "ghost.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing file")
	}
}

func TestGetRelatedDocs_EmptyFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for empty file")
	}
}

func TestGetRelatedDocs_NilLinkGraph(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("# Test\n"), 0o644)
	idx := index.New(dir)
	idx.Build()
	s := New(idx, nil) // nil link graph

	result, err := s.handleGetRelatedDocs(ctx, emptyReq, getRelatedDocsArgs{File: "test.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error when link graph is nil")
	}
}

// --- handleCheckDocHealth ---

func TestCheckDocHealth_FindsBrokenLinks(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleCheckDocHealth(ctx, emptyReq, checkDocHealthArgs{})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "nope.md") {
		t.Errorf("expected broken link to nope.md, got: %s", text)
	}
}

func TestCheckDocHealth_FilterByFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleCheckDocHealth(ctx, emptyReq, checkDocHealthArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	// README.md has a pending task, but no broken links
	if strings.Contains(text, "nope.md") {
		t.Error("should not show broken links from guide.md when filtering to README.md")
	}
}

func TestCheckDocHealth_FindsPendingTasks(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleCheckDocHealth(ctx, emptyReq, checkDocHealthArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "Fix bug") {
		t.Errorf("expected pending task 'Fix bug', got: %s", text)
	}
}

// --- handleGetDocStructure ---

func TestGetDocStructure_ExtractsHeadings(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocStructure(ctx, emptyReq, getDocStructureArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "Hello") {
		t.Errorf("expected heading 'Hello', got: %s", text)
	}
	if !strings.Contains(text, "#hello") {
		t.Errorf("expected slug '#hello', got: %s", text)
	}
}

func TestGetDocStructure_EmptyFile(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocStructure(ctx, emptyReq, getDocStructureArgs{File: ""})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for empty file")
	}
}

func TestGetDocStructure_NotFound(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocStructure(ctx, emptyReq, getDocStructureArgs{File: "ghost.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing file")
	}
}

func TestGetDocStructure_NonMarkdown(t *testing.T) {
	s := setupTestServer(t)
	result, err := s.handleGetDocStructure(ctx, emptyReq, getDocStructureArgs{File: "code.go"})
	if err != nil {
		t.Fatal(err)
	}
	text := textContent(result)
	if !strings.Contains(text, "No headings") {
		t.Errorf("expected 'No headings' for Go file, got: %s", text)
	}
}

func TestGetDocStructure_SizeGuard(t *testing.T) {
	s := setupTestServer(t)
	entry := s.idx.Lookup("README.md")
	origSize := entry.Size
	entry.Size = 11 * 1024 * 1024
	defer func() { entry.Size = origSize }()

	result, err := s.handleGetDocStructure(ctx, emptyReq, getDocStructureArgs{File: "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for oversized file")
	}
}

// --- validatePath ---

func TestValidatePath_Normal(t *testing.T) {
	s := setupTestServer(t)
	absPath, err := s.validatePath("README.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(absPath, "README.md") {
		t.Errorf("expected path ending in README.md, got: %s", absPath)
	}
}

func TestValidatePath_TraversalBlocked(t *testing.T) {
	s := setupTestServer(t)
	_, err := s.validatePath("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestValidatePath_NonexistentFile(t *testing.T) {
	s := setupTestServer(t)
	_, err := s.validatePath("nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// --- New ---

func TestNew_RegistersAllTools(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("# Test\n"), 0o644)
	idx := index.New(dir)
	idx.Build()
	s := New(idx, nil)
	if s.mcp == nil {
		t.Error("MCP server not initialized")
	}
}

// --- helpers ---

// textContent extracts the text from a CallToolResult.
func textContent(result *mcplib.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	tc, ok := result.Content[0].(mcplib.TextContent)
	if !ok {
		return ""
	}
	return tc.Text
}

