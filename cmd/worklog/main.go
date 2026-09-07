package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Vickyciou/worklog/internal/capture"
	"github.com/Vickyciou/worklog/internal/config"
)

const rootHelp = `worklog records and renders portable work journals.

Usage:
  worklog <command>

Getting started:
  worklog setup       Detect Git and system settings, then create a profile
  worklog day         Preview, write, or refresh a daily journal
  worklog settings    Update an existing profile with guided prompts

Commands:
  capture    Record a structured event for a Git commit
  inspect    Collect and print one profile's daily facts
  day        Preview, write, or refresh a daily journal
  setup      Create the first profile interactively
  settings   Update an existing profile interactively
  profile    Add, update, inspect, and validate profiles
  author     Add, inspect, and validate author identities
  repo       Add, inspect, and validate repository mappings
  doctor     Diagnose configuration and runtime prerequisites
  help       Show help for worklog
  version    Print version information

Options:
  -h, --help       Show help for worklog
  -v, --version    Print version information
`

const setupHelp = `Create the first worklog profile with guided defaults.

Usage:
  worklog setup

Behavior:
  Choose Work, Personal, or a custom profile name. The current Git repository
  and ~/Documents/worklog/<profile> are editable defaults. Worklog reads the
  author email from that repository and detects timezone and language from the
  environment or operating system. If a required value cannot be detected,
  setup asks only for that value.

  Setup offers Built-in plus installed Codex and Claude Code CLIs. Built-in uses
  no AI or network; provider presets reuse their CLI's current session without
  probing credentials. Setup shows resolved settings once before writing.
  Cancelling writes no configuration, state, or journal directory.

Examples:
  cd /path/to/repository
  worklog setup

Next:
  Run worklog day to preview today's journal. Register another repository with
  worklog repo add <profile> <path>.

Options:
  -h, --help    Show help for setup
`

const profileHelp = `Manage advanced profile routing and preferences.

A profile keeps work and personal repositories, author identities, timezone,
journal destination, and summary preferences separate.

Usage:
  worklog profile <add|set|list|show|validate>

Commands:
  add         Add another profile
  set         Update profile settings with flags
  list        List configured profile names
  show        Print one profile as JSON
  validate    Validate one profile or all profiles

Examples:
  worklog profile list
  worklog profile show work
  worklog profile add --help
  worklog profile set --help

Next:
  Add an author identity with worklog author add, then register a repository
  with worklog repo add.
`

const profileSetHelp = `Update an existing profile with explicit flags.

Usage:
  worklog profile set <name> [--language <language>] [--timezone <zone>] \
    [--journal-destination <path>] \
    [--daily-summary <built-in|codex|claude-code>] \
    [--deny-literal <value> ...]

At least one setting is required. Language is normalized, timezone must be a
valid IANA timezone, and ~ in the journal path resolves to the current home.
Changing the journal destination does not move or delete existing journals.
Repeated --deny-literal flags replace the profile's denied-literal list.

Examples:
  worklog profile set work --language en-US
  worklog profile set work --timezone Asia/Taipei \
    --journal-destination ~/Documents/worklog/work

Next:
  Run worklog settings work for the guided form, or worklog profile show work
  to inspect the saved result.
`

const settingsHelp = `Update an existing profile with guided prompts.

Usage:
  worklog settings [profile]

Behavior:
  A single configured profile is selected automatically. With multiple
  profiles, choose one from a numbered list or pass its name. Press Enter to
  keep the current language, IANA timezone, or journal folder.

  Worklog validates the complete candidate before one final confirmation.
  Changing the journal folder affects future output only; existing journals
  are not moved, copied, or deleted. Daily summary offers Built-in plus installed
  Codex and Claude Code CLIs; provider sessions remain owned by those CLIs.

Examples:
  worklog settings
  worklog settings work

Next:
  Run worklog profile show <name> to inspect the saved settings.

Options:
  -h, --help    Show help for settings
`

