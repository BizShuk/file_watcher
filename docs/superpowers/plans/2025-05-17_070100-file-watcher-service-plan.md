# File Watcher Service — Implementation Plan

## Goal

Build a Go-based file watcher service that:

1. Watches configured directories/files via `inotify` (Linux) / `fsnotify` (macOS)
2. On file create/update events → collects file size statistics (unique by path per hour bucket)
3. Keeps stats in runtime memory, flushes periodically (`batch_period` in config)
4. Outputs daily summary to stdout (future: LLM analysis via message channel)
5. Stores stats files locally with **1-week retention**

---

## Design Principles

### SOLID Principles

| Principle                     | Application                                                                                                                                     |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| **Single Responsibility (S)** | Each file has one job: `config.go` → config only, `stats.go` → stats only, etc.                                                                 |
| **Open/Closed (O)**           | `Notifier` interface is open for new implementations (stdout, LLM channel, webhook). Core logic closed to modification.                         |
| **Liskov Substitution (L)**   | Any `StatsCollector`, `Notifier`, or `Watcher` implementation can substitute the interface without breaking behavior.                           |
| **Interface Segregation (I)** | Small, focused interfaces: `StatCollector` (collect/flush/prune/clear), `Notifier` (notify), `WatcherOps` (add/start/close). No fat interfaces. |
| **Dependency Inversion (D)**  | `main.go` depends on interfaces (`StatsCollector`, `Notifier`, `WatcherOps`) not concrete implementations.                                      |

### Go Development Principles

| Principle                        | Application                                                                                         |
| -------------------------------- | --------------------------------------------------------------------------------------------------- |
| **Interfaces are implicit**      | No `implements` keyword — just define and use. Interfaces live near the consumer, not the producer. |
| **Composition over inheritance** | `Scheduler` composes `StatsCollector` and `Notifier` via interfaces, not inheritance.               |
| **Concurrency safety**           | `StatsCollector` uses `sync.RWMutex` to protect the in-memory map from concurrent reads/writes.     |
| **Error handling as values**     | All errors returned as values. No exceptions. `FlushHour()` returns `error`.                        |
| **Zero value is valid**          | Use `sync.Mutex{}` (zero value = unlocked), `time.Time{}` (zero = Jan 1, year 1) where possible.    |
| **Package is small**             | Flat structure, no deep nesting. Interfaces at package level.                                       |
| **Dependency injection**         | `main.go` wires dependencies explicitly — no global state, no singletons.                           |

---

## Project Structure (Flat + SOLID)

```
file_watcher/
├── main.go                   # Entry point + DI wiring + signal handling
├── config.go                 # Config loading + validation
├── config_test.go            # Config unit tests
├── watcher.go                # fsnotify wrapper (WatcherOps interface)
├── watcher_test.go           # Watcher unit tests
├── stats.go                  # StatsCollector interface + impl
├── stats_test.go             # Stats unit tests
├── notifier.go               # Notifier interface + StdoutNotifier impl
├── scheduler.go              # Batch scheduler (composes StatsCollector + Notifier)
├── go.mod
├── go.sum
└── plans/
```

> **Flat structure** — all source files at root level.
> **SOLID** — each component has a single responsibility, depends on interfaces.

---

## Interfaces (SOLID: I + D)

Defined at package level, implicit implementation via method signatures.

```go
// StatsCollector collects and flushes file metrics.
// (Liskov: any implementation substitutable)
type StatsCollector interface {
    AddOrUpdate(path string, size int64, modTime time.Time)
    FlushHour() error
    Prune(retentionDays int) error
    Clear()
}

// Notifier outputs a stats summary.
// (Liskov: StdoutNotifier, future LLM channel, webhook all substitutable)
type Notifier interface {
    Notify(summary string) error
}

// WatcherOps watches filesystem events and reports file changes.
// (Liskov: fsnotify implementation, future inotify-kernel impl substitutable)
type WatcherOps interface {
    Add(path string) error
    Start(handler func(path string, size int64, modTime time.Time)) error
    Close() error
}
```

---

## Config Schema (`~/.config/file_watcher/settings.json`)

