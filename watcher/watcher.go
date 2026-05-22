package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
)

// Watcher defines the file-watcher operations.
type Watcher interface {
	Add(path string) error
	Scan(ctx context.Context) error
	Close() error
	GetWarnings() []string
}

// fsWatcher wraps fsnotify.Watcher and implements Watcher.
type fsWatcher struct {
	done        chan struct{}
	doneOnce    sync.Once
	excludeList []string
	warnings    []string
	warnMu      sync.Mutex
	paths       []string
}

// New creates a new fsWatcher.
func New(excludeList []string) (*fsWatcher, error) {
	return &fsWatcher{
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
		w.paths = append(w.paths, path)
		return nil
	}

	if info.IsDir() {
		err := w.watchedWalk(path, func(p string) error {
			w.paths = append(w.paths, p)
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	}
	w.paths = append(w.paths, path)
	return nil
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
		_, err = os.Lstat(absTarget)
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

// Scan walks all registered paths and logs file information.
func (w *fsWatcher) Scan(ctx context.Context) error {
	for _, root := range w.paths {
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
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

			log.Info("scan file", "path", p, "size", info.Size())
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Close stops the watcher and releases resources.
func (w *fsWatcher) Close() error {
	w.doneOnce.Do(func() {
		close(w.done)
	})
	return nil
}

// GetWarnings returns all collected warnings.
func (w *fsWatcher) GetWarnings() []string {
	w.warnMu.Lock()
	defer w.warnMu.Unlock()
	res := make([]string, len(w.warnings))
	copy(res, w.warnings)
	return res
}