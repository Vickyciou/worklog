# 17 — 修改既有 profile settings

**What to build:** 讓使用者不必手動編輯 config JSON，就能安全修改既有 profile 的 language、timezone 與 journal destination，並為後續 Daily summary presets 提供共用的設定更新邊界。

**Blocked by:** 02 — 設定隔離的 profile 與 repository；15 — 精簡 setup 並提供任務導向 help。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] `worklog settings [profile]` 提供任務導向的互動式修改流程；只有一個 profile 時自動選取，多個 profiles 時要求使用者選擇。
- [x] `worklog profile set <name>` 提供可腳本化的 flags，可個別或同時修改 `--language`、`--timezone` 與 `--journal-destination`，且至少需要一個變更。
- [x] 互動流程顯示目前值與 resolved 新值，只確認一次；取消、輸入無效或驗證失敗時不改 config、不建立目錄。
- [x] Language 會正規化、timezone 必須是有效 IANA timezone、journal destination 支援 `~` 並儲存為 resolved absolute path。
- [x] Journal destination 變更只影響後續輸出，不自動移動、複製或刪除既有 journals，並在確認前清楚提示。
- [x] 更新只影響指定 profile；profile name 暫不允許修改，author identities 與 repository mappings 繼續由各自 commands 管理。
- [x] Settings 顯示目前 Daily summary mode；在 16 完成前固定為 Built-in，之後由相同更新邊界加入 presets。
- [x] `settings`、`profile set` 的 direct help、`help` aliases、錯誤 recovery 與 README examples 保持一致。
- [x] Acceptance tests 驗證單一與多 profile 選擇、各欄位及組合更新、路徑解析、其他 profile 不變、取消零副作用與 invalid update 不落盤。

## Comments

- 2026-09-04：完成 guided `settings` 與 scriptable `profile set`，共用完整 candidate validation 與 atomic config replacement；journal folder 變更不搬移或建立資料。
- 2026-09-04：`go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
