# 使用者隨手筆記與每日工作日誌整合

Status: ready-for-agent

## Problem Statement

目前 `worklog` 能從已註冊 repository 的 commits 收集工作事實，並由 `gen-commit` 在 commit 成功後透過 `worklog capture` 把結構化 `WorklogEvent` 寫入 Git notes。這條流程適合保存「完成了什麼」、「採取什麼作法」與 agent 能從任務和 diff 證實的技術決策，但無法可靠保存只有使用者本人知道的內容，例如學到了什麼、原本的假設如何被修正、做決策時真正考慮了哪些因素，以及仍待驗證的疑問。

使用者過去的 Bash 工作流程曾包含手動紀錄，但目前 Go CLI 沒有面向日常使用的 `worklog note` 指令。現有 `worklog capture` 是需要 repository、commit 與完整 JSON stdin 的進階整合 contract，而且每個 commit 只能有一筆 WorklogEvent；它不適合承載與 commit 無關、跨越多個 commits，或在尚未 commit 時產生的思考。

因此，當日即使同時有 commit activity 與使用者的反思，目前每日流程仍只能看見 commit facts。使用者若直接修改產出的 Obsidian 日誌，內容雖能保存在 managed block 外，卻不會進入 `inspect` 的 post-redaction payload，也無法在後續 `day --refresh` 時參與摘要與重新組織。

## Solution

新增面向使用者的 `worklog note` command family，讓使用者能以低摩擦方式記錄 learning、decision、question 或 observation。Manual note 是獨立於 Git commit 的一級工作事件，儲存在 profile 隔離的 Worklog state 中；它不會建立空 commit、不會覆寫 Git note，也不要求當下已經完成 commit。

`worklog note` 優先從目前所在的 registered repository 推導 profile；無法唯一推導時，使用明確且可修復的選擇規則。每筆 note 保存穩定識別碼、事件時間、profile、note type、使用者文字，以及 note type 所允許的 reason 或 evidence。使用者可以選擇把 note 關聯到目前 repository 的某個 commit，但這個關聯只作為本機 provenance，不是保存 note 的必要條件，也不會暴露 commit identifier 給 summarizer。

`worklog inspect` 與 `worklog day` 在同一個 profile、同一個 local date 內收集 commit facts 與 manual notes。兩種來源保持可辨識，不互相覆蓋，再經過相同的 profile-specific redaction boundary 形成版本化的 daily input。當日只要任一來源有 activity，該 profile 即為 active profile。

Daily summarization 同時接收 commit 與 manual facts。Commit facts 主要形成今日重點與完成事項；使用者明確輸入的 decisions、learnings、questions 與 observations 分別進入技術決策、今天學到的事、未解問題與工作筆記。使用者文字是 authoritative fact：summarizer 可以整理、分組與去除明確重複，但不得以從 commit 推測的內容覆寫、扭曲或捏造使用者的理由與學習。

若 AI summarizer 不可用、失敗或不支援新 contract，Built-in fallback 仍必須確定性地保留所有 manual notes，確保反思不會因 provider 問題消失。產出與 refresh 延續既有 preview、confirmation、atomic write 與 managed block 安全邊界；source digest 同時涵蓋 commit facts 與 manual notes。

## User Stories