```json
{
    "watch_list": ["/path/to/dir1", "/path/to/file1"],
    "admin": {
        "name": "Admin Name",
        "email": "admin@example.com",
        "webhook_url": ""
    },
    "batch_period": "1h",
    "stats_retention_days": 7
}
```

| Field                  | Type       | Description                                              |
| ---------------------- | ---------- | -------------------------------------------------------- |
| `watch_list`           | `[]string` | Directories/files to watch                               |
| `admin.name`           | `string`   | Admin display name                                       |
| `admin.email`          | `string`   | Admin email                                              |
| `admin.webhook_url`    | `string`   | Future: webhook for notifications                        |
| `batch_period`         | `string`   | Duration (e.g. `"1h"`, `"30m"`, `"5m"`). Flush frequency |
| `stats_retention_days` | `int`      | Days to keep stat files (default: 7)                     |

---

## Statistics Format (Metric Style — Path-Unique, Hour-Rounded)

One entry per unique file path per hour bucket.

**Stat file**: `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json` (hour-rounded)

```json
{
    "date": "2025-05-17T07:00:00Z",
    "entries": [
        {
            "path": "/var/log/app.log",
            "size_bytes": 1048576,
            "last_modified": "2025-05-17T10:30:00Z"
        },
        {
            "path": "/data/config/settings.json",
            "size_bytes": 2048,
            "last_modified": "2025-05-17T09:15:00Z"
        }
    ]
}
```

> **Unique key**: `path` is the unique identifier per hour bucket.
> **Filename**: `YYYY-MM-DDTHH.json` rounded down to the hour (e.g. `2025-05-17T07.json`).

---

## File Breakdown with Responsibilities

| File           | Responsibility                                                                               | SOLID class |
| -------------- | -------------------------------------------------------------------------------------------- | ----------- |
| `main.go`      | Entry point, wires DI, signal handling. Orchestrates all components.                         | Composer    |
| `config.go`    | Load + validate `~/.config/file_watcher/settings.json`.                                      | SRP         |
| `watcher.go`   | `fsnotify` wrapper. Implements `WatcherOps`. Thread-safe event dispatch.                     | OCP         |
| `stats.go`     | In-memory `map[path]StatEntry`. Implements `StatsCollector`. `sync.RWMutex` for concurrency. | SRP + OCP   |
| `notifier.go`  | `Notifier` interface + `StdoutNotifier` implementation.                                      | OCP + ISP   |
| `scheduler.go` | `time.Ticker` batch scheduler. Composes `StatsCollector` + `Notifier` via interfaces.        | DIP         |

---

## Implementation Steps

### Step 1: Project Setup

```bash
mkdir -p ~/projects/file_watcher
cd ~/projects/file_watcher
go mod init file_watcher
```

- Add dependency: `github.com/fsnotify/fsnotify`
- Create `plans/` directory

### Step 2: `config.go` (SRP)

- Structs: `Settings`, `Admin`
- `Load(path string) (*Settings, error)` — reads JSON, validates required fields
- Returns `error` on missing `watch_list`, unparseable `batch_period`
- `BatchPeriodDuration() (time.Duration, error)` — parses `batch_period` string

### Step 3: `stats.go` (SRP + Concurrency)

```go
type StatEntry struct {
    Size        int64
    LastModified time.Time
}

type fsStatsCollector struct {
    mu    sync.RWMutex
    data  map[string]StatEntry  // key = path
    today time.Time
}

func NewStatsCollector() *fsStatsCollector {
    return &fsStatsCollector{data: make(map[string]StatEntry)}
}
```

- `AddOrUpdate(path, size, modTime)` — `sync.RWMutex` write
- `FlushHour() error` — write `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`, return `error`
- `Prune(retentionDays) error` — read `os.ReadDir`, delete files older than retention
- `Clear()` — reset map, update `today` to current hour

### Step 4: `notifier.go` (OCP + ISP)

```go
type Notifier interface {
    Notify(summary string) error
}

type StdoutNotifier struct{}

func (s *StdoutNotifier) Notify(summary string) error {
    fmt.Println(summary)
    return nil
}
```

- `Notifier` interface — one method, focused
- Future: `LLMChannelNotifier`, `WebhookNotifier` implement the same interface

