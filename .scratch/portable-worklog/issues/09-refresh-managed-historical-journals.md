# 09 — 安全 refresh 歷史日誌

**What to build:** 讓使用者能重新產生指定日期的 managed content，同時保證日誌中人工補充的內容與既有有效檔案不會因 refresh 或寫入失敗而遺失。

**Blocked by:** 06 — 產生不依賴 AI 的每日工作日誌。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] 使用者可以明確指定歷史日期進行 preview、首次生成或 refresh。
- [x] Generated block 包含可重現的 source digest，refresh 只替換該 managed block。
- [x] Managed block 外的人工內容在 refresh 後 byte-for-byte 保留。
- [x] 已存在但沒有合法 managed boundaries 的檔案不會被整份覆寫，command 會清楚停止。
- [x] Refresh 前顯示 diff，只有確認後才執行 atomic replacement。
- [x] 注入 replacement 前的 write failure 時，既有 journal 保持完整且沒有殘留 partial file。

## Comments

- 2026-09-07：`worklog day [YYYY-MM-DD]` 支援歷史首次生成；`--refresh` 僅接受恰好一組含 SHA-256 source digest 的合法 managed boundaries。
- 2026-09-07：refresh 顯示 managed-block diff、確認後重查原始 bytes 並以同目錄 temporary file atomic rename；unit 與 acceptance tests 覆蓋人工內容保留、無效邊界、競態與 replacement 前失敗清理。