1. As a developer, I want to record a thought without making a commit, so that useful context is not lost while I am still investigating.
2. As a developer, I want a short `worklog note` command, so that capturing a thought interrupts my work as little as possible.
3. As a developer, I want to record what I learned, so that my journal contains more than a list of completed changes.
4. As a developer, I want to record a decision and its reason, so that I can later understand why I chose one option over another.
5. As a developer, I want to record an unresolved question, so that uncertainty is visible instead of being rewritten as a conclusion.
6. As a developer, I want to record an observation that is not yet a learning or decision, so that partial signals can still be retained.
7. As a developer, I want manual notes to be independent of commits, so that one thought can span several commits or precede all of them.
8. As a developer, I want an optional relationship between a note and a commit, so that I can preserve provenance when the relationship is meaningful.
9. As a developer, I want note capture to avoid creating empty or synthetic commits, so that repository history remains about source changes.
10. As a developer, I want note capture to avoid modifying existing Git notes, so that agent-captured WorklogEvents remain idempotent and conflict-safe.
11. As a developer working inside a registered repository, I want worklog to infer the profile, so that common note capture needs no routing flags.
12. As a developer with exactly one configured profile, I want note capture outside a repository to use that profile, so that I can record a thought from anywhere.
13. As a developer with several possible profiles, I want ambiguous note routing to stop with an actionable message, so that work and personal notes are never silently mixed.
14. As a developer, I want to select a profile explicitly, so that I can intentionally record a note outside its repository.
15. As a privacy-conscious employee, I want every manual note stored under exactly one profile, so that employer and personal reflections remain isolated.
16. As a developer, I want a note timestamp captured automatically, so that it appears on the correct local day without extra input.
17. As a developer, I want profile timezone rules applied to manual notes, so that notes around midnight join the same daily boundary as commits.
18. As a developer, I want an explicit historical timestamp option, so that I can recover a note I forgot to capture earlier without changing the system clock.
19. As a developer, I want malformed or incomplete notes rejected before persistence, so that later inspection does not fail on corrupted data.
20. As a developer, I want each successful note command to return a stable note identifier, so that I can recognize the saved event and refer to it in diagnostics.
21. As a developer, I want note writes to be atomic, so that interruption cannot leave a partially written event.
22. As a developer invoking note capture concurrently, I want each valid note retained exactly once, so that parallel shells cannot overwrite one another.
23. As a developer, I want manual note source files protected with private filesystem permissions, so that reflective content is not made broadly readable.
24. As a developer, I want `worklog inspect` to include commit facts and manual notes, so that I can audit the exact information used for my daily journal.
25. As a developer, I want inspected manual notes to pass through the same denied-literal and secret redaction rules as commit facts, so that the provider boundary remains consistent.
26. As a privacy-conscious employee, I want note identifiers, local storage paths, repository paths and commit hashes excluded from summarizer input, so that provenance metadata does not leak.
27. As a developer, I want commit facts and manual notes to remain distinguishable in the daily input, so that a summarizer does not mistake a personal reflection for completed work.
28. As a developer, I want daily events ordered deterministically by their profile-local occurrence time, so that repeated inspection and rendering are stable.
29. As a developer, I want deterministic tie-breaking for events with identical timestamps, so that source digests do not change between runs.
30. As a developer with commits and notes on the same day, I want both sources included in one profile-specific summary, so that the journal tells a coherent story of the day.
31. As a developer with only manual notes on a day, I want that profile to be considered active, so that reflection-only work can still produce a journal.
32. As a developer with only commits on a day, I want existing journal content and behavior to remain compatible, so that adopting notes does not degrade the current workflow.
33. As a developer active in several profiles, I want each profile's commits and notes summarized separately, so that no cross-profile request or journal is created.
34. As a developer, I want explicit user decisions to take precedence over AI inference, so that the journal preserves my actual reasoning.
35. As a developer, I want explicit learnings to appear under a dedicated section, so that they are retrievable independently from completed work.
36. As a developer, I want unresolved questions to remain questions, so that the summarizer cannot present speculation as fact.
37. As a developer, I want evidence attached to a learning when provided, so that I can remember what changed my mental model.
38. As a developer, I want a decision reason to remain attached to that decision, so that grouping or deduplication does not separate cause from choice.
39. As a developer, I want observations retained even when they cannot be confidently categorized further, so that the system does not force premature conclusions.
40. As a developer, I want duplicate commit and manual descriptions consolidated carefully, so that the journal avoids repetition without losing user-authored meaning.
41. As a developer, I want AI summarization failure to fall back without dropping manual notes, so that provider availability never determines whether my reflection survives.
42. As a developer using Built-in summary mode, I want manual notes rendered deterministically, so that reflection capture works without AI or network access.
43. As a developer using Codex or Claude Code presets, I want new summary fields written in the profile language, so that the journal remains linguistically consistent.
44. As a developer using a custom summarizer, I want a versioned input and output contract, so that I can migrate the adapter deliberately when manual notes are introduced.
45. As a developer, I want the journal preview to show where every note will appear before writing, so that I remain in control of the final artifact.
46. As a developer, I want `day --refresh` to incorporate notes added after the original journal was generated, so that the managed block can evolve safely.
47. As a developer, I want refresh to preserve all bytes outside the managed block, so that unrelated manual edits in Obsidian remain untouched.
48. As a developer, I want the managed source digest to include both event sources, so that refresh and stale-preview checks represent the complete daily input.
49. As a developer, I want no-activity behavior to distinguish truly empty days from note-only days, so that valid reflections are not reported as absent activity.
50. As a developer, I want actionable errors when profile resolution, timestamp parsing, storage, redaction or rendering fails, so that I can repair the workflow without inspecting implementation details.
51. As a maintainer, I want existing version-one WorklogEvents in Git notes to remain readable without rewriting them, so that manual-note support does not require repository migration.
52. As a maintainer, I want the feature covered through the existing external-process acceptance seam, so that tests prove observable CLI behavior rather than internal structure.
53. As a maintainer, I want manual data stored outside the source repository, so that cloning or publishing `worklog` never includes a user's reflections.
54. As a former employee, I want manual notes to remain local to the profile and machine unless I explicitly export them, so that changing jobs does not transport employer context automatically.

