# 設定匯出子命令實作計畫 (Settings Exporter Command Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在檔案監控服務 (file watcher) 中實作一個 `export` 子命令 (subcommand)，藉此讀取 `settings.json` 並以格式化後的 `JSON` 輸出至標準輸出 (stdout)。

**Architecture:** 新增 `export.go` 實作核心匯出邏輯 `runExport(w io.Writer) error`，以利單元測試 (unit test)；在 `main.go` 中解析命令列參數，當偵測到 `export` 參數時，將 `os.Stdout` 作為參數傳入 `runExport` 執行匯出。

**Tech Stack:** Go 1.20+, 標準庫 (`encoding/json`, `io`, `os`), `github.com/stretchr/testify/assert` (若有使用，或直接使用標準庫進行 assert)。

---

### Task 1: 實作設定匯出核心邏輯與單元測試 (Implement core export logic with TDD)

**Files:**
- Create: `export.go`
- Create: `export_test.go`

- [ ] **Step 1: 建立 export.go 並定義空函式**

建立 `export.go`，宣告空的 `runExport` 函式，回傳 `nil`。

```go
package main

import (
	"io"
)

// runExport 讀取設定檔並將內容格式化為 JSON 寫入 io.Writer
func runExport(w io.Writer) error {
	return nil
}
```

- [ ] **Step 2: 撰寫 export_test.go 中的測試案例**

建立 `export_test.go`，測試 `runExport` 函式在 mock 暫時家目錄的情況下，是否能將預設設定以正確的 JSON 格式寫入 buffer。

```go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunExport(t *testing.T) {
	// 建立暫時目錄作為測試用的 HOME
	tmpDir := t.TempDir()

	// 保存原本的 homeDirFn 並在測試結束後還原
	oldHomeDirFn := homeDirFn
	defer func() { homeDirFn = oldHomeDirFn }()

	// Mock 家目錄
	homeDirFn = func() string {
		return tmpDir
	}

	// 呼叫 runExport 進行匯出
	var buf bytes.Buffer
	err := runExport(&buf)
	if err != nil {
		t.Fatalf("runExport 執行失敗: %v", err)
	}

	// 解析輸出的 JSON
	var output Settings
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("無法解析輸出的 JSON: %v, 輸出內容: %s", err, buf.String())
	}

	// 驗證預設設定內容是否正確
	if len(output.WatchList) == 0 {
		t.Errorf("WatchList 不應為空")
	}
	if output.StatsRetentionDays != 7 {
		t.Errorf("預期的 StatsRetentionDays 為 7，得到 %d", output.StatsRetentionDays)
	}
}
```

- [ ] **Step 3: 執行測試以驗證其失敗**

在 `file_watcher` 專案目錄下執行測試命令：
Run: `go test -v -run TestRunExport`
Expected: FAIL 或者是 `t.Errorf("無法解析輸出的 JSON: ...")`，因為目前的 `runExport` 沒有輸出任何內容，測試應在此步驟失敗。

- [ ] **Step 4: 實作 runExport 的完整功能**

修改 `export.go`，實作讀取、格式化與輸出邏輯：

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// runExport 讀取設定檔並將內容格式化為 JSON 寫入 io.Writer
func runExport(w io.Writer) error {
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("載入設定檔失敗: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("格式化 JSON 失敗: %w", err)
	}

	_, err = fmt.Fprintln(w, string(data))
	if err != nil {
		return fmt.Errorf("寫入輸出失敗: %w", err)
	}

	return nil
}
```

- [ ] **Step 5: 再次執行測試以驗證其成功**

Run: `go test -v -run TestRunExport`
Expected: PASS

- [ ] **Step 6: Git 提交 Task 1 變更**

Run:
```bash
git add export.go export_test.go
git commit -m "feat: implement runExport function and its unit test"
```

---

### Task 2: 在 main.go 中整合 export 子命令 (Integrate export subcommand in main.go)

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 修改 main.go 以處理 export 參數**

在 `main.go` 的 `main` 函式中，新增對 `export` 參數的支援，如果第一個參數是 `export`，則呼叫 `runExport(os.Stdout)`，並在出錯時寫入 `os.Stderr` 並以 code `1` 退出。

修改 `main.go` 如下：

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
)

// runShow executes the show subcommand.
func runShow() error {
	return ShowCmd()
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "show":
			if err := runShow(); err != nil {
				fmt.Fprintf(os.Stderr, "show: %v\n", err)
				os.Exit(1)
			}
			return
		case "export":
			if err := runExport(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "export: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	period, err := cfg.BatchPeriodDuration()
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse batch period: %v\n", err)
		os.Exit(1)
	}

	watcher, err := NewWatcher(cfg.ExcludeList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create watcher: %v\n", err)
		os.Exit(1)
	}
	for _, p := range cfg.WatchList {
		log.Info("add path to watcher", "path", p)
		if err := watcher.Add(p); err != nil {
			fmt.Fprintf(os.Stderr, "add watch path %q: %v\n", p, err)
			os.Exit(1)
		}
	}

	collector := NewStatsCollector()
	handler := func(event fsnotify.Event) {
		var path string = event.Name
		var size int64 = 0
		var modTime int64 = time.Now().Unix()
		fileInfo, err := os.Stat(event.Name)
		if err == nil {
			size = fileInfo.Size()
			modTime = fileInfo.ModTime().Unix()
		}

		if event.Has(fsnotify.Remove) {
			collector.Remove(path)
			return
		}
		collector.AddOrUpdate(path, size, time.Unix(modTime, 0))
	}

	if err := watcher.Start(handler); err != nil {
		fmt.Fprintf(os.Stderr, "start watcher: %v\n", err)
		os.Exit(1)
	}

	notifier := &StdoutNotifier{}
	sched := NewScheduler(collector, notifier, period, cfg.StatsRetentionDays)
	if err := sched.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start scheduler: %v\n", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	sched.FlushNow()
	watcher.Close()
}
```

- [ ] **Step 2: 編譯並測試二進位檔**

執行建置命令以確認編譯無誤：
Run: `go build -o file_watcher .`
Expected: 編譯成功，生成二進位檔 `file_watcher`。

- [ ] **Step 3: 執行 export 命令進行手動驗證**

Run: `./file_watcher export`
Expected: 正確輸出 `JSON` 格式的設定檔內容（如果家目錄沒有設定檔，會先自動建立一個預設設定檔）。

- [ ] **Step 4: 執行所有專案測試確保沒有破壞現有機制**

Run: `go test -v ./...`
Expected: PASS 所有測試。

- [ ] **Step 5: Git 提交 Task 2 變更**

Run:
```bash
git add main.go
git commit -m "feat: integrate export subcommand into main entry point"
```
