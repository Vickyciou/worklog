package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorPassesCompleteConfigurationWithoutExposingPrivateValues(t *testing.T) {
	environment := newEnvironment(t)
	adapter := filepath.Join(environment.root, "healthy-adapter-private-name")
	invocationSentinel := filepath.Join(environment.root, "doctor-must-not-invoke")
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\nprintf invoked > \"$WORKLOG_DOCTOR_INVOKED\"\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write healthy adapter: %v", err)
	}
	environment.environment = append(environment.environment, "WORKLOG_DOCTOR_INVOKED="+invocationSentinel)
	const (
		email         = "doctor-private@example.invalid"
		deniedLiteral = "doctor-denied-private-value"
		capturedValue = "doctor-captured-private-value"
	)
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
		"--deny-literal", deniedLiteral,
		"--summarizer-command", adapter,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	commit := environment.createCommit(t)
	assertSucceeded(t, environment.runWithInput(t,
		`{"version":1,"outcome":"`+capturedValue+`"}`,
		"capture", environment.repository, commit,
	))

	result := environment.run(t, "doctor")
	assertSucceeded(t, result)
	for _, expected := range []string{
		"[OK] Git executable is available",
		"[OK] Config root exists and is writable",
		"[OK] State root exists and is writable",
		"[OK] Configuration is present and valid",
		`[OK] Profile "work" has valid author identities`,
		`[OK] Repository 1 for profile "work" supports the worklog Git notes ref`,
		`[OK] Profile "work" journal destination is writable`,
		`[OK] Profile "work" daily summary executable is available`,
		"Doctor found no problems.",
	} {
		assertContains(t, result.stdout, expected)
	}
	for _, private := range []string{email, deniedLiteral, capturedValue, adapter, environment.repository, environment.journalDestination} {
		if strings.Contains(result.stdout, private) || strings.Contains(result.stderr, private) {
			t.Fatalf("doctor exposed private value %q:\nstdout=%s\nstderr=%s", private, result.stdout, result.stderr)
		}
	}
	if _, err := os.Stat(invocationSentinel); !os.IsNotExist(err) {
		t.Fatalf("doctor invoked configured summarizer: %v", err)
	}
}

func TestDoctorReportsActionableRuntimeFailuresWithoutInvokingSummarizer(t *testing.T) {
	environment := newEnvironment(t)
	missingJournal := filepath.Join(environment.root, "missing-private-journal")
	missingCommand := "missing-private-summarizer"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "incomplete",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", filepath.Join(environment.root, "initial-incomplete-journal"),
		"--summarizer-command", missingCommand,
	))
	assertSucceeded(t, environment.run(t, "profile", "set", "incomplete", "--journal-destination", missingJournal))
	assertSucceeded(t, environment.run(t,
		"profile", "add", "brokenrepo",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "brokenrepo", "brokenrepo@example.invalid"))
	assertSucceeded(t, environment.run(t, "repo", "add", "brokenrepo", environment.repository))
	movedRepository := environment.repository + "-moved-private"
	if err := os.Rename(environment.repository, movedRepository); err != nil {
		t.Fatalf("move registered repository fixture: %v", err)
	}
	environment.environment = cleanEnvironment(environment.environment, map[string]string{"PATH": environment.temporaryDirectory})

	result := environment.runWithInputAt(t, environment.root, "", "doctor")
	if result.exitCode == 0 {
		t.Fatal("broken doctor configuration passed")
	}
	for _, expected := range []string{
		"[FAIL] Git executable is available",
		"Install Git and make sure it is available on PATH",
		`[FAIL] Repository 1 for profile "brokenrepo" is the registered Git root`,
		`[FAIL] Repository 1 for profile "brokenrepo" supports the worklog Git notes ref`,
		`[FAIL] Profile "incomplete" has author identities`,
		"worklog author add incomplete <email>",
		`[FAIL] Profile "incomplete" has registered repositories`,
		"worklog repo add incomplete <path>",
		`[FAIL] Profile "incomplete" journal destination is writable`,
		"worklog profile set incomplete --journal-destination <path>",
		`[FAIL] Profile "incomplete" daily summary executable is available`,
		"--daily-summary built-in",
	} {
		assertContains(t, result.stdout, expected)
	}
	assertContains(t, result.stderr, "doctor found")
	for _, private := range []string{missingJournal, missingCommand, environment.repository, movedRepository, "brokenrepo@example.invalid"} {
		if strings.Contains(result.stdout, private) || strings.Contains(result.stderr, private) {
			t.Fatalf("doctor exposed private value %q:\nstdout=%s\nstderr=%s", private, result.stdout, result.stderr)
		}
	}
}

func TestDoctorRejectsOverlappingMappingsWithoutPrintingConfigValues(t *testing.T) {
	environment := newEnvironment(t)
	configHome := filepath.Join(environment.configRoot, "worklog")
	stateHome := filepath.Join(environment.stateRoot, "worklog")
	for _, directory := range []string{configHome, stateHome} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create doctor root: %v", err)
		}
	}
	secretLiteral := "overlap-denied-private-value"
	configuration := map[string]any{
		"version": 1,
		"profiles": []map[string]any{
			{
				"name": "one", "authors": []string{"one@example.invalid"},
				"repositories": []string{environment.repository}, "timezone": "UTC", "language": "en",
				"journal_destination": filepath.Join(environment.root, "one-journal"), "denied_literals": []string{secretLiteral},
			},
			{
				"name": "two", "authors": []string{"two@example.invalid"},
				"repositories": []string{filepath.Join(environment.repository, "nested-private")}, "timezone": "UTC", "language": "en",
				"journal_destination": filepath.Join(environment.root, "two-journal"),
			},
		},
	}
	contents, err := json.Marshal(configuration)
	if err != nil {
		t.Fatalf("encode overlapping configuration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "config.json"), contents, 0o600); err != nil {
		t.Fatalf("write overlapping configuration: %v", err)
	}

	result := environment.runWithInputAt(t, environment.root, "", "doctor")
	if result.exitCode == 0 {
		t.Fatal("overlapping configuration passed doctor")
	}
	assertContains(t, result.stdout, "[FAIL] Configuration is present and valid")
	assertContains(t, result.stdout, "Remove overlapping repository mappings")
	for _, private := range []string{secretLiteral, environment.repository, "nested-private", "one@example.invalid", "two@example.invalid"} {
		if strings.Contains(result.stdout, private) || strings.Contains(result.stderr, private) {
			t.Fatalf("doctor exposed config value %q:\nstdout=%s\nstderr=%s", private, result.stdout, result.stderr)
		}
	}
}
