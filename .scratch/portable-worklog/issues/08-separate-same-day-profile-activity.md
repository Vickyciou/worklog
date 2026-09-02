# 08 — 分開處理同一天的多個 profiles

**What to build:** 讓不帶 profile 參數的 `worklog day` 自動找出當天有活動的 profiles，逐一執行完全隔離的收集、遮蔽、摘要、預覽及寫入流程。

**Blocked by:** 07 — 連接任意 command summarizer。

**Status:** ready-for-agent

- [ ] 同一天只有一個 active profile 時，`worklog day` 維持無額外參數的簡單流程。
- [ ] 同一天有多個 active profiles 時，自動建立各自的 DailyFacts、adapter request 與 journal output，不要求使用者記得指定 profile。
- [ ] 任一 summarizer request 只包含單一 profile 的 facts、redaction policy 與 preferences。
- [ ] 每個 profile 只寫入自己的 journal destination，內容不包含其他 profile 的 repository activity。
- [ ] Repository mapping 缺失或 ambiguous 時停止受影響的流程並提供修正方式，不會自動混合。
- [ ] Acceptance fixture 中的 work 與 personal commits 會產生兩次隔離 adapter requests 及兩份互不交叉的 journals。
