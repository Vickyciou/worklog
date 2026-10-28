# 13 — 發布可攜式 binaries

**What to build:** 讓使用者可以從 versioned release 安裝單一 `worklog` executable，在沒有 Go toolchain 或特定 AI agent 的新環境完成 setup、capture 與 day workflow。

**Blocked by:** 08 — 分開處理同一天的多個 profiles；09 — 安全 refresh 歷史日誌；10 — 診斷設定與隱私邊界；12 — 整合中央 gen-commit。

**Status:** done

**Milestone:** M3 — 發布與正式遷移

- [x] Project 明確宣告第一版支援的平台與架構，release pipeline 為每個目標產生單一 executable。
- [x] Versioned GitHub release 提供可辨識的 artifacts、checksums 與對應 source revision。
- [x] Release artifacts 不包含任何 developer config、captured notes、credentials、真實 journal path 或 employer identifiers。
- [x] 安裝說明涵蓋下載、驗證、放入 PATH、第一次 setup 與升級方式。
- [x] Fresh isolated environment 能只用 release binary 通過 version、setup、capture、inspect 與離線 day smoke tests。
- [x] Release failure 不會發布缺少平台 artifact 或未通過 acceptance tests 的版本。

## Comments

- 2026-10-27：第一版支援 macOS 與 Linux 的 arm64、amd64。Windows 沒有可驗證的環境，不發布未驗證的 binary。
- 2026-10-27：`scripts/release.sh` 以 `CGO_ENABLED=0 -trimpath` 建出四個 executables 與 `SHA256SUMS`，`worklog version` 顯示版本與 commit。腳本會檢查 binary 不含建置機器的 home 與 repository 路徑；未加 `-trimpath` 的 build 會被這個檢查抓到，release binaries 連使用者名稱都不含。
- 2026-10-27：`scripts/smoke-test.sh` 在隔離 HOME 且 PATH 不含 Go 的環境跑 version、setup、capture、inspect、day 與 status。本機 darwin_arm64 與 darwin_amd64（Rosetta）通過；指定錯誤版本時會失敗。
- 2026-10-27：`.github/workflows/release.yml` 在推送 `v*.*.*` tag 時依序執行 test、build、四個平台各自的 checksum 與 smoke test、publish；publish 依賴所有 smoke jobs。YAML 以 Ruby 解析確認 job 順序。actionlint 無法下載，未執行。
- 2026-10-27：尚未驗證：workflow 從未在 GitHub 上執行，Linux binaries 只在 CI 中測試，`ubuntu-24.04-arm` runner 對 private repository 是否可用也未確認；若不可用，該 job 失敗且不會發布。剩下的三項要等第一次推送 tag 後才能勾選，需要使用者授權 push。
- 2026-10-28：第一次推送 `v0.1.0` 時，Ubuntu 上 9 個 acceptance tests 失敗，原因是 runner 無法推測 Git 身分，`capture` 寫 note 失敗；本機 macOS 會從 hostname 推測所以沒發現。workflow 停在 test，沒有 build 也沒有建立 release，符合「失敗不發布」。修正後（`51defd0`、`8ce8617`）刪除舊 tag 並在 `0f37ff5` 重打 `v0.1.0`。
- 2026-10-28：第二次 run 40625454942 全部通過：兩平台 test、build、四平台 checksum 與 smoke（含 `ubuntu-24.04-arm`，repo 已公開）、publish。Release 附四個 executables 與 `SHA256SUMS`，release notes 註明來源 commit。照 README 用 `curl` 下載 darwin_arm64、`shasum -c` 回報 OK、`worklog version` 顯示 `worklog v0.1.0 (0f37ff55c243)`。
