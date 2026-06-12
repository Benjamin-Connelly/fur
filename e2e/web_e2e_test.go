// Package e2e holds browser-level end-to-end tests for fur's web mode. It is a
// separate Go module (own go.mod) so its heavy, Go-1.26-requiring browser
// dependency (chromedp + a modern cdproto) never bumps the main module's Go
// version or pollutes its dependency graph. Run with: cd e2e && go test ./...
// Tests skip when no Chrome/Chromium is installed.
package e2e

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/web"
)

func findBrowser() string {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome", "headless-shell"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// newE2EServer wraps the real fur web handler in an httptest.Server over a
// small tree and returns the base URL.
func newE2EServer(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# E2ETitleMarker\n\nbody paragraph\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := web.New(cfg, idx, index.NewLinkGraph(), nil)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

// browserContext builds a headless, no-sandbox chromedp context and a cleanup
// that cannot hang the test: chromedp.Cancel runs with a hard timeout so a
// wedged browser fails fast instead of blocking until the 10-minute test
// deadline.
func browserContext(t *testing.T, browser string) (context.Context, func()) {
	t.Helper()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browser),
		chromedp.Headless,
		chromedp.NoSandbox,
		chromedp.DisableGPU,
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	cleanup := func() {
		done := make(chan struct{})
		go func() { _ = chromedp.Cancel(ctx); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Log("chromedp.Cancel did not return within 10s; forcing allocator shutdown")
		}
		cancelCtx()
		cancelAlloc()
	}
	return ctx, cleanup
}

// TestE2E_MarkdownRenders loads a served markdown page in a real headless
// browser and asserts the server-rendered heading reaches the DOM. Hermetic:
// asserts only server-rendered content, so the page's CDN script fetches are
// irrelevant.
func TestE2E_MarkdownRenders(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Chromium found; skipping browser E2E")
	}
	url := newE2EServer(t)

	ctx, cleanup := browserContext(t, browser)
	defer cleanup()
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var bodyText string
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(url+"/README.md"),
		chromedp.WaitVisible("h1", chromedp.ByQuery),
		chromedp.Text("body", &bodyText, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("chromedp Run: %v", err)
	}
	if !strings.Contains(bodyText, "E2ETitleMarker") {
		t.Errorf("rendered page missing server-rendered heading; body=%q", bodyText)
	}
}

// TestE2E_SearchOverlay drives the self-hosted app.js (served from /__static,
// no CDN) to confirm the strict no-unsafe-inline CSP does not break fur's own
// scripts: the search toggle reveals the overlay.
func TestE2E_SearchOverlay(t *testing.T) {
	browser := findBrowser()
	if browser == "" {
		t.Skip("no Chrome/Chromium found; skipping browser E2E")
	}
	url := newE2EServer(t)

	ctx, cleanup := browserContext(t, browser)
	defer cleanup()
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var overlayHidden bool
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(url+"/README.md"),
		chromedp.WaitVisible("#search-toggle", chromedp.ByQuery),
		chromedp.Click("#search-toggle", chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(`document.getElementById('search-overlay').hasAttribute('hidden')`, &overlayHidden),
	); err != nil {
		t.Fatalf("chromedp Run: %v", err)
	}
	if overlayHidden {
		t.Error("search overlay still hidden after toggle; self-hosted app.js did not run under the strict CSP")
	}
}
