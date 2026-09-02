# 04 — 收集並檢查單一 profile 的每日事實

**What to build:** 讓使用者針對一個日期與 profile 收集完整但最小化的 DailyFacts，並在不呼叫 provider、不寫入日誌的情況下檢查將參與摘要的工作內容。

**Blocked by:** 02 — 設定隔離的 profile 與 repository；03 — 將結構化工作事實寫入 Git notes。

**Status:** ready-for-agent

**Milestone:** M1 — 最小可用流程

- [ ] 只掃描該 profile allowlist 中的 repositories，並依設定的 author identities、timezone 與 commit timestamp 選出目標日期活動。
- [ ] 同一 commit 從多個 refs 可達時只出現一次，預設忽略 merge commits 與 worklog notes ref 自身。
- [ ] 同時讀取新版 JSON events 與代表性的既有 v1 line-oriented notes，不自動改寫舊資料。
- [ ] 沒有 worklog note 的 commit 仍能以 Git metadata 提供不臆測決策或理由的基本事實。
- [ ] Inspect 行為不呼叫 summarizer、不建立或修改 journal note，並排除 hash、repository path、author email、raw diff 與 source code。
- [ ] 歷史日期與當地日期邊界皆由 acceptance fixtures 驗證。
