# 27 — `worklog day` 先用一行說明每個 profile

**What to build:** `worklog day` 在每個 profile 的預覽或差異之前，先印一行概覽，說明 profile、日期、commit 數與摘要方式，讓使用者不必讀完整份預覽就知道這次在處理什麼。

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

**Milestone:** M5 — 更順手的設定與說明

- [ ] 預覽與差異前印出 `<profile> · <date> · <N> commits · <summary>`，單一 commit 寫成 `1 commit`，摘要方式使用 setup 顯示的名稱（例如 `Built-in`、`Codex`、`Claude Code`、`Custom command`）。
- [ ] 概覽已經包含 profile 名稱，因此預覽與差異標題不再重複 `[profile]`。
- [ ] 已是最新的日誌維持現有的一行訊息，不另外印概覽。
- [ ] 路徑繼續以 `~/...` 顯示（Issue 22 已完成）。
- [ ] Acceptance tests 覆蓋單一 profile 與兩個 profiles 的概覽行。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 7。`~` 縮寫路徑已在 Issue 22 以 `displayPath` 完成，本 issue 只處理概覽行。
