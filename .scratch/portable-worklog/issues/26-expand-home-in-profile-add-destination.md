# 26 — `profile add` 展開日誌路徑中的 `~`

**What to build:** `worklog profile add <name> --journal-destination ~/...` 應把 `~` 展開成家目錄，和 `profile set`、guided setup 的行為一致。

**Blocked by:** None — can start immediately.

**Status:** complete

**Milestone:** M5 — 更順手的設定與說明

- [x] `profile add` 的 `--journal-destination` 以 `~` 或 `~/` 開頭時，存成家目錄下的絕對路徑。
- [x] 不以 `~` 開頭的相對路徑維持現有行為，以目前目錄解析成絕對路徑。
- [x] Acceptance test 以 `~/journal` 新增 profile，確認設定中的路徑位於家目錄下，且目前目錄沒有建立名為 `~` 的資料夾。

## Comments

- 2026-10-27：Issue 23 撰寫測試時觀察到，`profile add personal --journal-destination ~/journal` 在引號內傳入時會存成 `<目前目錄>/~/journal`。`profile set` 經由 `updatedProfile` 呼叫 `expandUserPath`，`addProfileAdvanced` 則直接呼叫 `config.NewProfile`，沒有展開。
- 2026-10-27：`addProfileAdvanced` 在建立 profile 前呼叫 `expandUserPath`。新增的 acceptance test 在修正前以 `<repository>/~/journals/personal` 失敗，修正後通過。
