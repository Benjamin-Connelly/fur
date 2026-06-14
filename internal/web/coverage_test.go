package web

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

func mustWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHandler confirms Handler() returns a working middleware-wrapped mux.
func TestHandler(t *testing.T) {
	s, _ := setupTestServer(t)
	h := s.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
	req := httptest.NewRequest("GET", "/README.md", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /README.md via Handler() = %d, want 200", rec.Code)
	}
}

// TestHandleAPISearch exercises the search endpoint: empty query, over-long
// query, and a real query (grep fallback path since no Bleve index is enabled).
func TestHandleAPISearch(t *testing.T) {
	s, _ := setupTestServer(t)
	h := s.Handler()

	do := func(q string) (int, string) {
		req := httptest.NewRequest("GET", "/__api/search?q="+q, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	if code, _ := do(""); code != http.StatusOK {
		t.Errorf("empty query = %d, want 200", code)
	}
	if code, _ := do(strings.Repeat("x", 250)); code != http.StatusOK {
		t.Errorf("over-long query = %d, want 200", code)
	}

	code, body := do("World")
	if code != http.StatusOK {
		t.Fatalf("search = %d, want 200", code)
	}
	// Body must be valid JSON (array of results, possibly empty).
	var results []searchResult
	if err := json.Unmarshal([]byte(body), &results); err != nil {
		t.Errorf("search response not valid JSON: %v\n%s", err, body)
	}
}

// TestHandleAPISearchBleve covers the fulltext (Bleve) branch of the search
// handler, which the grep-fallback tests skip.
func TestHandleAPISearchBleve(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "README.md", "# Hello\n\nThe quick brown fox jumps.\n")
	mustWrite(t, dir, "notes.md", "# Notes\n\nlazy dog content here\n")

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	if err := idx.BuildFulltext(t.TempDir()); err != nil {
		t.Fatalf("BuildFulltext: %v", err)
	}
	t.Cleanup(idx.CloseFulltext)

	s := New(cfg, idx, index.NewLinkGraph(), nil)
	t.Cleanup(func() { s.sse.Stop() })

	req := httptest.NewRequest("GET", "/__api/search?q=fox", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bleve search = %d, want 200", rec.Code)
	}
	var results []searchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("bleve search response not JSON: %v", err)
	}
}

// TestStartValidateBindError covers Start()'s early ValidateBind failure when a
// non-loopback host is configured without --listen-public.
func TestStartValidateBindError(t *testing.T) {
	s, _ := setupTestServer(t)
	s.cfg.Server.Host = "192.0.2.1" // TEST-NET-1, non-loopback
	s.cfg.Server.ListenPublic = false
	if err := s.Start(); err == nil {
		t.Error("Start should refuse a non-loopback bind without --listen-public")
	}
}

// TestStartStop runs Start on an ephemeral loopback port in a goroutine, waits
// until it serves, then Stop()s it — covering the listen + graceful-shutdown
// path of both methods.
func TestStartStop(t *testing.T) {
	s, _ := setupTestServer(t)

	// Grab a free port, then release it for Start to bind.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	s.cfg.Server.Host = "127.0.0.1"
	s.cfg.Server.Port = port

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start() }()

	// Poll until the server answers, then shut down.
	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
	deadline := time.Now().Add(3 * time.Second)
	var up bool
	for time.Now().Before(deadline) {
		resp, err := http.Get(base)
		if err == nil {
			resp.Body.Close()
			up = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !up {
		_ = s.Stop()
		<-errCh
		t.Fatal("server never came up")
	}

	if err := s.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	// Start returns once ListenAndServe unblocks via Shutdown.
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Error("Start did not return after Stop")
	}
}
