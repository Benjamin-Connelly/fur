package web

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// updateGolden regenerates the .html golden files: go test ./internal/web/
// -run TestGoldenMarkdown -update
var updateGolden = flag.Bool("update", false, "update golden files")

// TestGoldenMarkdown renders every testdata/golden/*.md fixture through the
// exact web Goldmark instance (NewMarkdown) and compares the HTML against the
// committed .html golden. The corpus includes adversarial inputs (raw HTML,
// <script>/<iframe>, javascript:/data: links) so a regression that starts
// passing raw HTML through — e.g. someone adding html.WithUnsafe() — changes
// the golden and fails review (lookit-9py.2.3). The security invariants are
// also asserted directly below so the guarantee does not rely on a human
// reading the golden diff.
func TestGoldenMarkdown(t *testing.T) {
	md := NewMarkdown()
	inputs, err := filepath.Glob("testdata/golden/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no golden fixtures found")
	}

	for _, in := range inputs {
		in := in
		name := strings.TrimSuffix(filepath.Base(in), ".md")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := md.Convert(src, &buf); err != nil {
				t.Fatalf("Convert: %v", err)
			}
			got := buf.Bytes()

			goldenPath := strings.TrimSuffix(in, ".md") + ".html"
			if *updateGolden {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("rendered HTML for %s differs from golden; run -update to inspect.\n--- got ---\n%s", name, got)
			}
		})
	}
}

// TestGoldenMarkdownSecurityInvariants asserts the security-relevant
// properties of the rendered HTML independently of the golden bytes, so the
// guarantee survives an accidental `-update` that bakes in a regression.
func TestGoldenMarkdownSecurityInvariants(t *testing.T) {
	md := NewMarkdown()
	render := func(t *testing.T, src string) string {
		t.Helper()
		var buf bytes.Buffer
		if err := md.Convert([]byte(src), &buf); err != nil {
			t.Fatalf("Convert: %v", err)
		}
		return buf.String()
	}

	t.Run("raw script tag is dropped, not emitted", func(t *testing.T) {
		out := render(t, "<script>alert(1)</script>\n")
		if strings.Contains(out, "<script") {
			t.Errorf("raw <script> passed through:\n%s", out)
		}
		// Goldmark (no WithUnsafe) replaces raw HTML blocks with a comment
		// marker and drops the contents entirely.
		if !strings.Contains(out, "raw HTML omitted") {
			t.Errorf("expected raw HTML to be omitted, got:\n%s", out)
		}
		if strings.Contains(out, "alert(1)") {
			t.Errorf("script body leaked into output:\n%s", out)
		}
	})

	t.Run("raw iframe is dropped", func(t *testing.T) {
		out := render(t, `<iframe src="https://evil"></iframe>`+"\n")
		if strings.Contains(out, "<iframe") {
			t.Errorf("raw <iframe> passed through:\n%s", out)
		}
	})

	t.Run("javascript and data link schemes are neutralized", func(t *testing.T) {
		out := render(t, "[x](javascript:alert(1)) [y](data:text/html,<script>alert(1)</script>)\n")
		if strings.Contains(out, `href="javascript:`) {
			t.Errorf("javascript: href survived:\n%s", out)
		}
		if strings.Contains(out, `href="data:text/html`) {
			t.Errorf("data:text/html href survived:\n%s", out)
		}
	})
}
