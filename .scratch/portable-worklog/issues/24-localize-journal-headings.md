# 24 — 日誌標題跟隨 profile language

**What to build:** 讓日誌中由 worklog 產生的固定文字（標題、section headings 與空白項目的佔位文字）依 profile language 顯示，使整份日誌的語言一致。worklog 在終端機印出的訊息維持英文。

**Blocked by:** 06 — 產生不依賴 AI 的每日工作日誌；19 — 讓 AI 摘要遵守 profile language。

**Status:** ready-for-agent

**Milestone:** M4 — 直覺的日常使用

- [ ] 提供 `zh` 與 `en` 兩套固定文字。`zh-*` 使用目前的台灣繁體中文文字，`en-*` 使用英文文字，其他語言使用英文文字。
- [ ] `zh-TW` profile 產生的日誌與目前輸出 byte-for-byte 相同，維持舊 Bash 與 Obsidian layout 的 parity。
- [ ] Frontmatter 的 `type: worklog-daily` 與 `tags: [工作日誌]` 不隨語言改變，避免既有 Obsidian 查詢失效。
- [ ] 改變 profile language 後更新既有日誌時，只替換 managed block 內的文字。
- [ ] `settings` 與 `setup` 的說明提到 language 同時影響 AI 摘要與日誌標題。
- [ ] Acceptance tests 覆蓋 `zh-TW` 輸出不變、`en-US` 產生英文標題，以及 frontmatter tags 不變。

## Comments

- 2026-10-27：使用者決定標題跟著 profile language 變，終端機訊息維持英文。英文版文字待實作時定稿，示意稿使用 `Work Log`、`Highlights` 這類名稱。
- 2026-10-27：Frontmatter tags 保持固定是本 issue 的預設決定，理由是 tags 用於 Obsidian 查詢，而不是給人閱讀。
