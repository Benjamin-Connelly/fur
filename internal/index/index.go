package index

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileEntry represents an indexed file.
type FileEntry struct {
	Path       string
	RelPath    string
	Size       int64
	ModTime    time.Time
	IsDir      bool
	IsMarkdown bool
}

// Stats holds aggregate statistics about the index.
type Stats struct {
	FileCount int
	DirCount  int
	TotalSize int64
}

// Options configures the indexing behavior.
type Options struct {
	IgnorePatterns []string // additional glob patterns to ignore
}

// Index maintains an in-memory file index for fast lookup and search.
type Index struct {
	root     string
	entries  []FileEntry
	byPath   map[string]*FileEntry
	opts     Options
	stats    Stats
	Fulltext *FulltextIndex
	mu       sync.RWMutex
}

// BuildFulltext creates and populates a fulltext search index. If cacheDir
// is empty, the index lives in memory only.
func (idx *Index) BuildFulltext(cacheDir string) error {
	ft, err := NewFulltextIndex(cacheDir)
	if err != nil {
		return err
	}
	if err := ft.BuildFrom(idx); err != nil {
		ft.Close()
		return err
	}
	idx.Fulltext = ft
	return nil
}

// CloseFulltext shuts down the fulltext index if one exists.
func (idx *Index) CloseFulltext() {
	if idx.Fulltext != nil {
		idx.Fulltext.Close()
	}
}

// New creates a new Index rooted at the given directory.
func New(root string) *Index {
	return &Index{
		root:   root,
		byPath: make(map[string]*FileEntry),
	}
}

// NewWithOptions creates a new Index with custom options.
func NewWithOptions(root string, opts Options) *Index {
	return &Index{
		root:   root,
		byPath: make(map[string]*FileEntry),
		opts:   opts,
	}
}

// hiddenDirs are always skipped during indexing.
var hiddenDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, ".bzr": true,
}

// Build walks the root directory and populates the index.
func (idx *Index) Build() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.entries = nil
	idx.byPath = make(map[string]*FileEntry)
	idx.stats = Stats{}

	gitignore := loadGitignore(idx.root)

	err := filepath.WalkDir(idx.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()

		// Skip hidden directories (.git, .hg, etc.)
		if d.IsDir() && path != idx.root {
			if hiddenDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
		}

		rel, err := filepath.Rel(idx.root, path)
		if err != nil {
			return nil
		}

		// Skip root itself
		if rel == "." {
			return nil
		}

		// Check .gitignore patterns
		if gitignore.match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check custom ignore patterns
		if idx.matchesIgnorePatterns(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entry := FileEntry{
			Path:       path,
			RelPath:    rel,
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			IsDir:      d.IsDir(),
			IsMarkdown: isMarkdown(name),
		}

		idx.entries = append(idx.entries, entry)

		if d.IsDir() {
			idx.stats.DirCount++
		} else {
			idx.stats.FileCount++
			idx.stats.TotalSize += info.Size()
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Build byPath map after walk completes so pointers into the
	// finalized entries slice are stable (append during walk can
	// reallocate the backing array, invalidating earlier pointers).
	for i := range idx.entries {
		idx.byPath[idx.entries[i].RelPath] = &idx.entries[i]
	}

	return nil
}

// Rebuild performs an incremental update by re-walking the tree.
// It replaces the index atomically.
func (idx *Index) Rebuild() error {
	newIdx := NewWithOptions(idx.root, idx.opts)
	if err := newIdx.Build(); err != nil {
		return err
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.entries = newIdx.entries
	idx.byPath = newIdx.byPath
	idx.stats = newIdx.stats
	return nil
}

// Entries returns a copy of all indexed entries.
func (idx *Index) Entries() []FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]FileEntry, len(idx.entries))
	copy(result, idx.entries)
	return result
}

// MarkdownFiles returns only markdown file entries.
func (idx *Index) MarkdownFiles() []FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	var result []FileEntry
	for _, e := range idx.entries {
		if e.IsMarkdown {
			result = append(result, e)
		}
	}
	return result
}

// Lookup returns the entry for a relative path, or nil if not found.
func (idx *Index) Lookup(relPath string) *FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.byPath[relPath]
}

// Root returns the index root directory.
func (idx *Index) Root() string {
	return idx.root
}

// Stats returns aggregate statistics about the index.
func (idx *Index) Stats() Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.stats
}

func (idx *Index) matchesIgnorePatterns(rel string) bool {
	for _, pattern := range idx.opts.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(rel)); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, rel); matched {
			return true
		}
	}
	return false
}

func isMarkdown(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".markdown" || ext == ".mdown"
}

// gitignoreRules holds parsed .gitignore patterns.
type gitignoreRules struct {
	patterns []gitignorePattern
}

type gitignorePattern struct {
	pattern  string
	negate   bool
	dirOnly  bool
	hasSlash bool // pattern contains a slash (anchored to root)
}

func loadGitignore(root string) gitignoreRules {
	path := filepath.Join(root, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return gitignoreRules{}
	}
	defer f.Close()

	var rules gitignoreRules
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p := gitignorePattern{}

		if strings.HasPrefix(line, "!") {
			p.negate = true
			line = line[1:]
		}

		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		// A pattern with a slash (not just trailing) is anchored
		p.hasSlash = strings.Contains(line, "/")
		p.pattern = line

		rules.patterns = append(rules.patterns, p)
	}
	return rules
}

func (g gitignoreRules) match(rel string, isDir bool) bool {
	matched := false
	for _, p := range g.patterns {
		if p.dirOnly && !isDir {
			continue
		}

		doesMatch := false
		if p.hasSlash {
			// Anchored: match against the full relative path
			doesMatch = matchGlob(p.pattern, rel)
		} else {
			// Unanchored: match against the basename, or any path segment
			doesMatch = matchGlob(p.pattern, filepath.Base(rel))
			if !doesMatch {
				doesMatch = matchGlob(p.pattern, rel)
			}
		}

		if doesMatch {
			matched = !p.negate
		}
	}
	return matched
}

// matchGlob handles simple glob matching including ** for directory wildcards.
func matchGlob(pattern, name string) bool {
	// Handle ** patterns by trying filepath.Match on segments
	if strings.Contains(pattern, "**") {
		// Convert ** to match any number of path segments
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			if prefix == "" && suffix == "" {
				return true
			}
			if prefix == "" {
				// **/suffix - match suffix against end of path
				if matched, _ := filepath.Match(suffix, name); matched {
					return true
				}
				if matched, _ := filepath.Match(suffix, filepath.Base(name)); matched {
					return true
				}
				return false
			}
			if suffix == "" {
				// prefix/** - match prefix against start of path
				return strings.HasPrefix(name, prefix+"/") || name == prefix
			}
		}
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}
