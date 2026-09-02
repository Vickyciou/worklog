# 06 — 產生不依賴 AI 的每日工作日誌

**What to build:** 讓已完成設定的使用者只執行 `worklog day`，就能從單一 profile 的 DailyFacts 預覽並安全寫入格式穩定的繁體中文 Obsidian 工作日誌，即使完全沒有 summarizer 也可使用。

**Blocked by:** 04 — 收集並檢查單一 profile 的每日事實。

**Status:** ready-for-agent

- [ ] 無 summarizer 時會產生 deterministic DailySummary，而不是要求使用者補裝 AI provider。
- [ ] Core deterministic render YAML frontmatter 與 Markdown，使用預設台灣繁體中文 headings 及正確的 property types。
- [ ] Technology names 會正規化、去重並限制數量，不會把敏感識別資訊當成 tech tag。
- [ ] 預設 destination 支援依 profile、年份與月份組織日誌，provider 無法決定檔案路徑。
- [ ] 寫入前提供 preview 或 diff；目的地不明確時拒絕寫入。
- [ ] 確認寫入採 atomic replacement，並在生成內容外預留明確的 managed-block boundaries。