## Implementation Decisions

- Add a user-facing `note` command family alongside the existing advanced `capture` contract. `capture` remains commit-scoped and is not widened to append unrelated manual content.
- The shortest non-interactive form is `worklog note <text>`. It records an `observation` unless `--type` selects `learning`, `decision` or `question`.
- Running `worklog note` without text starts a concise guided prompt when interactive. Non-interactive callers may supply the note text through stdin. Exactly one note is created per invocation.
- Supported note types are `observation`, `learning`, `decision` and `question`. All types require non-empty primary text. `learning` may include evidence. `decision` includes a non-empty reason; guided mode asks for it, and the explicit form rejects a decision without one. Other types reject decision-only fields rather than silently ignoring them.
- Provide `--profile` for explicit routing and `--at` for an advanced RFC3339 occurrence time. The default occurrence time is the current instant. Display and daily selection use the selected profile's IANA timezone.
- Resolve profile in this order: an explicit `--profile`; the unique profile owning the current registered repository; the only configured profile. If none or several profiles remain possible, fail before writing and print the exact corrective form using `--profile` or repository registration.
- Permit an optional `--related-commit <revision>` only when the command is run within a registered repository owned by the selected profile. Resolve it to a commit before writing. Store the resolved relationship as private local provenance, but exclude hashes and revisions from inspected or summarized content.
- Introduce a versioned `ManualNote` document distinct from `WorklogEvent`. It contains a schema version, stable identifier, profile name, occurred-at instant, note type, primary text, optional reason, optional evidence and optional commit relationship metadata.
- Store ManualNotes below Worklog's existing state root, partitioned by profile and profile-local date. Use one immutable file per note so concurrent note commands cannot overwrite an existing event and so a damaged write cannot corrupt an entire day's notes.
- Generate note identifiers locally with collision-resistant randomness. Identifiers are operational metadata: return them after capture and include them in local diagnostics, but do not send them to a summarizer or render them into the journal by default.
- Create state directories with owner-only permissions, write note files with owner read/write permissions, and install each file atomically. A failed validation or write leaves no partial note.
- Keep ManualNotes after journal generation and refresh. Source events remain necessary for deterministic regeneration; automatic pruning, archival and synchronization are separate concerns.
- Leave the version-one Git-note WorklogEvent contract unchanged. Existing JSON and legacy line-oriented commit notes remain readable, idempotent and conflict-safe.
- Extend daily collection to gather two independent sources for each profile-local date: registered-repository commit facts and profile-scoped ManualNotes. A ManualNote never substitutes for a commit fact, and a commit fact never overwrites a ManualNote.
- Treat a profile as active when either source contains at least one event. Note-only activity therefore reaches preview and journal generation. Missing repository mappings must not erase otherwise valid ManualNotes; repository configuration problems remain visible through doctor and actionable warnings.
- Define a new version of the post-redaction daily input contract that represents source type explicitly and carries commit facts plus manual note semantics without overloading an outcome field. Preserve deterministic ordering by occurrence time, source type and stable internal identity.
- Apply profile-specific denied literals and common secret-pattern redaction to every user-authored text field before `inspect` output or summarizer invocation. Raw ManualNotes stay local and are never modified by redaction.
- Keep `inspect` as the exact provider-boundary audit command. Its JSON includes the new contract version and redacted event content, while excluding stable note identifiers, source file paths, repository paths, author identities, commit identifiers and raw source metadata.
- Version the summarizer input change explicitly. Update Built-in, Codex and Claude Code adapters together. Custom adapters that reject or mishandle the new version are treated as unavailable and trigger the existing deterministic fallback with an actionable migration message.
- Extend the validated daily summary model with explicit learnings, questions and observations while retaining focus, completed topics, decisions, next steps and technologies. Decision reasons and learning evidence remain attached to their parent items.
- Instruct AI presets that user-authored ManualNotes are authoritative statements. They may group semantically equivalent items and improve organization in the profile language, but must not invent learnings, answer unresolved questions, manufacture evidence, or replace an explicit reason with an inferred one.
- When a commit fact and ManualNote appear duplicative, preserve the ManualNote's wording and semantic type. Deduplication may remove repeated completion prose, but may not remove a user-authored decision, learning, question or observation from its dedicated output section.
- Extend Built-in summarization deterministically: commit outcomes populate focus and completed work; decisions from either source populate technical decisions; learning notes populate learnings; question notes populate unresolved questions; observation notes populate work notes; note evidence and reasons are retained verbatim after text cleaning.
- Render managed sections in this order: title, 今日重點, 完成事項, 技術決策, 今天學到的事, 未解問題, 工作筆記, 後續行動. Empty reflection-only sections may be omitted to keep commit-only journals compatible; existing required sections retain current empty-state behavior.
- Compute the managed source digest from the complete post-redaction daily input, including commit and manual events. Adding a note after initial generation therefore produces a changed refresh diff.
- Preserve the existing `day` safety model: complete preview before confirmation, one independent plan per active profile, explicit confirmation, atomic initial write, stale-preview detection and managed-block-only refresh.
- Continue to keep profile requests separate. Manual notes from one profile must never influence another profile's activity detection, provider input, source digest, preview or destination.
- Extend `doctor` to validate the ManualNote state root, directory permissions and readability without opening or printing note contents.
- Extend root and task-oriented help with concise `note` examples, supported types, profile resolution rules, privacy behavior, historical capture and the next `inspect` command.
- Do not automatically invoke an AI provider during note capture. Capture remains fast, offline and deterministic; summarization occurs only through the existing `inspect` and `day` workflows.
- The temporary specification directory under `docs/` is a development artifact and may be removed after the feature is implemented and its durable user-facing documentation has been incorporated into the normal project documentation.

