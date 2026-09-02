# 07 — 連接任意 command summarizer

**What to build:** 讓每個 profile 可以選擇任意 command-line AI、local model 或不使用 AI，透過固定 JSON contract 取得摘要，同時由 core 保有格式與安全控制權。

**Blocked by:** 05 — 在 summarizer 邊界執行隱私遮蔽；06 — 產生不依賴 AI 的每日工作日誌。

**Status:** ready-for-agent

- [ ] Adapter 只從 stdin 接收 post-redaction DailyFacts JSON，並只從 stdout 回傳 DailySummary JSON。
- [ ] DailySummary schema 驗證 focus、topic-grouped completed items、decisions、next steps 與 public technologies。
- [ ] Provider 輸出無法直接提供 final Markdown 或選擇 journal destination。
- [ ] Command 不存在、timeout、nonzero exit、empty stdout、malformed JSON 或 schema-invalid JSON 時使用 deterministic fallback。
- [ ] Provider credentials 僅能由 adapter 的環境或系統 credential storage 取得，不寫進 worklog config。
- [ ] Acceptance tests 使用 fake summarizer 分別驗證成功、每種失敗模式與 deterministic result。
