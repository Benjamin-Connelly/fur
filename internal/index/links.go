package index

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Benjamin-Connelly/lookit/internal/render"
)

// Link represents a link from one file to another.
type Link struct {
	Source         string // relative path of the source file
	Target         string // relative path or URL of the target
	Fragment       string // anchor fragment (e.g., "heading-name" from #heading-name)
	Text           string // link text
	Line           int    // line number in source
	Broken         bool   // true if target cannot be resolved
	BrokenFragment bool   // true if target exists but #fragment doesn't match any heading
}

// LinkGraph maintains bidirectional link relationships between files.
type LinkGraph struct {
	// Forward links: source -> []Link
	forward map[string][]Link
	// Backlinks: target -> []Link
	backward map[string][]Link
	mu       sync.RWMutex
}

// NewLinkGraph creates an empty link graph.
func NewLinkGraph() *LinkGraph {
	return &LinkGraph{
		forward:  make(map[string][]Link),
		backward: make(map[string][]Link),
	}
}

// SetLinks replaces all links originating from the given source file.
func (g *LinkGraph) SetLinks(source string, links []Link) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Remove old backlinks for this source
	for _, oldLink := range g.forward[source] {
		g.removeBacklink(oldLink.Target, source)
	}

	// Set new forward links
	g.forward[source] = links

	// Add new backlinks
	for _, link := range links {
		g.backward[link.Target] = append(g.backward[link.Target], link)
	}
}

// ForwardLinks returns all links originating from the given file.
func (g *LinkGraph) ForwardLinks(source string) []Link {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Link, len(g.forward[source]))
	copy(result, g.forward[source])
	return result
}

// Backlinks returns all links pointing to the given file.
func (g *LinkGraph) Backlinks(target string) []Link {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Link, len(g.backward[target]))
	copy(result, g.backward[target])
	return result
}

// BrokenLinks returns all links in the graph that are marked as broken.
func (g *LinkGraph) BrokenLinks() []Link {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var broken []Link
	for _, links := range g.forward {
		for _, link := range links {
			if link.Broken {
				broken = append(broken, link)
			}
		}
	}
	return broken
}

func (g *LinkGraph) removeBacklink(target, source string) {
	backlinks := g.backward[target]
	filtered := backlinks[:0]
	for _, bl := range backlinks {
		if bl.Source != source {
			filtered = append(filtered, bl)
		}
	}
	g.backward[target] = filtered
}

var (
	// Matches [text](target) markdown links
	mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Matches [[wikilink]] style links
	wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

// ExtractLinks parses markdown content for links, resolves relative paths,
// and marks broken links based on whether targets exist in the index.
func ExtractLinks(filePath, content string, idx *Index) []Link {
	sourceDir := filepath.Dir(filePath)
	lines := strings.Split(content, "\n")
	var links []Link

	for lineNum, line := range lines {
		// Standard markdown links: [text](target)
		for _, match := range mdLinkRe.FindAllStringSubmatch(line, -1) {
			text := match[1]
			target := match[2]

			// Extract fragment anchor
			var fragment string
			if i := strings.Index(target, "#"); i >= 0 {
				fragment = target[i+1:]
				target = target[:i]
			}
			target = strings.TrimSpace(target)

			// Skip external URLs and empty targets (but allow pure #fragment)
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if target == "" && fragment == "" {
				continue
			}

			// Pure #fragment links target the current file
			resolved := filePath
			broken := false
			if target != "" {
				resolved = resolveRelPath(sourceDir, target, idx.Root())
				broken = idx.Lookup(resolved) == nil
			}

			links = append(links, Link{
				Source:   filePath,
				Target:   resolved,
				Fragment: fragment,
				Text:     text,
				Line:     lineNum + 1,
				Broken:   broken,
			})
		}

		// Wikilinks: [[target]] or [[target|text]]
		for _, match := range wikiLinkRe.FindAllStringSubmatch(line, -1) {
			raw := match[1]
			text := raw
			target := raw

			// Handle [[target|display text]] syntax
			if i := strings.Index(raw, "|"); i >= 0 {
				target = raw[:i]
				text = raw[i+1:]
			}

			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}

			// Try to resolve wikilink: look for matching file in the index
			resolved := resolveWikilink(target, idx)
			broken := resolved == "" || idx.Lookup(resolved) == nil

			if resolved == "" {
				resolved = target
			}

			links = append(links, Link{
				Source: filePath,
				Target: resolved,
				Text:   strings.TrimSpace(text),
				Line:   lineNum + 1,
				Broken: broken,
			})
		}
	}

	return links
}

// BuildFromIndex reads all markdown files from the index, extracts links,
// and populates the graph.
func (g *LinkGraph) BuildFromIndex(idx *Index) {
	mdFiles := idx.MarkdownFiles()
	for _, entry := range mdFiles {
		content, err := os.ReadFile(entry.Path)
		if err != nil {
			continue
		}
		links := ExtractLinks(entry.RelPath, string(content), idx)
		g.SetLinks(entry.RelPath, links)
	}
}

// ValidateFragments checks all links with fragments and marks those whose
// anchor doesn't match any heading in the target file. Call after BuildFromIndex.
func (g *LinkGraph) ValidateFragments(idx *Index) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Cache file headings to avoid re-reading the same file
	slugCache := make(map[string]map[string]bool)

	for source, links := range g.forward {
		for i := range links {
			link := &g.forward[source][i]
			if link.Fragment == "" || link.Broken {
				continue
			}

			entry := idx.Lookup(link.Target)
			if entry == nil {
				continue
			}

			slugs, ok := slugCache[link.Target]
			if !ok {
				data, err := os.ReadFile(entry.Path)
				if err != nil {
					continue
				}
				slugs = render.HeadingSlugs(string(data))
				slugCache[link.Target] = slugs
			}

			frag := strings.ToLower(link.Fragment)
			if !slugs[frag] {
				link.BrokenFragment = true
			}
		}
	}
}

// BrokenFragmentLinks returns all links where the target exists but the
// fragment doesn't match any heading.
func (g *LinkGraph) BrokenFragmentLinks() []Link {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var broken []Link
	for _, links := range g.forward {
		for _, link := range links {
			if link.BrokenFragment {
				broken = append(broken, link)
			}
		}
	}
	return broken
}

// resolveRelPath resolves a relative link target against a source directory,
// returning a path relative to the index root.
func resolveRelPath(sourceDir, target, root string) string {
	// Join source dir with target
	joined := filepath.Join(sourceDir, target)
	// Clean the path
	cleaned := filepath.Clean(joined)
	// Ensure no traversal above root
	if strings.HasPrefix(cleaned, "..") {
		return target
	}
	return cleaned
}

// resolveWikilink tries to find a matching file for a wikilink target.
// It searches for exact matches and common markdown extensions.
func resolveWikilink(target string, idx *Index) string {
	// Normalize: replace spaces with path-friendly chars
	normalized := strings.ReplaceAll(target, " ", "-")

	candidates := []string{
		target,
		normalized,
		target + ".md",
		normalized + ".md",
		target + ".markdown",
		normalized + ".markdown",
	}

	for _, candidate := range candidates {
		if entry := idx.Lookup(candidate); entry != nil {
			return candidate
		}
	}

	// Try case-insensitive search through all entries
	lowerTarget := strings.ToLower(normalized)
	entries := idx.Entries()
	for _, e := range entries {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(e.RelPath), filepath.Ext(e.RelPath)))
		if base == lowerTarget {
			return e.RelPath
		}
	}

	return ""
}
