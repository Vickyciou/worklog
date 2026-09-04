package acceptance_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDaySendsTheInspectedPayloadToAValidatedCommandSummarizer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is covered by portable internal summarizer tests on Windows")
	}
	environment := newEnvironment(t)
	summarizer := writeFakeSummarizer(t, environment.root)
	payloadPath := filepath.Join(environment.root, "adapter-payload.json")
	workingDirectoryPath := filepath.Join(environment.root, "adapter-working-directory")
	environment.environment = append(environment.environment,
		"WORKLOG_FAKE_MODE=success",
		"WORKLOG_FAKE_PAYLOAD="+payloadPath,
		"WORKLOG_FAKE_WORKDIR="+workingDirectoryPath,
	)
	const email = "summarizer@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
		"--deny-literal", "private-client",
		"--summarizer-command", summarizer,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)
	commit := createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "metadata fallback", "")
	event := `{"version":1,"outcome":"private-client adapter source","approach":"token=fixture-secret","technologies":["Go"]}`
	assertSucceeded(t, environment.runWithInput(t, event, "capture", environment.repository, commit))
	inspected := environment.run(t, "inspect", "work", date)
	assertSucceeded(t, inspected)

	result := environment.runWithInput(t, "y\n", "day")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Adapter output")
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("read adapter payload: %v", err)
	}
	if string(payload) != inspected.stdout {
		t.Fatalf("adapter stdin differs from inspect output\n--- stdin ---\n%s--- inspect ---\n%s", payload, inspected.stdout)
	}
	for _, forbidden := range []string{"private-client", "fixture-secret", commit, environment.repository, email} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("adapter received forbidden value %q:\n%s", forbidden, payload)
		}
	}
	workingDirectory, err := os.ReadFile(workingDirectoryPath)
	if err != nil {
		t.Fatalf("read adapter working directory: %v", err)
	}
	expectedWorkingDirectory := filepath.Join(environment.stateRoot, "worklog")
	expectedWorkingDirectory, err = filepath.EvalSymlinks(expectedWorkingDirectory)
	if err != nil {
		t.Fatalf("resolve expected adapter working directory: %v", err)
	}
	if strings.TrimSpace(string(workingDirectory)) != expectedWorkingDirectory {
		t.Fatalf("adapter working directory = %q, want %q", strings.TrimSpace(string(workingDirectory)), expectedWorkingDirectory)
	}
	if strings.Contains(string(workingDirectory), environment.repository) {
		t.Fatal("adapter ran inside the registered repository")
	}
}

func TestDayFallsBackForEveryCommandSummarizerFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is covered by portable internal summarizer tests on Windows")
	}
	tests := []struct {
		name    string
		mode    string
		failure string
		missing bool
	}{
		{name: "missing command", failure: "command unavailable", missing: true},
		{name: "nonzero", mode: "nonzero", failure: "command exited unsuccessfully"},
		{name: "empty stdout", mode: "empty", failure: "command returned empty output"},
		{name: "malformed JSON", mode: "malformed", failure: "command returned invalid DailySummary JSON"},
		{name: "schema-invalid JSON", mode: "invalid", failure: "command returned invalid DailySummary JSON"},
		{name: "timeout", mode: "timeout", failure: "command timed out"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := newEnvironment(t)
			command := "worklog-summarizer-does-not-exist"
			if !test.missing {
				command = writeFakeSummarizer(t, environment.root)
			}
			environment.environment = append(environment.environment,
				"WORKLOG_FAKE_MODE="+test.mode,
				"WORKLOG_SUMMARIZER_TIMEOUT=500ms",
			)
			const email = "fallback@example.invalid"
			assertSucceeded(t, environment.run(t,
				"profile", "add", "work",
				"--timezone", "UTC",
				"--language", "en",
				"--journal-destination", environment.journalDestination,
				"--summarizer-command", command,
			))
			assertSucceeded(t, environment.run(t, "author", "add", "work", email))
			assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
			now := time.Now().UTC().Truncate(time.Second)
			createCommitAt(t, environment, environment.repository, email,
				now.Format(time.RFC3339), now.Format(time.RFC3339), "Deterministic fallback output", "")

			result := environment.runWithInput(t, "y\n", "day")

			assertSucceeded(t, result)
			assertContains(t, result.stdout, "Summarizer unavailable ("+test.failure+"); using Built-in.")
			assertContains(t, result.stdout, "Deterministic fallback output")
			if strings.Contains(result.stdout, "provider-secret-that-must-not-leak") || strings.Contains(result.stderr, "provider-secret-that-must-not-leak") {
				t.Fatal("provider stderr leaked")
			}
		})
	}
}

func writeFakeSummarizer(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "fake-summarizer")
	contents := `#!/bin/sh
if [ -n "$WORKLOG_FAKE_PAYLOAD" ]; then
  /bin/cat > "$WORKLOG_FAKE_PAYLOAD"
else
  /bin/cat >/dev/null
fi
if [ -n "$WORKLOG_FAKE_WORKDIR" ]; then
  /bin/pwd > "$WORKLOG_FAKE_WORKDIR"
fi
case "$WORKLOG_FAKE_MODE" in
  success)
    printf '%s\n' '{"version":1,"focus":["Adapter focus"],"completed":[{"topic":"Provider","items":[{"outcome":"Adapter output"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}'
    ;;
  nonzero)
    printf '%s\n' 'provider-secret-that-must-not-leak' >&2
    exit 9
    ;;
  empty)
    ;;
  malformed)
    printf '%s\n' 'not json'
    ;;
  invalid)
    printf '%s\n' '{"version":1,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":[],"markdown":"# forbidden"}'
    ;;
  timeout)
    sleep 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatalf("write fake summarizer: %v", err)
	}
	return path
}
