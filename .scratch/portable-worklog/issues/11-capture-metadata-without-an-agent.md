# 11 — 提供沒有 AI agent 時的 capture fallback

**What to build:** 提供可選的通用 post-commit fallback，讓使用者直接使用 Git commit、未經任何 AI agent 時，仍能自動留下最小且可驗證的工作事件。

**Blocked by:** 03 — 將結構化工作事實寫入 Git notes。

**Status:** wont-do

**Milestone:** M3 — 發布與正式遷移

- [ ] 使用者可明確安裝或啟用 generic post-commit integration，不會無聲修改所有 repositories。
- [ ] 一次正常 Git commit 會自動呼叫公開 capture contract，並為新 commit 建立 WorklogEvent。
- [ ] Fallback 只記錄 Git metadata 支持的 outcome 與 source metadata，不捏造 decisions、reasons 或 next steps。
- [ ] 重複觸發遵守 capture idempotency，既有不同 note 不會被覆寫。
- [ ] Integration 不取代或破壞 repository 已存在的 unrelated hook 行為。
- [ ] Acceptance test 在臨時 repository 完成 commit 後直接檢查生成的 Git note。

## Comments

- 2026-10-27：取消，使用者同意。沒有 note 的 commit 已經由 `dailyfacts` 以 commit subject 作為 outcome 收進日誌，fallback 能記錄的內容不會增加任何資訊。另外 post-commit hook 會在 agent capture 之前寫入 note；以實際 binary 驗證，之後 gen-commit 的完整 event 會被 `commit already has a different worklog event` 拒絕，導致 agent 紀錄變少。
