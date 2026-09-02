# 03 — 將結構化工作事實寫入 Git notes

**What to build:** 提供穩定的 capture contract，讓 agent 或其他整合能為剛完成的 commit 寫入可驗證、版本化且不會被重試破壞的工作事實。

**Blocked by:** 01 — 建立可編譯 CLI 與驗收測試邊界。

**Status:** ready-for-agent

- [ ] `worklog capture` 的 help 清楚描述輸入 contract，且可以為指定 repository 與 commit 接收結構化工作事實。
- [ ] 新事件以版本化 WorklogEvent JSON 寫入專用 Git notes ref，必填 outcome，選填欄位符合規格允許的事實類型。
- [ ] 相同事件重試時結果為 idempotent，不會產生重複內容。
- [ ] 同一 commit 已有不同事件時停止並保留原 note，不會靜默覆寫。
- [ ] 不存在的 repository、commit、缺少 outcome 或 schema-invalid input 會失敗且不留下部分 note。
- [ ] Acceptance tests 從 executable 外部檢查 exit status、輸出訊息與實際 Git note。
