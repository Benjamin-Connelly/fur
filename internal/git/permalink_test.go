package git

import (
	"testing"
)

func TestNormalizeRemoteURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/user/repo.git", "https://github.com/user/repo"},
		{"https://github.com/user/repo", "https://github.com/user/repo"},
		{"git@github.com:user/repo.git", "https://github.com/user/repo"},
		{"git@github.com:user/repo", "https://github.com/user/repo"},
		{"ssh://git@github.com/user/repo.git", "https://github.com/user/repo"},
		{"git://github.com/user/repo.git", "https://github.com/user/repo"},
		{"ssh://git@gitlab.com:2222/user/repo.git", "https://gitlab.com/user/repo"},
	}
	for _, tt := range tests {
		got := normalizeRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectStyle(t *testing.T) {
	tests := []struct {
		url  string
		want PermalinkStyle
	}{
		{"https://github.com/user/repo", PermalinkGitHub},
		{"git@github.com:user/repo", PermalinkGitHub},
		{"https://gitlab.com/user/repo", PermalinkGitLab},
		{"https://bitbucket.org/user/repo", PermalinkBitbucket},
		{"https://codeberg.org/user/repo", PermalinkCodeberg},
		{"https://gitea.example.com/user/repo", PermalinkGitea},
		{"https://unknown.example.com/user/repo", PermalinkGitHub}, // default
	}
	for _, tt := range tests {
		got := detectStyle(tt.url)
		if got != tt.want {
			t.Errorf("detectStyle(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestBuildFileLink(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		style     PermalinkStyle
		ref       string
		filePath  string
		startLine int
		endLine   int
		want      string
	}{
		{
			"github no line",
			"https://github.com/user/repo", PermalinkGitHub, "abc123", "src/main.go", 0, 0,
			"https://github.com/user/repo/blob/abc123/src/main.go",
		},
		{
			"github single line",
			"https://github.com/user/repo", PermalinkGitHub, "abc123", "main.go", 42, 0,
			"https://github.com/user/repo/blob/abc123/main.go#L42",
		},
		{
			"github line range",
			"https://github.com/user/repo", PermalinkGitHub, "abc123", "main.go", 10, 20,
			"https://github.com/user/repo/blob/abc123/main.go#L10-L20",
		},
		{
			"gitlab",
			"https://gitlab.com/user/repo", PermalinkGitLab, "abc123", "main.go", 10, 20,
			"https://gitlab.com/user/repo/-/blob/abc123/main.go#L10-20",
		},
		{
			"bitbucket",
			"https://bitbucket.org/user/repo", PermalinkBitbucket, "abc123", "main.go", 5, 0,
			"https://bitbucket.org/user/repo/src/abc123/main.go#lines-5",
		},
		{
			"bitbucket range",
			"https://bitbucket.org/user/repo", PermalinkBitbucket, "abc", "f.go", 1, 10,
			"https://bitbucket.org/user/repo/src/abc/f.go#lines-1:10",
		},
		{
			"gitea",
			"https://gitea.io/user/repo", PermalinkGitea, "abc", "f.go", 5, 0,
			"https://gitea.io/user/repo/src/commit/abc/f.go#L5",
		},
		{
			"codeberg range",
			"https://codeberg.org/user/repo", PermalinkCodeberg, "abc", "f.go", 1, 5,
			"https://codeberg.org/user/repo/src/commit/abc/f.go#L1-L5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFileLink(tt.baseURL, tt.style, tt.ref, tt.filePath, tt.startLine, tt.endLine)
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestLineFragment(t *testing.T) {
	// No line
	if f := lineFragment(PermalinkGitHub, 0, 0); f != "" {
		t.Errorf("expected empty, got %q", f)
	}
	// Negative line
	if f := lineFragment(PermalinkGitHub, -1, 0); f != "" {
		t.Errorf("expected empty for negative, got %q", f)
	}
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"docs/guide.markdown", true},
		{"notes.mdown", true},
		{"spec.mkd", true},
		{"UPPER.MD", true},
		{"main.go", false},
		{"style.css", false},
		{"data.json", false},
		{"no-extension", false},
	}
	for _, tt := range tests {
		got := isMarkdownFile(tt.path)
		if got != tt.want {
			t.Errorf("isMarkdownFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestBuildFileLink_MarkdownPlain(t *testing.T) {
	// Markdown files with line numbers should get ?plain=1
	got := buildFileLink("https://github.com/user/repo", PermalinkGitHub, "abc123", "README.md", 10, 0)
	want := "https://github.com/user/repo/blob/abc123/README.md?plain=1#L10"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// Markdown without line numbers should NOT get ?plain=1
	got = buildFileLink("https://github.com/user/repo", PermalinkGitHub, "abc123", "README.md", 0, 0)
	want = "https://github.com/user/repo/blob/abc123/README.md"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// Non-markdown with line numbers should NOT get ?plain=1
	got = buildFileLink("https://github.com/user/repo", PermalinkGitHub, "abc123", "main.go", 10, 0)
	want = "https://github.com/user/repo/blob/abc123/main.go#L10"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestLineFragment_AllStyles(t *testing.T) {
	tests := []struct {
		name  string
		style PermalinkStyle
		start int
		end   int
		want  string
	}{
		{"github single", PermalinkGitHub, 5, 0, "#L5"},
		{"github range", PermalinkGitHub, 5, 15, "#L5-L15"},
		{"gitlab single", PermalinkGitLab, 5, 0, "#L5"},
		{"gitlab range", PermalinkGitLab, 5, 15, "#L5-15"},
		{"bitbucket single", PermalinkBitbucket, 5, 0, "#lines-5"},
		{"bitbucket range", PermalinkBitbucket, 5, 15, "#lines-5:15"},
		{"gitea single", PermalinkGitea, 5, 0, "#L5"},
		{"gitea range", PermalinkGitea, 5, 15, "#L5-L15"},
		{"codeberg single", PermalinkCodeberg, 5, 0, "#L5"},
		{"codeberg range", PermalinkCodeberg, 5, 15, "#L5-L15"},
		{"zero line", PermalinkGitHub, 0, 0, ""},
		{"negative line", PermalinkGitLab, -1, 0, ""},
		{"end equals start", PermalinkGitHub, 5, 5, "#L5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineFragment(tt.style, tt.start, tt.end)
			if got != tt.want {
				t.Errorf("lineFragment(%v, %d, %d) = %q, want %q", tt.style, tt.start, tt.end, got, tt.want)
			}
		})
	}
}
