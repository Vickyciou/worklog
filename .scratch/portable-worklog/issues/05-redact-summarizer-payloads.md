# 05 — 在 summarizer 邊界執行隱私遮蔽

**What to build:** 在任何 DailyFacts 離開本機核心前套用 profile 專屬與通用的隱私規則，並讓使用者可以稽核 provider 真正會收到的 payload。

**Blocked by:** 04 — 收集並檢查單一 profile 的每日事實。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] Profile 可以設定禁止傳出的 literals，遮蔽規則只作用於該 profile。
- [x] Common secret patterns 與禁止 literals 在 adapter invocation 前完成偵測及遮蔽。
- [x] `worklog inspect` 顯示與 adapter stdin 完全相同的 post-redaction DailyFacts，但不呼叫 provider 或寫入 journal。
- [x] Payload 不含 commit hash、repository path、author email、raw code、raw diff 或未明確允許的 commit body。
- [x] Fake summarizer 一旦收到 fixture hash、path、email、denied literal 或 secret 就使 acceptance test 失敗。
- [x] 不同 profiles 的 redaction 設定及資料不會互相出現在 inspect 結果中。

## Comments

- 2026-09-04：新增共用 post-redaction `DailyFacts` boundary；`inspect` 與 `day` 均先套用 profile denied literals、secret/token、hash、path、email、raw diff 與 fenced-code 規則。
- 2026-09-04：acceptance fixture 將 `inspect` 輸出視為未來 adapter stdin，遇到任何禁止值即失敗，並驗證兩個 profiles 的規則與資料隔離。
