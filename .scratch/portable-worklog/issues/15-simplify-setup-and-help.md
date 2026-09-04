# 15 — 精簡 setup 並提供任務導向 help

**What to build:** 讓第一次使用者只回答必要的人類選擇，由 worklog 自動取得 Git 與系統已知資料，並提供一致、可發現的 command help。

**Blocked by:** 02 — 設定隔離的 profile 與 repository。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] Setup 以附用途說明的 Work、Personal、Custom 選單取得 profile；Work 與 Personal 只是相同能力的 routing labels，Custom 才詢問名稱。
- [x] Repository 以目前 canonical Git root 為可編輯預設；journal destination 以 `~/Documents/worklog/<profile>` 為可編輯預設，兩者都支援 `~`，並在確認前顯示 resolved absolute path。
- [x] Author email 從選定 repository 的 effective `git config user.email` 取得；timezone 與 language 從環境及作業系統取得。只有無法安全取得必要值時才個別詢問。
- [x] Setup 顯示完整 resolved 設定並只做一次最終確認；取消不建立 profile、目錄、config 或 state。
- [x] M2 summarizer 尚未啟用前，setup 使用 deterministic built-in summary，不再要求一般使用者輸入無法立即生效的 raw summarizer command。
- [x] Root help 以首次設定與每日使用為優先；setup、profile、profile add、author、repo、repo add、inspect、capture、day 均支援 `<command> --help`，並有對等的 `help <command> [subcommand]` 路徑。
- [x] Help 說明用途、必要參數、常見範例及下一個可採取的動作；錯誤訊息指向最接近的 help topic。
- [x] Acceptance tests 驗證自動取得、缺值 fallback、Work/Personal/Custom、預設與自訂路徑、取消零副作用及 help aliases。

## Comments

- 2026-09-04：完成 guided setup、任務導向 help 與缺值 fallback 驗收；`go test ./...` 及 `go vet ./...` 通過。
