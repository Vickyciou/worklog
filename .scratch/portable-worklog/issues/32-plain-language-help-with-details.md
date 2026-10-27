# 32 — 指令說明改用白話，技術細節移到 `--details`

**What to build:** `day`、`inspect`、`status`、`doctor` 的 `--help` 只說明它會做什麼、怎麼用、下一步；隱私邊界、摘要格式驗證、managed block 等技術細節移到 `--details`。

**Blocked by:** 31 — Root help 依使用頻率分組。

**Status:** ready-for-agent

**Milestone:** M5 — 更順手的設定與說明

- [ ] 上述指令的 `--help` 不使用 DailyFacts、DailySummary、managed block、schema、payload 等內部術語。
- [ ] `worklog <command> --details` 與 `worklog help <command> --details` 印出完整技術說明，內容不少於目前 help 的技術資訊。
- [ ] 有 details 的指令在 help 最後提示 `worklog <command> --details`。
- [ ] `capture --help` 維持完整 JSON 契約，因為 `gen-commit` 依賴它作為輸入規格。
- [ ] Acceptance tests 覆蓋 help 不含內部術語、details 包含原有技術資訊，以及兩種 details 入口。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 5。`setup`、`settings`、`profile` 等設定類 help 已在 Issue 15 精簡，本 issue 不更動它們的結構。
