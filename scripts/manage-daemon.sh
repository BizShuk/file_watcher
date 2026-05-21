#!/bin/bash

# file_watcher 守護程序管理器 (file_watcher Daemon Manager)
# 支援 macOS (launchd) 與 Linux (systemd --user)

set -e

ulimit -n 65536

# 定義變數
OS="$(uname -s)"
BINARY_NAME="file_watcher"
PLIST_LABEL="com.user.file_watcher"
PLIST_PATH="$HOME/Library/LaunchAgents/${PLIST_LABEL}.plist"
SYSTEMD_DIR="$HOME/.config/systemd/user"
SYSTEMD_PATH="${SYSTEMD_DIR}/${BINARY_NAME}.service"
CONFIG_DIR="$HOME/.config/file_watcher"
LOG_OUT="${CONFIG_DIR}/daemon.log"
LOG_ERR="${CONFIG_DIR}/daemon.err"

# 確保設定目錄存在
mkdir -p "${CONFIG_DIR}"

# 取得二進位檔路徑
get_binary_path() {
    local bin_path
    # 嘗試從 go env 尋找 GOBIN 與 GOPATH
    local go_bin
    go_bin=$(go env GOBIN 2>/dev/null || echo "")
    local go_path
    go_path=$(go env GOPATH 2>/dev/null || echo "")

    if [ -n "${go_bin}" ]; then
        bin_path="${go_bin}/${BINARY_NAME}"
    elif [ -n "${go_path}" ]; then
        # GOPATH 可能包含多個路徑（以冒號分隔），取第一個
        local first_gopath
        first_gopath=$(echo "${go_path}" | cut -d':' -f1)
        bin_path="${first_gopath}/bin/${BINARY_NAME}"
    else
        bin_path="$HOME/go/bin/${BINARY_NAME}"
    fi

    # 檢查二進位檔是否存在，若不存在則執行安裝
    if [ ! -f "${bin_path}" ]; then
        echo "未在 ${bin_path} 找到二進位檔，正在進行 go install..."
        go install .
    fi

    # 再次確認
    if [ ! -f "${bin_path}" ]; then
        echo "錯誤: 無法安裝或找到 ${BINARY_NAME} 二進位檔" >&2
        exit 1
    fi

    echo "${bin_path}"
}

# 安裝 macOS daemon (launchd)
install_mac() {
    local bin_path
    bin_path=$(get_binary_path)
    echo "正在為 macOS 配置 launchd..."

    # 準備環境變數 plist 區段
    local env_section=""
    if [ -n "${SLACK_BOT_TOKEN}" ] && [ -n "${SLACK_CHANNEL_ID}" ]; then
        env_section="<key>EnvironmentVariables</key>
        <dict>
            <key>SLACK_BOT_TOKEN</key>
            <string>${SLACK_BOT_TOKEN}</string>
            <key>SLACK_CHANNEL_ID</key>
            <string>${SLACK_CHANNEL_ID}</string>
        </dict>"
    fi

    cat <<EOF > "${PLIST_PATH}"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${bin_path}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_OUT}</string>
    <key>StandardErrorPath</key>
    <string>${LOG_ERR}</string>
    ${env_section}
</dict>
</plist>
EOF

    chmod 644 "${PLIST_PATH}"
    echo "已成功生成 LaunchAgent 設定檔: ${PLIST_PATH}"
    echo "執行 './manage-daemon.sh start' 以啟動服務"
}

# 移除 macOS daemon
uninstall_mac() {
    echo "正在移除 macOS launchd 設定..."
    if launchctl list | grep -q "${PLIST_LABEL}"; then
        echo "正在停止服務..."
        launchctl bootout gui/$(id -u) "${PLIST_PATH}" 2>/dev/null || launchctl unload "${PLIST_PATH}" 2>/dev/null || true
    fi
    if [ -f "${PLIST_PATH}" ]; then
        rm -f "${PLIST_PATH}"
        echo "已移除 ${PLIST_PATH}"
    fi
    echo "解除安裝完成"
}

# 啟動 macOS daemon
start_mac() {
    if [ ! -f "${PLIST_PATH}" ]; then
        echo "錯誤: 尚未安裝服務，請先執行 './manage-daemon.sh install'" >&2
        exit 1
    fi
    echo "正在載入並啟動服務..."
    launchctl bootstrap gui/$(id -u) "${PLIST_PATH}" 2>/dev/null || launchctl load "${PLIST_PATH}"
    echo "服務已啟動"
}

# 停止 macOS daemon
stop_mac() {
    if [ ! -f "${PLIST_PATH}" ]; then
        echo "錯誤: 服務檔不存在" >&2
        exit 1
    fi
    echo "正在停止並解除載入服務..."
    launchctl bootout gui/$(id -u) "${PLIST_PATH}" 2>/dev/null || launchctl unload "${PLIST_PATH}"
    echo "服務已停止"
}

