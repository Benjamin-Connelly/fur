package index

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/spf13/afero"
)

// SearchResult holds a single fulltext search hit.
type SearchResult struct {
	Path     string   // relative file path
	Score    float64  // BM25 relevance score
	Snippets []string // highlighted content fragments
	Title    string   // filename
}

// FulltextIndex wraps a Bleve index for content search.
type FulltextIndex struct {
	idx  bleve.Index
	path string // on-disk path (empty = memory-only)
	fs   afero.Fs
	mu   sync.RWMutex
}

// buildMapping creates the document mapping with title (boosted), content, and path fields.
func buildMapping() mapping.IndexMapping {
	titleField := bleve.NewTextFieldMapping()
	titleField.Store = true
	titleField.IncludeTermVectors = true

	contentField := bleve.NewTextFieldMapping()
	contentField.Store = true
	contentField.IncludeTermVectors = true

	pathField := bleve.NewKeywordFieldMapping()

	docMapping := bleve.NewDocumentMapping()
	docMapping.AddFieldMappingsAt("title", titleField)
	docMapping.AddFieldMappingsAt("content", contentField)
	docMapping.AddFieldMappingsAt("path", pathField)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = docMapping
	indexMapping.DefaultAnalyzer = "standard"

	return indexMapping
}

// NewFulltextIndex creates a Bleve index. If cacheDir is non-empty, the index
// is persisted at cacheDir/index.bleve; otherwise it uses an in-memory store.
func NewFulltextIndex(cacheDir string) (*FulltextIndex, error) {
	m := buildMapping()

	ft := &FulltextIndex{fs: afero.NewOsFs()}

	if cacheDir == "" {
		idx, err := bleve.NewMemOnly(m)
		if err != nil {
			return nil, err
		}
		ft.idx = idx
		return ft, nil
	}

	indexPath := filepath.Join(cacheDir, "index.bleve")
	ft.path = indexPath

	// Try opening existing index first. Re-tighten perms on open: an index
	// created by an older fur (0o755) or left loose by a co-located adversary
	// would otherwise stay world-readable. The Bleve index mirrors the content
	// of every browsed file, so on a multi-user box loose perms are a
	// cross-user disclosure (audit Chains F and H).
	idx, err := bleve.Open(indexPath)
	if err == nil {
		secureCachePerms(cacheDir, indexPath)
		ft.idx = idx
		return ft, nil
	}

	// Create the cache directory if needed, owner-only (0700).
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, err
	}

	// Remove stale index directory to get a clean start
	os.RemoveAll(indexPath)

	idx, err = bleve.New(indexPath, m)
	if err != nil {
		return nil, err
	}
	// Bleve creates its store with the process umask (typically 0o755);
	// clamp the cache dir and the index tree to owner-only.
	secureCachePerms(cacheDir, indexPath)
	ft.idx = idx
	return ft, nil
}

// secureCachePerms clamps the fur cache directory and the Bleve index tree to
// owner-only access (0700 dirs, 0600 files). Best-effort: permission errors
// are ignored because a usable-but-loose cache still beats refusing to run,
// and the disclosure risk is the operator's own home directory.
//
// The cache dir is clamped first so that, before we descend, no other user
// can traverse in to swap a symlink; the index tree is then walked through an
// os.Root so every chmod is root-scoped and cannot follow a symlink out of
// the cache (avoids the WalkDir+Chmod TOCTOU class).
func secureCachePerms(cacheDir, indexPath string) {
	_ = os.Chmod(cacheDir, 0o700)

	root, err := os.OpenRoot(indexPath)
	if err != nil {
		return
	}
	defer root.Close()

	_ = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = root.Chmod(p, 0o700)
		} else {
			_ = root.Chmod(p, 0o600)
		}
		return nil
	})
}

// BuildFrom reads all markdown files from the file index and batch-indexes them.
func (ft *FulltextIndex) BuildFrom(idx *Index) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	entries := idx.MarkdownFiles()
	root := idx.Root()

	batch := ft.idx.NewBatch()
	for _, e := range entries {
		data, err := afero.ReadFile(idx.Fs(), filepath.Join(root, e.RelPath))
		if err != nil {
			continue
		}
		doc := map[string]interface{}{
			"title":   filepath.Base(e.RelPath),
			"content": string(data),
			"path":    e.RelPath,
		}
		_ = batch.Index(e.RelPath, doc)
	}

	return ft.idx.Batch(batch)
}

// Update re-indexes a single file. absPath is the full path on disk,
// relPath is the index-relative key.
func (ft *FulltextIndex) Update(absPath, relPath string) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	data, err := afero.ReadFile(ft.fs, absPath)
	if err != nil {
		return err
	}
	doc := map[string]interface{}{
		"title":   filepath.Base(relPath),
		"content": string(data),
		"path":    relPath,
	}
	return ft.idx.Index(relPath, doc)
}

// Remove deletes a document from the index.
func (ft *FulltextIndex) Remove(relPath string) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return ft.idx.Delete(relPath)
}

// Search runs a match query and returns results with highlighted snippets.
func (ft *FulltextIndex) Search(query string, maxResults int) ([]SearchResult, error) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	if query == "" {
		return nil, nil
	}

	mq := bleve.NewMatchQuery(query)
	req := bleve.NewSearchRequestOptions(mq, maxResults, 0, false)
	req.Fields = []string{"title", "path"}
	req.Highlight = bleve.NewHighlight()

	res, err := ft.idx.Search(req)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(res.Hits))
	for _, hit := range res.Hits {
		sr := SearchResult{
			Path:  hit.ID,
			Score: hit.Score,
		}
		if t, ok := hit.Fields["title"].(string); ok {
			sr.Title = t
		}
		if frags, ok := hit.Fragments["content"]; ok {
			sr.Snippets = frags
		}
		results = append(results, sr)
	}
	return results, nil
}

// Close shuts down the underlying Bleve index.
func (ft *FulltextIndex) Close() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return ft.idx.Close()
}
