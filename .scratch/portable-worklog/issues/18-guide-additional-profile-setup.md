# 18 — 引導新增完整 profile

**What to build:** 讓使用者新增第二個或後續 profile 時沿用 setup 的低理解成本流程，一次建立 profile、初始 author identity 與 repository mapping，同時保留原有 flags 供腳本使用。

**Blocked by:** 02 — 設定隔離的 profile 與 repository；15 — 精簡 setup 並提供任務導向 help；16 — 提供 AI CLI 摘要 presets。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] `worklog profile add` 提供 Work、Personal、Custom 選單；`worklog profile add <name>` 可略過名稱選擇，其餘流程相同。
- [x] Guided add 以目前 canonical Git root 為可編輯預設，自動取得 effective author email、timezone 與 language，並詢問 journal folder 與 Daily summary。
- [x] 最終摘要包含 resolved profile、repository、author、timezone、language、journal folder 與 Daily summary，且只確認一次。
- [x] 確認後一次保存 profile、初始 author identity 與 repository mapping；取消、驗證錯誤或 mapping 衝突時不修改 config、不建立 journal directory。
- [x] 原本帶完整 flags 的 `profile add` 保持可腳本化，且 help 與 README 清楚區分 guided 與 advanced 用法。
- [x] Acceptance tests 覆蓋選單、指定名稱、完整保存、取消零副作用與既有 advanced contract。

## Comments

- 2026-09-08：完成 setup-style guided add；已有 `work` 或 `personal` 時，選單會標示並預選第一個尚未建立的常用 profile。
- 2026-09-08：完整 `go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
