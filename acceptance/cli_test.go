package acceptance_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var worklogBinary string

func TestMain(m *testing.M) {
	buildRoot, err := os.MkdirTemp("", "worklog-acceptance-build-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create isolated build root: %v\n", err)
		os.Exit(1)
	}
	worklogBinary = filepath.Join(buildRoot, "worklog")
	if runtime.GOOS == "windows" {
		worklogBinary += ".exe"
	}

	command := exec.Command("go", "build", "-o", worklogBinary, "./cmd/worklog")
	command.Dir = repositoryRoot()
	command.Env = isolatedGoEnvironment(buildRoot)
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "build worklog acceptance binary: %v\n%s", buildErr, output)
		if removeErr := os.RemoveAll(buildRoot); removeErr != nil {
			fmt.Fprintf(os.Stderr, "remove isolated build root: %v\n", removeErr)
		}
		os.Exit(1)
	}

	exitCode := m.Run()
	if removeErr := os.RemoveAll(buildRoot); removeErr != nil {
		fmt.Fprintf(os.Stderr, "remove isolated build root: %v\n", removeErr)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func TestRootHelp(t *testing.T) {
	const expected = `worklog turns your Git commits into a daily work journal.

Usage:
  worklog <command>

Running worklog without a command shows today's status.

Every day:
  day           Preview, write, or update today's journal
  status        Show today's activity and journal state

Set up:
  setup         First-time setup; run it inside a Git repository
  settings      Change a profile's language, timezone, folder, or summary
  repo add      Add a Git repository to a profile
  profile add   Add another profile with guided prompts

When something looks wrong:
  doctor        Find setup problems and how to fix them
  inspect       Show the facts worklog collected for a day

Advanced and integrations:
  profile       List, show, update, and validate profiles
  author        Add, list, and validate author emails
  repo          List, show, and validate repository mappings
  capture       Record a structured event for a commit (for commit tools)
  version       Print version information

Options:
  -h, --help       Show help for worklog
  -v, --version    Print version information

Run worklog help <command> for one command's details.
`
	for _, arguments := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		t.Run(strings.Join(arguments, "_"), func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, arguments...)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
			if result.stdout != expected {
				t.Fatalf("stdout mismatch\n--- got ---\n%s--- want ---\n%s", result.stdout, expected)
			}
		})
	}
}

func TestCommandHelpIsAvailableThroughDirectAndHelpForms(t *testing.T) {
	topics := []struct {
		name       string
		direct     []string
		help       []string
		expectText string
	}{
		{name: "setup", direct: []string{"setup", "--help"}, help: []string{"help", "setup"}, expectText: "Worklog reads the\n  author email"},
		{name: "settings", direct: []string{"settings", "--help"}, help: []string{"help", "settings"}, expectText: "Changing the journal folder affects future output only"},
		{name: "profile", direct: []string{"profile", "--help"}, help: []string{"help", "profile"}, expectText: "worklog profile <add|set|list|show|validate>"},
		{name: "profile add", direct: []string{"profile", "add", "--help"}, help: []string{"help", "profile", "add"}, expectText: "same guided flow as setup"},
		{name: "profile set", direct: []string{"profile", "set", "--help"}, help: []string{"help", "profile", "set"}, expectText: "Changing the journal destination does not move"},
		{name: "author", direct: []string{"author", "--help"}, help: []string{"help", "author"}, expectText: "Manage Git author emails"},
		{name: "repo", direct: []string{"repo", "--help"}, help: []string{"help", "repo"}, expectText: "every repository can belong\nto only one profile"},
		{name: "repo add", direct: []string{"repo", "add", "--help"}, help: []string{"help", "repo", "add"}, expectText: "worklog repo add [path] [--profile <name>]"},
		{name: "day", direct: []string{"day", "--help"}, help: []string{"help", "day"}, expectText: "It writes only after you type y"},
		{name: "doctor", direct: []string{"doctor", "--help"}, help: []string{"help", "doctor"}, expectText: "Every problem comes with the command that fixes"},
	}

	for _, topic := range topics {
		t.Run(topic.name, func(t *testing.T) {
			environment := newEnvironment(t)
			direct := environment.run(t, topic.direct...)
			alias := environment.run(t, topic.help...)
			if direct.exitCode != 0 || alias.exitCode != 0 {
				t.Fatalf("help failed: direct=%d %q; alias=%d %q", direct.exitCode, direct.stderr, alias.exitCode, alias.stderr)
			}
			if direct.stdout != alias.stdout {
				t.Fatalf("help aliases differ\n--- direct ---\n%s--- alias ---\n%s", direct.stdout, alias.stdout)
			}
			assertContains(t, direct.stdout, topic.expectText)
		})
	}
}

