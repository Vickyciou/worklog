# 10 — 診斷設定與隱私邊界

**What to build:** 提供 `worklog doctor`，讓使用者在日常生成失敗前就能發現 Git、profiles、repository mappings、journal destinations 與 summarizer commands 的設定問題。

**Blocked by:** 02 — 設定隔離的 profile 與 repository；03 — 將結構化工作事實寫入 Git notes；07 — 連接任意 command summarizer。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] Doctor 檢查 Git 可用性、config 與 state roots、必要 profile 欄位及 author identities。
- [x] Doctor 檢查 registered repositories、重疊 mappings、Git notes 可用性與 journal destination 狀態。
- [x] Doctor 檢查 configured summarizer command 是否可執行，但不需要真的傳送工作內容。
- [x] 每個失敗項目提供可採取的修正建議，並以 exit status 區分成功與失敗。
- [x] 診斷輸出不顯示 credentials、captured content、denied literals 的原值或其他 secrets。
- [x] 完整與故意破壞的臨時設定皆有 acceptance coverage。

## Comments

- 2026-09-07：新增 `worklog doctor` 與 help；以 `[OK]`/`[FAIL]` 檢查 Git、roots、configuration、profiles、authors、repository root、Git notes、journal destination 與 summary executable，失敗時提供可執行修正方向並回傳 non-zero。
- 2026-09-07：doctor 只做 executable discovery，從不呼叫 summarizer/provider 或 login probe；輸出不包含 repository/journal path、command、author email、denied literal 或 captured facts。
- 2026-09-07：acceptance 覆蓋完整設定、缺少 mapping、失效 repository、缺少 journal/command 與重疊 mappings，並以 private sentinels 驗證診斷輸出邊界。
