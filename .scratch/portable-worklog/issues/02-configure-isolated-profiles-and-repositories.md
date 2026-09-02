# 02 — 設定隔離的 profile 與 repository

**What to build:** 讓使用者透過 setup、profile 與 repo commands 建立可攜但彼此隔離的工作環境，將 repository、作者身分、時區及日誌目的地確定地歸屬到單一 profile。

**Blocked by:** 01 — 建立可編譯 CLI 與驗收測試邊界。

**Status:** ready-for-agent

**Milestone:** M1 — 最小可用流程

- [ ] 互動式 setup 可以建立包含時區、語言、journal destination 與可選 summarizer 設定的第一個 profile。
- [ ] 使用者可以新增、檢視及驗證 profile、author identities 與明確的 repository allowlist。
- [ ] 每個 repository 只能屬於一個 profile；重複或重疊 mapping 會停止並說明衝突。
- [ ] Config 與 state 預設存放在 source repository 外，acceptance tests 可用環境變數覆寫其 roots。
- [ ] Setup 不會把 provider credentials 或其他 secrets 寫入設定，顯示設定時也不會洩漏它們。
- [ ] 完成一次 setup 後，日常 command 不需要重複傳入 profile、destination 或 timezone。
