# Portable worklog milestones

Milestones group the approved implementation tickets into delivery stages. A ticket's `Blocked by` field remains the source of truth for technical eligibility.

Assign one ticket to one agent context. A ticket joins the technical frontier only after every declared blocker is complete and verified. Work milestones in order unless the user explicitly reprioritizes an eligible later-stage ticket.

## M1 — 最小可用流程

**Outcome:** 使用者可以完成一次設定，正常 commit 後自動 capture，並為單一 profile 產生不依賴 AI 的每日工作日誌。

**Tickets:** 01、02、03、04、06、12

**Assignment waves:**

1. 01 — 建立可編譯 CLI 與驗收測試邊界
2. 02 — 設定隔離的 profile 與 repository；03 — 將結構化工作事實寫入 Git notes
3. 04 — 收集並檢查單一 profile 的每日事實；12 — 整合中央 gen-commit
4. 06 — 產生不依賴 AI 的每日工作日誌

**Status:** complete（2026-09-04）

M1 完成條件已達成：01、02、03、04、06、12 的 acceptance criteria 全部通過，包括中央 `gen-commit` 至 daily collection 的臨時 repository 端到端驗證。

## M2 — 安全且完整的日常使用

**Outcome:** 使用者可以安全連接任意 command summarizer，隔離同一天的多個 profiles，refresh 歷史日誌並診斷設定問題。

**Tickets:** 05、07、08、09、10、15、16、17、18、19

**Assignment waves:**

1. 05 — 在 summarizer 邊界執行隱私遮蔽；09 — 安全 refresh 歷史日誌；15 — 精簡 setup 並提供任務導向 help
2. 07 — 連接任意 command summarizer；17 — 修改既有 profile settings
3. 08 — 分開處理同一天的多個 profiles；10 — 診斷設定與隱私邊界；16 — 提供 AI CLI 摘要 presets
4. 18 — 引導新增完整 profile（M2 完成後的易用性 follow-up）
5. 19 — 讓 AI 摘要遵守 profile language（M2 完成後的 correctness follow-up）

**Status:** complete（2026-09-07；Issues 18、19 follow-ups 於 2026-09-08、2026-09-10 完成）

M2 完成條件已達成：05、07、08、09、10、15、16、17、18、19 的 acceptance criteria 全部通過，包括 post-redaction adapter boundary、validated summarizer fallback、隔離的多 profile daily flow、managed historical refresh、guided setup/settings/additional-profile flow、遵守 profile language 的 authenticated CLI presets，以及不洩漏 private values 的 doctor diagnostics。

## M3 — 發布與正式遷移

**Outcome:** 使用者可以安裝 versioned binary，選擇非 agent capture fallback，並在通過 parity gate 後安全取代既有 Bash 流程。

**Tickets:** 11、13、14

**Assignment waves:**

1. 11 — 提供沒有 AI agent 時的 capture fallback
2. 13 — 發布可攜式 binaries
3. 14 — 驗證 Bash parity 與遷移條件

M3 預定在 M2 完成後開始。Ticket 13 也需要 11 與 M1 的 12 完成。M3 完成條件：11、13、14 的 acceptance criteria 全部通過。
