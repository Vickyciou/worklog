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

**Tickets:** 13、14（11 已於 2026-10-27 取消）

**Assignment waves:**

1. 13 — 發布可攜式 binaries
2. 14 — 驗證 Bash parity 與遷移條件

M3 預定在 M2 完成後開始。Ticket 13 需要 M1 的 12 完成。M3 完成條件：13、14 的 acceptance criteria 全部通過。

## M4 — 直覺的日常使用

**Outcome:** 使用者每天只需要 `worklog day`、看預覽、回答 `y`；同一天重跑、拒絕寫入或沒有變動時都不會遇到錯誤，並能用 `worklog status` 確認今天的狀態。

**Tickets:** 20、21、22、23、24、25

**Assignment waves:**

1. 20 — 輸入關閉時取消 guided prompts；21 — 拒絕寫入日誌視為正常結束；24 — 日誌標題跟隨 profile language
2. 22 — 讓 `worklog day` 自動更新既有日誌
3. 25 — 更新日誌時只顯示有變動的行（22 完成後的易用性 follow-up）
4. 23 — 新增 `worklog status` 顯示今日狀態

**Status:** complete（2026-10-27）

M4 不依賴 M3。使用者於 2026-10-27 決定先完成 M4 再進行 M3。M4 完成條件：20、21、22、23、24、25 的 acceptance criteria 全部通過。

## M5 — 更順手的設定與說明

**Outcome:** 使用者不必記得 profile 名稱或內部術語。`day` 先用一行說明在處理什麼，`profile show` 一眼看懂，setup 明確詢問語言，help 依使用頻率分組並用白話說明。

**Tickets:** 26、27、28、29、30、31、32

**Assignment waves:**

1. 26 — `profile add` 展開日誌路徑中的 `~`；27 — `worklog day` 先用一行說明每個 profile；28 — `profile show` 預設顯示易讀格式；29 — Setup 明確詢問日誌語言
2. 30 — `inspect` 與 `repo add` 自動判斷 profile
3. 31 — Root help 依使用頻率分組
4. 32 — 指令說明改用白話，技術細節移到 `--details`

**Status:** complete（2026-10-27）

M5 不依賴 M3。使用者於 2026-10-27 決定接著 M4 進行 M5，並排除打錯字提示的建議。M5 完成條件：26 至 32 的 acceptance criteria 全部通過。
