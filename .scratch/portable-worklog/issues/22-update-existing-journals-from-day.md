# 22 — 讓 `worklog day` 自動更新既有日誌

**What to build:** 同一天再次執行 `worklog day` 時，不再因日誌已存在而失敗。worklog 依每個 profile 的日誌狀態自動選擇建立、更新或告知已是最新，使用者不需要記得 `--refresh`。

**Blocked by:** 09 — Refresh 受管理的歷史日誌；21 — 拒絕寫入日誌視為正常結束。

**Status:** ready-for-agent

**Milestone:** M4 — 直覺的日常使用

- [ ] 每個 active profile 的日誌狀態只有三種：尚未建立、已是最新、有新變動。狀態由 journal 是否存在，以及 managed block 的 source digest 是否等於本次 post-redaction DailyFacts 的 digest 決定。
- [ ] 尚未建立時維持目前的完整預覽與 `Write this journal? [y/N]` 流程。
- [ ] 有新變動時顯示 managed block 差異與 `Update this journal? [y/N]`，確認後只替換 managed block，保留 block 外所有 bytes。
- [ ] 已是最新時印出 `✓ <profile>: journal for <date> is already up to date.`，不呼叫 summarizer、不詢問、不寫入。
- [ ] 既有檔案沒有恰好一個有效 managed block 時，仍拒絕修改並說明原因與修復方式。
- [ ] 每個 profile 獨立判斷狀態；同一天可以有一個 profile 建立、另一個更新。
- [ ] 指定歷史日期時套用相同規則。
- [ ] 保留 `--refresh`，意義維持「只更新既有日誌」；日誌不存在時照舊報錯。root help 與 `day --help` 不再把它列為必要步驟。
- [ ] 保留預覽後檔案被改動時拒絕安裝的 stale-preview 保護。
- [ ] 成功訊息使用 `~` 縮寫家目錄路徑。
- [ ] 改變 profile language 後，既有日誌不得被判斷為「已是最新」，以便標題更新為新語言。可行做法是讓 digest 涵蓋影響 render 的 profile 設定（至少 language）。
- [ ] Acceptance tests 覆蓋：同日新增 commit 後更新、沒有新 commit 時顯示已是最新且 fake summarizer 未被呼叫、兩個 profiles 一個建立一個更新、無效 managed block 被拒絕，以及 `--refresh` 的既有行為。

## Comments

- 2026-10-27：目前第二次執行 `worklog day` 會回報 `journal already exists; refresh is required to replace it` 並 exit 1；改用 `--refresh` 且沒有任何變動時，仍會顯示 `(no managed changes)` 並詢問 `Refresh this journal? [y/N]`。
