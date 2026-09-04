# 01 — 建立可編譯 CLI 與驗收測試邊界

**What to build:** 提供第一個可編譯、可直接執行的 `worklog` Go CLI，並建立以外部 process 驗證使用者可見行為的 acceptance seam，讓後續功能都能在隔離環境中安全擴充。

**Blocked by:** None — can start immediately.

**Status:** complete

**Milestone:** M1 — 最小可用流程

- [x] Source 可以產生不依賴 Go runtime 的 `worklog` executable，並提供穩定的 root help 與 version 輸出。
- [x] 未知 command 或不合法參數會回傳非零 exit status 與可理解的錯誤訊息。
- [x] Acceptance harness 會編譯 CLI，並以外部 process 在臨時 home、config root、state root、Git repository 與 journal destination 中執行。
- [x] Acceptance tests 不會讀取或寫入開發者真實的 home、Git repositories 或日誌目錄。
- [x] Repository 說明可讓新的 contributor 使用固定命令完成 build 與 acceptance test。
