# 16 — 提供 AI CLI 摘要 presets

**What to build:** 在安全的 summarizer contract 可用後，讓使用者以 Built-in、Codex 或 Claude Code 選項設定摘要；worklog 直接沿用 provider CLI 自己管理的登入 session，不要求或預先探查 credential。

**Blocked by:** 07 — 連接任意 command summarizer；17 — 修改既有 profile settings。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] Setup 與 settings 將欄位命名為 Daily summary，而不是 Summarizer command，並說明 AI 是選用功能、Built-in 不使用 AI 或網路。
- [x] 以 executable discovery 判斷 Codex 或 Claude Code 是否可選，不另做 login-status 或 credential probe；CLI 不存在時清楚提示並保留 Built-in。
- [x] Codex 與 Claude Code presets 直接沿用各 CLI 當下可用的 session，由 provider CLI 管理登入、授權、usage limits 與 billing，worklog 不取得或顯示 credential。
- [x] Presets 仍遵守相同的 post-redaction DailyFacts stdin 與 DailySummary stdout schema，不讓 provider 選擇 journal path 或輸出 final Markdown。
- [x] Provider process 在不含 registered repository source tree 的 worklog state directory 執行，只收到 post-redaction payload；nonzero、空輸出或 schema-invalid output 時顯示安全的失敗原因並使用 deterministic fallback。
- [x] 第一版不在 setup、settings 或其他一般使用者流程提供 Custom command 選項；Codex 與 Claude Code presets 共用內部 summarizer contract。
- [x] Acceptance tests 以 fake Codex 與 Claude Code executables 驗證 executable discovery、成功呼叫、nonzero、空輸出、payload boundary、schema validation 及 fallback，不依賴真實 provider、登入狀態或網路。

## Comments

- 2026-09-04：參考既有 `~/Documents/worklog/worklog.sh`，移除預先判斷登入狀態的設計。既有流程直接呼叫 provider CLI，讓 CLI 自己使用現有 session，失敗後退回離線摘要；新版保留相同簡單模型並加上結構化輸出驗證。
- 2026-09-04：第一版只提供 Built-in、Codex 與 Claude Code，不提供使用者可設定的 Custom command。
- 2026-09-04：完成 executable-discovered presets；Codex 使用官方穩定 `codex exec` 非互動模式、read-only sandbox 與 output schema，Claude Code 使用 print/safe mode 與 JSON schema。兩者都不探查登入狀態。
- 2026-09-04：fake provider acceptance 覆蓋兩個 presets 的 success、nonzero、empty、schema-invalid、payload redaction、state-directory execution 與 deterministic fallback。
