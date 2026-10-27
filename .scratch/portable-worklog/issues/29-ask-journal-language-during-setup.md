# 29 — Setup 明確詢問日誌語言

**What to build:** Guided setup 與 guided `profile add` 每次都詢問日誌語言，以偵測到的系統語言作為預設值，讓使用者知道語言會決定日誌標題與 AI 摘要的語言。

**Blocked by:** None — can start immediately.

**Status:** complete

**Milestone:** M5 — 更順手的設定與說明

- [x] Repository 之後、Journal folder 之前顯示 `Journal language [<detected>]:`，按 Enter 接受預設值。
- [x] 無法偵測時預設值為 `zh-TW`，與目前的 fallback 相同。
- [x] 輸入值沿用現有的語言正規化；空白或無效值不寫入任何設定。
- [x] Help 與 README 說明 setup 會詢問語言。
- [x] Acceptance tests 覆蓋接受偵測值與手動輸入不同語言兩種情況。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 6。目前 `collectGuidedProfile` 只在偵測失敗時才詢問語言，review 時的測試環境偵測成 `en-AU` 卻沒有詢問。
- 2026-10-27：`collectGuidedProfile` 每次顯示 `Journal language [<default>]:`，預設值為偵測到的語言，偵測失敗時為 `zh-TW`。13 個 guided input 測試補上語言的回答；新增覆寫偵測語言的測試，首個 setup 測試確認提示含偵測值。setup、profile add 的 help 與 README 已更新。`go test ./...` 通過。
