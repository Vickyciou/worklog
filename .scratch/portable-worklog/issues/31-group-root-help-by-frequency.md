# 31 — Root help 依使用頻率分組

**What to build:** `worklog help` 依使用頻率分組，每天用的指令在最上面，設定類其次，排錯工具與進階整合指令在最後，每個指令只出現一次。

**Blocked by:** 30 — `inspect` 與 `repo add` 自動判斷 profile。

**Status:** ready-for-agent

**Milestone:** M5 — 更順手的設定與說明

- [ ] 第一行說明 worklog 會把 Git commits 整理成每日工作日誌。
- [ ] 分組依序為 Every day（`day`、`status`）、Set up（`setup`、`settings`、`repo add`、`profile add`）、When something looks wrong（`doctor`、`inspect`）、Advanced and integrations（`profile`、`author`、`repo`、`capture`、`version`）。
- [ ] 保留 `Running worklog without a command shows today's status.` 與 Options 區塊。
- [ ] Acceptance test 的 root help 預期更新。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 8、9。目前 `day`、`status` 等指令在 Getting started 與 Commands 各出現一次，`capture` 排在 Commands 第一個。
