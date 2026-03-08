package index

import (
	"os"
	"path/filepath"
	"sync"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
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

	ft := &FulltextIndex{}

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

	// Try opening existing index first
	idx, err := bleve.Open(indexPath)
	if err == nil {
		ft.idx = idx
		return ft, nil
	}

	// Create the cache directory if needed
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}

	// Remove stale index directory to get a clean start
	os.RemoveAll(indexPath)

	idx, err = bleve.New(indexPath, m)
	if err != nil {
		return nil, err
	}
	ft.idx = idx
	return ft, nil
}

// BuildFrom reads all markdown files from the file index and batch-indexes them.
func (ft *FulltextIndex) BuildFrom(idx *Index) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	entries := idx.MarkdownFiles()
	root := idx.Root()

	batch := ft.idx.NewBatch()
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(root, e.RelPath))
		if err != nil {
			continue
		}
		doc := map[string]interface{}{
			"title":   filepath.Base(e.RelPath),
			"content": string(data),
			"path":    e.RelPath,
		}
		batch.Index(e.RelPath, doc)
	}

	return ft.idx.Batch(batch)
}

// Update re-indexes a single file. absPath is the full path on disk,
// relPath is the index-relative key.
func (ft *FulltextIndex) Update(absPath, relPath string) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	data, err := os.ReadFile(absPath)
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
