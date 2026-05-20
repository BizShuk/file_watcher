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
