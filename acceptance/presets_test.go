package acceptance_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSetupAndSettingsDiscoverPresetsWithoutCredentialProbes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures are covered by portable preset runner tests on Windows")
	}
	environment := newEnvironment(t)
	providerBin := filepath.Join(environment.root, "provider-bin")
	writeFakePresetCLIs(t, providerBin)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"PATH":        providerBin + string(os.PathListSeparator) + systemCommandPath(t),
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "presets@example.invalid"))
	invocationLog := filepath.Join(environment.root, "provider-invocation")
	environment.environment = append(environment.environment, "WORKLOG_PRESET_ARGS="+invocationLog)
	setupInput := strings.Join([]string{"1", "", environment.journalDestination, "2", "", ""}, "\n")

	setup := environment.runWithInput(t, setupInput, "setup")

	assertSucceeded(t, setup)
	assertContains(t, setup.stdout, "Codex       available; uses the CLI's current session")
	assertContains(t, setup.stdout, "Claude Code available; uses the CLI's current session")
	if _, err := os.Stat(invocationLog); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("setup probed provider credentials or invoked a provider: %v", err)
	}
	show := environment.run(t, "profile", "show", "work", "--json")
	assertSucceeded(t, show)
	assertContains(t, show.stdout, `"summary_mode": "codex"`)

	settingsInput := strings.Join([]string{"", "", "", "3", "", ""}, "\n")
	settings := environment.runWithInput(t, settingsInput, "settings")
	assertSucceeded(t, settings)
	show = environment.run(t, "profile", "show", "work", "--json")
	assertSucceeded(t, show)
	assertContains(t, show.stdout, `"summary_mode": "claude-code"`)
	if _, err := os.Stat(invocationLog); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("settings probed provider credentials or invoked a provider: %v", err)
	}
}

func TestUnavailablePresetKeepsBuiltInAvailable(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"PATH":        systemCommandPath(t),
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "presets@example.invalid"))
	input := strings.Join([]string{"1", "", environment.journalDestination, "2", ""}, "\n")

	result := environment.runWithInput(t, input, "setup")

	if result.exitCode == 0 {
		t.Fatal("unavailable preset setup succeeded")
	}
	for _, expected := range []string{"Codex", "not found", "choose Built-in"} {
		assertContains(t, result.stderr, expected)
	}
	assertContains(t, result.stdout, "Built-in")
	if _, err := os.Stat(filepath.Join(environment.configRoot, "worklog", "config.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unavailable preset created configuration: %v", err)
	}
}

func TestPresetSummaryUsesProfileLanguage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures are covered by portable preset runner tests on Windows")
	}
	environment := newEnvironment(t)
	providerBin := filepath.Join(environment.root, "provider-bin")
	writeFakePresetCLIs(t, providerBin)
	argumentsPath := filepath.Join(environment.root, "provider-arguments")
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"PATH": providerBin + string(os.PathListSeparator) + systemCommandPath(t),
	})
	environment.environment = append(environment.environment,
		"WORKLOG_PRESET_MODE=language",
		"WORKLOG_PRESET_ARGS="+argumentsPath,
	)
	const email = "language@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "Asia/Taipei",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
		"--daily-summary", "claude-code",
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	now := time.Date(2026, 9, 10, 9, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*60*60))
	createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "Implement language-aware summaries", "")

	result := environment.runWithInput(t, "y\n", "day", now.Format(time.DateOnly))

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "繁體中文摘要")
	if strings.Contains(result.stdout, "English summary") {
		t.Fatal("English provider summary was rendered for a zh-TW profile")
	}
	arguments, err := os.ReadFile(argumentsPath)
	if err != nil {
		t.Fatalf("read provider arguments: %v", err)
	}
	assertContains(t, string(arguments), "Target language: zh-TW.")
}

