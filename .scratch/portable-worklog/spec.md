# 可攜式、與 Agent 及 AI Provider 解耦的工作日誌 CLI

Status: ready-for-agent

## Problem Statement

使用者目前依靠一支個人化 Bash script，從 Git commits、Git notes 與手動紀錄產生 Obsidian 工作日誌。現有流程已能運作，但綁定單一使用者路徑、廣泛掃描本機 repositories、依賴 Claude CLI，並讓 Bash 同時負責資料蒐集、隱私遮蔽、模型提示、格式驗證與 Markdown 寫入。這使工具難以跨公司、跨電腦、跨 AI agent 與跨 summarizer 使用，也提高公司資料和個人 side project 被混合處理的風險。

使用者需要一套可以由 Git 管理並帶到新環境的通用流程。平常使用時不應要求使用者記得額外記錄；正常執行中央管理的 `gen-commit` 後，工作事實就應自動保存。每天理想上只需要執行一次 `worklog day`，工具便能依 profile 分流、摘要、預覽並寫入正確日誌位置。

## Solution

建立一個以 Go 實作、可編譯為單一執行檔的 `worklog` CLI。核心以版本化結構資料與 Git notes 保存 commit 階段的工作事實，以 profile 和 repository allowlist 隔離公司與個人資料，以 command adapter 連接任意 summarizer，並由核心程式驗證 summary schema、確定性產生 Markdown。

Agent integration 只依賴穩定的 `worklog capture` contract。中央 `gen-commit` skill 可繼續透過 Git 與 symlink 分享給 Codex、Claude Code 或其他可執行 shell command 的 agent；公司專用 commit skills 留在各自公司 repository，只選擇性呼叫相同 contract，不進入通用核心。

首次使用由低理解成本的互動式 setup 完成。Setup 從目前 Git repository 與作業系統自動取得 author email、timezone 與 language；使用者只需從有用途說明的 Work、Personal 或 Custom profile 選項中選擇，確認可編輯的 repository 與 journal destination 預設值，並選擇摘要方式。日常使用只需正常 commit，然後執行 `worklog day`。若同一天有多個 profile 的活動，工具自動分開產生多份日誌，不把資料放入同一次 summarizer request。重新執行時只 refresh 受管理的生成區塊，保留使用者人工補充。

## User Stories

