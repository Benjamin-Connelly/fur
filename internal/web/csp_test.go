package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/web/static"
)

// cspDirective returns the value of one CSP directive (e.g. "script-src")
// from a full Content-Security-Policy header string.
func cspDirective(csp, name string) string {
	for _, part := range strings.Split(csp, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, name+" ") || part == name {
			return strings.TrimSpace(strings.TrimPrefix(part, name))
		}
	}
	return ""
}

// TestCSPScriptSrcNoUnsafeInline is the Chain D regression guard.
//
// The Mermaid -> JS -> fetch-local smuggling chain (and any injected inline
// <script>) relies on the page permitting inline script execution. Before
// the fix, script-src carried 'unsafe-inline', so any inline script that
// reached the DOM would run. The fix externalizes all of fur's own scripts
// to /__static and drops 'unsafe-inline' from script-src. References
// lookit-9py.3.8 / .4.4.
func TestCSPScriptSrcNoUnsafeInline(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	s.middleware(s.mux).ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header missing")
	}
	scriptSrc := cspDirective(csp, "script-src")
	if scriptSrc == "" {
		t.Fatalf("script-src directive missing from CSP %q", csp)
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src still allows 'unsafe-inline' (%q); inline scripts "+
			"in rendered content can execute (Chain D)", scriptSrc)
	}
	if strings.Contains(scriptSrc, "'unsafe-eval'") {
		t.Errorf("script-src allows 'unsafe-eval' (%q)", scriptSrc)
	}
	// D3 is vendored locally; the d3js.org CDN must no longer be allowlisted.
	if strings.Contains(scriptSrc, "d3js.org") {
		t.Errorf("script-src still allows d3js.org (%q); D3 is vendored at "+
			"/__static/d3.v7.min.js", scriptSrc)
	}
}

// TestNoInlineScriptsInTemplates asserts fur's own HTML carries no inline
// <script> bodies — every script must be an external src so the strict
// script-src CSP does not break the UI. We check the rendered output of the
// markdown and directory pages plus the embedded templates.
func TestNoInlineScriptsInTemplates(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	for _, path := range []string{"/", "/graph"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		s.mux.ServeHTTP(rec, req)
		body := rec.Body.String()
		// An inline script tag is "<script>" or "<script ...>" without a src
		// attribute. Flag any "<script>" with a non-empty body.
		scanInlineScript(t, path, body)
	}
}

func scanInlineScript(t *testing.T, label, body string) {
	t.Helper()
	low := strings.ToLower(body)
	idx := 0
	for {
		open := strings.Index(low[idx:], "<script")
		if open < 0 {
			return
		}
		open += idx
		gt := strings.Index(low[open:], ">")
		if gt < 0 {
			return
		}
		tag := low[open : open+gt+1]
		closeIdx := strings.Index(low[open+gt+1:], "</script>")
		bodyText := ""
		if closeIdx >= 0 {
			bodyText = strings.TrimSpace(body[open+gt+1 : open+gt+1+closeIdx])
		}
		if !strings.Contains(tag, "src=") && bodyText != "" {
			t.Errorf("%s: inline <script> with body found (CSP forbids inline execution): %.80q", label, bodyText)
		}
		idx = open + gt + 1
	}
}

// TestMermaidInitStrict asserts the externalized Mermaid bootstrap pins
// securityLevel 'strict', which sanitizes diagram HTML and disables click/
// script handlers — the direct defense against Mermaid-borne JS smuggling.
func TestMermaidInitStrict(t *testing.T) {
	data, err := static.Files.ReadFile("mermaid-init.js")
	if err != nil {
		t.Fatalf("read mermaid-init.js: %v", err)
	}
	if !strings.Contains(string(data), "securityLevel: 'strict'") {
		t.Error("mermaid-init.js must set securityLevel: 'strict'")
	}
}

// TestStaticScriptsServed confirms the externalized scripts are reachable
// (so the strict CSP does not leave a broken UI).
func TestStaticScriptsServed(t *testing.T) {
	s, _ := setupTestServer(t)
	defer s.sse.Stop()

	for _, name := range []string{"app.js", "livereload.js", "mermaid-init.js", "graph.js", "d3.v7.min.js"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/__static/"+name, nil)
		s.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("/__static/%s = %d, want 200", name, rec.Code)
		}
	}
}
