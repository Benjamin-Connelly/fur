package web

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

// setupTestServer creates a Server backed by a temp directory with test files.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n\nWorld.\n\n```mermaid\ngraph TD\n  A-->B\n```\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("# Guide\n\n[Back](../README.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false

	idx := index.New(dir)
	idx.Build()

	links := index.NewLinkGraph()
	links.SetLinks("README.md", []index.Link{
		{Source: "README.md", Target: "docs/guide.md", Text: "Guide"},
	})

	s := New(cfg, idx, links, nil)
	return s, dir
}

// --- SSEBroker tests ---

func TestNewSSEBroker(t *testing.T) {
	b := NewSSEBroker()
	if b == nil {
		t.Fatal("NewSSEBroker returned nil")
	}
	if b.clients == nil {
		t.Error("clients map not initialized")
	}
	b.Stop()
}

func TestSSEBrokerRegisterUnregister(t *testing.T) {
	b := NewSSEBroker()
	defer b.Stop()

	ch := make(chan string, 8)
	b.register <- ch

	// Give the goroutine time to process
	time.Sleep(10 * time.Millisecond)

	b.unregister <- ch
	time.Sleep(10 * time.Millisecond)

	// Channel should be closed after unregister
	_, open := <-ch
	if open {
		t.Error("channel should be closed after unregister")
	}
}

func TestSSEBrokerBroadcast(t *testing.T) {
	b := NewSSEBroker()
	defer b.Stop()

	ch := make(chan string, 8)
	b.register <- ch
	time.Sleep(10 * time.Millisecond)

	b.broadcast <- "reload"
	select {
	case msg := <-ch:
		if msg != "reload" {
			t.Errorf("got %q, want %q", msg, "reload")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestSSEBrokerNotify(t *testing.T) {
	b := NewSSEBroker()
	defer b.Stop()

	ch := make(chan string, 8)
	b.register <- ch
	time.Sleep(10 * time.Millisecond)

	b.Notify("test.md")

	select {
	case msg := <-ch:
		if msg != "test.md" {
			t.Errorf("got %q, want %q", msg, "test.md")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestSSEBrokerNotifyAfterStop(t *testing.T) {
	b := NewSSEBroker()
	b.Stop()
	time.Sleep(10 * time.Millisecond)

	// Should not panic or block indefinitely
	done := make(chan struct{})
	go func() {
		b.Notify("should-not-block")
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(time.Second):
		t.Fatal("Notify blocked after Stop")
	}
}

func TestSSEBrokerStopClosesClients(t *testing.T) {
	b := NewSSEBroker()

	ch := make(chan string, 8)
	b.register <- ch
	time.Sleep(10 * time.Millisecond)

	b.Stop()
	time.Sleep(10 * time.Millisecond)

	_, open := <-ch
	if open {
		t.Error("client channel should be closed after Stop")
	}
}

// --- middleware tests ---

func TestMiddlewareSetsSecurityHeaders(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := s.middleware(inner)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}
	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header missing")
	}

	pp := rec.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("Permissions-Policy header missing")
	}
}

// --- etagMiddleware tests ---

func TestEtagMiddlewareGeneratesETag(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("hello"))
	})
	handler := etagMiddleware(inner)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag header not set")
	}

	wantEtag := fmt.Sprintf(`"%x"`, md5.Sum([]byte("hello")))
	if etag != wantEtag {
		t.Errorf("ETag = %q, want %q", etag, wantEtag)
	}
}

func TestEtagMiddlewareReturns304OnMatch(t *testing.T) {
	body := []byte("hello etag")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(body)
	})
	handler := etagMiddleware(inner)

	etag := fmt.Sprintf(`"%x"`, md5.Sum(body))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotModified)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("body should be empty on 304, got %d bytes", rec.Body.Len())
	}
}

func TestEtagMiddlewarePassthroughNon200(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})
	handler := etagMiddleware(inner)

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	etag := rec.Header().Get("ETag")
	if etag != "" {
		t.Errorf("ETag should not be set for non-200, got %q", etag)
	}
}

// --- responseRecorder tests ---

func TestResponseRecorderCapturesStatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	rr := &responseRecorder{ResponseWriter: w, statusCode: 200}

	rr.WriteHeader(http.StatusCreated)
	if rr.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want %d", rr.statusCode, http.StatusCreated)
	}
	if !rr.captured {
		t.Error("captured should be true after WriteHeader")
	}

	rr.Header().Set("Content-Type", "text/plain")
	n, err := rr.Write([]byte("test body"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 9 {
		t.Errorf("Write returned %d, want 9", n)
	}
	if string(rr.body) != "test body" {
		t.Errorf("body = %q, want %q", rr.body, "test body")
	}
	if rr.contentType != "text/plain" {
		t.Errorf("contentType = %q, want %q", rr.contentType, "text/plain")
	}
}

func TestResponseRecorderMultipleWrites(t *testing.T) {
	w := httptest.NewRecorder()
	rr := &responseRecorder{ResponseWriter: w, statusCode: 200}

	rr.Header().Set("Content-Type", "text/plain")
	rr.Write([]byte("hello "))
	rr.Write([]byte("world"))

	if string(rr.body) != "hello world" {
		t.Errorf("body = %q, want %q", rr.body, "hello world")
	}
}

// --- formatSize tests ---

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

// --- handleCustomCSS tests ---

func TestHandleCustomCSSServesFile(t *testing.T) {
	s, dir := setupTestServer(t)
	defer s.sse.Stop()

	cssContent := "body { color: red; }"
	cssPath := filepath.Join(dir, "custom.css")
	os.WriteFile(cssPath, []byte(cssContent), 0o644)
	s.cfg.Server.CustomCSS = cssPath

	req := httptest.NewRequest("GET", "/__custom.css", nil)
	rec := httptest.NewRecorder()
	s.handleCustomCSS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != cssContent {
		t.Errorf("body = %q, want %q", rec.Body.String(), cssContent)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
}

func TestHandleCustomCSSReturns404ForEmptyPath(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	s.cfg.Server.CustomCSS = ""

	req := httptest.NewRequest("GET", "/__custom.css", nil)
	rec := httptest.NewRecorder()
	s.handleCustomCSS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleCustomCSSReturns404ForMissingFile(t *testing.T) {
	s, dir := setupTestServer(t)
	defer s.sse.Stop()

	s.cfg.Server.CustomCSS = filepath.Join(dir, "nonexistent.css")

	req := httptest.NewRequest("GET", "/__custom.css", nil)
	rec := httptest.NewRecorder()
	s.handleCustomCSS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleCustomCSSRelativePath(t *testing.T) {
	s, dir := setupTestServer(t)
	defer s.sse.Stop()

	cssContent := ".relative { color: blue; }"
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(cssContent), 0o644)
	s.cfg.Server.CustomCSS = "style.css"

	req := httptest.NewRequest("GET", "/__custom.css", nil)
	rec := httptest.NewRecorder()
	s.handleCustomCSS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != cssContent {
		t.Errorf("body = %q, want %q", rec.Body.String(), cssContent)
	}
}
