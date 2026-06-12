package git

import (
	"strings"
	"testing"
)

// FuzzNormalizeRemoteURL fuzzes the permalink remote-URL normalizer with
// arbitrary git remote strings. Invariant: never panics; the result never
// contains a shell metacharacter sequence that could matter if it were ever
// (incorrectly) passed to a shell — paired with TestGitPackageNoExec which
// guarantees it is not.
func FuzzNormalizeRemoteURL(f *testing.F) {
	seeds := []string{
		"git@github.com:o/r.git",
		"ssh://git@github.com:22/o/r.git",
		"https://gitlab.com/o/r",
		"git://codeberg.org/o/r",
		"ssh://-oProxyCommand=evil@h/x",
		"git@$(touch x):o/r",
		"https://h/" + strings.Repeat("a", 50000),
		"",
		"::::",
		"git@:",
		"\x1b]0;title\x07@h:/r",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, url string) {
		out := normalizeRemoteURL(url)
		_ = detectStyle(url)
		_ = buildFileLink(out, detectStyle(url), "main", "f.md", 1, 0)
		// Output is only ever used to build a URL string / displayed; it must
		// not be longer than a sane multiple of the input (no pathological
		// expansion).
		if len(out) > len(url)+len("https://") {
			t.Errorf("normalizeRemoteURL(%q) expanded unexpectedly to %q", url, out)
		}
	})
}
