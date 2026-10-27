# 22 — 讓 `worklog day` 自動更新既有日誌

**What to build:** 同一天再次執行 `worklog day` 時，不再因日誌已存在而失敗。worklog 依每個 profile 的日誌狀態自動選擇建立、更新或告知已是最新，使用者不需要記得 `--refresh`。

**Blocked by:** 09 — Refresh 受管理的歷史日誌；21 — 拒絕寫入日誌視為正常結束。

**Status:** complete

**Milestone:** M4 — 直覺的日常使用

- [x] 每個 active profile 的日誌狀態只有三種：尚未建立、已是最新、有新變動。狀態由 journal 是否存在，以及 managed block 的 source digest 是否等於本次 post-redaction DailyFacts 的 digest 決定。
- [x] 尚未建立時維持目前的完整預覽與 `Write this journal? [y/N]` 流程。
- [x] 有新變動時顯示 managed block 差異與 `Update this journal? [y/N]`，確認後只替換 managed block，保留 block 外所有 bytes。
- [x] 已是最新時印出 `✓ <profile>: journal for <date> is already up to date.`，不呼叫 summarizer、不詢問、不寫入。
- [x] 既有檔案沒有恰好一個有效 managed block 時，仍拒絕修改並說明原因與修復方式。
- [x] 每個 profile 獨立判斷狀態；同一天可以有一個 profile 建立、另一個更新。
- [x] 指定歷史日期時套用相同規則。
- [x] 保留 `--refresh`，用於強制重新產生既有日誌；日誌不存在時照舊報錯。`day --help` 與 README 不再把它列為必要步驟。
- [x] 保留預覽後檔案被改動時拒絕安裝的 stale-preview 保護。
- [x] 成功訊息使用 `~` 縮寫家目錄路徑。
- [x] 改變 profile language 或 daily summary 後，使用者有明確方式重新產生既有日誌：`--refresh` 即使 facts 未變也重新產生，「已是最新」訊息與 `settings` 都提示這個指令。
- [x] Acceptance tests 覆蓋：同日新增 commit 後更新、沒有新 commit 時顯示已是最新且 fake summarizer 未被呼叫、兩個 profiles 一個建立一個更新、無效 managed block 被拒絕，以及 `--refresh` 的既有行為。

## Comments

- 2026-10-27：目前第二次執行 `worklog day` 會回報 `journal already exists; refresh is required to replace it` 並 exit 1；改用 `--refresh` 且沒有任何變動時，仍會顯示 `(no managed changes)` 並詢問 `Refresh this journal? [y/N]`。
- 2026-10-27：決定不把 language 併入 source digest。Spec 與既有 acceptance test 定義 digest 為完整 post-redaction DailyFacts 的 SHA-256；改變定義會讓所有既有日誌在升級後被判斷為有變動。改為讓 `--refresh` 強制重新產生，並在「已是最新」訊息與 `settings` 變更 language 或 daily summary 時提示。
- 2026-10-27：`planActiveProfiles` 以 `classifyJournal` 判斷 missing、current、changed 三種狀態；current 不呼叫 summarizer。`journal` 新增 `SourceDigest` 與 `ManagedDigest`，`CheckAvailable` 只剩內部使用，改為未匯出。預覽、差異與成功訊息以 `~` 顯示家目錄下的路徑。
- 2026-10-27：新增 acceptance tests 涵蓋同日重跑已是最新且 summarizer 未被呼叫、新增 commit 後更新並保留 block 外內容、`--refresh` 強制重新產生，以及兩個 profiles 狀態各自判斷。以實際 binary 在獨立設定目錄驗證：重跑顯示已是最新、新增 commit 後顯示差異並更新、再跑一次又顯示已是最新。`go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
- 2026-10-27：Follow-up 觀察：差異目前是整個 managed block 先全刪再全加，還不是逐行差異，新增一個 commit 也會印出整份日誌兩次。另外 `profile add --journal-destination` 不會展開字面上的 `~`（shell 會先展開，所以一般使用不受影響），與 `profile set` 不一致。兩者都不在本 issue 範圍內。
