# 25 — 更新日誌時只顯示有變動的行

**What to build:** `worklog day` 更新既有日誌時，只顯示新增或刪除的行以及少量上下文，讓使用者一眼看出這次多了什麼，而不是看到整份舊日誌與整份新日誌各印一次。

**Blocked by:** 22 — 讓 `worklog day` 自動更新既有日誌。

**Status:** complete

**Milestone:** M4 — 直覺的日常使用

- [x] 差異以行為單位計算，新增行以 `+ ` 開頭、刪除行以 `- ` 開頭、上下文以兩個空格開頭。
- [x] 每處變動前後最多顯示兩行上下文；省略的未變動區段以 `  ...` 表示。
- [x] 不顯示 managed block 的開始與結束標記行，source digest 的變動不出現在差異中。
- [x] 內容完全沒有變動時（例如 `--refresh` 重新產生出相同內容），顯示 `  (no visible changes)`，仍照常詢問是否更新。
- [x] 確認後寫入的內容與目前相同：只替換 managed block，保留 block 外所有 bytes。
- [x] 測試覆蓋只新增行、同時有新增與刪除、遠處變動之間的省略，以及沒有可見變動。

## Comments

- 2026-10-27：Issue 22 實作後觀察到，同一天多一個 commit 時，差異會先以 `-` 印出整個舊 block、再以 `+` 印出整個新 block，包含 digest 標記行。
- 2026-10-27：`managedDiff` 改為比對 managed 標記之間的行，以最長共同子序列計算逐行差異，刪除行排在新增行之前，並只保留變動前後兩行上下文。Unit tests 覆蓋新增、取代、遠處變動、空白上下文行與沒有可見變動；acceptance tests 確認差異不含 source digest、未變動的行不會被標成刪除。
- 2026-10-27：以實際 binary 驗證：同一天多一個 commit 時只顯示 `+ - Add retry metrics` 與其上下文；`--refresh` 產生相同內容時顯示 `(no visible changes)`。`go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
