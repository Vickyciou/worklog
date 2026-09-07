# 08 — 分開處理同一天的多個 profiles

**What to build:** 讓不帶 profile 參數的 `worklog day` 自動找出當天有活動的 profiles，逐一執行完全隔離的收集、遮蔽、摘要、預覽及寫入流程。

**Blocked by:** 07 — 連接任意 command summarizer。

**Status:** complete

**Milestone:** M2 — 安全且完整的日常使用

- [x] 同一天只有一個 active profile 時，`worklog day` 維持無額外參數的簡單流程。
- [x] 同一天有多個 active profiles 時，自動建立各自的 DailyFacts、adapter request 與 journal output，不要求使用者記得指定 profile。
- [x] 任一 summarizer request 只包含單一 profile 的 facts、redaction policy 與 preferences。
- [x] 每個 profile 只寫入自己的 journal destination，內容不包含其他 profile 的 repository activity。
- [x] Repository mapping 缺失或 ambiguous 時停止受影響的流程並提供修正方式，不會自動混合。
- [x] Acceptance fixture 中的 work 與 personal commits 會產生兩次隔離 adapter requests 及兩份互不交叉的 journals。

## Comments

- 2026-09-07：`day` 先收集並 preflight 全部 active profiles，再依 profile 逐一摘要、預覽、確認與寫入；無日期時每個 profile 使用自己的 local date。
- 2026-09-07：缺少 repository/author mapping、無法收集的 repository 或重複 active destination 會在 provider invocation 與任何 journal write 前停止，並提供 `repo add`、`author add`、`repo validate` 或 `profile set` 修正方向。
- 2026-09-07：雙 profile fake adapter acceptance 驗證兩份獨立 post-redaction request 與兩份不交叉 journal；無活動時不建立 journal。