func TestCaptureHelpDescribesLiveInputContract(t *testing.T) {
	const expected = `Record a versioned WorklogEvent as a Git note.

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
	for _, arguments := range [][]string{{"capture", "--help"}, {"capture", "-h"}, {"help", "capture"}} {
		t.Run(strings.Join(arguments, "_"), func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, arguments...)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
			if result.stdout != expected {
				t.Fatalf("stdout mismatch\n--- got ---\n%s--- want ---\n%s", result.stdout, expected)
			}
		})
	}
}

func TestInspectDetailsDescribeCollectionAndPrivacyContract(t *testing.T) {
	const expected = `Technical details for worklog inspect.

  Collection scans only repositories registered to the profile, matches
  configured author identities, uses commit timestamps, deduplicates commits
  across refs, and ignores merge commits and refs/notes/worklog by default.

  Output is JSON and excludes commit hashes, repository paths, author emails,
  commit bodies, raw diffs, source code, and capture source metadata. Inspect
  applies the profile's denied literals and common secret redaction, then prints
  the exact DailyFacts payload an adapter would receive. It never invokes the
  configured summarizer or writes a journal.
`
	for _, arguments := range [][]string{{"inspect", "--details"}, {"help", "inspect", "--details"}} {
		t.Run(strings.Join(arguments, "_"), func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, arguments...)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
			if result.stdout != expected {
				t.Fatalf("stdout mismatch\n--- got ---\n%s--- want ---\n%s", result.stdout, expected)
			}
		})
	}
}

func TestEverydayHelpAvoidsInternalTermsAndPointsToDetails(t *testing.T) {
	environment := newEnvironment(t)
	internalTerms := []string{"DailyFacts", "DailySummary", "managed block", "managed-block", "schema", "payload", "adapter"}
	for _, test := range []struct {
		command       string
		details       bool
		detailsExpect string
	}{
		{command: "day", details: true, detailsExpect: "exactly one valid managed block"},
		{command: "inspect", details: true, detailsExpect: "DailyFacts payload"},
		{command: "doctor", details: true, detailsExpect: "It does not read captured note content"},
		{command: "status"},
	} {
		help := environment.run(t, test.command, "--help")
		assertSucceeded(t, help)
		for _, term := range internalTerms {
			if strings.Contains(help.stdout, term) {
				t.Errorf("%s --help contains internal term %q", test.command, term)
			}
		}
		if !test.details {
			if strings.Contains(help.stdout, "--details") {
				t.Errorf("%s --help mentions --details without providing them", test.command)
			}
			continue
		}
		assertContains(t, help.stdout, "--details")
		for _, arguments := range [][]string{{test.command, "--details"}, {"help", test.command, "--details"}} {
			details := environment.run(t, arguments...)
			assertSucceeded(t, details)
			assertContains(t, details.stdout, test.detailsExpect)
		}
	}
}

func TestCaptureWritesVersionedEventToDedicatedGitNotesRef(t *testing.T) {
	environment := newEnvironment(t)
	commit := environment.createCommit(t)
	input := `{
  "technologies": ["Go", "Git"],
  "version": 1,
  "source": {"name": "gen-commit", "kind": "agent"},
  "decisions": [{"reason": "Keep provider integrations decoupled", "decision": "Use Git notes"}],
  "outcome": "Recorded structured work facts",
  "next_steps": ["Collect daily facts"],
  "approach": "Validate before writing"
}`

	result := environment.runWithInput(t, input, "capture", environment.repository, commit)

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
	}
	if result.stdout != fmt.Sprintf("Recorded worklog event for commit %s.\n", commit) {
		t.Fatalf("stdout = %q", result.stdout)
	}

	note := environment.git(t, "notes", "--ref=refs/notes/worklog", "show", commit)
	if note.exitCode != 0 {
		t.Fatalf("read Git note: %s", note.stderr)
	}
	const expectedNote = "{\"version\":1,\"outcome\":\"Recorded structured work facts\",\"approach\":\"Validate before writing\",\"decisions\":[{\"decision\":\"Use Git notes\",\"reason\":\"Keep provider integrations decoupled\"}],\"next_steps\":[\"Collect daily facts\"],\"technologies\":[\"Go\",\"Git\"],\"source\":{\"kind\":\"agent\",\"name\":\"gen-commit\"}}\n"
	if note.stdout != expectedNote {
		t.Fatalf("Git note mismatch\n--- got ---\n%s--- want ---\n%s", note.stdout, expectedNote)
	}
}

func TestCaptureIsIdempotentForTheSameEvent(t *testing.T) {
	environment := newEnvironment(t)
	commit := environment.createCommit(t)
	firstInput := `{"version":1,"outcome":"Recorded structured work facts","technologies":["Go"]}`
	secondInput := "{\n  \"technologies\": [\"Go\"],\n  \"outcome\": \"Recorded structured work facts\",\n  \"version\": 1\n}\n"

	first := environment.runWithInput(t, firstInput, "capture", environment.repository, commit)
	if first.exitCode != 0 {
		t.Fatalf("first capture exit code = %d; stderr:\n%s", first.exitCode, first.stderr)
	}
	refBefore := environment.git(t, "rev-parse", "refs/notes/worklog")

	second := environment.runWithInput(t, secondInput, "capture", environment.repository, commit)
	if second.exitCode != 0 {
		t.Fatalf("second capture exit code = %d; stderr:\n%s", second.exitCode, second.stderr)
	}
	if second.stderr != "" {
		t.Fatalf("second capture stderr = %q, want empty", second.stderr)
	}
	if second.stdout != fmt.Sprintf("Worklog event for commit %s is already recorded.\n", commit) {
		t.Fatalf("second capture stdout = %q", second.stdout)
	}
	refAfter := environment.git(t, "rev-parse", "refs/notes/worklog")
	if refBefore.exitCode != 0 || refAfter.exitCode != 0 || refBefore.stdout != refAfter.stdout {
		t.Fatalf("notes ref changed on retry: before=%q after=%q", refBefore.stdout, refAfter.stdout)
	}
}

func TestCaptureRejectsAConflictingEventAndPreservesTheOriginal(t *testing.T) {
	environment := newEnvironment(t)
	commit := environment.createCommit(t)
	original := `{"version":1,"outcome":"Original outcome"}`
	conflict := `{"version":1,"outcome":"Different outcome"}`

	first := environment.runWithInput(t, original, "capture", environment.repository, commit)
	if first.exitCode != 0 {
		t.Fatalf("first capture exit code = %d; stderr:\n%s", first.exitCode, first.stderr)
	}
	noteBefore := environment.git(t, "notes", "--ref=refs/notes/worklog", "show", commit)
	refBefore := environment.git(t, "rev-parse", "refs/notes/worklog")

	second := environment.runWithInput(t, conflict, "capture", environment.repository, commit)
	if second.exitCode == 0 {
		t.Fatal("conflicting capture exit code = 0, want non-zero")
	}
	if second.stdout != "" {
		t.Fatalf("conflicting capture stdout = %q, want empty", second.stdout)
	}
	assertContains(t, second.stderr, "commit already has a different worklog event")

	noteAfter := environment.git(t, "notes", "--ref=refs/notes/worklog", "show", commit)
	refAfter := environment.git(t, "rev-parse", "refs/notes/worklog")
	if noteBefore.stdout != noteAfter.stdout || refBefore.stdout != refAfter.stdout {
		t.Fatalf("original note changed after conflict: before=%q after=%q", noteBefore.stdout, noteAfter.stdout)
	}
}

func TestCaptureRejectsInvalidInputWithoutWritingANote(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{name: "missing version", input: `{"outcome":"Done"}`, expectedError: "version is required"},
		{name: "missing outcome", input: `{"version":1}`, expectedError: "outcome is required"},
		{name: "unsupported version", input: `{"version":2,"outcome":"Done"}`, expectedError: "version must be 1"},
		{name: "null optional field", input: `{"version":1,"outcome":"Done","next_steps":null}`, expectedError: "next_steps must not be null"},
		{name: "unknown field", input: `{"version":1,"outcome":"Done","detail":"raw"}`, expectedError: `unknown field "detail"`},
		{name: "empty approach", input: `{"version":1,"outcome":"Done","approach":" "}`, expectedError: "approach must be non-empty"},
		{name: "decision without reason", input: `{"version":1,"outcome":"Done","decisions":[{"decision":"Choose notes","reason":""}]}`, expectedError: "decisions[0].reason must be non-empty"},
		{name: "empty next step", input: `{"version":1,"outcome":"Done","next_steps":[""]}`, expectedError: "next_steps[0] must be non-empty"},
		{name: "empty technology", input: `{"version":1,"outcome":"Done","technologies":[" "]}`, expectedError: "technologies[0] must be non-empty"},
		{name: "source without kind", input: `{"version":1,"outcome":"Done","source":{"kind":""}}`, expectedError: "source.kind must be non-empty"},
		{name: "null source name", input: `{"version":1,"outcome":"Done","source":{"kind":"agent","name":null}}`, expectedError: "source.name must not be null"},
		{name: "multiple JSON values", input: `{"version":1,"outcome":"Done"} {}`, expectedError: "expected exactly one JSON object"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := newEnvironment(t)
			commit := environment.createCommit(t)

			result := environment.runWithInput(t, test.input, "capture", environment.repository, commit)

			if result.exitCode == 0 {
				t.Fatal("exit code = 0, want non-zero")
			}
			if result.stdout != "" {
				t.Fatalf("stdout = %q, want empty", result.stdout)
			}
			assertContains(t, result.stderr, test.expectedError)
			notesRef := environment.git(t, "show-ref", "--verify", "--quiet", "refs/notes/worklog")
			if notesRef.exitCode == 0 {
				t.Fatal("invalid capture created the worklog notes ref")
			}
		})
	}
}

func TestCaptureRejectsMissingRepositoryOrCommitWithoutWritingANote(t *testing.T) {
	const input = `{"version":1,"outcome":"Done"}`

	t.Run("missing repository", func(t *testing.T) {
		environment := newEnvironment(t)
		result := environment.runWithInput(t, input, "capture", filepath.Join(environment.root, "missing"), "HEAD")

		if result.exitCode == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		assertContains(t, result.stderr, "access repository")
	})

	t.Run("missing commit", func(t *testing.T) {
		environment := newEnvironment(t)
		environment.createCommit(t)
		result := environment.runWithInput(t, input, "capture", environment.repository, "deadbeef")

		if result.exitCode == 0 {
			t.Fatal("exit code = 0, want non-zero")
		}
		assertContains(t, result.stderr, `resolve commit "deadbeef"`)
		notesRef := environment.git(t, "show-ref", "--verify", "--quiet", "refs/notes/worklog")
		if notesRef.exitCode == 0 {
			t.Fatal("missing commit capture created the worklog notes ref")
		}
	})
}

func TestVersion(t *testing.T) {
	for _, argument := range []string{"version", "--version", "-v"} {
		t.Run(argument, func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, argument)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stdout != "worklog dev\n" {
				t.Fatalf("stdout = %q, want %q", result.stdout, "worklog dev\n")
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	environment := newEnvironment(t)

	result := environment.run(t, "unknown")

	if result.exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
	assertContains(t, result.stderr, `unknown command "unknown"`)
	assertContains(t, result.stderr, "Run 'worklog --help' for usage.")
}

func TestIllegalArguments(t *testing.T) {
	environment := newEnvironment(t)

	result := environment.run(t, "version", "extra")

	if result.exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
	assertContains(t, result.stderr, `"version" does not accept arguments`)
}

func TestCommandErrorsPointToClosestHelpTopic(t *testing.T) {
	tests := []struct {
		name      string
		arguments []string
		help      string
	}{
		{name: "setup", arguments: []string{"setup", "extra"}, help: "worklog setup --help"},
		{name: "settings", arguments: []string{"settings", "one", "two"}, help: "worklog settings --help"},
		{name: "profile add", arguments: []string{"profile", "add"}, help: "worklog profile add --help"},
		{name: "profile set", arguments: []string{"profile", "set"}, help: "worklog profile set --help"},
		{name: "profile", arguments: []string{"profile", "unknown"}, help: "worklog profile --help"},
		{name: "inspect", arguments: []string{"inspect"}, help: "worklog inspect --help"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, test.arguments...)

			if result.exitCode == 0 {
				t.Fatal("exit code = 0, want non-zero")
			}
			assertContains(t, result.stderr, "Run '"+test.help+"' for help.")
		})
	}
}

type acceptanceEnvironment struct {
	root               string
	home               string
	configRoot         string
	stateRoot          string
	repository         string
	journalDestination string
	temporaryDirectory string
	environment        []string
}

func newEnvironment(t *testing.T) acceptanceEnvironment {
	t.Helper()

	root := t.TempDir()
	environment := acceptanceEnvironment{
		root:               root,
		home:               filepath.Join(root, "home"),
		configRoot:         filepath.Join(root, "config"),
		stateRoot:          filepath.Join(root, "state"),
		repository:         filepath.Join(root, "repository"),
		journalDestination: filepath.Join(root, "journal"),
		temporaryDirectory: filepath.Join(root, "tmp"),
	}

	for _, directory := range []string{
		environment.home,
		environment.configRoot,
		environment.stateRoot,
		environment.repository,
		environment.journalDestination,
		environment.temporaryDirectory,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create isolated directory %q: %v", directory, err)
		}
	}

	environment.environment = cleanEnvironment(os.Environ(), map[string]string{
		"HOME":                environment.home,
		"USERPROFILE":         environment.home,
		"XDG_CONFIG_HOME":     environment.configRoot,
		"XDG_STATE_HOME":      environment.stateRoot,
		"XDG_CACHE_HOME":      filepath.Join(environment.stateRoot, "cache"),
		"XDG_DATA_HOME":       filepath.Join(environment.stateRoot, "data"),
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_GLOBAL":   filepath.Join(root, "gitconfig"),
		"GIT_TERMINAL_PROMPT": "0",
		"TMPDIR":              environment.temporaryDirectory,
	}, "GIT_", "WORKLOG_", "XDG_")

	git := exec.Command("git", "init", "--quiet", environment.repository)
	git.Env = environment.environment
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("initialize isolated Git repository: %v\n%s", err, output)
	}

	return environment
}

func (environment acceptanceEnvironment) run(t *testing.T, arguments ...string) processResult {
	return environment.runWithInput(t, "", arguments...)
}

func (environment acceptanceEnvironment) runWithInput(t *testing.T, input string, arguments ...string) processResult {
	return environment.runWithInputAt(t, environment.repository, input, arguments...)
}

func (environment acceptanceEnvironment) runWithInputAt(t *testing.T, directory, input string, arguments ...string) processResult {
	t.Helper()

	command := exec.Command(worklogBinary, arguments...)
	command.Dir = directory
	command.Env = environment.environment
	command.Stdin = strings.NewReader(input)
	return runCommand(t, command)
}

func (environment acceptanceEnvironment) git(t *testing.T, arguments ...string) processResult {
	t.Helper()

	commandArguments := append([]string{"-C", environment.repository}, arguments...)
	command := exec.Command("git", commandArguments...)
	command.Env = environment.environment
	return runCommand(t, command)
}

func (environment acceptanceEnvironment) createCommit(t *testing.T) string {
	t.Helper()

	for _, setting := range [][]string{
		{"config", "user.name", "Worklog Acceptance"},
		{"config", "user.email", "worklog@example.invalid"},
	} {
		result := environment.git(t, setting...)
		if result.exitCode != 0 {
			t.Fatalf("configure Git fixture: %s", result.stderr)
		}
	}
	commit := environment.git(t, "commit", "--quiet", "--allow-empty", "-m", "fixture commit")
	if commit.exitCode != 0 {
		t.Fatalf("create Git fixture commit: %s", commit.stderr)
	}
	hash := environment.git(t, "rev-parse", "HEAD")
	if hash.exitCode != 0 {
		t.Fatalf("resolve Git fixture commit: %s", hash.stderr)
	}
	return strings.TrimSpace(hash.stdout)
}

func runCommand(t *testing.T, command *exec.Cmd) processResult {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("execute %q: %v", command.Path, err)
		}
		exitCode = exitError.ExitCode()
	}

	return processResult{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}

type processResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func repositoryRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("locate acceptance test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
}

func isolatedGoEnvironment(root string) []string {
	return cleanEnvironment(os.Environ(), map[string]string{
		"HOME":           filepath.Join(root, "home"),
		"USERPROFILE":    filepath.Join(root, "home"),
		"GOCACHE":        filepath.Join(root, "go-cache"),
		"GOPATH":         filepath.Join(root, "go-path"),
		"GOMODCACHE":     filepath.Join(root, "go-mod-cache"),
		"XDG_CACHE_HOME": filepath.Join(root, "cache"),
		"GOENV":          "off",
		"GOTOOLCHAIN":    "local",
	}, "GO", "CGO_")
}

func cleanEnvironment(current []string, replacements map[string]string, removedPrefixes ...string) []string {
	result := make([]string, 0, len(current)+len(replacements))
	for _, variable := range current {
		name, _, found := strings.Cut(variable, "=")
		if _, replace := replacements[name]; found && replace {
			continue
		}
		remove := false
		for _, prefix := range removedPrefixes {
			if strings.HasPrefix(name, prefix) {
				remove = true
				break
			}
		}
		if remove {
			continue
		}
		result = append(result, variable)
	}
	for name, value := range replacements {
		result = append(result, name+"="+value)
	}
	return result
}

func assertContains(t *testing.T, value string, expectedSubstring string) {
	t.Helper()
	if !strings.Contains(value, expectedSubstring) {
		t.Fatalf("%q does not contain %q", value, expectedSubstring)
	}
}
