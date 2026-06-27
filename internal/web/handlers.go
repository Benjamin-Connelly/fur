package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	gitpkg "github.com/Benjamin-Connelly/fur/internal/git"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/plugin"
	"github.com/Benjamin-Connelly/fur/internal/render"
	"github.com/Benjamin-Connelly/fur/internal/tasks"
	"github.com/Benjamin-Connelly/fur/internal/web/templates"
	"github.com/spf13/afero"
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
		if _, err := s.idx.ValidatePath(relPath); err != nil {
			http.NotFound(w, r)
			return
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
	HasMermaid   bool // page contains a mermaid diagram → load the mermaid bundle
}

type tocHeading struct {
	Level int
	Text  string
	Slug  string
}

// slugify converts a heading text to a URL-safe anchor ID.
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
		_ = s.plugins.Run(plugin.HookBeforeRender, ctx)
		sourceStr = ctx.Content
	}

	// Render markdown to HTML using Goldmark
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(sourceStr), &buf); err != nil {
		http.Error(w, "Markdown render error", http.StatusInternalServerError)
		return
	}

	// Extract headings for TOC. Use the centralized AnchorSlugs so duplicate
	// headings get the same disambiguated slugs the document API and TUI use
	// (audit Chain M).
	headings := render.ExtractHeadings(string(source))
	slugs := render.AnchorSlugs(string(source))
	var tocHeadings []tocHeading
	for i, h := range headings {
		tocHeadings = append(tocHeadings, tocHeading{
			Level: h.Level,
			Text:  h.Text,
			Slug:  slugs[i],
		})
	}

	// Gather links from the link graph
	var backlinks []index.Link
	var forwardLinks []index.Link
	if s.links != nil {
		backlinks = s.links.Backlinks(relPath)
		forwardLinks = s.links.ForwardLinks(relPath)
	}

	// Replace mermaid fenced code blocks so mermaid.js renders them client-side.
	// hasMermaid drives conditional loading of the (2.6MB) mermaid bundle — it
	// is included only when the page actually contains a diagram.
	hasMermaid := false
	rendered := mermaidBlockRe.ReplaceAllStringFunc(buf.String(), func(match string) string {
		inner := mermaidBlockRe.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		hasMermaid = true
		return `<pre class="mermaid">` + inner[1] + `</pre>`
	})

	// Fire AfterRender hook on rendered HTML
	if s.plugins != nil {
		ctx := &plugin.HookContext{Content: rendered, FilePath: relPath}
		_ = s.plugins.Run(plugin.HookAfterRender, ctx)
		rendered = ctx.Content
	}

	data := markdownPageData{
		pageData:     s.buildPageData(relPath),
		RelPath:      relPath,
		RenderedHTML: template.HTML(rendered),
		Headings:     tocHeadings,
		Backlinks:    backlinks,
		ForwardLinks: forwardLinks,
		HasMermaid:   hasMermaid,
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

// imageContentTypes is the allowlist of extensions handleFile serves as raw
// image bytes (with the mapped Content-Type) instead of routing through the
// syntax highlighter. Keeping it an explicit map — rather than deferring to
// mime.TypeByExtension — bounds exactly what handleFile will emit as a
// non-HTML body, which is a deliberate security boundary. These are all raster
// formats that cannot carry script.
//
// SVG is handled separately in handleFile (it is an active document): it is
// served as image/svg+xml but under a per-response `sandbox` CSP that gives it
// an opaque origin with scripting disabled, so it renders as an image without
// the XSS surface. It is therefore not in this raster-only map.
var imageContentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".bmp":  "image/bmp",
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request, relPath string) {
	absPath := filepath.Join(s.idx.Root(), relPath)
	source, err := afero.ReadFile(s.fs, absPath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(relPath))

	// SVG is an active document (it can carry <script>, event handlers,
	// <foreignObject>). It renders as an image via <img> (script-safe by spec),
	// but navigating directly to the .svg URL loads it as a top-level document.
	// Serve image/svg+xml, but override the response CSP with `sandbox` (opaque
	// origin; scripts, forms and frames disabled) plus default-src 'none'.
	// Combined with the global script-src 'self', this neutralizes every active
	// vector on direct navigation; sandbox does not affect <img> embedding (it
	// applies only to documents), so inline SVG images still render.
	if ext == ".svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
		// #nosec G705 -- the per-response sandbox CSP gives the document an
		// opaque origin with scripting disabled, so the SVG cannot execute or
		// reach fur's origin even when opened directly.
		w.Write(source)
		return
	}

	// Other images are served as raw bytes so <img> references in rendered
	// markdown resolve; the ETag middleware still wraps this for caching.
	if ct, ok := imageContentTypes[ext]; ok {
		w.Header().Set("Content-Type", ct)
		// #nosec G705 -- not XSS: bytes are served with a non-HTML image/*
		// Content-Type and the middleware's X-Content-Type-Options: nosniff, so
		// the browser cannot interpret them as HTML. (SVG is handled above.)
		w.Write(source)
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
	if s.idx.GetFulltext() != nil {
		bleveResults, err := s.idx.GetFulltext().Search(query, 100)
		if err == nil {
			var results []searchResult
			for _, br := range bleveResults {
				// Confine results to the current served root. The Bleve index is a
				// persistent global cache (~/.cache/fur/index.bleve) that accumulates
				// entries from every root fur has ever served and is not scoped or
				// cleared per-root, so a hit may be a stale entry from a different
				// directory. Direct reads delegate to ValidatePath; search must honor
				// the same boundary or it discloses paths and content snippets from
				// outside the served tree. The in-memory index only holds the current
				// root's files, so a nil Lookup means the hit is out-of-root.
				if s.idx.Lookup(br.Path) == nil {
					continue
				}
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
			sanitized := strings.NewReplacer("\n", "", "\r", "").Replace(msg)
			fmt.Fprintf(w, "data: %s\n\n", sanitized)
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

func (s *Server) handleAPIDocument(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, `missing "file" query parameter`, http.StatusBadRequest)
		return
	}

	// Delegate to the shared chokepoint. ValidatePath rejects path-traversal
	// strings and resolves symlinks, refusing targets outside the serve root.
	// Returns the absolute path on success so we don't recompute it.
	absPath, err := s.idx.ValidatePath(filePath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	entry := s.idx.Lookup(filePath)
	if entry == nil {
		http.Error(w, "file not found in index", http.StatusNotFound)
		return
	}

	data, err := afero.ReadFile(s.fs, absPath)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	content := string(data)

	// Extract headings. Slugs come from the centralized render.AnchorSlugs so
	// the API, the server-side TOC, and the TUI fragment scroller all agree on
	// duplicate disambiguation (audit Chain M).
	headings := render.ExtractHeadings(content)
	slugs := render.AnchorSlugs(content)
	type headingJSON struct {
		Level int    `json:"level"`
		Text  string `json:"text"`
		Slug  string `json:"slug"`
		Line  int    `json:"line"`
	}
	hdgs := make([]headingJSON, 0, len(headings))
	for i, h := range headings {
		hdgs = append(hdgs, headingJSON{
			Level: h.Level,
			Text:  h.Text,
			Slug:  slugs[i],
			Line:  h.Line,
		})
	}

	// Extract links (forward + backlinks)
	type linkJSON struct {
		Source   string `json:"source"`
		Target   string `json:"target"`
		Text     string `json:"text"`
		Line     int    `json:"line,omitempty"`
		Broken   bool   `json:"broken,omitempty"`
		Fragment string `json:"fragment,omitempty"`
	}
	var fwd, back []linkJSON
	if s.links != nil {
		for _, l := range s.links.ForwardLinks(filePath) {
			fwd = append(fwd, linkJSON{
				Source:   l.Source,
				Target:   l.Target,
				Text:     l.Text,
				Line:     l.Line,
				Broken:   l.Broken,
				Fragment: l.Fragment,
			})
		}
		for _, l := range s.links.Backlinks(filePath) {
			back = append(back, linkJSON{
				Source: l.Source,
				Target: l.Target,
				Text:   l.Text,
				Line:   l.Line,
			})
		}
	}

	result := struct {
		File         string        `json:"file"`
		Size         int64         `json:"size"`
		IsMarkdown   bool          `json:"isMarkdown"`
		Headings     []headingJSON `json:"headings"`
		ForwardLinks []linkJSON    `json:"forwardLinks"`
		Backlinks    []linkJSON    `json:"backlinks"`
	}{
		File:         filePath,
		Size:         entry.Size,
		IsMarkdown:   entry.IsMarkdown,
		Headings:     hdgs,
		ForwardLinks: fwd,
		Backlinks:    back,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleAPITasks(w http.ResponseWriter, r *http.Request) {
	pendingOnly := r.URL.Query().Get("pending") == "true"

	var allTasks []tasks.Task
	for _, entry := range s.idx.MarkdownFiles() {
		if entry.Size > 10*1024*1024 {
			continue // skip files > 10MB
		}
		absPath := filepath.Join(s.idx.Root(), entry.RelPath)
		data, err := afero.ReadFile(s.fs, absPath)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks.Extract(entry.RelPath, string(data))...)
		if len(allTasks) > 1000 {
			break
		}
	}

	if pendingOnly {
		allTasks = tasks.Pending(allTasks)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allTasks)
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