1. As a developer, I want to install worklog as a single executable, so that I do not need to manage a language runtime.
2. As a developer changing computers, I want to restore the tool from Git-managed source and configuration templates, so that setup is repeatable.
3. As a developer joining a new company, I want to create a fresh work profile, so that previous employer data is not carried into the new environment.
4. As a side-project developer, I want a personal profile, so that personal work remains separate from employer work.
5. As a developer, I want each repository registered to exactly one profile, so that routing is deterministic.
6. As a developer, I want worklog to reject ambiguous repository mappings, so that data is never silently mixed.
7. As a developer, I want `worklog day` to discover active profiles automatically, so that I do not need to type a profile every day.
8. As a developer active in multiple profiles on one day, I want separate journal generation per profile, so that each data boundary remains isolated.
9. As a developer, I want repository discovery to use explicit allowlists, so that unrelated repositories are not scanned.
10. As a developer using different Git email addresses, I want profiles to recognize configured author identities, so that all of my commits are collected.
11. As a developer, I want commit-time work facts recorded automatically, so that I do not need to remember `worklog note`.
12. As a developer using the central `gen-commit` skill, I want every successful commit to invoke the same capture contract, so that all supported agents behave consistently.
13. As a developer using a company-specific commit skill, I want it to call the generic capture contract without becoming part of the portable tool, so that company workflow remains local.
14. As a developer committing without an AI agent, I want worklog to fall back to Git metadata, so that journal generation still works.
15. As a developer, I want captured events stored as versioned structured Git notes, so that they can be validated and migrated.
16. As an existing user, I want current version-one Git notes to remain readable, so that migration does not discard recent records.
17. As a developer, I want one worklog note per commit to be idempotent, so that retries do not create duplicates.
18. As a developer, I want conflicting worklog note writes to stop instead of overwrite, so that captured facts are not silently lost.
19. As a privacy-conscious employee, I want summarizers to receive structured and redacted facts by default, so that source code and internal paths are not exposed.
20. As a privacy-conscious employee, I want profile-specific redaction rules, so that each employer's identifiers are handled independently.
21. As a developer, I want hashes, repository paths, author emails, secrets, and raw code excluded from summarizer input, so that the provider sees only necessary facts.
22. As a developer, I want to inspect the exact redacted payload before it is sent, so that I can audit data boundaries.
23. As a developer, I want to choose a summarizer per profile, so that company and personal work can follow different policies.
24. As a developer, I want to use Claude, another command-line AI, a local model, or no AI, so that the workflow is provider-independent.
25. As an adapter author, I want a stable JSON stdin/stdout contract, so that a new summarizer can be added without changing the core.
26. As a developer, I want summarizer failures to fall back to deterministic output, so that the daily journal is still available.
27. As a developer, I want invalid summarizer output rejected by schema validation, so that malformed Markdown is never written.
28. As a developer, I want Markdown and YAML rendered by the core, so that headings, frontmatter and list types stay stable.
29. As a developer, I want technology names deduplicated and bounded, so that frontmatter remains useful.
30. As a developer, I want output paths, tags, language and templates configurable per profile, so that the same binary fits different journals.
31. As an Obsidian user, I want daily notes organized by year and month, so that the existing journal convention remains supported.
32. As an Obsidian user, I want generated content marked as a managed block, so that manual additions remain outside the refresh boundary.
33. As an Obsidian user, I want refresh to replace only generated content, so that rerunning a date cannot erase my notes.
34. As a developer, I want a preview or diff before the journal is written, so that I remain in control of mutations.
35. As a developer, I want journal writes to be atomic, so that interruption cannot leave a partial note.
36. As a developer, I want `worklog day` with no date to use the configured local date and timezone, so that daily usage is effortless.
37. As a developer, I want to generate or refresh a historical date explicitly, so that missed or corrected journals can be recovered.
38. As a new user, I want an interactive setup command, so that I do not need to author configuration manually.
39. As a new user, I want a doctor command, so that missing Git, profile mappings, journal destinations and summarizer commands are diagnosed clearly.
40. As a developer, I want setup to leave secrets out of configuration files, so that credentials stay in environment variables or system credential storage.
41. As a maintainer, I want configuration and state stored outside the source repository, so that cloning the code never imports another environment's data.
42. As a maintainer, I want portable release binaries for supported platforms, so that users can install without a Go toolchain.
43. As a maintainer, I want the current Bash implementation preserved until behavior parity is demonstrated, so that migration is reversible.
44. As a maintainer, I want the executable switch to happen only after acceptance tests pass, so that the existing daily workflow is not disrupted.
45. As a former employee, I want tool code to be portable while employer profiles and notes remain behind, so that legal and privacy boundaries are respected.
46. As a new user, I want setup to infer facts the computer already knows, so that I only answer questions that require a human choice.
47. As a new user, I want profile choices described with concrete Work, Personal and Custom examples, so that I do not need to understand the configuration model first.
48. As a subscriber to a supported AI coding CLI, I want worklog to reuse my provider-managed Codex or Claude Code session, so that worklog never asks me to copy an API key.
49. As a user seeking help, I want task-oriented command help with examples and exact recovery actions, so that I do not need to inspect configuration files.
50. As a user, I want to change an existing profile's language, timezone, journal destination and daily summary preference without editing JSON, so that settings can evolve safely after setup.

## Implementation Decisions

