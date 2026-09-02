package main

import (
	"fmt"
	"os"

	"github.com/Vickyciou/worklog/internal/capture"
)

const rootHelp = `worklog records and renders portable work journals.

Usage:
  worklog <command>

Commands:
  capture    Record a structured event for a Git commit
  help       Show help for worklog
  version    Print version information

Options:
  -h, --help       Show help for worklog
  -v, --version    Print version information
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

Options:
  -h, --help    Show help for capture
`

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Print(rootHelp)
		return 0
	}

	switch args[0] {
	case "help":
		if len(args) == 2 && args[1] == "capture" {
			fmt.Print(captureHelp)
			return 0
		}
		if len(args) != 1 {
			return usageError("%q does not accept arguments", args[0])
		}
		fmt.Print(rootHelp)
		return 0
	case "-h", "--help":
		if len(args) != 1 {
			return usageError("%q does not accept arguments", args[0])
		}
		fmt.Print(rootHelp)
		return 0
	case "capture":
		return runCapture(args[1:])
	case "version", "-v", "--version":
		if len(args) != 1 {
			return usageError("%q does not accept arguments", args[0])
		}
		fmt.Printf("worklog %s\n", version)
		return 0
	default:
		return usageError("unknown command %q", args[0])
	}
}

func runCapture(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(captureHelp)
		return 0
	}
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "worklog: capture requires <repository> and <commit>")
		fmt.Fprintln(os.Stderr, "Run 'worklog capture --help' for usage.")
		return 2
	}

	result, err := capture.Run(args[0], args[1], os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worklog: capture failed: %v\n", err)
		return 1
	}
	if result.Created {
		fmt.Printf("Recorded worklog event for commit %s.\n", result.Commit)
	} else {
		fmt.Printf("Worklog event for commit %s is already recorded.\n", result.Commit)
	}
	return 0
}

func usageError(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "worklog: "+format+"\n", args...)
	fmt.Fprintln(os.Stderr, "Run 'worklog --help' for usage.")
	return 2
}
