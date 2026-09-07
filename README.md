# worklog

`worklog` is a portable command-line tool for recording and rendering work journals.

Building and testing require Go 1.23 or newer. The compiled CLI does not require a
Go installation at runtime.

## Build

From the repository root, build the CLI with:

```sh
go build -o ./bin/worklog ./cmd/worklog
```

## Acceptance tests

Run the external-process acceptance suite with:

```sh
go test ./...
```

The suite compiles its own CLI executable and runs it with isolated temporary home,
configuration, state, Git repository, and journal destination directories.

## Configuration

Run the guided setup once from the first Git repository:

```sh
worklog setup
```

Choose a Work, Personal, or Custom profile. The current Git root and a
profile-specific journal folder are editable defaults. Setup reads the effective
author email from Git, detects timezone and language, and shows every resolved value
for one final confirmation. If Git has no author email or the timezone cannot be
detected safely, setup asks only for the missing value. Daily summary choices are
Built-in, Codex, and Claude Code. Built-in needs no AI or network; the other choices
appear only when their executable is available and reuse the CLI's current session.
Worklog never probes or stores provider credentials.

Discover commands and examples through either help form:

```sh
worklog --help
worklog setup --help
worklog settings --help
worklog profile add --help
worklog profile set --help
worklog help repo
worklog help repo add
```

Use the configuration command families to add, inspect, and validate entries:

```sh
worklog profile add personal --timezone Asia/Taipei --language zh-TW --journal-destination /path/to/journal
worklog profile list
worklog profile show personal
worklog profile validate

worklog settings personal
worklog profile set personal --language en-US --timezone Asia/Taipei
worklog profile set personal --journal-destination ~/Documents/worklog/personal
worklog profile set personal --daily-summary codex
worklog profile set personal --deny-literal private-project --deny-literal client-code

worklog author add personal developer@example.com
worklog author list personal
worklog author validate personal

worklog repo add personal /path/to/repository
worklog repo list
worklog repo show /path/to/repository
worklog repo validate
```

`settings [profile]` is the guided update path. With one profile it selects it
automatically; with several profiles it asks which one to change. `profile set`
offers the same language, IANA timezone, journal-folder, and daily-summary update
boundary for scripts. Both validate the complete profile before atomically replacing config.
Changing the journal folder affects future output only and never moves, copies, or
deletes existing journals.

`repo show` resolves a registered repository to the profile, timezone, language, and
journal destination that daily commands should use. A repository can belong to only
one profile. Exact duplicate paths and overlapping parent or child mappings are
rejected.

## Generate or refresh a daily journal

After setup, daily use needs no profile flags:

```sh
worklog day
worklog day 2026-09-02
worklog day 2026-09-02 --refresh
```

`day` discovers every profile with activity, using each profile's current local date
when no date is passed. Each profile gets its own post-redaction `DailyFacts`, summary
request, preview, confirmation, and journal destination; profile facts are never
combined. Profiles without registered repositories or author identities stop before
any provider invocation or write and report the command needed to repair the mapping.
Built-in creates a deterministic summary. Codex, Claude Code, and an advanced
summarizer command all use the same stdin/stdout boundary and must return a
schema-valid `DailySummary` JSON object on stdout. Missing, timed-out, failed, empty,
malformed, or invalid commands fall back deterministically without exposing provider
stderr. The core prints the complete Markdown preview before asking for confirmation.
Only `y` or `yes` writes. The core chooses the destination as
`<journal-destination>/<YYYY>/<MM>/<YYYY-MM-DD>.md`; configured providers never
choose a path.

The rendered Obsidian note uses YAML date, string, and list properties, bounded and
deduplicated public technology names, Taiwan Traditional Chinese headings, and an
explicit managed block with a DailyFacts source digest. A confirmed first write is
installed atomically. Passing `YYYY-MM-DD` previews and generates that historical
date when it does not exist. Existing notes are refused unless `--refresh` is
explicit. Refresh validates exactly one managed block, displays its diff, preserves
all bytes outside the block, rechecks the previewed file, and atomically replaces it
only after confirmation.

## Inspect daily facts

Inspect one profile's facts for its current local date, or pass a historical date:

```sh
worklog inspect personal
worklog inspect personal 2026-09-02
```

Collection scans only the profile's registered repositories. It matches author
identities and uses commit timestamps in the profile timezone, deduplicates commits
reachable from multiple refs, and ignores merge commits and `refs/notes/worklog`.
The JSON output includes structured worklog notes and commit-subject fallback facts,
but excludes hashes, repository paths, author emails, commit bodies, diffs, source
code, and capture source metadata. Before output, it replaces profile-specific
denied literals and common secret, token, hash, path, email, raw diff, and fenced-code
patterns. This is the exact post-redaction `DailyFacts` boundary used by summarizer
adapters; `inspect` does not invoke one or write a journal.

## Diagnose configuration

Run the non-destructive diagnostic before daily generation when an environment or
mapping has changed:

```sh
worklog doctor
```

Doctor checks Git, config and state roots, required profile settings, authors,
non-overlapping registered repositories, Git notes access, journal destinations, and
summary executables. Every failure includes a repair action and produces a non-zero
exit status. It never reads captured note content, invokes a provider, probes provider
credentials, or prints configured paths, commands, author emails, denied literals, or
captured facts.

## Advanced command summarizers

Guided setup and settings expose only Built-in, Codex, and Claude Code. Advanced
configuration retains the raw command contract without adding a Custom UI option:

```sh
worklog profile add local --timezone Asia/Taipei --language zh-TW \
  --journal-destination ~/Documents/worklog/local \
  --summarizer-command '/path/to/summary-adapter --format json'
```

Worklog executes the command from its state directory, writes exactly the inspected
post-redaction `DailyFacts` JSON to stdin, and accepts only one strict
`DailySummary` JSON object on stdout. The supported document contains `version`,
`focus`, topic-grouped `completed` items, `decisions`, `next_steps`, and canonical
public `technologies`; it cannot select a journal path or return final Markdown.
Credentials must come from the command environment or system credential storage.
The default timeout is two minutes and can be overridden with
`WORKLOG_SUMMARIZER_TIMEOUT` using a Go duration such as `30s`.

Configuration and runtime state stay outside the source repository. Defaults follow
the platform user configuration directory and XDG state directory conventions. Set
`WORKLOG_CONFIG_HOME` or `WORKLOG_STATE_HOME` to replace either application root;
tests use these overrides to remain isolated from real user data.
