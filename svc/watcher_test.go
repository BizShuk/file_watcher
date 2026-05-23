package svc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWatcher(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	w.Close()
}

func TestWatcherAdd_file(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile.txt")
	os.WriteFile(tmp, []byte("hello"), 0o644)
	if err := w.Add(tmp); err != nil {
		t.Errorf("expected no error adding file, got %v", err)
	}
}

func TestWatcherAdd_directory(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	dir := filepath.Join(t.TempDir(), "subdir")
	os.Mkdir(dir, 0o755)
	if err := w.Add(dir); err != nil {
		t.Errorf("expected no error adding dir, got %v", err)
	}
}

func TestWatcherAdd_nonexistent(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = w.Add("/nonexistent/path")
	if err == nil {
		t.Error("expected error adding nonexistent path")
	}
}

func TestWatcherScan(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tempDir := t.TempDir()

	subdir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subdir, 0o755)

	file1 := filepath.Join(tempDir, "file1.txt")
	os.WriteFile(file1, []byte("hello"), 0o644)

	file2 := filepath.Join(subdir, "file2.txt")
	os.WriteFile(file2, []byte("world"), 0o644)

	if err := w.Add(tempDir); err != nil {
		t.Fatalf("expected no error adding dir, got %v", err)
	}

	ctx := context.Background()
	if _, err := w.Scan(ctx); err != nil {
		t.Errorf("expected no error from scan, got %v", err)
	}
}

func TestWatcherClose(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}

func TestWatcher_Symlinks(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tempDir := t.TempDir()

	targetFile := filepath.Join(tempDir, "target.txt")
	os.WriteFile(targetFile, []byte("target"), 0o644)
	validLink := filepath.Join(tempDir, "valid_link")
	err = os.Symlink(targetFile, validLink)
	if err != nil {
		t.Skip("symlink creation failed, skipping test")
		return
	}

	brokenLink := filepath.Join(tempDir, "broken_link")
	err = os.Symlink(filepath.Join(tempDir, "nonexistent.txt"), brokenLink)
	if err != nil {
		t.Skip("symlink creation failed, skipping test")
		return
	}

	err = w.Add(validLink)
	if err != nil {
		t.Errorf("expected no error adding valid symlink, got %v", err)
	}

	err = w.Add(brokenLink)
	if err != nil {
		t.Errorf("expected no error adding broken symlink, got %v", err)
	}

	warns := w.GetWarnings()
	if len(warns) == 0 {
		t.Error("expected at least one warning for broken symlink, got none")
	}

	w2, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	parentDir := filepath.Join(tempDir, "parent")
	os.Mkdir(parentDir, 0o755)

	childBrokenLink := filepath.Join(parentDir, "child_broken_link")
	os.Symlink(filepath.Join(tempDir, "nonexistent.txt"), childBrokenLink)

	err = w2.Add(parentDir)
	if err != nil {
		t.Errorf("expected no error adding parent directory with broken symlink, got %v", err)
	}

	warns2 := w2.GetWarnings()
	if len(warns2) == 0 {
		t.Error("expected warnings to contain warning about child broken symlink")
	}
}
