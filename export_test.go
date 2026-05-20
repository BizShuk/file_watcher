package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunExport(t *testing.T) {
	// 建立暫時目錄作為測試用的 HOME
	tmpDir := t.TempDir()

	// 保存原本的 homeDirFn 並在測試結束後還原
	oldHomeDirFn := homeDirFn
	defer func() { homeDirFn = oldHomeDirFn }()

	// Mock 家目錄
	homeDirFn = func() string {
		return tmpDir
	}

	// 呼叫 runExport 進行匯出
	var buf bytes.Buffer
	err := runExport(&buf)
	if err != nil {
		t.Fatalf("runExport 執行失敗: %v", err)
	}

	// 解析輸出的 JSON
	var output Settings
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("無法解析輸出的 JSON: %v, 輸出內容: %s", err, buf.String())
	}

	// 驗證預設設定內容是否正確
	if len(output.WatchList) == 0 {
		t.Errorf("WatchList 不應為空")
	}
	if output.StatsRetentionDays != 7 {
		t.Errorf("預期的 StatsRetentionDays 為 7，得到 %d", output.StatsRetentionDays)
	}
}