- Implement the core as a Go CLI and distribute platform-specific single binaries. Runtime behavior must not depend on an installed Go toolchain, Python environment, or modern Bash.
- Keep source code, schemas, prompts, generic integrations, examples and tests under Git. Store user profiles, caches, state, credentials and captured work data outside the source repository.
- Provide `setup`, `profile`, `repo`, `capture`, `inspect`, `day` and `doctor` command families. Optimize the everyday path so that `worklog day` requires no flags after setup.
- Keep setup progressively disclosed. Prompt for a Work, Personal or Custom profile; show the current Git root and a profile-specific journal folder as editable defaults; infer author email, IANA timezone and language; ask for an inferred value only when it cannot be resolved safely; then show one final confirmation before writing.
- Provide `settings [profile]` as the guided path for changing an existing profile and `profile set` as the scriptable advanced contract. Validate the complete candidate config before atomic replacement; never move existing journals as a side effect of changing their destination.
- Treat `work`, `personal` and custom profile names as routing labels with identical capabilities. They do not silently change privacy or collection behavior.
- Present summarization as a user goal rather than a raw command. Offer deterministic built-in output by default and, once the command adapter is available, offer executable-discovered Codex and Claude Code presets. The first preset release has no user-facing Custom command option. A preset simply invokes the provider CLI and lets it reuse its own session; worklog never probes, imports, copies or stores provider credentials, and falls back deterministically when invocation fails.
- Keep `capture`, author identities, repository mappings and raw summarizer commands available as advanced contracts, but omit them from the shortest first-run explanation. Root help leads with setup, daily generation, adding the current repository, settings and diagnostics; detailed help remains available for every command and subcommand.
- Treat a profile as a routing and privacy boundary. Each profile owns repository allowlists, author identities, timezone, journal destination, redaction policy, summarizer and rendering preferences.
- Require each registered repository to resolve to exactly one profile. Reject overlapping mappings during setup and doctor checks.
- When a date contains activity from multiple profiles, process each profile in an independent collection, redaction, summarization, preview and write cycle. Never construct a cross-profile summarizer payload.
- Use Git commits as the durable activity index and a dedicated Git notes ref as the structured event store. Exclude the notes ref itself from activity traversal.
- Define a versioned WorklogEvent JSON schema. Require an outcome and allow supported optional facts for approach, explicit decisions and reasons, explicit next steps, public technologies and capture source metadata.
- Preserve read compatibility with the current line-oriented version-one note format. New writes use the JSON schema; migration does not rewrite existing notes automatically.
- Make capture idempotent when the same event already exists. Reject a different event for the same commit unless a future explicit edit workflow is used.
- Keep `gen-commit` in the separately Git-managed central skills collection. Its generic integration reads the live capture contract from CLI help and records each successful commit without embedding provider or profile logic.
- Keep company-specific commit skills outside this repository. They may call the public capture contract when installed.
- Provide an optional generic post-commit fallback that records metadata when no agent integration is active. It cannot invent decisions or rationale.
- Define summarization as a command adapter protocol. The core sends a redacted DailyFacts JSON document on stdin and accepts a DailySummary JSON document on stdout; a nonzero exit, timeout, empty result or invalid schema triggers deterministic fallback.
- Keep provider credentials outside worklog configuration. Provider adapters may read environment variables or platform credential storage.
- Make structured-only privacy the default. Summarizer input excludes hashes, raw commit bodies unless explicitly allowed, repository paths, author emails, raw diffs and source code.
- Apply profile-specific literal redaction and common secret-pattern checks before invoking an adapter. Redaction is defense in depth, not permission to send unrestricted material.
- Make the exact post-redaction DailyFacts payload available through an inspect command without invoking a provider or writing a journal.
- Validate DailySummary structurally before rendering. The supported summary model contains focus statements, topic-grouped completed items, decisions, next steps and public technology names.
- Render YAML frontmatter and Markdown deterministically in the core. Providers never emit final Markdown or choose filesystem paths.
- Support a default Taiwan Traditional Chinese daily template compatible with the current journal headings and property types, while allowing profile-level language and template selection.
- Mark generated note content with explicit managed-block boundaries and a source digest. Refresh updates only that block and preserves content outside it.
- Default to non-destructive behavior: preview changes, refuse ambiguous destinations, and use atomic replacement after confirmation. Refresh is the explicit regeneration operation.
- Derive a day's activity using configured timezone and documented commit timestamp semantics. Deduplicate commits reachable from multiple refs and ignore merge commits by default.
- Do not sync or push worklog Git notes automatically. Any future export or remote synchronization requires an explicit operation and separate privacy design.
- Supply setup and doctor diagnostics that explain actionable fixes without exposing secrets or captured content.
- Use the current Bash workflow as a behavioral reference and migration source. Switch the user's executable link only after the Go CLI passes the agreed acceptance seam against representative fixtures.
- Keep the repository generic: no employer names, real Vault paths, private repository identifiers, credentials, production notes or company-specific adapters enter tracked history.

