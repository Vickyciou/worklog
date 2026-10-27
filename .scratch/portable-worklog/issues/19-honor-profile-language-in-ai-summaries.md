# 19 — 讓 AI 摘要遵守 profile language

**What to build:** 修正 AI preset 未使用 profile language 的缺口，讓不同來源語言不會使同一個 `zh-TW` profile 意外產生英文摘要，同時保留技術名稱與識別字的準確性。

**Blocked by:** 16 — 提供 AI CLI 摘要 presets；17 — 修改既有 profile settings。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] Codex 與 Claude Code preset prompt 明確包含 normalized profile language。
- [x] Focus、topic、outcome、approach、decision、reason 與 next steps 被要求使用目標語言。
- [x] Public technology names 與 identifiers 在翻譯會降低準確性時保留原文。
- [x] DailyFacts/redaction stdin boundary 不因語言提示而增加 private data。
- [x] Built-in 明確保留 captured text，不宣稱提供翻譯。
- [x] Acceptance regression 以 `zh-TW` profile 與英文來源驗證 provider 收到語言要求並回傳繁體中文摘要。

## Comments

- 2026-09-10：確認 `language` 原先只被保存與驗證，未進入 AI preset prompt；兩個 `zh-TW` profiles 會因來源語言不同而產生不同語言的摘要。
- 2026-09-10：Codex 與 Claude Code 共用的 preset prompt 現在帶入 profile language，且不改變 post-redaction DailyFacts stdin。
- 2026-09-10：語言 regression、完整 `go test ./...`、`go vet ./...` 與 `git diff --check` 通過。
