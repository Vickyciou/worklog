# 28 — `profile show` 預設顯示易讀格式

**What to build:** `worklog profile show <name>` 預設以對齊的欄位顯示設定，讓人一眼看懂；需要給程式讀取時加 `--json` 取得現有 JSON。

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

**Milestone:** M5 — 更順手的設定與說明

- [ ] 預設輸出 profile 名稱，以及 Language、Timezone、Journal folder、Daily summary、Authors、Repositories；路徑以 `~/...` 顯示，多個 authors 或 repositories 各占一行。
- [ ] 有 denied literals 時只顯示數量，不印出內容。
- [ ] 最後提示 `Change these with worklog settings <name>.`
- [ ] `--json` 輸出與目前完全相同的 JSON。
- [ ] 讀取 JSON 的既有 acceptance tests 改用 `--json`；新增易讀格式的 acceptance test。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 12。目前 `showProfile` 只呼叫 `writeJSON`。
- 2026-10-27：denied literals 只顯示數量，避免在終端機或截圖中露出使用者特意遮蔽的字串。
