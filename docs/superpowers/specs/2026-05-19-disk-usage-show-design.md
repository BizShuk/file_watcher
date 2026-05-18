# Disk Usage Show Command Design

## Overview

Add a `show` subcommand to the file_watcher binary that displays disk usage growth as a horizontal bar chart, comparing the initial recorded size vs. the latest recorded size for each monitored file path.

## Subcommand Integration

- **Invocation**: `./file_watcher show`
- **Arguments**: None (self-contained)
- **Configuration**: Uses existing `~/.config/file_watcher/settings.json` and `~/.config/file_watcher/stats/` directory

## Behavior

1. Read all JSON stat files from the stats directory (`~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`)
2. For each unique file path across all stat files:
   - Find the **earliest** record (initial size) and **latest** record (current size)
   - Compute: `size_change = latest_size - initial_size`
   - Compute: `growth_pct = (size_change / initial_size) * 100` (if initial_size > 0)
3. Sort by absolute `size_change` descending
4. Display Top 20 entries as a horizontal bar chart
5. Show both absolute growth (formatted bytes) and percentage

## Output Format

```
磁碟使用量增長報告（初始 vs 最新）
================================================================================

/path/to/file1      ████████████████████████████  50.5MB  (+125%)
/path/to/file2      █████████████████████████     30.2MB  (+80%)
...

Legend: bar = growth amount, parentheses = growth percentage
```

### Bar Chart Details

- Bar width is fixed (60 characters), normalized to the maximum growth value
- Bytes formatted as: `B`, `KB`, `MB`, `GB` (auto-scale)
- New files (no initial record): display as `NEW` with `NEW` label
- Zero-growth files: display as `-` with `(0%)`

## Error Handling

- Empty stats directory: display "目前沒有任何統計資料" and exit gracefully
- Single record only: growth shown as `0` or `NEW` if only latest exists
- Read errors: skip the unreadable file, continue processing others
- Files with initial_size == 0: show absolute size as growth

## Implementation

### Files to Modify

- `main.go`: Add CLI argument parsing for `show` subcommand

### New Files

- `cmd/show.go`: `show` command implementation
  - `Run()` function to execute the report logic
  - `readAllStats()` to load and aggregate stat files
  - `computeGrowth()` to calculate size changes
  - `formatBytes()` to convert bytes to human-readable format
  - `printBarChart()` to render the horizontal bar chart

### Key Functions

```go
// cmd/show.go
func Run(args []string) error
func readAllStats(statsDir string) (map[string][]StatFileEntry, error)
func computeGrowth(entries map[string][]StatFileEntry) []GrowthEntry
func formatBytes(bytes int64) string
func printBarChart(entries []GrowthEntry, maxWidth int)
```

## Testing

- Unit tests for `formatBytes()` with edge cases (0, 1KB, 1MB, 1GB)
- Unit tests for `computeGrowth()` with mocked stat data
- Integration test verifying `show` command output format
