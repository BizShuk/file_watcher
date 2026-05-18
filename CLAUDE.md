# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
go build -o file_watcher .        # Build the binary
go test ./...                       # Run all tests
go test -v ./...                    # Run all tests with verbose output
go test -run TestWatcherAdd_file   # Run a single test
go test -race ./...                 # Run tests with race detector
```

## Architecture

The project is a file watcher that monitors directories for changes and periodically flushes file stats to disk.

### Core Components

```
main.go          # Entry point, wires together Watcher → StatsCollector → Scheduler → Notifier
watcher.go       # Wraps fsnotify.Watcher, recursively watches directories, filters exclude list
stats.go         # StatsCollector interface + fsStatsCollector implementation (thread-safe)
scheduler.go     # Periodically flushes stats (via FlushHour) and prunes old files
notifier.go      # Notifier interface with StdoutNotifier implementation
config.go        # Settings struct + Load() from ~/.config/file_watcher/settings.json
utils/config.go  # LoadOrCreate() helper for JSON config files
```

### Key Interfaces (DIP via ISP)

- `StatsCollector`: `AddOrUpdate`, `Remove`, `FlushHour`, `Prune`, `Clear`
- `Notifier`: `Notify(summary string) error`
- `WatcherOps`: `Add`, `Start`, `Close`

### Data Flow

1. `fsWatcher.Start()` listens to fsnotify events and calls the handler
2. Handler calls `collector.AddOrUpdate(path, size, modTime)` for each file event
3. `Scheduler.run()` ticks every `batchPeriod` and calls `flush()`
4. `flush()` writes stats to `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json` then prunes old files
5. `Notifier.Notify()` delivers a summary (stdout for now)

### Configuration

Config path: `~/.config/file_watcher/settings.json` (auto-created from embedded `settings.default.json`)

```json
{
  "watch_list": ["/tmp"],
  "exclude_list": [".git"],
  "batch_period": "1h",
  "stats_retention_days": 7
}
```

Stats stored in: `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`

### Thread Safety

`fsStatsCollector` uses `sync.RWMutex` to protect its map. Stats files are written atomically (no locking needed).