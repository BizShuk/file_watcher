# Disk Usage Show Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `show` subcommand to file_watcher that displays disk usage growth with a bar chart comparing initial vs. latest recorded sizes.

**Architecture:** Modify `main.go` to add CLI argument parsing for `show` subcommand. Create `cmd/show.go` with report logic that reads all stat JSON files, computes growth per file, and renders a horizontal bar chart.

**Tech Stack:** Go standard library only (no external dependencies beyond existing ones)

---

## File Structure

```
file_watcher/
├── main.go           # Modify: add CLI arg parsing, dispatch show command
├── cmd/
│   └── show.go       # Create: show command implementation
├── stats.go          # No changes (StatFileEntry already defined)
└── stats_test.go     # No changes
```

---

## Task 1: Modify main.go to add CLI argument parsing

**Files:**
- Modify: `main.go:1-77`

- [ ] **Step 1: Read current main.go**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
)

func main() {
	cfg, err := Load()
	// ... rest of file
```

- [ ] **Step 2: Add CLI argument parsing**

Replace the main function to check for `show` subcommand:

```go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "show" {
		if err := runShow(); err != nil {
			fmt.Fprintf(os.Stderr, "show: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// existing main logic follows
	cfg, err := Load()
	// ... rest unchanged
}
```

- [ ] **Step 3: Add runShow function call**

After existing imports, add:

```go
// runShow executes the show subcommand.
func runShow() error {
	return ShowCmd()
}
```

- [ ] **Step 4: Run tests to verify**

```bash
go build -o file_watcher .
./file_watcher show
```

Expected: Should run without error (may show empty stats message)

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: add show subcommand scaffold

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 2: Create cmd/show.go with show command implementation

**Files:**
- Create: `cmd/show.go`

- [ ] **Step 1: Write the ShowCmd function**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ShowCmd runs the show subcommand to display disk usage growth.
func ShowCmd() error {
	statsDir := defaultStatsDir()

	entries, err := readAllStats(statsDir)
	if err != nil {
		return fmt.Errorf("read stats: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("目前沒有任何統計資料")
		return nil
	}

	growth := computeGrowth(entries)
	if len(growth) == 0 {
		fmt.Println("無法計算增長資料")
		return nil
	}

	printBarChart(growth)
	return nil
}

// GrowthEntry holds the computed growth for a file path.
type GrowthEntry struct {
	Path         string
	InitialSize  int64
	LatestSize   int64
	SizeChange   int64
	GrowthPct    float64
	IsNew        bool
}

// readAllStats reads all stat files and returns a map of path -> sorted entries by time.
func readAllStats(statsDir string) (map[string][]StatFileEntry, error) {
	result := make(map[string][]StatFileEntry)

	patterns := []string{statsDir + "/*.json"}
	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue // skip unreadable files
		}

		var statFile StatFile
		if err := json.Unmarshal(data, &statFile); err != nil {
			continue
		}

		for _, entry := range statFile.Entries {
			result[entry.Path] = append(result[entry.Path], entry)
		}
	}

	// Sort each path's entries by LastModified time
	for path, entries := range result {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].LastModified.Before(entries[j].LastModified)
		})
		result[path] = entries
	}

	return result, nil
}

// computeGrowth calculates size change from initial to latest for each path.
func computeGrowth(entries map[string][]StatFileEntry) []GrowthEntry {
	var growth []GrowthEntry

	for path, pathEntries := range entries {
		if len(pathEntries) < 1 {
			continue
		}

		initial := pathEntries[0]
		latest := pathEntries[len(pathEntries)-1]

		initialSize := initial.Size
		latestSize := latest.Size
		sizeChange := latestSize - initialSize

		isNew := len(pathEntries) == 1 && initialSize > 0

		var growthPct float64
		if initialSize > 0 {
			growthPct = float64(sizeChange) / float64(initialSize) * 100
		}

		growth = append(growth, GrowthEntry{
			Path:        path,
			InitialSize: initialSize,
			LatestSize:  latestSize,
			SizeChange:  sizeChange,
			GrowthPct:   growthPct,
			IsNew:       isNew,
		})
	}

	// Sort by absolute size change descending
	sort.Slice(growth, func(i, j int) bool {
		return growth[i].SizeChange > growth[j].SizeChange
	})

	// Limit to top 20
	if len(growth) > 20 {
		growth = growth[:20]
	}

	return growth
}

// formatBytes converts bytes to human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}

// printBarChart renders the growth entries as a horizontal bar chart.
func printBarChart(entries []GrowthEntry) {
	fmt.Println("磁碟使用量增長報告（初始 vs 最新）")
	fmt.Println("================================================================================")
	fmt.Println()

	const maxWidth = 60
	var maxChange int64
	for _, e := range entries {
		if e.SizeChange > maxChange {
			maxChange = e.SizeChange
		}
	}

	for _, e := range entries {
		var barLen int
		if maxChange > 0 {
			barLen = int(float64(e.SizeChange) / float64(maxChange) * float64(maxWidth))
		}
		bar := strings.Repeat("█", barLen)

		sizeStr := formatBytes(e.SizeChange)
		if e.IsNew {
			sizeStr = formatBytes(e.LatestSize) + " (NEW)"
		} else if e.SizeChange == 0 {
			sizeStr = "- (0%)"
		} else {
			sizeStr = fmt.Sprintf("%s (+%.0f%%)", sizeStr, e.GrowthPct)
		}

		// Truncate path if too long
		path := e.Path
		if len(path) > 50 {
			path = "..." + path[len(path)-47:]
		}

		fmt.Printf("%-50s %s  %s\n", path, bar, sizeStr)
	}

	fmt.Println()
	fmt.Println("Legend: bar = growth amount, parentheses = growth percentage")
}
```

- [ ] **Step 2: Run build to verify**

```bash
go build -o file_watcher .
```

Expected: Should compile without errors

- [ ] **Step 3: Test with sample data**

Create test stat files:

```bash
mkdir -p ~/.config/file_watcher/stats
cat > /tmp/test_stats.json << 'EOF'
{
  "date": "2026-05-18T10:00:00Z",
  "entries": [
    {"path": "/tmp/test1.txt", "size_bytes": 1000, "last_modified": "2026-05-18T10:00:00Z"},
    {"path": "/tmp/test2.txt", "size_bytes": 5000, "last_modified": "2026-05-18T10:00:00Z"}
  ]
}
EOF
```

Run show:

```bash
./file_watcher show
```

Expected: Should display "目前沒有任何統計資料" since files are not in stats dir

- [ ] **Step 4: Commit**

```bash
git add cmd/show.go main.go
git commit -m "feat: implement show command with bar chart display

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 3: Add unit tests for show command

**Files:**
- Create: `cmd/show_test.go`

- [ ] **Step 1: Write tests for formatBytes**

```go
package main

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{1073741824, "1.0GB"},
		{1610612736, "1.5GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestComputeGrowth(t *testing.T) {
	entries := map[string][]StatFileEntry{
		"/tmp/test1.txt": {
			{Path: "/tmp/test1.txt", Size: 1000, LastModified: parseTime("2026-05-18T10:00:00Z")},
			{Path: "/tmp/test1.txt", Size: 2000, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
		"/tmp/test2.txt": {
			{Path: "/tmp/test2.txt", Size: 5000, LastModified: parseTime("2026-05-18T10:00:00Z")},
			{Path: "/tmp/test2.txt", Size: 4500, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
	}

	growth := computeGrowth(entries)

	if len(growth) != 2 {
		t.Errorf("expected 2 entries, got %d", len(growth))
	}

	// First entry should be test1.txt (1000 growth)
	if growth[0].Path != "/tmp/test1.txt" {
		t.Errorf("expected /tmp/test1.txt, got %s", growth[0].Path)
	}
	if growth[0].SizeChange != 1000 {
		t.Errorf("expected SizeChange=1000, got %d", growth[0].SizeChange)
	}

	// Second entry should be test2.txt (-500 growth)
	if growth[1].Path != "/tmp/test2.txt" {
		t.Errorf("expected /tmp/test2.txt, got %s", growth[1].Path)
	}
	if growth[1].SizeChange != -500 {
		t.Errorf("expected SizeChange=-500, got %d", growth[1].SizeChange)
	}
}

func TestComputeGrowthNewFile(t *testing.T) {
	entries := map[string][]StatFileEntry{
		"/tmp/new.txt": {
			{Path: "/tmp/new.txt", Size: 5000, LastModified: parseTime("2026-05-18T11:00:00Z")},
		},
	}

	growth := computeGrowth(entries)

	if len(growth) != 1 {
		t.Errorf("expected 1 entry, got %d", len(growth))
	}
	if !growth[0].IsNew {
		t.Errorf("expected IsNew=true, got %v", growth[0].IsNew)
	}
}
```

- [ ] **Step 2: Add parseTime helper for tests**

Add to `cmd/show.go`:

```go
import "time"

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./... -v -run "TestFormatBytes|TestComputeGrowth"
```

Expected: All tests pass

- [ ] **Step 4: Run all tests with race detector**

```bash
go test -race ./...
```

Expected: All tests pass with no race conditions

- [ ] **Step 5: Commit**

```bash
git add cmd/show_test.go cmd/show.go
git commit -m "test: add unit tests for show command

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Self-Review Checklist

- [ ] Spec coverage: All requirements in spec have corresponding implementation
- [ ] No placeholders: All steps have complete code
- [ ] Type consistency: StatFileEntry, GrowthEntry, formatBytes all consistent
- [ ] Tests: formatBytes and computeGrowth have tests
- [ ] Error handling: readAllStats skips unreadable files, empty stats handled