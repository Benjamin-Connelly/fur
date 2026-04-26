package index

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the filesystem for changes and re-indexes affected files.
type Watcher struct {
	index    *Index
	graph    *LinkGraph
	watcher  *fsnotify.Watcher
	onChange func(path string) // callback for file changes
	done     chan struct{}
	debounce time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	pending  map[string]struct{}
}

// NewWatcher creates a file watcher that updates the index on changes.
// If graph is non-nil, it will be rebuilt when markdown files change.
func NewWatcher(idx *Index, graph *LinkGraph, onChange func(path string)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		index:    idx,
		graph:    graph,
		watcher:  fw,
		onChange: onChange,
		done:     make(chan struct{}),
		debounce: 100 * time.Millisecond,
		pending:  make(map[string]struct{}),
	}

	go w.loop()

	return w, nil
}

// Start begins watching the index root directory recursively.
func (w *Watcher) Start() error {
	return w.addRecursive(w.index.Root())
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// If a new directory was created, watch it
				if event.Has(fsnotify.Create) {
					w.maybeWatchDir(event.Name)
				}

				rel, err := filepath.Rel(w.index.Root(), event.Name)
				if err != nil {
					continue
				}

				w.mu.Lock()
				w.pending[rel] = struct{}{}
				w.scheduleBuild()
				w.mu.Unlock()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		case <-w.done:
			return
		}
	}
}

// scheduleBuild resets the debounce timer. Must be called with w.mu held.
func (w *Watcher) scheduleBuild() {
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, w.doBuild)
}

// doBuild re-indexes and notifies callbacks for pending changes.
func (w *Watcher) doBuild() {
	w.mu.Lock()
	paths := make([]string, 0, len(w.pending))
	hasMd := false
	for p := range w.pending {
		paths = append(paths, p)
		if isMarkdown(filepath.Base(p)) {
			hasMd = true
		}
	}
	w.pending = make(map[string]struct{})
	w.mu.Unlock()

	if err := w.index.Rebuild(); err != nil {
		log.Printf("watcher: re-index failed: %v", err)
		return
	}

	// Rebuild link graph if markdown files changed
	if hasMd && w.graph != nil {
		w.graph.BuildFromIndex(w.index)
	}

	// Update fulltext index for changed markdown files
	if w.index.Fulltext != nil {
		for _, p := range paths {
			if !isMarkdown(filepath.Base(p)) {
				continue
			}
			absPath := filepath.Join(w.index.Root(), p)
			if _, err := w.index.Fs().Stat(absPath); err != nil {
				// File was deleted
				if delErr := w.index.Fulltext.Remove(p); delErr != nil {
					log.Printf("watcher: fulltext remove %s: %v", p, delErr)
				}
			} else {
				if updErr := w.index.Fulltext.Update(absPath, p); updErr != nil {
					log.Printf("watcher: fulltext update %s: %v", p, updErr)
				}
			}
		}
	}

	if w.onChange != nil {
		for _, rel := range paths {
			w.onChange(rel)
		}
	}
}

// maybeWatchDir adds a new directory to the watcher if it exists and is not
// filtered. VCS metadata dirs are always skipped; other dotdirs respect the
// index's ShowHidden option.
func (w *Watcher) maybeWatchDir(path string) {
	info, err := w.index.Fs().Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	name := filepath.Base(path)
	if hiddenDirs[name] {
		return
	}
	if !w.index.opts.ShowHidden && strings.HasPrefix(name, ".") {
		return
	}
	if err := w.watcher.Add(path); err != nil {
		log.Printf("watcher: failed to watch new dir %s: %v", path, err)
	}
}

func (w *Watcher) addRecursive(root string) error {
	entries := w.index.Entries()
	dirs := make(map[string]bool)
	dirs[root] = true
	showHidden := w.index.opts.ShowHidden
	for _, e := range entries {
		if !e.IsDir {
			continue
		}
		base := filepath.Base(e.Path)
		if hiddenDirs[base] {
			continue
		}
		if !showHidden && strings.HasPrefix(base, ".") {
			continue
		}
		dirs[e.Path] = true
	}
	for dir := range dirs {
		if err := w.watcher.Add(dir); err != nil {
			log.Printf("watcher: skipping %s: %v", dir, err)
		}
	}
	return nil
}
