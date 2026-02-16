package index

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the vault for file changes and triggers re-indexing.
type Watcher struct {
	indexer  *Indexer
	watcher  *fsnotify.Watcher
	root     string
	debounce map[string]*time.Timer
	mu       sync.Mutex
	onChange func() // callback after index changes
}

func NewWatcher(indexer *Indexer, root string, onChange func()) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		indexer:  indexer,
		watcher:  fw,
		root:     root,
		debounce: make(map[string]*time.Timer),
		onChange: onChange,
	}

	// Add vault root and subdirectories
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			fw.Add(path)
		}
		return nil
	})

	return w, nil
}

// Start begins watching for changes. Blocks until Stop is called.
func (w *Watcher) Start() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Only care about markdown files
	if !strings.HasSuffix(path, ".md") {
		// But watch new directories
		if event.Has(fsnotify.Create) {
			info, err := os.Stat(path)
			if err == nil && info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
				w.watcher.Add(path)
			}
		}
		return
	}

	// Debounce: wait 200ms before processing
	w.mu.Lock()
	if timer, ok := w.debounce[path]; ok {
		timer.Stop()
	}
	w.debounce[path] = time.AfterFunc(200*time.Millisecond, func() {
		w.mu.Lock()
		delete(w.debounce, path)
		w.mu.Unlock()

		if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
			w.indexer.RemoveFile(path)
		} else {
			w.indexer.IndexFile(path)
		}

		if w.onChange != nil {
			w.onChange()
		}
	})
	w.mu.Unlock()
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	return w.watcher.Close()
}