const profileAddHelp = `Add another profile using explicit advanced settings.

Usage:
  worklog profile add <name> --timezone <zone> --language <language> \\
    --journal-destination <path> \\
    [--daily-summary <built-in|codex|claude-code>] \\
    [--deny-literal <value> ...] \\
    [--summarizer-command <command>]

Examples:
  worklog profile add personal --timezone Asia/Taipei --language zh-TW \\
    --journal-destination ~/Documents/worklog/personal

Credentials must not be written in a summarizer command. Let the command read
them from its environment or system credential storage.

Next:
  Run worklog author add <profile> <email>, then worklog repo add <profile>
  <path>.
`

const authorHelp = `Manage Git author emails recognized by a profile.

Usage:
  worklog author <add|list|validate>

Examples:
  worklog author add work developer@example.com
  worklog author list work
  worklog author validate work

Next:
  Register the author's repository with worklog repo add <profile> <path>.
`

const repoHelp = `Manage which Git repositories belong to each profile.

One profile can contain multiple repositories, but every repository can belong
to only one profile. Worklog scans registered repositories only.

Usage:
  worklog repo <add|list|show|validate>

Examples:
  worklog repo add work /path/to/repository
  worklog repo list work
  worklog repo show /path/to/repository
  worklog repo validate

Next:
  Run worklog inspect <profile> to review the facts worklog will use.
`

const repoAddHelp = `Add one Git repository to a profile.

Usage:
  worklog repo add <profile> <path>

Example:
  worklog repo add work /path/to/repository

The path must resolve to a Git repository and must not overlap a repository
already owned by another profile.

Next:
  Run worklog repo validate <path>, then worklog inspect <profile>.
`

const inspectHelp = `Collect and print one profile's DailyFacts without side effects.

Usage:
  worklog inspect <profile> [YYYY-MM-DD]

Behavior:
  The date defaults to today in the profile's configured timezone. Collection
  scans only repositories registered to the profile, matches configured author
  identities, uses commit timestamps, deduplicates commits across refs, and
  ignores merge commits and refs/notes/worklog by default.

  Output is JSON and excludes commit hashes, repository paths, author emails,
  commit bodies, raw diffs, source code, and capture source metadata. Inspect
  applies the profile's denied literals and common secret redaction, then prints
  the exact DailyFacts payload an adapter would receive. It never invokes the
  configured summarizer or writes a journal.

Next:
  Run worklog day to preview today's journal.

Options:
  -h, --help    Show help for inspect
`

const dayHelp = `Preview, write, or refresh a daily journal.

Usage:
  worklog day [YYYY-MM-DD] [--refresh]

Behavior:
  Worklog discovers every profile with activity on the requested day. With no date,
  each profile uses today in its own timezone; YYYY-MM-DD selects a historical date.
  Each active profile gets a separate post-redaction DailyFacts payload, summary,
  preview, confirmation, and journal destination. Codex, Claude
  Code, and an advanced summarizer command all receive that boundary and must
  return a schema-valid DailySummary JSON object. Missing, timed-out, failed,
  empty, malformed, or invalid providers use the deterministic Built-in fallback.
  Worklog then renders and previews the complete journal. The
  destination is:

    <journal-destination>/<YYYY>/<YYYY-MM>/<YYYY-MM-DD>.md

  Only an explicit y or yes writes the journal. A historical date can be generated
  when its destination is absent. Existing journals are refused unless --refresh
  is explicit. Refresh requires exactly one valid managed block, previews its diff,
  preserves every byte outside it, and atomically replaces the file after confirmation.

  Profiles without registered repositories or author identities stop before any
  provider call or write and include the command needed to repair the mapping.

Next:
  Run worklog inspect <profile> first when you want to review the source facts.

Options:
  --refresh     Replace only an existing journal's managed block
  -h, --help    Show help for day
`

const doctorHelp = `Diagnose worklog configuration and runtime prerequisites.

Usage:
  worklog doctor

Behavior:
  Doctor checks Git, config and state roots, profiles, author identities,
  repository mappings, the Git notes ref, journal destinations, and configured
  summary executables. It does not read captured note content, invoke a summary
  provider, probe provider credentials, or print paths, commands, denied literals,
  credentials, or captured facts.

  Every failed check includes an action to repair it. Any failure produces a
  non-zero exit status.

Options:
  -h, --help    Show help for doctor
`

