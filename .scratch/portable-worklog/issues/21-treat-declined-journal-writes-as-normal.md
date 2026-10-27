# 21 — 拒絕寫入日誌視為正常結束

**What to build:** 使用者看完預覽後回答 `n` 是正常選擇。worklog 應平靜結束，不把它當成錯誤；同一天有多個 active profiles 時，拒絕其中一個不應中斷其他 profiles。

**Blocked by:** 08 — 分開處理同一天的多個 profiles；09 — Refresh 受管理的歷史日誌。

**Status:** ready-for-agent

**Milestone:** M4 — 直覺的日常使用

- [ ] 拒絕新日誌時印出 `Not written. Run worklog day again when you're ready.`；拒絕更新時印出對應的 `Not updated.` 訊息。兩者都不印 `worklog:` 錯誤前綴或 help 提示。
- [ ] 所有 profiles 都處理完且沒有實際錯誤時 exit 0，包含全部被拒絕的情況。
- [ ] 多個 active profiles 時，拒絕其中一個後繼續預覽與確認下一個 profile。
- [ ] 寫入失敗、驗證失敗等真正的錯誤仍維持非零 exit status。
- [ ] Acceptance tests 覆蓋單一 profile 拒絕、兩個 profiles 中拒絕第一個並寫入第二個，以及拒絕後 journal destination 不存在。

## Comments

- 2026-10-27：目前回答 `n` 會回傳 `journal write not confirmed` 或 `journal refresh not confirmed` 錯誤並 exit 1。`runDay` 遇到第一個錯誤就 return，因此同一天第一個 profile 被拒絕時，後面的 profiles 不會被處理（依 `cmd/worklog/config_commands.go` 的迴圈推論）。
