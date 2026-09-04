package summarizer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Vickyciou/worklog/internal/dailyfacts"
)

const validSummary = `{"version":1,"focus":["Validate adapter"],"completed":[{"topic":"Summarizer","items":[{"outcome":"Accepted structured output"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}`

func TestRunSendsExactDailyFactsAndAcceptsValidatedSummary(t *testing.T) {
	command, payloadPath := helperCommand(t, "success")
	facts := dailyfacts.Document{Version: 1, Profile: "work", Date: "2026-09-04", Timezone: "UTC", Facts: []dailyfacts.Fact{}}

	summary, failure := Run(context.Background(), command, facts, t.TempDir(), time.Second)

	if failure != "" {
		t.Fatalf("Run failure = %q", failure)
	}
	if len(summary.Focus) != 1 || summary.Focus[0] != "Validate adapter" {
		t.Fatalf("summary = %+v", summary)
	}
	expected, err := dailyfacts.Encode(facts)
	if err != nil {
		t.Fatalf("encode expected payload: %v", err)
	}
	actual, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("read helper payload: %v", err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("payload mismatch\n--- got ---\n%s--- want ---\n%s", actual, expected)
	}
}

func TestRunClassifiesFailuresWithoutReturningProviderOutput(t *testing.T) {
	tests := []struct {
		mode    string
		timeout time.Duration
		want    Failure
	}{
		{mode: "nonzero", timeout: time.Second, want: FailureNonzero},
		{mode: "empty", timeout: time.Second, want: FailureEmpty},
		{mode: "malformed", timeout: time.Second, want: FailureInvalid},
		{mode: "invalid", timeout: time.Second, want: FailureInvalid},
		{mode: "timeout", timeout: 20 * time.Millisecond, want: FailureTimeout},
	}
	facts := dailyfacts.Document{Version: 1, Profile: "work", Date: "2026-09-04", Timezone: "UTC", Facts: []dailyfacts.Fact{}}
	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			command, _ := helperCommand(t, test.mode)
			_, failure := Run(context.Background(), command, facts, t.TempDir(), test.timeout)
			if failure != test.want {
				t.Fatalf("Run failure = %q, want %q", failure, test.want)
			}
		})
	}
	if _, failure := Run(context.Background(), "worklog-summarizer-does-not-exist", facts, t.TempDir(), time.Second); failure != FailureUnavailable {
		t.Fatalf("missing command failure = %q, want %q", failure, FailureUnavailable)
	}
}

func TestSummarizerHelperProcess(t *testing.T) {
	if os.Getenv("WORKLOG_SUMMARIZER_HELPER") != "1" {
		return
	}
	payload, _ := io.ReadAll(os.Stdin)
	if path := os.Getenv("WORKLOG_SUMMARIZER_PAYLOAD"); path != "" {
		_ = os.WriteFile(path, payload, 0o600)
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "success":
		fmt.Fprintln(os.Stdout, validSummary)
	case "nonzero":
		fmt.Fprintln(os.Stderr, "provider-secret-that-must-not-leak")
		os.Exit(7)
	case "empty":
	case "malformed":
		fmt.Fprintln(os.Stdout, "not json")
	case "invalid":
		fmt.Fprintln(os.Stdout, `{"version":1,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":[],"markdown":"# forbidden"}`)
	case "timeout":
		time.Sleep(time.Second)
	}
	os.Exit(0)
}

func helperCommand(t *testing.T, mode string) (string, string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}
	payloadPath := t.TempDir() + string(os.PathSeparator) + "payload.json"
	t.Setenv("WORKLOG_SUMMARIZER_HELPER", "1")
	t.Setenv("WORKLOG_SUMMARIZER_PAYLOAD", payloadPath)
	return quoteArgument(executable) + " -test.run=TestSummarizerHelperProcess -- " + mode, payloadPath
}

func quoteArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
