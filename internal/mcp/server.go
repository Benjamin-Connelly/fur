package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/afero"

	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/render"
	"github.com/Benjamin-Connelly/lookit/internal/tasks"
)

// Server wraps lookit's index and link graph as MCP tools.
type Server struct {
	idx   *index.Index
	links *index.LinkGraph
	mcp   *server.MCPServer
}

// New creates an MCP server with all lookit tools registered.
func New(idx *index.Index, links *index.LinkGraph) *Server {
	s := &Server{
		idx:   idx,
		links: links,
		mcp: server.NewMCPServer("lookit", "0.4.0",
			server.WithToolCapabilities(false),
		),
	}
	s.registerTools()
	return s
}

// Serve starts the MCP server on stdin/stdout.
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcp)
}

// --- Tool Input Types ---

type searchDocsArgs struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"Search query string"`
	Type  string `json:"type,omitempty" jsonschema_description:"Search type: filename (fuzzy) or content (fulltext)" jsonschema:"enum=filename,enum=content,default=filename"`
}

type getDocumentArgs struct {
	File string `json:"file" jsonschema:"required" jsonschema_description:"Relative file path within the indexed directory"`
}

type getRelatedDocsArgs struct {
	File      string `json:"file" jsonschema:"required" jsonschema_description:"Relative file path to find related documents for"`
	Direction string `json:"direction,omitempty" jsonschema_description:"Link direction: forward, back, or both" jsonschema:"enum=forward,enum=back,enum=both,default=both"`
}

type checkDocHealthArgs struct {
	File string `json:"file,omitempty" jsonschema_description:"Specific file to check (omit for all files)"`
}

type getDocStructureArgs struct {
	File string `json:"file" jsonschema:"required" jsonschema_description:"Relative file path to extract structure from"`
}

func (s *Server) registerTools() {
	// search_docs — fuzzy or fulltext search
	s.mcp.AddTool(
		mcp.NewTool("search_docs",
			mcp.WithDescription("Search documentation files by filename or content"),
			mcp.WithInputSchema[searchDocsArgs](),
		),
		mcp.NewTypedToolHandler(s.handleSearchDocs),
	)

	// get_document — read file content
	s.mcp.AddTool(
		mcp.NewTool("get_document",
			mcp.WithDescription("Read a document's content as plain text"),
			mcp.WithInputSchema[getDocumentArgs](),
		),
		mcp.NewTypedToolHandler(s.handleGetDocument),
	)

	// get_related_docs — forward links, backlinks, or both
	s.mcp.AddTool(
		mcp.NewTool("get_related_docs",
			mcp.WithDescription("Find documents linked to or from a file (forward links, backlinks, or both)"),
			mcp.WithInputSchema[getRelatedDocsArgs](),
		),
		mcp.NewTypedToolHandler(s.handleGetRelatedDocs),
	)

	// check_doc_health — broken links and pending tasks
	s.mcp.AddTool(
		mcp.NewTool("check_doc_health",
			mcp.WithDescription("Check documentation health: broken links, broken fragments, and pending tasks"),
			mcp.WithInputSchema[checkDocHealthArgs](),
		),
		mcp.NewTypedToolHandler(s.handleCheckDocHealth),
	)

	// get_doc_structure — headings and anchors
	s.mcp.AddTool(
		mcp.NewTool("get_doc_structure",
			mcp.WithDescription("Extract document structure: headings with anchor slugs and nesting levels"),
			mcp.WithInputSchema[getDocStructureArgs](),
		),
		mcp.NewTypedToolHandler(s.handleGetDocStructure),
	)
}

// validatePath checks that the resolved path stays within the index root.
// Prevents symlink escapes and path traversal.
func (s *Server) validatePath(relPath string) (string, error) {
	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}
	absPath := filepath.Join(s.idx.Root(), relPath)
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("file not found")
	}
	root := s.idx.Root()
	if !strings.HasPrefix(resolved, root+string(filepath.Separator)) && resolved != root {
		return "", fmt.Errorf("path escapes index root")
	}
	return absPath, nil
}

// --- Tool Handlers ---

func (s *Server) handleSearchDocs(ctx context.Context, req mcp.CallToolRequest, args searchDocsArgs) (*mcp.CallToolResult, error) {
	if args.Query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}
	if len(args.Query) > 500 {
		return mcp.NewToolResultError("query too long (max 500 chars)"), nil
	}

	if args.Type == "content" && s.idx.Fulltext != nil {
		results, err := s.idx.Fulltext.Search(args.Query, 20)
		if err != nil {
			return mcp.NewToolResultError("search failed: " + err.Error()), nil
		}
		var lines []string
		for _, r := range results {
			snippet := ""
			if len(r.Snippets) > 0 {
				snippet = " — " + r.Snippets[0]
			}
			lines = append(lines, fmt.Sprintf("%s (score: %.2f)%s", r.Path, r.Score, snippet))
		}
		if len(lines) == 0 {
			return mcp.NewToolResultText("No results found."), nil
		}
		return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
	}

	// Default: fuzzy filename search
	results := s.idx.FuzzySearch(args.Query, 20)
	var lines []string
	for _, r := range results {
		lines = append(lines, r.RelPath)
	}
	if len(lines) == 0 {
		return mcp.NewToolResultText("No results found."), nil
	}
	return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
}

func (s *Server) handleGetDocument(ctx context.Context, req mcp.CallToolRequest, args getDocumentArgs) (*mcp.CallToolResult, error) {
	if args.File == "" {
		return mcp.NewToolResultError("file is required"), nil
	}

	entry := s.idx.Lookup(args.File)
	if entry == nil {
		return mcp.NewToolResultError("file not found: " + args.File), nil
	}

	absPath, err := s.validatePath(args.File)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if entry.Size > 10*1024*1024 {
		return mcp.NewToolResultError("file too large (>10MB)"), nil
	}

	data, err := afero.ReadFile(s.idx.Fs(), absPath)
	if err != nil {
		return mcp.NewToolResultError("read error: " + err.Error()), nil
	}

	// Binary check
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	for _, b := range sample {
		if b == 0 {
			return mcp.NewToolResultError("binary file, cannot display"), nil
		}
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleGetRelatedDocs(ctx context.Context, req mcp.CallToolRequest, args getRelatedDocsArgs) (*mcp.CallToolResult, error) {
	if args.File == "" {
		return mcp.NewToolResultError("file is required"), nil
	}
	if s.idx.Lookup(args.File) == nil {
		return mcp.NewToolResultError("file not found: " + args.File), nil
	}
	if s.links == nil {
		return mcp.NewToolResultError("link graph not available"), nil
	}

	dir := args.Direction
	if dir == "" {
		dir = "both"
	}

	var lines []string

	if dir == "forward" || dir == "both" {
		fwd := s.links.ForwardLinks(args.File)
		if len(fwd) > 0 {
			lines = append(lines, "Forward links:")
			for _, l := range fwd {
				status := ""
				if l.Broken {
					status = " [BROKEN]"
				}
				frag := ""
				if l.Fragment != "" {
					frag = "#" + l.Fragment
				}
				lines = append(lines, fmt.Sprintf("  → %s%s (%s)%s", l.Target, frag, l.Text, status))
			}
		}
	}

	if dir == "back" || dir == "both" {
		back := s.links.Backlinks(args.File)
		if len(back) > 0 {
			lines = append(lines, "Backlinks:")
			for _, l := range back {
				lines = append(lines, fmt.Sprintf("  ← %s:%d (%s)", l.Source, l.Line, l.Text))
			}
		}
	}

	if len(lines) == 0 {
		return mcp.NewToolResultText("No related documents found."), nil
	}
	return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
}

func (s *Server) handleCheckDocHealth(ctx context.Context, req mcp.CallToolRequest, args checkDocHealthArgs) (*mcp.CallToolResult, error) {
	var lines []string

	// Broken links
	if s.links != nil {
		broken := s.links.BrokenLinks()
		brokenFrags := s.links.BrokenFragmentLinks()

		if args.File != "" {
			// Filter to specific file
			var filtered []index.Link
			for _, l := range broken {
				if l.Source == args.File {
					filtered = append(filtered, l)
				}
			}
			broken = filtered

			filtered = nil
			for _, l := range brokenFrags {
				if l.Source == args.File {
					filtered = append(filtered, l)
				}
			}
			brokenFrags = filtered
		}

		if len(broken) > 0 {
			lines = append(lines, fmt.Sprintf("Broken links (%d):", len(broken)))
			for _, l := range broken {
				lines = append(lines, fmt.Sprintf("  %s:%d → %s (%s)", l.Source, l.Line, l.Target, l.Text))
			}
		}
		if len(brokenFrags) > 0 {
			lines = append(lines, fmt.Sprintf("Broken fragments (%d):", len(brokenFrags)))
			for _, l := range brokenFrags {
				lines = append(lines, fmt.Sprintf("  %s:%d → %s#%s (%s)", l.Source, l.Line, l.Target, l.Fragment, l.Text))
			}
		}
	}

	// Pending tasks
	var allTasks []tasks.Task
	mdFiles := s.idx.MarkdownFiles()
	for _, entry := range mdFiles {
		if args.File != "" && entry.RelPath != args.File {
			continue
		}
		if entry.Size > 10*1024*1024 {
			continue // skip files > 10MB
		}
		absPath := filepath.Join(s.idx.Root(), entry.RelPath)
		data, err := afero.ReadFile(s.idx.Fs(), absPath)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks.Extract(entry.RelPath, string(data))...)
		if len(allTasks) > 1000 {
			break
		}
	}
	pending := tasks.Pending(allTasks)

	if len(pending) > 0 {
		lines = append(lines, fmt.Sprintf("Pending tasks (%d):", len(pending)))
		for _, t := range pending {
			pri := ""
			if t.Priority != "" {
				pri = " !" + t.Priority
			}
			lines = append(lines, fmt.Sprintf("  %s:%d%s %s", t.File, t.Line, pri, t.Text))
		}
	}

	if len(lines) == 0 {
		return mcp.NewToolResultText("No issues found. Documentation is healthy."), nil
	}
	return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
}

func (s *Server) handleGetDocStructure(ctx context.Context, req mcp.CallToolRequest, args getDocStructureArgs) (*mcp.CallToolResult, error) {
	if args.File == "" {
		return mcp.NewToolResultError("file is required"), nil
	}

	entry := s.idx.Lookup(args.File)
	if entry == nil {
		return mcp.NewToolResultError("file not found: " + args.File), nil
	}

	absPath, err := s.validatePath(args.File)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if entry.Size > 10*1024*1024 {
		return mcp.NewToolResultError("file too large (>10MB)"), nil
	}

	data, err := afero.ReadFile(s.idx.Fs(), absPath)
	if err != nil {
		return mcp.NewToolResultError("read error: " + err.Error()), nil
	}

	content := string(data)
	headings := render.ExtractHeadings(content)

	slugCounts := make(map[string]int)
	var lines []string
	for _, h := range headings {
		slug := render.Slugify(h.Text)
		n := slugCounts[slug]
		slugCounts[slug]++
		if n > 0 {
			slug = fmt.Sprintf("%s-%d", slug, n)
		}
		indent := strings.Repeat("  ", h.Level-1)
		lines = append(lines, fmt.Sprintf("%s%s (#%s) [line %d]", indent, h.Text, slug, h.Line))
	}

	if len(lines) == 0 {
		return mcp.NewToolResultText("No headings found."), nil
	}

	header := fmt.Sprintf("Structure of %s:\n", args.File)
	return mcp.NewToolResultText(header + strings.Join(lines, "\n")), nil
}
