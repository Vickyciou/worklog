# 13 — 發布可攜式 binaries

**What to build:** 讓使用者可以從 versioned release 安裝單一 `worklog` executable，在沒有 Go toolchain 或特定 AI agent 的新環境完成 setup、capture 與 day workflow。

**Blocked by:** 08 — 分開處理同一天的多個 profiles；09 — 安全 refresh 歷史日誌；10 — 診斷設定與隱私邊界；11 — 提供沒有 AI agent 時的 capture fallback；12 — 整合中央 gen-commit。

**Status:** ready-for-agent

**Milestone:** M3 — 發布與正式遷移

- [ ] Project 明確宣告第一版支援的平台與架構，release pipeline 為每個目標產生單一 executable。
- [ ] Versioned GitHub release 提供可辨識的 artifacts、checksums 與對應 source revision。
- [ ] Release artifacts 不包含任何 developer config、captured notes、credentials、真實 journal path 或 employer identifiers。
- [ ] 安裝說明涵蓋下載、驗證、放入 PATH、第一次 setup 與升級方式。
- [ ] Fresh isolated environment 能只用 release binary 通過 version、setup、capture、inspect 與離線 day smoke tests。
- [ ] Release failure 不會發布缺少平台 artifact 或未通過 acceptance tests 的版本。
