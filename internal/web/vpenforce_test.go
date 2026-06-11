package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

// pathAcceptingRoutes lists every internal/web route whose handler resolves
// request-supplied bytes into a filesystem path. Adding a new such route
// requires (a) appending here AND (b) ensuring the handler delegates to
// Index.ValidatePath — TestEveryWebHandlerDelegatesToValidatePath fails
// otherwise.
//
// Routes intentionally absent:
//   - /__static/: served from an embed.FS; the request path is not
//     resolved against the host filesystem.
//   - /__custom.css: the CSS path comes from server configuration, not the
//     request. The associated trust concern (per-project config overrides,
//     FUR_SERVER_CUSTOM_CSS env pivot) is Chain A/L territory, not
//     request-driven traversal.
//   - /__api/files, /__api/search, /__api/graph, /__api/tasks, /__events,
//     /graph: no path-shaped input arrives from the request.
var pathAcceptingRoutes = []struct {
	name     string
	buildURL func(payload string) string
}{
	{"GET / (URL path)", func(p string) string { return "/" + p }},
	{"GET /__api/document?file=", func(p string) string { return "/__api/document?file=" + p }},
}

// TestEveryWebHandlerDelegatesToValidatePath is the runtime invariant guard
// for the "every web handler accepting path-shaped input MUST delegate to
// Index.ValidatePath" rule.
//
// For each path-accepting route × adversarial-payload combination, the
// response must (a) carry a canonical rejection code (403 or 404) and (b)
// NOT include bytes from outside the serve root. The symlink-escape case
// is the load-bearing one — a markdown-named symlink inside the root whose
// target lives outside is exactly what inline strings.Contains("..")
// checks fail to detect.
//
// References: lookit-9py.4.9 (HARDEN_VPENFORCE); SECURITY-INVENTORY.md §15;
// bd memory "every-web-handler-accepting-a-path-shaped-input".
func TestEveryWebHandlerDelegatesToValidatePath(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	const canary = "SECRET-VPENFORCE-CANARY-9YP-4-9"
	secretPath := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secretPath, []byte("# "+canary+"\nleaked\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	// Benign in-root file so Index.Build has something to walk besides
	// the symlink. Mirrors a realistic served repo.
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# ok\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	if err := os.Symlink(secretPath, filepath.Join(root, "escape.md")); err != nil {
		t.Skipf("os.Symlink unsupported: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	idx := index.New(root)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := New(cfg, idx, index.NewLinkGraph(), nil)
	defer s.sse.Stop()

	// httptest.NewServer + real client so that the standard http.ServeMux's
	// automatic path-cleaning redirects (e.g., "/docs/../../etc/passwd" -> 307
	// Location: /etc/passwd) are followed end-to-end. The final response is
	// what we assert against; a 307 to a safe destination is fine.
	ts := httptest.NewServer(s.mux)
	defer ts.Close()
	client := ts.Client()

	payloads := []struct {
		name    string
		payload string
	}{
		{"traversal dot-dot", "../etc/passwd"},
		{"traversal nested", "docs/../../etc/passwd"},
		{"traversal long", "README.md/../../../etc/passwd"},
		{"traversal url-encoded", "..%2f..%2fetc/passwd"},
		{"symlink escape", "escape.md"},
	}

	for _, route := range pathAcceptingRoutes {
		for _, p := range payloads {
			t.Run(route.name+"/"+p.name, func(t *testing.T) {
				target := route.buildURL(p.payload)
				resp, err := client.Get(ts.URL + target)
				if err != nil {
					t.Fatalf("GET %s: %v", target, err)
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
					t.Errorf("target=%q: final status=%d, want 403 or 404; handler must delegate to Index.ValidatePath",
						target, resp.StatusCode)
				}
				if strings.Contains(string(body), canary) {
					t.Errorf("target=%q: response leaked content from outside the serve root (canary %q present)",
						target, canary)
				}
			})
		}
	}
}
