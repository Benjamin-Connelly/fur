package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	gitpkg "github.com/Benjamin-Connelly/lookit/internal/git"
	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/plugin"
	"github.com/Benjamin-Connelly/lookit/internal/render"
	"github.com/Benjamin-Connelly/lookit/internal/web/templates"
	"github.com/spf13/afero"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Common template data shared by all pages.
type pageData struct {
	Title         string
	Breadcrumbs   []breadcrumb
	GitBranch     string
	ExtraCSS      template.CSS
	CustomCSSPath string
}

type breadcrumb struct {
	Name string
	Href string
}

func (s *Server) buildPageData(relPath string) pageData {
	title := relPath
	if title == "." {
		title = filepath.Base(s.idx.Root())
	}

	pd := pageData{Title: title}

	// Build breadcrumbs
	if relPath != "." {
		parts := strings.Split(relPath, "/")
		for i, part := range parts {
			pd.Breadcrumbs = append(pd.Breadcrumbs, breadcrumb{
				Name: part,
				Href: "/" + strings.Join(parts[:i+1], "/"),
			})
		}
	}

	// Git branch
	if s.cfg.Git.Enabled {
		repo, err := gitpkg.Open(s.idx.Root())
		if err == nil {
			if branch, err := repo.Branch(); err == nil {
				pd.GitBranch = branch
			}
		}
	}

	// Chroma CSS for syntax highlighting
	css, err := s.code.CSS()
	if err == nil {
		pd.ExtraCSS = template.CSS(css)
	}

	// Custom CSS override
	if s.cfg.Server.CustomCSS != "" {
		pd.CustomCSSPath = "/__custom.css"
	}

	return pd
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	cleanPath := filepath.Clean(r.URL.Path)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	relPath := strings.TrimPrefix(cleanPath, "/")
	if relPath == "" {
		relPath = "."
	}

	// Verify resolved path stays within the served root
	if relPath != "." {
		absPath := filepath.Join(s.idx.Root(), relPath)
		resolved, err := filepath.EvalSymlinks(absPath)
		if err == nil {
			rootPrefix := s.idx.Root() + string(os.PathSeparator)
			if !strings.HasPrefix(resolved, rootPrefix) && resolved != s.idx.Root() {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
	}

	entry := s.idx.Lookup(relPath)
	if entry == nil && relPath != "." {
		http.NotFound(w, r)
		return
	}

	if entry != nil && entry.IsDir {
		s.handleDirectory(w, r, relPath)
		return
	}

	if entry != nil && entry.IsMarkdown {
		s.handleMarkdown(w, r, relPath)
		return
	}

	if entry != nil {
		s.handleFile(w, r, relPath)
		return
	}

	// Root directory
	s.handleDirectory(w, r, ".")
}

// Directory listing data
type dirPageData struct {
	pageData
	ParentHref string
	GitEnabled bool
	Entries    []dirEntry
}

type dirEntry struct {
	Name      string
	Path      string
	IsDir     bool
	SizeStr   string
	ModTime   string
	GitStatus string
	GitClass  string
}

func (s *Server) handleDirectory(w http.ResponseWriter, r *http.Request, relPath string) {
	entries := s.idx.Entries()

	// Filter to direct children of this directory
	var dirEntries []dirEntry
	for _, e := range entries {
		dir := filepath.Dir(e.RelPath)
		if dir == "." {
			dir = ""
		}
		target := relPath
		if target == "." {
			target = ""
		}
		if dir != target || e.RelPath == "." {
			continue
		}

		de := dirEntry{
			Name:    filepath.Base(e.RelPath),
			Path:    e.RelPath,
			IsDir:   e.IsDir,
			SizeStr: formatSize(e.Size),
			ModTime: e.ModTime.Format("Jan 02, 2006 15:04"),
		}
		dirEntries = append(dirEntries, de)
	}

	// Sort: directories first, then alphabetical
	sortDirEntries(dirEntries)

	// Git status badges
	var gitStatuses map[string]gitpkg.FileStatus
	if s.cfg.Git.Enabled {
		repo, err := gitpkg.Open(s.idx.Root())
		if err == nil {
			statuses, err := repo.Status()
			if err == nil {
				gitStatuses = make(map[string]gitpkg.FileStatus, len(statuses))
				for _, fs := range statuses {
					gitStatuses[fs.Path] = fs
				}
			}
		}
	}

	if gitStatuses != nil {
		for i := range dirEntries {
			if fs, ok := gitStatuses[dirEntries[i].Path]; ok {
				dirEntries[i].GitStatus, dirEntries[i].GitClass = gitStatusLabel(fs)
			}
		}
	}

	var parentHref string
	if relPath != "." {
		parent := filepath.Dir(relPath)
		if parent == "." {
			parentHref = "/"
		} else {
			parentHref = "/" + parent
		}
	}

	data := dirPageData{
		pageData:   s.buildPageData(relPath),
		ParentHref: parentHref,
		GitEnabled: s.cfg.Git.Enabled && gitStatuses != nil,
		Entries:    dirEntries,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := templates.PageTemplates["directory.html"].ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

// Markdown view data
type markdownPageData struct {
	pageData
	RelPath      string
	RenderedHTML template.HTML
	Headings     []tocHeading
	Backlinks    []index.Link
	ForwardLinks []index.Link
}

type tocHeading struct {
	Level int
	Text  string
	Slug  string
}

// slugify converts a heading text to a URL-safe anchor ID.
func slugify(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *Server) handleMarkdown(w http.ResponseWriter, r *http.Request, relPath string) {
	absPath := filepath.Join(s.idx.Root(), relPath)
	source, err := afero.ReadFile(s.fs, absPath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Fire BeforeRender hook on source
	sourceStr := string(source)
	if s.plugins != nil {
		ctx := &plugin.HookContext{Content: sourceStr, FilePath: relPath}
		s.plugins.Run(plugin.HookBeforeRender, ctx)
		sourceStr = ctx.Content
	}

	// Render markdown to HTML using Goldmark
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.Emoji,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(sourceStr), &buf); err != nil {
		http.Error(w, "Markdown render error", http.StatusInternalServerError)
		return
	}

	// Extract headings for TOC
	headings := render.ExtractHeadings(string(source))
	var tocHeadings []tocHeading
	for _, h := range headings {
		tocHeadings = append(tocHeadings, tocHeading{
			Level: h.Level,
			Text:  h.Text,
			Slug:  slugify(h.Text),
		})
	}

	// Gather links from the link graph
	var backlinks []index.Link
	var forwardLinks []index.Link
	if s.links != nil {
		backlinks = s.links.Backlinks(relPath)
		forwardLinks = s.links.ForwardLinks(relPath)
	}

	// Replace mermaid fenced code blocks so mermaid.js renders them client-side
	rendered := mermaidBlockRe.ReplaceAllStringFunc(buf.String(), func(match string) string {
		inner := mermaidBlockRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		return `<pre class="mermaid">` + inner[1] + `</pre>`
	})

	// Fire AfterRender hook on rendered HTML
	if s.plugins != nil {
		ctx := &plugin.HookContext{Content: rendered, FilePath: relPath}
		s.plugins.Run(plugin.HookAfterRender, ctx)
		rendered = ctx.Content
	}

	data := markdownPageData{
		pageData:     s.buildPageData(relPath),
		RelPath:      relPath,
		RenderedHTML: template.HTML(rendered),
		Headings:     tocHeadings,
		Backlinks:    backlinks,
		ForwardLinks: forwardLinks,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var out bytes.Buffer
	if err := templates.PageTemplates["markdown.html"].ExecuteTemplate(&out, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out.WriteTo(w)
}

// Code view data
type codePageData struct {
	pageData
	Language        string
	SizeStr         string
	HighlightedHTML template.HTML
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request, relPath string) {
	absPath := filepath.Join(s.idx.Root(), relPath)
	source, err := afero.ReadFile(s.fs, absPath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(relPath)
	highlighted, err := s.code.Highlight(filename, string(source))
	if err != nil {
		highlighted = template.HTMLEscapeString(string(source))
	}

	entry := s.idx.Lookup(relPath)
	var size int64
	if entry != nil {
		size = entry.Size
	}

	data := codePageData{
		pageData:        s.buildPageData(relPath),
		Language:        s.code.GetLanguage(filename),
		SizeStr:         formatSize(size),
		HighlightedHTML: template.HTML(highlighted),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := templates.PageTemplates["code.html"].ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func (s *Server) handleAPIFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var entries interface{}
	if query != "" {
		entries = s.idx.FuzzySearch(query, 50)
	} else {
		entries = s.idx.Entries()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// searchResult represents a single grep match.
type searchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

var grepLineRe = regexp.MustCompile(`^([^:]+):(\d+):(.*)$`)

// mermaidBlockRe matches goldmark-rendered mermaid fenced code blocks.
var mermaidBlockRe = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)

func (s *Server) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(query) > 200 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]searchResult{})
		return
	}

	// Use Bleve fulltext search when available
	if s.idx.Fulltext != nil {
		bleveResults, err := s.idx.Fulltext.Search(query, 100)
		if err == nil {
			var results []searchResult
			for _, br := range bleveResults {
				content := ""
				if len(br.Snippets) > 0 {
					content = br.Snippets[0]
				}
				results = append(results, searchResult{
					File:    br.Path,
					Line:    0,
					Content: content,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
			return
		}
		// Fall through to grep on error
	}

	// Use git grep if in a git repo, otherwise fall back to grep.
	// Use "--" to separate flags from the pattern to prevent flag injection.
	// Use a 5-second timeout to prevent ReDoS from pathological patterns.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if gitpkg.IsRepo(s.idx.Root()) {
		cmd = exec.CommandContext(ctx, "git", "grep", "-n", "--no-color", "-I", "-F", "--", query)
	} else {
		cmd = exec.CommandContext(ctx, "grep", "-rn", "--no-color", "-I", "-F", "--", query, ".")
	}
	cmd.Dir = s.idx.Root()

	output, _ := cmd.Output() // ignore exit code (1 = no matches)

	var results []searchResult
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		m := grepLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNum := 0
		fmt.Sscanf(m[2], "%d", &lineNum)
		results = append(results, searchResult{
			File:    m[1],
			Line:    lineNum,
			Content: m[3],
		})
		if len(results) >= 100 {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgCh := make(chan string, 8)
	s.sse.register <- msgCh
	defer func() {
		s.sse.unregister <- msgCh
	}()

	ctx := r.Context()
	for {
		select {
		case msg := <-msgCh:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// Helper functions

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func sortDirEntries(entries []dirEntry) {
	// Simple insertion sort: dirs first, then alphabetical
	for i := 1; i < len(entries); i++ {
		j := i
		for j > 0 && dirEntryLess(entries[j], entries[j-1]) {
			entries[j], entries[j-1] = entries[j-1], entries[j]
			j--
		}
	}
}

func dirEntryLess(a, b dirEntry) bool {
	if a.IsDir != b.IsDir {
		return a.IsDir
	}
	return strings.ToLower(a.Name) < strings.ToLower(b.Name)
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	data := s.buildPageData("graph")
	data.Title = "Link Graph"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := templates.PageTemplates["graph.html"].ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func (s *Server) handleAPIGraph(w http.ResponseWriter, r *http.Request) {
	type graphNode struct {
		ID         string `json:"id"`
		Label      string `json:"label"`
		IsMarkdown bool   `json:"isMarkdown"`
		Links      int    `json:"links"`
	}
	type graphLink struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	type graphData struct {
		Nodes []graphNode `json:"nodes"`
		Links []graphLink `json:"links"`
	}

	nodeSet := make(map[string]bool)
	var links []graphLink

	if s.links != nil {
		for _, entry := range s.idx.Entries() {
			if !entry.IsMarkdown {
				continue
			}
			fwd := s.links.ForwardLinks(entry.RelPath)
			if len(fwd) == 0 {
				continue
			}
			nodeSet[entry.RelPath] = true
			for _, link := range fwd {
				if link.Broken {
					continue
				}
				nodeSet[link.Target] = true
				links = append(links, graphLink{Source: entry.RelPath, Target: link.Target})
			}
		}
	}

	var nodes []graphNode
	for id := range nodeSet {
		label := filepath.Base(id)
		linkCount := len(s.links.ForwardLinks(id)) + len(s.links.Backlinks(id))
		nodes = append(nodes, graphNode{
			ID:         id,
			Label:      label,
			IsMarkdown: strings.HasSuffix(id, ".md"),
			Links:      linkCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graphData{Nodes: nodes, Links: links})
}

func gitStatusLabel(fs gitpkg.FileStatus) (label, class string) {
	code := fs.Worktree
	if code == ' ' {
		code = fs.Staging
	}
	switch code {
	case gitpkg.Modified:
		return "M", "modified"
	case gitpkg.Added:
		return "A", "added"
	case gitpkg.Deleted:
		return "D", "deleted"
	case gitpkg.Renamed:
		return "R", "modified"
	case gitpkg.Copied:
		return "C", "added"
	case gitpkg.Untracked:
		return "?", "untracked"
	default:
		return "", ""
	}
}