## Testing Decisions

- Use one primary acceptance seam: execute the compiled `worklog` CLI as an external process against isolated temporary homes, Git repositories, profiles, journal destinations and fake summarizer commands.
- Assert observable behavior rather than internal functions. Tests inspect process exit status, stdout and stderr contracts, Git notes, adapter payloads, generated files and preserved manual content.
- Cover setup, repository registration, capture, idempotency, conflict rejection, profile routing, multi-profile separation, payload inspection, redaction, provider success, provider timeout or failure, invalid summary fallback, deterministic rendering, historical dates and refresh.
- Cover setup inference and fallback prompts with isolated Git config, locale and timezone fixtures. Assert that pressing Enter accepts repository and journal defaults, custom values remain possible, cancellation writes nothing, and help works through both `<command> --help` and `help <command> [subcommand]` forms.
- Verify privacy at the adapter boundary by using a fake summarizer that fails if it receives a hash, unregistered repository data, path, email, configured denied literal or fixture secret.
- Verify profile isolation with same-day commits in work and personal fixture repositories. The fake adapter must receive two separate requests and each journal destination must contain only its own facts.
- Verify managed refresh by adding manual content outside the generated block, rerunning the date and asserting byte-for-byte preservation of the manual content.
- Verify compatibility by loading representative version-one notes produced by the existing Bash script and new JSON notes in the same fixture day.
- Verify fallback by running with no summarizer, an unavailable command, nonzero exit, empty stdout, malformed JSON and schema-invalid JSON; every case must produce deterministic output or a clear non-writing error according to the command contract.
- Verify atomic safety by injecting a write failure before replacement and asserting the previous journal remains intact.
- Verify configuration portability by running all acceptance cases with environment-controlled config and state roots, without referring to the developer's real home directory.
- The existing Bash script provides behavioral examples but no reusable test seam. New acceptance fixtures establish the first project prior art; narrower unit tests are optional only where they clarify pure schema or rendering edge cases without duplicating CLI assertions.

## Out of Scope

- Moving any employer-specific commit skill into the portable repository.
- Combining different profiles into one daily summary, one provider request or one journal note.
- Uploading source code, raw diffs or unrestricted commit history to a summarizer.
- Automatically pushing, fetching or synchronizing the dedicated Git notes ref.
- Reconstructing decisions or rationale that were never captured.
- Backfilling structured notes for historical commits automatically.
- Supporting non-Git activity sources in the first release.
- Providing a graphical application, web dashboard, cloud account or hosted synchronization service.
- Shipping first-party implementations for every AI provider. The stable command adapter contract is the extension point.
- Replacing the current Bash command before compatibility and acceptance checks pass.
- Migrating or transporting employer data when changing jobs.

## Further Notes

- The portable asset is the tool and its contracts, not the collected work data. A new employer receives a fresh local profile and repository mapping.
- The central skills repository remains the distribution source for `gen-commit` across agents through symlinks. The worklog repository owns only the CLI and the generic capture contract.
- The repository should begin private. Public release can be reconsidered after tracked history and fixtures are confirmed free of proprietary identifiers.
- The preferred end-user flow is: install once, run interactive setup once, commit normally, then run `worklog day` at the end of the day.