const captureHelp = `Record a versioned WorklogEvent as a Git note.

Usage:
  worklog capture <repository> <commit>

Input:
  Read exactly one JSON object from stdin. Unknown fields are rejected.

  {
    "version": 1,
    "outcome": "required non-empty string",
    "approach": "optional non-empty string",
    "decisions": [
      {"decision": "non-empty string", "reason": "non-empty string"}
    ],
    "next_steps": ["non-empty string"],
    "technologies": ["non-empty public technology name"],
    "source": {"kind": "non-empty string", "name": "optional non-empty string"}
  }

  version and outcome are required. All other fields are optional. Empty arrays
  are accepted. The canonical JSON event is stored at refs/notes/worklog.

Behavior:
  Repeating the same event succeeds without changing the note. A different event
  for a commit that already has a note is rejected without overwriting it.

Next:
  Run worklog inspect <profile> to verify the event in the daily facts.

Options:
  -h, --help    Show help for capture
`

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	return runWithIO(args, os.Stdin, os.Stdout, os.Stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, rootHelp)
		return 0
	}

	switch args[0] {
	case "help":
		help, found := helpText(args[1:])
		if !found {
			return usageError(stderr, "unknown help topic %q", strings.Join(args[1:], " "))
		}
		fmt.Fprint(stdout, help)
		return 0
	case "-h", "--help":
		if len(args) != 1 {
			return usageError(stderr, "%q does not accept arguments", args[0])
		}
		fmt.Fprint(stdout, rootHelp)
		return 0
	case "capture":
		return runCapture(args[1:], stdin, stdout, stderr)
	case "version", "-v", "--version":
		if len(args) != 1 {
			return usageError(stderr, "%q does not accept arguments", args[0])
		}
		fmt.Fprintf(stdout, "worklog %s\n", version)
		return 0
	case "setup", "settings", "profile", "author", "repo", "inspect", "day", "doctor":
		paths, err := config.DefaultPaths()
		if err != nil {
			return commandError(stderr, err)
		}
		application := cli{stdin: stdin, stdout: stdout, paths: paths}
		if err := application.execute(args[0], args[1:]); err != nil {
			return commandErrorWithHelp(stderr, err, closestHelpTopic(args))
		}
		return 0
	default:
		return usageError(stderr, "unknown command %q", args[0])
	}
}

func helpText(topic []string) (string, bool) {
	if len(topic) == 0 {
		return rootHelp, true
	}
	key := strings.Join(topic, " ")
	help := map[string]string{
		"setup":       setupHelp,
		"settings":    settingsHelp,
		"profile":     profileHelp,
		"profile add": profileAddHelp,
		"profile set": profileSetHelp,
		"author":      authorHelp,
		"repo":        repoHelp,
		"repo add":    repoAddHelp,
		"capture":     captureHelp,
		"inspect":     inspectHelp,
		"day":         dayHelp,
		"doctor":      doctorHelp,
	}
	value, found := help[key]
	return value, found
}

func runCapture(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(stdout, captureHelp)
		return 0
	}
	if len(args) != 2 {
		fmt.Fprintln(stderr, "worklog: capture requires <repository> and <commit>")
		fmt.Fprintln(stderr, "Run 'worklog capture --help' for usage.")
		return 2
	}

	result, err := capture.Run(args[0], args[1], stdin)
	if err != nil {
		fmt.Fprintf(stderr, "worklog: capture failed: %v\n", err)
		return 1
	}
	if result.Created {
		fmt.Fprintf(stdout, "Recorded worklog event for commit %s.\n", result.Commit)
	} else {
		fmt.Fprintf(stdout, "Worklog event for commit %s is already recorded.\n", result.Commit)
	}
	return 0
}

func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "worklog: "+format+"\n", args...)
	fmt.Fprintln(stderr, "Run 'worklog --help' for usage.")
	return 2
}

func commandError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "worklog: %v\n", err)
	return 1
}

func commandErrorWithHelp(stderr io.Writer, err error, topic string) int {
	fmt.Fprintf(stderr, "worklog: %v\n", err)
	fmt.Fprintf(stderr, "Run 'worklog %s --help' for help.\n", topic)
	return 1
}

func closestHelpTopic(args []string) string {
	if len(args) > 1 {
		candidate := strings.Join(args[:2], " ")
		if _, found := helpText(args[:2]); found {
			return candidate
		}
	}
	return args[0]
}
