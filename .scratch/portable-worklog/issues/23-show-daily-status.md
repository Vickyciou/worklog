# 23 — 新增 `worklog status` 顯示今日狀態

**What to build:** 新增唯讀的 `worklog status`，讓使用者一眼看到今天每個 profile 的活動與日誌狀態，以及下一步該執行的指令。不帶參數執行 `worklog` 時也顯示這個畫面。

**Blocked by:** 22 — 讓 `worklog day` 自動更新既有日誌。

**Status:** complete

**Milestone:** M4 — 直覺的日常使用

- [x] 每個 profile 一行，顯示今天（依該 profile 的 timezone）的 commit 數與日誌狀態：沒有活動、尚未建立、已是最新、有新變動。日誌狀態沿用 22 的判斷方式。
- [x] 最後一行依狀態提示下一步：有尚未建立或有新變動的日誌時提示 `worklog day`，全部最新時說明今天已完成。
- [x] Profile 缺少 repository 或 author 等設定問題時，在該 profile 那一行標示問題並提示 `worklog doctor`，不中斷其他 profiles 的顯示。
- [x] 尚未設定時提示 `worklog setup`。
- [x] `status` 不呼叫 summarizer、不寫入任何檔案，也不印出 commit 內容、author email 或 denied literals。
- [x] 不帶參數的 `worklog` 顯示 status；`worklog help` 與 `worklog --help` 照舊顯示說明。
- [x] Root help 的日常區塊加入 `status`。
- [x] Acceptance tests 覆蓋未設定、沒有活動、尚未建立、已是最新、有新變動，以及一個 profile 設定有問題時其他 profile 照常顯示。

## Comments

- 2026-10-27：目前要知道今天的狀況，需要自行組合 `profile list`、`inspect`、`doctor` 與查看日誌目錄。
- 2026-10-27：新增 `cmd/worklog/status.go`，每個 profile 一行顯示日期、commit 數與狀態（no activity、not written yet、up to date、has new changes、needs attention），日誌狀態沿用 `classifyJournal`。不帶參數的 `worklog` 改為顯示 status，`help`、`--help`、`-h` 照舊顯示說明。
- 2026-10-27：有 profile 需要處理時，下一步提示 `worklog doctor` 優先於 `worklog day`，因為 `day` 在任何 profile 缺少 repository 或 author 時會在寫入前停止。
- 2026-10-27：Acceptance tests 覆蓋未設定、沒有活動、尚未建立、已是最新、有新變動、設定有問題的 profile，以及不印出 commit subject 與 author email。以實際 binary 驗證已是最新與新增 commit 後顯示 has new changes。`go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
