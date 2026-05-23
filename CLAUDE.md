# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Run and Test Commands

```bash
# Build the binary
go build -o file_watcher .

# Run directly
go run main.go start               # Start the watcher
go run main.go show                # Show disk usage growth
go run main.go export              # Export configuration

# Run the binary
./file_watcher start               # Start the watcher
./file_watcher show                # Show disk usage growth
./file_watcher export              # Export configuration

# Run tests
go test ./...                      # Run all tests
go test -v ./...                   # Run all tests with verbose output
go test -run TestWatcherAdd_file  # Run a single test
go test -race ./...                # Run tests with race detector
```


## Architecture

The project is a file watcher that monitors directories for changes and periodically flushes file stats to disk.

### Core Components

```
main.go              # Entry point, CLI commands setup
runner/runner.go     # Wires components (DI entry point) and drives execution loop
config/config.go     # Settings schema and validation
watcher/watcher.go   # Scans/watches directories
stats/collector.go   # Thread-safe stats collection and serialization
warning/sink.go      # Thread-safe warning collection
show/                # Disk usage growth computation and rendering
gosdk/notify         # External dependency for notification interface and notifiers
```

### Key Interfaces (DIP via ISP)

- `stats.Recorder` and `stats.Flusher` (implemented by `stats.Collector`)
- `notify.Notifier` (imported from `gosdk/notify`, implemented by `StdoutNotifier`, `SlackNotifier`, `Multi`)
- `watcher.Watcher`

### Data Flow

1. `runner.Run()` starts the scheduler.
2. Scheduler runs jobs:
   - `scan` job calls `watcher.Scan()` to check files.
   - `flush` job writes stats using `stats.Flusher.FlushHour()` and prunes old files.
3. Upon shutdown, a final flush drains warnings and flushes stats.
4. `notify.Notifier.Notify()` delivers the final report.

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