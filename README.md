# worklog

worklog turns your Git commits into a daily work journal.

At the end of the day, run `worklog day`. It gathers what you committed, shows you
the journal it would write, and saves it as a Markdown file once you say yes. The
journals work well in Obsidian, and they stay on your own computer.

```text
$ worklog
  work   2026-10-27   2 commits   not written yet

Next: worklog day
```

## How it works

1. **You commit as usual.** Every commit you make in a registered repository
   counts. Its subject line becomes an entry in the journal.
2. **Optionally, your AI agent adds detail.** When you commit with an agent such as
   Claude Code or Codex, it can record what you did, how, and why. worklog stores
   this as a Git note on the commit, so the journal gets richer entries. See
   [Recording more detail](#recording-more-detail).
3. **You run `worklog day`.** worklog writes one journal per profile per day, for
   example `~/Documents/worklog/work/2026/2026-10/2026-10-27.md`.

Profiles keep different parts of your life apart. A typical setup has a `work`
profile and a `personal` profile, each with its own repositories, email, timezone,
language, and journal folder. Their journals never mix.

## Install

### From a release

Releases include macOS and Linux builds for Apple silicon / arm64 and for Intel /
amd64. You don't need Go installed.

1. Download the build for your computer and the checksum file. The repository is
   private, so use the GitHub CLI with an account that can read it:

   ```sh
   version=v0.1.0
   platform=darwin_arm64   # or darwin_amd64, linux_amd64, linux_arm64
   gh release download "$version" --repo Vickyciou/worklog \
     --pattern "worklog_${version}_${platform}" --pattern SHA256SUMS
   ```

2. Check that the download is intact. The line for your file must end in `OK`:

   ```sh
   shasum -a 256 -c SHA256SUMS --ignore-missing
   ```

3. Install it as `worklog` somewhere on your `PATH`:

   ```sh
   install -m 755 "worklog_${version}_${platform}" ~/bin/worklog
   worklog version
   ```

   If you downloaded the macOS file with a browser instead, also run
   `xattr -d com.apple.quarantine ~/bin/worklog`.

To upgrade, repeat these steps with the new version. Your settings, notes, and
journals are stored separately and stay as they are.

### From source

You need Go 1.23 or newer:

```sh
go build -o ./bin/worklog ./cmd/worklog
```

## First-time setup

Go to a Git repository you work in and run:

```sh
cd ~/code/my-project
worklog setup
```

Setup asks a few questions, and most have a ready answer you can accept with
Enter:

- **Profile.** Work, Personal, or a name you choose.
- **Repository.** The repository you're in.
- **Journal language.** Your system language. It sets the journal headings, and
  the language AI summaries are written in.
- **Journal folder.** `~/Documents/worklog/<profile>`.
- **Daily summary.** How each journal gets written. See
  [Choosing a daily summary](#choosing-a-daily-summary).

Your email comes from Git and your timezone from the system. Setup asks for them
only when it can't find them. You see everything once more before anything is
saved.

## Every day

### See where you are

```sh
worklog
```

Running `worklog` on its own shows today's commits and the journal state for each
profile. The last line tells you what to run next. It only reads, so it's safe to
run any time.

### Write today's journal

```sh
worklog day
```

For each profile with commits today, you see a one-line overview and then the full
journal. Type `y` to save it. Anything else leaves it unwritten, and you can run
`worklog day` again later.

A journal looks like this:

```markdown
---
date: 2026-10-27
type: worklog-daily
tech: [Go]
tags: [工作日誌]
---

<!-- worklog:managed:start source-digest=sha256:… -->
# Work Log - 2026-10-27

## Highlights

- Added login retry with backoff
- Fix token refresh race

## Completed

### Work

- Added login retry with backoff
  - Approach: Retry three times with exponential backoff
- Fix token refresh race

## Decisions

- Cap retries at three (reason: Avoid locking out accounts)

## Next Steps

- Add retry metrics

<!-- worklog:managed:end -->
```

Headings are in Traditional Chinese when the profile language starts with `zh`
(for example `zh-TW`), and in English otherwise. The `tags` line stays the same in
both so your Obsidian searches keep working.

### Commit more later? Run it again

Running `worklog day` again on the same day shows only what changed, with `+` for
added lines and `-` for removed ones, and asks before updating:

```text
  - Add login retry
  - Fix token refresh race
+ - Add retry metrics
```

If nothing changed, it says the journal is already up to date. Anything you write
outside the two `worklog:managed` marker lines is yours. worklog never touches it.

### A past day, or a rebuild

```sh
worklog day 2026-10-20            # write or update a past day
worklog day 2026-10-20 --refresh  # rebuild even though no commits changed
```

Use `--refresh` after changing a profile's language or daily summary, to rewrite
an existing journal with the new setting.

## Choosing a daily summary

| Choice | What it does | Needs |
| --- | --- | --- |
| **Built-in** | Lists your work as recorded, with no rewriting. | Nothing. No AI and no network. |
| **Claude Code** | Asks Claude Code to write the journal in the profile's language. | The `claude` CLI, already signed in. |
| **Codex** | Asks Codex to write the journal in the profile's language. | The `codex` CLI, already signed in. |

worklog reuses the CLI's existing sign-in and never stores or checks your
credentials. If the AI isn't available, fails, or returns something unusable,
worklog falls back to Built-in for that run and tells you why.

The AI receives only a cleaned-up list of your work. It never sees code, diffs,
commit hashes, file paths, or email addresses. To see exactly what it receives, run
`worklog inspect`.

## Changing your setup

| I want to… | Run |
| --- | --- |
| Add the repository I'm in | `worklog repo add` |
| Add a repository to a specific profile | `worklog repo add ~/code/api --profile work` |
| Change language, timezone, folder, or summary | `worklog settings` |
| See a profile's settings | `worklog profile show work` |
| Add another profile, such as personal | `worklog profile add` |
| Add another email for a profile | `worklog author add work me@company.example` |
| List the repositories in a profile | `worklog repo list work` |

With only one profile, worklog uses it automatically. With several, it asks which
one you mean, or takes `--profile`.

A repository belongs to exactly one profile. Moving the journal folder only
affects new journals. Existing ones stay where they are.

### Keeping private words out

If some words shouldn't appear in journals or reach an AI, such as a client name or
an internal codename, list them for the profile:

```sh
worklog profile set work --deny-literal client-codename --deny-literal internal-host
```

worklog replaces them before anything is summarized. It also removes common
secrets, tokens, paths, and emails on its own. `profile show` only shows how many
words are hidden, never the words themselves.

## Recording more detail

A commit subject alone says what changed. A note can add how you did it, why you
made a decision, and what's next. Agents and scripts record notes with
`worklog capture`:

```sh
echo '{"version":1,"outcome":"Added login retry with backoff",
       "decisions":[{"decision":"Cap retries at three","reason":"Avoid locking out accounts"}],
       "next_steps":["Add retry metrics"]}' \
  | worklog capture . HEAD
```

`worklog capture --help` lists every field. Recording the same note twice is
harmless. A different note for a commit that already has one is refused, so
nothing gets overwritten. The `gen-commit` skill does this automatically after each
commit it makes.

## When something looks wrong

```sh
worklog doctor     # finds setup problems and tells you how to fix each one
worklog inspect    # shows the facts worklog collected for today
worklog inspect work 2026-10-20
```

`doctor` never prints your commits, paths, or private words, and never calls an
AI. Every command has `--help`. `day`, `inspect`, and `doctor` also have
`--details` with the exact privacy and file-safety rules.

## Where things are stored

- **Journals** go in each profile's journal folder.
- **Notes** live in your repositories under the Git ref `refs/notes/worklog`.
- **Settings** go in your user configuration folder, for example
  `~/Library/Application Support/worklog` on macOS. Set `WORKLOG_CONFIG_HOME` to
  use another folder.
- **Working files** go in `~/.local/state/worklog`. Set `WORKLOG_STATE_HOME` to
  use another folder.

Nothing is stored inside the worklog source code.

## Advanced: your own summary command

Instead of Built-in, Claude Code, or Codex, a profile can use any command you
write:

```sh
worklog profile add local --timezone Asia/Taipei --language zh-TW \
  --journal-destination ~/Documents/worklog/local \
  --summarizer-command '/path/to/summary-adapter --format json'
```

The command receives the same JSON that `worklog inspect` prints, on stdin. It
must print one summary JSON object on stdout with `version`, `focus`, `completed`,
`decisions`, `next_steps`, and `technologies`. worklog still decides the file path
and writes the Markdown itself. The command has two minutes by default. Change
that with `WORKLOG_SUMMARIZER_TIMEOUT`, for example `30s`. Keep credentials in the
command's environment or your system keychain, never in the command line. See
`worklog day --details` for the full rules.

## For contributors

Run the tests:

```sh
go test ./...
```

The acceptance tests build the CLI and run it as a separate program, with
temporary home, settings, repository, and journal folders. They never touch your
real data.

Publish a release by pushing a version tag from `main`:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions then runs the tests on Linux and macOS. It builds all four
platforms with `scripts/release.sh` and tries each build on its own platform with
`scripts/smoke-test.sh`, in a clean environment without Go. The release is
published only when every step passes. To try a release build locally:

```sh
scripts/release.sh v0.0.0
scripts/smoke-test.sh dist/worklog_v0.0.0_darwin_arm64 v0.0.0
```
