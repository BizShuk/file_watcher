# file_watcher 守護程序部署指南 (file_watcher Daemon Deployment Guide)

本指南說明如何將 `file_watcher` 作為守護程序 (`daemon`) 執行於背景，並在程式遭意外終止（如 `killed`）時自動重啟。

本專案提供了一個自動化腳本 `scripts/manage-daemon.sh`，可簡化安裝與管理流程。以下為自動管理與手動配置的詳細說明。

---

## 快速開始：自動管理

我們提供了一個自動化腳本 `scripts/manage-daemon.sh` 用於安裝、移除與管理服務。

### 1. 安裝服務
若您希望啟用 `Slack` 通知功能，請在安裝前設定環境變數：
```bash
export SLACK_BOT_TOKEN="xoxb-your-token"
export SLACK_CHANNEL_ID="Cyourchannel"
./scripts/manage-daemon.sh install
```
如果不需要 Slack 通知，直接執行：
```bash
./scripts/manage-daemon.sh install
```
此步驟將會自動：
- 執行 `go install .` 來建置並安裝 `file_watcher` 到您的 `go/bin` 目錄。
- 偵測作業系統，並在對應路徑生成服務設定檔。

### 2. 啟動服務
```bash
./scripts/manage-daemon.sh start
```

### 3. 查看狀態與日誌
```bash
./scripts/manage-daemon.sh status
```

### 4. 停止服務
```bash
./scripts/manage-daemon.sh stop
```

### 5. 移除服務與設定檔
```bash
./scripts/manage-daemon.sh uninstall
```

---

## 手動配置說明

若您不希望使用管理腳本，也可以手動建立以下配置。

### macOS (launchd)

在 macOS 上，守護程序是透過 `launchd` 管理。我們推薦將其配置為 `LaunchAgent`，運行在當前用戶的 session 中。

#### 1. 建立設定檔
建立檔案 `~/Library/LaunchAgents/com.user.file_watcher.plist`，內容如下：
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.file_watcher</string>
    <key>ProgramArguments</key>
    <array>
        <!-- 請將下方的路徑修改為您實際的 file_watcher 執行檔絕對路徑 -->
        <string>/Users/yourusername/go/bin/file_watcher</string>
    </array>
    <!-- 開機或登入時自動啟動 -->
    <key>RunAtLoad</key>
    <true/>
    <!-- 保持運行（若被 killed 會自動重啟） -->
    <key>KeepAlive</key>
    <true/>
    <!-- 日誌輸出路徑 -->
    <key>StandardOutPath</key>
    <string>/Users/yourusername/.config/file_watcher/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/yourusername/.config/file_watcher/daemon.err</string>
</dict>
</plist>
```

#### 2. 啟動服務
```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.user.file_watcher.plist
```

#### 3. 停止服務
```bash
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.user.file_watcher.plist
```

---

### Linux (systemd)

在 Linux 上，我們可以使用 `systemd` 的 `user` 級別服務。這讓您不需要 `sudo` 權限便能部署，且能繼承用戶權限。

#### 1. 建立服務檔
建立檔案 `~/.config/systemd/user/file_watcher.service`：
```ini
[Unit]
Description=File Watcher Daemon Service
After=network.target

[Service]
Type=simple
# 請將下方的路徑修改為您實際的 file_watcher 執行檔絕對路徑
ExecStart=/home/yourusername/go/bin/file_watcher
# 意外終止時自動重啟
Restart=always
RestartSec=5
WorkingDirectory=/home/yourusername
StandardOutput=append:/home/yourusername/.config/file_watcher/daemon.log
StandardError=append:/home/yourusername/.config/file_watcher/daemon.err

[Install]
WantedBy=default.target
```

#### 2. 重新載入並啟動服務
```bash
systemctl --user daemon-reload
systemctl --user enable file_watcher.service
systemctl --user start file_watcher.service
```

#### 3. 檢視服務狀態
```bash
systemctl --user status file_watcher.service
```

#### 4. 停止服務
```bash
systemctl --user stop file_watcher.service
```

#### 5. 允許持久化運行 (Linger)
預設情況下，當使用者登出時，`systemd --user` 服務會被停止。如果您希望在登出後服務依然運行，請執行：
```bash
loginctl enable-linger $USER
```
