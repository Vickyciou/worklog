# 12 — 整合中央 gen-commit

**What to build:** 讓中央管理並透過 symlink 分享的 `gen-commit` 在任何支援 shell command 的 agent 中，都能於 commit 成功後使用同一個 worklog capture contract 保存事實。

**Blocked by:** 03 — 將結構化工作事實寫入 Git notes。

**Status:** complete

**Milestone:** M1 — 最小可用流程

- [x] Generic gen-commit integration 每次執行只讀取一次目前 CLI help，並依 live contract 建立 capture input。
- [x] 成功 commit 後自動記錄可由 inspected diff 與任務內容證實的工作事實，不需要使用者另外執行 note command。
- [x] Integration 不包含 AI provider 選擇、profile routing、employer 名稱或 company-specific workflow。
- [x] Capture 失敗不會撤銷成功 commit，但會把可採取行動的錯誤回報給使用者。
- [x] 同一份中央 skill 可繼續透過 Git 與 symlinks 供不同 agents 使用，不複製出各 provider 版本。
- [x] 使用臨時 repository 的驗證證明 gen-commit 產生的 note 能被 worklog daily collection 讀取。

## Comments

- 2026-09-04：確認中央 `gen-commit` 只讀一次 live `worklog capture --help`，成功 commit 後送出單一 JSON event；兩個 agent 端均 symlink 到同一中央來源。
- 2026-09-04：臨時 repository 端到端驗證成功，capture 產生的 Git note 可由 `worklog inspect` 收集；中央 skills manager status 通過。
