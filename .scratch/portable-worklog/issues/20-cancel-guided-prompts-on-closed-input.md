# 20 — 輸入關閉時取消 guided prompts

**What to build:** 修正 guided prompts 把 EOF 當成「按 Enter 接受預設值」的問題，讓沒有人回答的 setup、settings 或 profile add 一律取消且不寫入任何設定。

**Blocked by:** 15 — 精簡 setup 並提供任務導向 help；17 — 修改既有 profile settings；18 — 引導新增完整 profile。

**Status:** ready-for-agent

**Milestone:** M4 — 直覺的日常使用

- [ ] 任何 guided prompt 或 `[Y/n]` 確認在讀到內容前遇到 EOF 時，停止流程並回報 `No answer received (input closed). Nothing was saved.`。
- [ ] 取消時不修改 config、不建立 state 或 journal directory，exit status 非零。
- [ ] 錯誤訊息提示可腳本化的替代指令，例如 `worklog profile set` 或帶 flags 的 `worklog profile add`。
- [ ] 有內容但沒有結尾換行的最後一行仍視為有效回答。
- [ ] `worklog day` 的 `[y/N]` 確認在 EOF 時維持不寫入。
- [ ] Acceptance tests 以 stdin 關閉的方式執行 `setup`、`settings`、`profile add`，驗證 config 不變且 exit status 非零。

## Comments

- 2026-10-27：重現方式為 `worklog settings < /dev/null`。每題套用預設值，最後印出 `Updated profile "work".` 且 exit 0。原因在 `cmd/worklog/config_commands.go` 的 `prompt` 與 `confirm`：兩者把 `io.EOF` 當成空字串，而空字串代表接受預設值，`confirm` 的預設又是 yes。`setup` 與 `profile add` 共用同一組函式，依程式碼推論有相同問題，尚未實際執行驗證。
