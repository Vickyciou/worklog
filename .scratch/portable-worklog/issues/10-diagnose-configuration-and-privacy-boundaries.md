# 10 — 診斷設定與隱私邊界

**What to build:** 提供 `worklog doctor`，讓使用者在日常生成失敗前就能發現 Git、profiles、repository mappings、journal destinations 與 summarizer commands 的設定問題。

**Blocked by:** 02 — 設定隔離的 profile 與 repository；03 — 將結構化工作事實寫入 Git notes；07 — 連接任意 command summarizer。

**Status:** ready-for-agent

- [ ] Doctor 檢查 Git 可用性、config 與 state roots、必要 profile 欄位及 author identities。
- [ ] Doctor 檢查 registered repositories、重疊 mappings、Git notes 可用性與 journal destination 狀態。
- [ ] Doctor 檢查 configured summarizer command 是否可執行，但不需要真的傳送工作內容。
- [ ] 每個失敗項目提供可採取的修正建議，並以 exit status 區分成功與失敗。
- [ ] 診斷輸出不顯示 credentials、captured content、denied literals 的原值或其他 secrets。
- [ ] 完整與故意破壞的臨時設定皆有 acceptance coverage。
