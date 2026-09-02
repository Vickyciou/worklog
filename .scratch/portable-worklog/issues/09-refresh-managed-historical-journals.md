# 09 — 安全 refresh 歷史日誌

**What to build:** 讓使用者能重新產生指定日期的 managed content，同時保證日誌中人工補充的內容與既有有效檔案不會因 refresh 或寫入失敗而遺失。

**Blocked by:** 06 — 產生不依賴 AI 的每日工作日誌。

**Status:** ready-for-agent

- [ ] 使用者可以明確指定歷史日期進行 preview、首次生成或 refresh。
- [ ] Generated block 包含可重現的 source digest，refresh 只替換該 managed block。
- [ ] Managed block 外的人工內容在 refresh 後 byte-for-byte 保留。
- [ ] 已存在但沒有合法 managed boundaries 的檔案不會被整份覆寫，command 會清楚停止。
- [ ] Refresh 前顯示 diff，只有確認後才執行 atomic replacement。
- [ ] 注入 replacement 前的 write failure 時，既有 journal 保持完整且沒有殘留 partial file。