### Step 5: `watcher.go` (OCP + Concurrency)

```go
type WatcherOps interface {
    Add(path string) error
    Start(handler func(path string, size int64, modTime time.Time)) error
    Close() error
}

type fsWatcher struct {
    watcher  *fsnotify.Watcher
    done     chan struct{}
}
```

- `Add(path)` — recursive for directories, single file for files
- `Start(handler)` — goroutine listens on `watcher.Events`, dispatches to handler
- `Close()` — close watcher, signal `done` channel

### Step 6: `scheduler.go` (DIP)

```go
type Scheduler struct {
    collector StatsCollector  // interface (DIP)
    notifier  Notifier        // interface (DIP)
    period    time.Duration
    ticker    *time.Ticker
    done      chan struct{}
}
```

- `Start()` — `time.Ticker` fires every `period`. On tick: `FlushHour()`, `Notify(summary)`, `Clear()`
- Also runs `Prune(retentionDays)` on each tick
- `FlushNow()` — for SIGTERM/SIGINT graceful flush
- Depends on interfaces, not concrete types (DIP)

### Step 7: `main.go` (Composer + DI)

```go
func main() {
    cfg, err := config.Load("~/.config/file_watcher/settings.json")
    if err != nil { log.Fatal(err) }

    collector := stats.NewStatsCollector()
    notifier := &notifier.StdoutNotifier{}
    sched := scheduler.NewScheduler(collector, notifier, cfg.BatchPeriod, cfg.RetentionDays)

    w, err := watcher.New()
    if err != nil { log.Fatal(err) }
    for _, p := range cfg.WatchList { w.Add(p) }

    handler := func(path string, size int64, modTime time.Time) {
        collector.AddOrUpdate(path, size, modTime)
    }
    go w.Start(handler)
    go sched.Start()

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    sched.FlushNow()
    w.Close()
}
```

- Explicit DI — no globals, no singletons
- Graceful shutdown: flush pending stats before exit

### Step 8: Tests

- `config_test.go`: valid/invalid JSON, missing fields, bad duration
- `stats_test.go`: AddOrUpdate, FlushHour output, Clear, concurrent AddOrUpdate
- `watcher_test.go`: mock fsnotify events, verify handler called correctly

### Step 9: Settings.json Example

```json
{
    "watch_list": ["/tmp/watch"],
    "admin": {
        "name": "Shuk",
        "email": "shuk@example.com",
        "webhook_url": ""
    },
    "batch_period": "1h",
    "stats_retention_days": 7
}
```

---

## Build & Run

```bash
cd ~/projects/file_watcher
go build -o file_watcher .
./file_watcher
```

---

## TODOs for Future Iterations

### TODO: Grafana Integration

- Add `metrics_port` to `settings.json`
- Serve Prometheus metrics endpoint (`/metrics`)
- Expose: `file_watcher_size_bytes{path="..."}`
- Add `grafana/dashboard.json` for import

### TODO: LLM Integration

- New `LLMNotifier` implementing `Notifier` interface
- Consumes `chan StatsSummary` message channel
- Sends to LLM (OpenAI/Anthropic) for structured analysis
- Config: `llm_provider`, `llm_api_key`, `llm_prompt_template` in `settings.json`
- **Open/Closed**: add LLM notifier without modifying `stats.go` or `scheduler.go`

---

## SOLID Checklist

| Component      | S         | O   | L   | I   | D   |
| -------------- | --------- | --- | --- | --- | --- |
| `config.go`    | ✅        | ✅  | ✅  | ✅  | ✅  |
| `stats.go`     | ✅        | ✅  | ✅  | ✅  | ✅  |
| `notifier.go`  | ✅        | ✅  | ✅  | ✅  | ✅  |
| `watcher.go`   | ✅        | ✅  | ✅  | ✅  | ✅  |
| `scheduler.go` | ✅        | ✅  | ✅  | ✅  | ✅  |
| `main.go`      | ✅ (orch) | ✅  | ✅  | ✅  | ✅  |

> **S**ingle Responsibility, **O**pen/Closed, **L**iskov, **I**nterface Segregation, **D**ependency Inversion

---

## Open Questions

None — all decisions captured. Ready to implement.