## Testing Decisions

- Use one primary test seam: compile the CLI and execute it as an external process against isolated temporary config home, state home, Git repositories, profiles, fake summarizers and journal destinations. This is the same acceptance seam already used by setup, capture, inspect, day, redaction, presets and refresh.
- Add an end-to-end acceptance case that creates a commit WorklogEvent and several ManualNotes on the same profile-local date, then verifies that `inspect` exposes both source types and `day` renders completed work, decisions, learnings, questions and observations in their correct sections.
- Assert observable behavior only: command arguments, stdin, exit status, stdout/stderr, persisted note documents, inspected JSON, provider stdin, journal preview, confirmation behavior and final Markdown. Avoid tests coupled to private helper functions or directory traversal implementation.
- Test the shortest positional-text note form, every note type, guided input, stdin input, explicit profile routing, inferred repository routing, single-profile fallback routing and ambiguous routing failure.
- Test field validation, including blank primary text, decision without reason, unsupported type, incompatible flags, invalid timestamp, unknown profile, unregistered repository and invalid related commit. Every rejected command must leave state unchanged.
- Test timezone boundaries with notes immediately before and after local midnight and with an explicit historical occurrence time. Selection must use profile-local dates, matching commit collection semantics.
- Test atomicity and concurrency by launching multiple note commands into the same profile and date, then verifying every successful note exists exactly once and no temporary or partial file is collected.
- Test filesystem protection where the platform supports POSIX permissions: state directories remain owner-only and ManualNote files remain owner-readable and owner-writable only.
- Test `inspect` as the privacy boundary using profile denied literals, path-like text, emails, token fixtures, hashes and source metadata. Raw stored notes should retain user input while inspected JSON and fake summarizer stdin contain only redacted allowed fields.
- Test that note identifiers, filenames, state paths, commit hashes, repository paths and profile author emails never appear in inspected output, provider stdin or rendered Markdown.
- Test deterministic event ordering and tie-breaking by repeating `inspect` and `day` against identical state and asserting byte-identical payloads and stable digests.
- Test note-only activity: a profile with ManualNotes and no commits for the date must receive a preview and be eligible for journal writing rather than producing the no-activity message.
- Test commit-only regression behavior: existing version-one JSON Git notes, legacy line-oriented notes and commit-subject fallback remain readable and render the established sections without invented reflection content.
- Test multi-profile isolation with same-day commits and notes in work and personal profiles. The fake summarizer must receive separate requests, and each destination must contain only its own profile's events.
- Test AI preset instructions and schema validation using fake Codex and Claude Code executables. Valid new summary output is accepted; wrong versions, missing required fields, invented unknown fields, malformed JSON, timeout and nonzero exit all use deterministic fallback.
- Test a legacy or incompatible custom summarizer against a day containing ManualNotes. The fallback journal must still include every explicit note, and the user must receive an actionable contract-version message without provider stderr or private content.
- Test deterministic Built-in rendering for each ManualNote type, attached decision reasons, attached learning evidence, duplicate text across commit and manual sources, and bounded public technology normalization.
- Test initial journal preview and refusal semantics: no file or destination directory is created before affirmative confirmation.
- Test refresh after adding a ManualNote to an existing journal. The managed block and digest must change, the preview must show the reflection section, and all bytes outside the managed block must remain byte-for-byte identical.
- Test stale-preview protection by changing a journal or ManualNote source after preview but before replacement; worklog must refuse to install content derived from stale state.
- Test doctor without reading note contents. Permission or state-root failures must produce repair actions while note text, identifiers and configured private values remain absent from diagnostics.
- Prefer narrower unit tests only for pure schema validation, deterministic ordering, redaction or rendering edge cases that would make the external-process acceptance suite needlessly opaque. They supplement rather than replace the primary seam.

