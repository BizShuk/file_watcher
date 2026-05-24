# 📁 File Watcher — Project Summary

`Date:` 2026-05-23
`Repo:` `~/projects/file_watcher`

---

## 🔧 Architecture (SOLID)

| File / Package | Responsibility |
|----------------|----------------|
| `main.go` | Entry point, CLI command invocation |
| `cmd/` | Cobra subcommands (`RootCmd`, `StartCmd`, `ExportCmd`, `ShowCmd`) |
| `config/config.go` | Loads config via `sdkutils.CreateIfNotExist`, validation |
| `settings.default.json` | Embedded default (`//go:embed`) |
| `svc/` | Integrated services: watcher, collector (stats), sink (warning), show (growth chart) |
| `handler/` | Application runner and scheduler orchestration (life-cycle driver) |

---

## 📝 Config Format

```json
{
  "watch_list": ["/tmp"],
  "admin": { "name": "admin", "email": "admin@localhost", "webhook_url": "" },
  "batch_period": "1h",
  "stats_retention_days": 7
}
```

`Config path:` `~/.config/file_watcher/settings.json`

---

## 📋 Changelog

### 2026-05-17 — v0.1

- [x] File watcher with fsnotify recursive watch
- [x] Stats collector with in-memory map + RWMutex
- [x] Hourly batch flush to `~/.config/file_watcher/stats/YYYY-MM-DDTHH.json`
- [x] 7-day retention auto-prune
- [x] `StdoutNotifier`
- [x] Graceful SIGINT/SIGTERM shutdown with `FlushNow`
- [x] `settings.default.json` embedded via `//go:embed`
- [x] Auto-create config from default if missing
- [x] Extracted `utils/config.go` as reusable SDK
- [x] Module: `github.com/bizshuk/file_watcher`
- [x] SOLID design, 10 unit tests passing

### 2026-05-23 — Package Refactoring

- [x] Merged `show`, `stats`, `warning`, `watcher` packages into a single `svc` package to eliminate circular import risks and consolidate services
- [x] Moved `runner` package into `handler` package to clarify life-cycle routing

---

## 📌 TODO

### 1. Grafana Integration
- Prometheus metrics endpoint (`/metrics`)
- `grafana/dashboard.json`

### 2. LLM Integration
- Replace `StdoutNotifier` with message channel -> LLM analysis
- Detect anomalies, structured insights

---

## ✅ Build & Test

```
go build -buildvcs=false .   ✅ BUILD OK
go test ./...                ok   github.com/bizshuk/file_watcher   0.523s
```

---

## 🧪 Run

```bash
cd ~/projects/file_watcher && ./file_watcher
```