func TestCodexAndClaudeCodePresetsUseTheSharedValidatedBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures are covered by portable preset runner tests on Windows")
	}
	providers := []struct {
		mode       string
		executable string
		safeFlag   string
	}{
		{mode: "codex", executable: "codex", safeFlag: "--sandbox read-only"},
		{mode: "claude-code", executable: "claude", safeFlag: "--safe-mode"},
	}
	modes := []struct {
		name    string
		mode    string
		failure string
	}{
		{name: "success", mode: "success"},
		{name: "nonzero", mode: "nonzero", failure: "command exited unsuccessfully"},
		{name: "empty", mode: "empty", failure: "command returned empty output"},
		{name: "invalid", mode: "invalid", failure: "command returned invalid DailySummary JSON"},
	}
	for _, provider := range providers {
		for _, providerMode := range modes {
			t.Run(provider.mode+"/"+providerMode.name, func(t *testing.T) {
				environment := newEnvironment(t)
				providerBin := filepath.Join(environment.root, "provider-bin")
				writeFakePresetCLIs(t, providerBin)
				payloadPath := filepath.Join(environment.root, "provider-payload")
				argumentsPath := filepath.Join(environment.root, "provider-arguments")
				workingDirectoryPath := filepath.Join(environment.root, "provider-working-directory")
				environment.environment = cleanEnvironment(environment.environment, map[string]string{
					"PATH": providerBin + string(os.PathListSeparator) + systemCommandPath(t),
				})
				environment.environment = append(environment.environment,
					"WORKLOG_PRESET_MODE="+providerMode.mode,
					"WORKLOG_PRESET_PAYLOAD="+payloadPath,
					"WORKLOG_PRESET_ARGS="+argumentsPath,
					"WORKLOG_PRESET_WORKDIR="+workingDirectoryPath,
				)
				const email = "preset@example.invalid"
				assertSucceeded(t, environment.run(t,
					"profile", "add", "work",
					"--timezone", "UTC",
					"--language", "en",
					"--journal-destination", environment.journalDestination,
					"--deny-literal", "private-preset",
					"--daily-summary", provider.mode,
				))
				assertSucceeded(t, environment.run(t, "author", "add", "work", email))
				assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
				now := time.Now().UTC().Truncate(time.Second)
				commit := createCommitAt(t, environment, environment.repository, email,
					now.Format(time.RFC3339), now.Format(time.RFC3339), "Preset deterministic fallback", "")
				event := `{"version":1,"outcome":"private-preset source","approach":"token=provider-secret","technologies":["Go"]}`
				assertSucceeded(t, environment.runWithInput(t, event, "capture", environment.repository, commit))
				inspected := environment.run(t, "inspect", "work", now.Format(time.DateOnly))
				assertSucceeded(t, inspected)

				result := environment.runWithInput(t, "y\n", "day")

				assertSucceeded(t, result)
				payload, err := os.ReadFile(payloadPath)
				if err != nil {
					t.Fatalf("read provider payload: %v", err)
				}
				if string(payload) != inspected.stdout {
					t.Fatalf("preset stdin differs from inspect output\n--- stdin ---\n%s--- inspect ---\n%s", payload, inspected.stdout)
				}
				for _, forbidden := range []string{"private-preset", "provider-secret", commit, environment.repository, email} {
					if strings.Contains(string(payload), forbidden) {
						t.Fatalf("preset received forbidden value %q:\n%s", forbidden, payload)
					}
				}
				arguments, err := os.ReadFile(argumentsPath)
				if err != nil {
					t.Fatalf("read provider arguments: %v", err)
				}
				assertContains(t, string(arguments), provider.safeFlag)
				workingDirectory, err := os.ReadFile(workingDirectoryPath)
				if err != nil {
					t.Fatalf("read provider working directory: %v", err)
				}
				if strings.Contains(string(workingDirectory), environment.repository) {
					t.Fatal("preset ran in registered repository")
				}
				if providerMode.failure == "" {
					assertContains(t, result.stdout, "Preset adapter output")
				} else {
					assertContains(t, result.stdout, providerMode.failure)
					assertContains(t, result.stdout, "[REDACTED] source")
				}
				if strings.Contains(result.stdout, "provider-stderr-secret") || strings.Contains(result.stderr, "provider-stderr-secret") {
					t.Fatal("provider stderr leaked")
				}
			})
		}
	}
}

func writeFakePresetCLIs(t *testing.T, directory string) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatalf("create provider bin: %v", err)
	}
	contents := `#!/bin/sh
if [ -n "$WORKLOG_PRESET_ARGS" ]; then
  printf '%s\n' "$*" > "$WORKLOG_PRESET_ARGS"
fi
if [ -n "$WORKLOG_PRESET_WORKDIR" ]; then
  /bin/pwd > "$WORKLOG_PRESET_WORKDIR"
fi
if [ -n "$WORKLOG_PRESET_PAYLOAD" ]; then
  /bin/cat > "$WORKLOG_PRESET_PAYLOAD"
else
  /bin/cat >/dev/null
fi
case "$WORKLOG_PRESET_MODE" in
  success)
    printf '%s\n' '{"version":1,"focus":["Preset focus"],"completed":[{"topic":"Provider","items":[{"outcome":"Preset adapter output"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}'
    ;;
  language)
    case "$*" in
      *"Target language: zh-TW."*)
        printf '%s\n' '{"version":1,"focus":["繁體中文摘要"],"completed":[{"topic":"語言","items":[{"outcome":"依照 profile 語言產生摘要"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}'
        ;;
      *)
        printf '%s\n' '{"version":1,"focus":["English summary"],"completed":[{"topic":"Language","items":[{"outcome":"Ignored profile language"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}'
        ;;
    esac
    ;;
  nonzero)
    printf '%s\n' 'provider-stderr-secret' >&2
    exit 8
    ;;
  empty)
    ;;
  invalid)
    printf '%s\n' '{"version":1,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":[],"journal_path":"/forbidden"}'
    ;;
esac
`
	for _, executable := range []string{"codex", "claude"} {
		if err := os.WriteFile(filepath.Join(directory, executable), []byte(contents), 0o700); err != nil {
			t.Fatalf("write fake %s: %v", executable, err)
		}
	}
}

func systemCommandPath(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locate Git: %v", err)
	}
	return filepath.Dir(git) + string(os.PathListSeparator) + "/bin:/usr/bin"
}