## Out of Scope

- Automatically inferring a user's learning, motivation, confidence or decision rationale from commits, diffs or conversation history.
- Turning every commit into a manual note or every manual note into a Git note.
- Creating empty, synthetic or metadata-only commits to act as note storage.
- Replacing the existing `gen-commit` integration or changing its responsibility for commit-scoped WorklogEvents.
- Editing, deleting, pruning or archiving ManualNotes in the first delivery. Corrections can be recorded as a new note until a separately designed mutation workflow exists.
- Automatically synchronizing ManualNotes across computers, pushing them to Git, uploading them to cloud storage or exporting them with employer data.
- Reading arbitrary Obsidian notes as summarizer input or using manual text outside the managed block as an implicit Worklog event.
- Automatically answering unresolved questions during daily summarization.
- Adding confidence scoring, mood tracking, time tracking, project management, reminders or weekly knowledge extraction.
- Backfilling historical learnings or rationales that the user never recorded.
- Adding a GUI, menu-bar application, Raycast extension, mobile capture interface or editor plugin. These may call the CLI contract in future work.
- Supporting non-text attachments such as screenshots, audio, PDFs or links with fetched remote content.
- Changing the journal destination layout or weakening preview, confirmation, profile isolation, redaction, managed-block or atomic-write guarantees.

## Further Notes

- The product distinction is intentional: `capture` records commit-scoped facts for integrations; `note` records user-authored reflection. Both contribute to a daily journal, but neither is an alias for the other.
- A manual note is authoritative about what the user thought, not proof that a technical claim is universally true. Learning evidence and unresolved questions preserve this distinction.
- Keeping raw notes in local state allows deterministic refresh without placing private reflections in the source repository. The Obsidian journal remains the human-facing durable artifact after generation.
- Manual note support intentionally expands beyond the first portable release's original Git-only activity-source boundary. It should therefore be delivered as an explicit contract evolution with updated help and adapter migration behavior.
- The test seam described above is the expected implementation acceptance boundary: `worklog note` creates state, `worklog inspect` exposes the redacted combined daily input, and `worklog day` previews or writes the resulting managed journal.
- This specification is intentionally stored in a temporary folder under `docs/` at the user's request. Remove that folder after implementation, verification and durable documentation updates are complete.
