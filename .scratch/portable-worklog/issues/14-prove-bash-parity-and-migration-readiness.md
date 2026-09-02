# 14 — 驗證 Bash parity 與遷移條件

**What to build:** 使用代表性 fixtures 證明 Go CLI 已涵蓋現行 Bash 日常流程及安全要求，提供明確的 go/no-go 結果、切換步驟與可逆的 rollback 方法。

**Blocked by:** 13 — 發布可攜式 binaries。

**Status:** ready-for-agent

- [ ] Acceptance suite 涵蓋 setup、registration、capture、v1 compatibility、redaction、single-profile day、multi-profile separation、fallback、historical refresh 與 atomic safety。
- [ ] 代表性 Bash 與 Go outputs 會以相同輸入比較；預期差異有明確說明且符合 specification。
- [ ] Fixtures 不包含真實 employer identifiers、private repository names、credentials、production notes 或個人 journal paths。
- [ ] Migration report 明確列出已達成、尚未達成及阻止切換的條件。
- [ ] 切換 executable link 只會在 acceptance suite 全部通過並取得使用者授權後進行。
- [ ] Rollback 可以恢復既有 Bash command，且不會刪除或改寫既有 profiles、Git notes 或 journals。
