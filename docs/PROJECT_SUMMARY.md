# 📁 File Watcher — Project Summary

**Date:** 2026-05-17
**Repo:** `~/projects/file_watcher`

---

## 🔧 Architecture (SOLID)

| File | Responsibility |
|------|---------------|
| `main.go` | Entry point, wires DI, graceful shutdown |
| `config.go` | Loads config via `utils.LoadOrCreate`, validation |
| `settings.default.json` | Embedded default (`//go:embed`) |
| `stats.go` | In-memory map + RWMutex, hourly flush to JSON |
| `watcher.go` | fsnotify wrapper, recursive watch |
| `scheduler.go` | time.Ticker batch scheduler |
| `notifier.go` | Notifier interface + `StdoutNotifier` |
| `utils/config.go` | Reusable `LoadOrCreate(path, defaultJSON, out)` SDK |

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

**Config path:** `~/.config/file_watcher/settings.json`

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
- [x] Module: `github.com/shuk/file_watcher`
- [x] SOLID design, 10 unit tests passing

---

## 📌 TODO

### 1. Grafana Integration
- Prometheus metrics endpoint (`/metrics`)
- `grafana/dashboard.json`

### 2. LLM Integration
- Replace `StdoutNotifier` with message channel → LLM analysis
- Detect anomalies, structured insights

---

## ✅ Build & Test

```
go build -buildvcs=false .   ✅ BUILD OK
go test ./...                ok   github.com/shuk/file_watcher   0.570s
```

---

## 🧪 Run

```bash
cd ~/projects/file_watcher && ./file_watcher
```