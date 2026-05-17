package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// FileHandler is called for each file event.
// (ISP: focused callback interface, no fat methods)
type FileHandler func(path string, size int64, modTime int64, isRemove bool)

// WatcherOps defines the file-watcher operations.
// (DIP: main.go depends on this abstraction, not fsnotify directly)
type WatcherOps interface {
	Add(path string) error
	Start(handler FileHandler) error
	Close() error
}

// fsWatcher wraps fsnotify.Watcher and implements WatcherOps.
type fsWatcher struct {
	wrapped *fsnotify.Watcher
	done    chan struct{}
}

// NewWatcher creates a new fsWatcher.
func NewWatcher() (*fsWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}
	return &fsWatcher{wrapped: w, done: make(chan struct{})}, nil
}

// Add registers a path to be watched.
// If path is a directory, it is watched recursively.
func (w *fsWatcher) Add(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path %q: %w", path, err)
	}
	if info.IsDir() {
		// Watch recursively — fsnotify doesn't auto-recurse.
		if err := w.watchedWalk(path, func(p string) error {
			return w.wrapped.Add(p)
		}); err != nil {
			return err
		}
		return nil
	}
	return w.wrapped.Add(path)
}

// watchedWalk recursively processes a directory tree.
func (w *fsWatcher) watchedWalk(root string, fn func(string) error) error {
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if info.IsDir() {
			if err := fn(p); err != nil {
				return err
			}
		}
		return nil
	})
}

// Start begins watching and dispatches events to the handler.
// It blocks until Close is called or an error occurs.
func (w *fsWatcher) Start(handler FileHandler) error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-w.done:
				cancel()
				return
			case <-ctx.Done():
				return
			case event := <-w.wrapped.Events:
				if event.Has(fsnotify.Remove) {
					handler(event.Name, 0, 0, true)
					continue
				} 
				
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					info, err := os.Stat(event.Name)
					if err != nil {
						continue // file gone or inaccessible
					}
					handler(event.Name, info.Size(), info.ModTime().Unix(), false)
				}
			case err := <-w.wrapped.Errors:
				// Log and continue — many fsnotify errors are transient.
				fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
			}
		}
	}()
	<-ctx.Done()
	return ctx.Err()
}

// Close stops the watcher and releases resources.
func (w *fsWatcher) Close() error {
	if w.done != nil {
		close(w.done)
		w.done = nil
	}
	return w.wrapped.Close()
}