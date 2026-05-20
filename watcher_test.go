package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
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
	os.WriteFile(tmp, []byte("hello"), 0644)
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
	os.Mkdir(dir, 0755)
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

func TestWatcherStart_contextCancel(t *testing.T) {
	w, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	tmp := filepath.Join(t.TempDir(), "testfile.txt")
	os.WriteFile(tmp, []byte("hello"), 0644)
	w.Add(tmp)

	done := make(chan struct{})
	go func() {
		_ = w.Start(func(event fsnotify.Event) {})
		close(done)
	}()

	// Cancel after a short delay.
	time.Sleep(100 * time.Millisecond)
	w.Close()

	<-done
	// Should not panic or error on close
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

	// 1. 建立一個正常的軟連結
	targetFile := filepath.Join(tempDir, "target.txt")
	os.WriteFile(targetFile, []byte("target"), 0644)
	validLink := filepath.Join(tempDir, "valid_link")
	err = os.Symlink(targetFile, validLink)
	if err != nil {
		t.Skip("symlink creation failed (might lack privileges on Windows), skipping test")
		return
	}

	// 2. 建立一個無效的軟連結 (broken symlink)
	brokenLink := filepath.Join(tempDir, "broken_link")
	err = os.Symlink(filepath.Join(tempDir, "nonexistent.txt"), brokenLink)
	if err != nil {
		t.Skip("symlink creation failed, skipping test")
		return
	}

	// 測試：加入正常的軟連結，應能正常註冊且沒有 warning
	err = w.Add(validLink)
	if err != nil {
		t.Errorf("expected no error adding valid symlink, got %v", err)
	}

	// 測試：加入無效的軟連結，應能正常處理 (不崩潰) 且記錄 warning
	err = w.Add(brokenLink)
	if err != nil {
		t.Errorf("expected no error adding broken symlink (should gracefully skip/log), got %v", err)
	}

	warns := w.GetWarnings()
	if len(warns) == 0 {
		t.Error("expected at least one warning for broken symlink, got none")
	}

	// 3. 測試父目錄包含無效軟連結的遞迴 Add 行為
	w2, err := NewWatcher(make([]string, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	parentDir := filepath.Join(tempDir, "parent")
	os.Mkdir(parentDir, 0755)
	
	// 在父目錄下建立 broken symlink
	childBrokenLink := filepath.Join(parentDir, "child_broken_link")
	os.Symlink(filepath.Join(tempDir, "nonexistent.txt"), childBrokenLink)

	err = w2.Add(parentDir)
	if err != nil {
		t.Errorf("expected no error adding parent directory with broken symlink, got %v", err)
	}

	warns2 := w2.GetWarnings()
	found := false
	for _, wn := range warns2 {
		if filepath.Base(wn) != "" { // 只要有抓到 warnings
			found = true
		}
	}
	if !found {
		t.Error("expected warnings to contain warning about child broken symlink")
	}
}
