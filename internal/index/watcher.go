package index

import (
	"errors"
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
	onChange func()      // callback after index changes
	onError  func(error) // callback on fatal errors

	closed bool
}

func NewWatcher(indexer *Indexer, root string, onChange func(), onError func(error)) (*Watcher, error) {
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
		onError:  onError,
	}

	// Add vault root and subdirectories
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			if err := fw.Add(path); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, errors.Join(err, fw.Close())
	}

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

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.fatal(err)
			return
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
				if err := w.watcher.Add(path); err != nil {
					w.fatal(err)
					return
				}
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
			if err := w.indexer.RemoveFile(path); err != nil {
				w.fatal(err)
				return
			}
		} else {
			if err := w.indexer.IndexFile(path); err != nil {
				w.fatal(err)
				return
			}
		}

		if w.onChange != nil {
			w.onChange()
		}
	})
	w.mu.Unlock()
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	return w.watcher.Close()
}