# 檢視 macOS daemon 狀態
status_mac() {
    echo "=== 系統服務狀態 (System Service Status) ==="
    if launchctl list | grep -q "${PLIST_LABEL}"; then
        echo "服務狀態: 運行中 (Running)"
        launchctl list | grep "${PLIST_LABEL}"
    else
        echo "服務狀態: 未運行 (Not Running)"
    fi
    echo "=== 最近的日誌 (Recent Logs) ==="
    if [ -f "${LOG_OUT}" ]; then
        echo "[stdout] (最後 10 行):"
        tail -n 10 "${LOG_OUT}"
    fi
    if [ -f "${LOG_ERR}" ]; then
        echo "[stderr] (最後 10 行):"
        tail -n 10 "${LOG_ERR}"
    fi
}

# 安裝 Linux daemon (systemd)
install_linux() {
    local bin_path
    bin_path=$(get_binary_path)
    echo "正在為 Linux 配置 systemd user service..."

    mkdir -p "${SYSTEMD_DIR}"

    # 準備環境變數 systemd 格式
    local env_section=""
    if [ -n "${SLACK_BOT_TOKEN}" ] && [ -n "${SLACK_CHANNEL_ID}" ]; then
        env_section="Environment=\"SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}\" \"SLACK_CHANNEL_ID=${SLACK_CHANNEL_ID}\""
    fi

    cat <<EOF > "${SYSTEMD_PATH}"
[Unit]
Description=File Watcher Daemon Service
After=network.target

[Service]
Type=simple
ExecStart=${bin_path}
Restart=always
RestartSec=5
WorkingDirectory=$HOME
StandardOutput=append:${LOG_OUT}
StandardError=append:${LOG_ERR}
${env_section}

[Install]
WantedBy=default.target
EOF

    chmod 644 "${SYSTEMD_PATH}"
    systemctl --user daemon-reload
    echo "已成功生成 systemd 設定檔: ${SYSTEMD_PATH}"
    echo "執行 './manage-daemon.sh start' 以啟動服務"
}

# 移除 Linux daemon
uninstall_linux() {
    echo "正在移除 Linux systemd 設定..."
    if systemctl --user is-active --quiet "${BINARY_NAME}"; then
        echo "正在停止服務..."
        systemctl --user stop "${BINARY_NAME}"
    fi
    if systemctl --user is-enabled --quiet "${BINARY_NAME}" 2>/dev/null; then
        systemctl --user disable "${BINARY_NAME}"
    fi
    if [ -f "${SYSTEMD_PATH}" ]; then
        rm -f "${SYSTEMD_PATH}"
        echo "已移除 ${SYSTEMD_PATH}"
        systemctl --user daemon-reload
    fi
    echo "解除安裝完成"
}

# 啟動 Linux daemon
start_linux() {
    if [ ! -f "${SYSTEMD_PATH}" ]; then
        echo "錯誤: 尚未安裝服務，請先執行 './manage-daemon.sh install'" >&2
        exit 1
    fi
    echo "正在載入並啟用服務..."
    systemctl --user enable "${BINARY_NAME}"
    systemctl --user start "${BINARY_NAME}"
    echo "服務已啟動"
}

# 停止 Linux daemon
stop_linux() {
    echo "正在停止服務..."
    systemctl --user stop "${BINARY_NAME}"
    systemctl --user disable "${BINARY_NAME}" 2>/dev/null || true
    echo "服務已停止"
}

# 檢視 Linux daemon 狀態
status_linux() {
    echo "=== 系統服務狀態 (System Service Status) ==="
    systemctl --user status "${BINARY_NAME}" || true
    echo "=== 最近的日誌 (Recent Logs) ==="
    if [ -f "${LOG_OUT}" ]; then
        echo "[stdout] (最後 10 行):"
        tail -n 10 "${LOG_OUT}"
    fi
    if [ -f "${LOG_ERR}" ]; then
        echo "[stderr] (最後 10 行):"
        tail -n 10 "${LOG_ERR}"
    fi
}

# 主控制流程
case "$1" in
    install)
        if [ "${OS}" = "Darwin" ]; then
            install_mac
        else
            install_linux
        fi
        ;;
    uninstall)
        if [ "${OS}" = "Darwin" ]; then
            uninstall_mac
        else
            uninstall_linux
        fi
        ;;
    start)
        if [ "${OS}" = "Darwin" ]; then
            start_mac
        else
            start_linux
        fi
        ;;
    stop)
        if [ "${OS}" = "Darwin" ]; then
            stop_mac
        else
            stop_linux
        fi
        ;;
    status)
        if [ "${OS}" = "Darwin" ]; then
            status_mac
        else
            status_linux
        fi
        ;;
    *)
        echo "使用方式: $0 {install|uninstall|start|stop|status}"
        exit 1
        ;;
esac
