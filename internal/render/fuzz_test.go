package render

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzSlugify fuzzes the anchor-slug chokepoint. Invariants: never panics;
// always returns valid UTF-8; the output contains only the slug alphabet
// ([a-z0-9-_]); and it is idempotent (slugifying a slug yields the same slug),
// which is what keeps fragment resolution stable.
func FuzzSlugify(f *testing.F) {
	for _, s := range []string{"", "Hello World", "café", "café", "# 🚀", "---", "a.b!c", "Cyrillic а", "ﬁle", strings.Repeat("x", 5000)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := Slugify(s)
		if !utf8.ValidString(got) {
			t.Errorf("Slugify(%q) produced invalid UTF-8: %q", s, got)
		}
		for _, r := range got {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				t.Errorf("Slugify(%q)=%q contains out-of-alphabet rune %q", s, got, r)
			}
		}
		if again := Slugify(got); again != got {
			t.Errorf("Slugify not idempotent: Slugify(%q)=%q, Slugify(that)=%q", s, got, again)
		}
	})
}

// FuzzMarkdownRender fuzzes the heading/link extraction and slug-mapping path
// over arbitrary markdown. Invariant: never panics, and AnchorSlugs returns
// exactly one slug per extracted heading.
func FuzzMarkdownRender(f *testing.F) {
	seeds := []string{
		"# H1\n## H2\n",
		"[link](../../../etc/passwd)\n",
		"[[wikilink]]\n",
		"<script>alert(1)</script>\n",
		"```mermaid\ngraph TD\nA-->B\n```\n",
		"![img](data:text/html,<script>x</script>)\n",
		"# ‮RTL override\n",
		strings.Repeat("#", 10000) + " deep\n",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		headings := ExtractHeadings(src)
		_ = ExtractLinks(src)
		slugs := AnchorSlugs(src)
		if len(slugs) != len(headings) {
			t.Errorf("AnchorSlugs returned %d slugs for %d headings", len(slugs), len(headings))
		}
		// Slugs must be unique (the property fragment resolution relies on).
		seen := map[string]bool{}
		for _, s := range slugs {
			if seen[s] {
				t.Errorf("duplicate slug %q from AnchorSlugs over %q", s, src)
			}
			seen[s] = true
		}
	})
}
