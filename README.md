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

Run the interactive setup once from the first Git repository to register:

```sh
worklog setup
```

The setup records a profile name, author email, repository, IANA timezone, language,
journal destination, and an optional summarizer command. Credentials are never
requested; summarizer credentials must come from environment variables or system
credential storage.

Use the configuration command families to add, inspect, and validate entries:

```sh
worklog profile add personal --timezone Asia/Taipei --language zh-TW --journal-destination /path/to/journal
worklog profile list
worklog profile show personal
worklog profile validate

worklog author add personal developer@example.com
worklog author list personal
worklog author validate personal

worklog repo add personal /path/to/repository
worklog repo list
worklog repo show /path/to/repository
worklog repo validate
```

`repo show` resolves a registered repository to the profile, timezone, language, and
journal destination that daily commands should use. A repository can belong to only
one profile. Exact duplicate paths and overlapping parent or child mappings are
rejected.

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
code, and capture source metadata. `inspect` does not invoke a summarizer or write a
journal.

Configuration and runtime state stay outside the source repository. Defaults follow
the platform user configuration directory and XDG state directory conventions. Set
`WORKLOG_CONFIG_HOME` or `WORKLOG_STATE_HOME` to replace either application root;
tests use these overrides to remain isolated from real user data.
