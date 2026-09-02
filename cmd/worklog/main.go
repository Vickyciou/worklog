package main

import (
	"fmt"
	"os"
)

const rootHelp = `worklog records and renders portable work journals.

Usage:
  worklog <command>

Commands:
  help       Show help for worklog
  version    Print version information

Options:
  -h, --help       Show help for worklog
  -v, --version    Print version information
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
	case "help", "-h", "--help":
		if len(args) != 1 {
			return usageError("%q does not accept arguments", args[0])
		}
		fmt.Print(rootHelp)
		return 0
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

func usageError(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "worklog: "+format+"\n", args...)
	fmt.Fprintln(os.Stderr, "Run 'worklog --help' for usage.")
	return 2
}
