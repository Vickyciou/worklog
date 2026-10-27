# 30 — `inspect` 與 `repo add` 自動判斷 profile

**What to build:** `worklog inspect` 與 `worklog repo add` 的 profile 改為選填。能判斷時直接使用，判斷不出來才詢問。`repo add` 改成 `repo add [path] [--profile name]`，在 repository 裡直接執行即可加入。

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

**Milestone:** M5 — 更順手的設定與說明

- [ ] `inspect [profile] [YYYY-MM-DD]`：符合 `YYYY-MM-DD` 的參數視為日期。沒有指定 profile 時，依序使用目前目錄所屬 repository 的 profile、唯一的 profile，最後以編號清單詢問。
- [ ] `repo add [path] [--profile name]`：path 預設為目前目錄。沒有 `--profile` 時使用唯一的 profile，否則以編號清單詢問；輸入關閉時取消且不寫入。
- [ ] 舊寫法 `repo add <profile> <path>` 不保留。兩個位置參數時失敗並提示 `Use: worklog repo add <path> --profile <profile>`。
- [ ] 已存在的 repository mappings 不受影響。
- [ ] Help、README、`day` 與 `doctor` 的修復提示改用新寫法。
- [ ] Acceptance tests 覆蓋在 repository 內不帶參數加入、`--profile` 指定、多個 profiles 時詢問、舊寫法的提示，以及 `inspect` 的三種判斷方式。

## Comments

- 2026-10-27：來自 2026-10-27 CLI usability review 的建議 10。使用者於 2026-10-27 決定不保留舊寫法；已加入的 repositories 存在設定檔中，不受語法變更影響。
