package watcher

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

// Handler is called for each file event.
type Handler func(path string, op fsnotify.Op) error

// Watcher defines the file-watcher operations.
type Watcher interface {
	Add(path string) error
	Start(ctx context.Context, handler Handler) error
	Close() error
	GetWarnings() []string
}

// fsWatcher wraps fsnotify.Watcher and implements Watcher.
type fsWatcher struct {
	wrapped     *fsnotify.Watcher
	done        chan struct{}
	doneOnce    sync.Once
	excludeList []string
	warnings    []string
	warnMu      sync.Mutex
	wg          sync.WaitGroup
}

// New creates a new fsWatcher.
func New(excludeList []string) (*fsWatcher, error) {
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

func (w *fsWatcher) watchedWalk(root string, fn func(string) error) error {
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
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

func (w *fsWatcher) addWarning(msg string) {
	w.warnMu.Lock()
	defer w.warnMu.Unlock()
	w.warnings = append(w.warnings, msg)
}

// Start begins watching and dispatches events to the handler.
func (w *fsWatcher) Start(ctx context.Context, handler Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
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

				if err := handler(event.Name, event.Op); err != nil {
					log.Warn("handler error", "path", event.Name, "err", err)
				}
			case err := <-w.wrapped.Errors:
				log.Warn("watcher error", "err", err)
			}
		}
	}()
	<-ctx.Done()
	return ctx.Err()
}

// Close stops the watcher and releases resources.
func (w *fsWatcher) Close() error {
	w.doneOnce.Do(func() {
		close(w.done)
	})
	w.wg.Wait()
	return w.wrapped.Close()
}

// GetWarnings returns all collected warnings.
func (w *fsWatcher) GetWarnings() []string {
	w.warnMu.Lock()
	defer w.warnMu.Unlock()
	res := make([]string, len(w.warnings))
	copy(res, w.warnings)
	return res
}