package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	w.Close()
}

func TestWatcherAdd_file(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile.txt")
	os.WriteFile(tmp, []byte("hello"), 0644)
	if err := w.Add(tmp); err != nil {
		t.Errorf("expected no error adding file, got %v", err)
	}
}

func TestWatcherAdd_directory(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	dir := filepath.Join(t.TempDir(), "subdir")
	os.Mkdir(dir, 0755)
	if err := w.Add(dir); err != nil {
		t.Errorf("expected no error adding dir, got %v", err)
	}
}

func TestWatcherAdd_nonexistent(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = w.Add("/nonexistent/path")
	if err == nil {
		t.Error("expected error adding nonexistent path")
	}
}

func TestWatcherStart_contextCancel(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile.txt")
	os.WriteFile(tmp, []byte("hello"), 0644)
	w.Add(tmp)

	done := make(chan struct{})
	go func() {
		_ = w.Start(func(event fsnotify.Event, path string, size int64, modTime int64) {})
		close(done)
	}()

	// Cancel after a short delay.
	time.Sleep(100 * time.Millisecond)
	w.Close()

	<-done
	// Should not panic or error on close
}

func TestWatcherClose(t *testing.T) {
	w, err := NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}