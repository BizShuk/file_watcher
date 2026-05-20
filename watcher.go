package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
)

// FileHandler is called for each file event.
// (ISP: focused callback interface, no fat methods)
type FileHandler func(event fsnotify.Event)

// WatcherOps defines the file-watcher operations.
// (DIP: main.go depends on this abstraction, not fsnotify directly)
type WatcherOps interface {
	Add(path string) error
	Start(handler FileHandler) error
	Close() error
	GetWarnings() []string
}

// fsWatcher wraps fsnotify.Watcher and implements WatcherOps.
type fsWatcher struct {
	wrapped     *fsnotify.Watcher
	done        chan struct{}
	excludeList []string
	warnings    []string
	warnMu      sync.Mutex
}

// NewWatcher creates a new fsWatcher.
func NewWatcher(excludeList []string) (*fsWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}
	return &fsWatcher{
		wrapped:     w,
		done:        make(chan struct{}),
		excludeList: excludeList,
	}, nil
}

// Add registers a path to be watched.
// If path is a directory, it is watched recursively.
func (w *fsWatcher) Add(path string) error {
	isBroken, target := w.checkBrokenSymlink(path)
	if isBroken {
		w.addWarning(fmt.Sprintf("broken symlink detected: %s -> %s (target not found)", path, target))
		return nil
	}

	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("lstat path %q: %w", path, err)
	}

	// purely monitor the soft link itself
	if info.Mode()&os.ModeSymlink != 0 {
		return w.wrapped.Add(path)
	}

	if info.IsDir() {
		err := w.watchedWalk(path, func(p string) error {
			return w.wrapped.Add(p)
		})
		if err != nil {
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

		if info.Mode()&os.ModeSymlink != 0 {
			isBroken, target := w.checkBrokenSymlink(p)
			if isBroken {
				w.addWarning(fmt.Sprintf("broken symlink detected: %s -> %s (target not found)", p, target))
			}
			return nil
		}

		for _, ext := range w.excludeList {
			if strings.HasSuffix(p, ext) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if info.IsDir() {
			if err := fn(p); err != nil {
				w.addWarning(fmt.Sprintf("failed to watch directory %s: %v", p, err))
				log.Warn("Failed to watch directory", "path", p, "err", err)
			}
		}
		return nil
	})
}

// GetWarnings returns all warnings collected during execution.
func (w *fsWatcher) GetWarnings() []string {
	w.warnMu.Lock()
	defer w.warnMu.Unlock()
	res := make([]string, len(w.warnings))
	copy(res, w.warnings)
	return res
}

func (w *fsWatcher) addWarning(msg string) {
	w.warnMu.Lock()
	defer w.warnMu.Unlock()
	w.warnings = append(w.warnings, msg)
}

func (w *fsWatcher) checkBrokenSymlink(path string) (bool, string) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, ""
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return true, ""
		}
		absTarget := target
		if !filepath.IsAbs(target) {
			absTarget = filepath.Join(filepath.Dir(path), target)
		}
		_, err = os.Stat(absTarget)
		if err != nil && os.IsNotExist(err) {
			return true, target
		}
	}
	return false, ""
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
				skip := false
				for _, ext := range w.excludeList {
					if strings.HasSuffix(event.Name, ext) {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
				log.Info("Event", "name", event.Name, "op", event.Op)

				handler(event)
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
