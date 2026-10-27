package acceptance_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInteractiveSetupCreatesIsolatedConfiguration(t *testing.T) {
	environment := newEnvironment(t)
	secret := "must-not-appear-in-worklog-config"
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"PROVIDER_API_TOKEN": secret,
		"TZ":                 "Asia/Taipei",
		"LC_ALL":             "",
		"LC_MESSAGES":        "",
		"LANG":               "zh_TW.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
	input := strings.Join([]string{
		"1",
		"",
		environment.journalDestination,
		"",
		"y",
		"",
	}, "\n")

	result := environment.runWithInput(t, input, "setup")

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", result.exitCode, result.stdout, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
	}
	assertContains(t, result.stdout, `Created profile "work".`)
	assertContains(t, result.stdout, "Daily summary:  Built-in (no AI or network)")
	if strings.Contains(result.stdout, secret) {
		t.Fatal("setup output leaked provider secret")
	}

	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	contents, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	configuration := string(contents)
	for _, expected := range []string{
		`"version": 1`,
		`"name": "work"`,
		`"developer@example.com"`,
		`"timezone": "Asia/Taipei"`,
		`"language": "zh-TW"`,
		`"journal_destination": `,
	} {
		assertContains(t, configuration, expected)
	}
	if strings.Contains(configuration, secret) {
		t.Fatal("configuration stored provider secret from environment")
	}
	if strings.Contains(configuration, repositoryRoot()) {
		t.Fatal("configuration was written into or tied to the source repository")
	}
	if _, err := os.Stat(filepath.Join(environment.stateRoot, "worklog")); err != nil {
		t.Fatalf("state home was not created outside source repository: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(configurationPath)
		if err != nil {
			t.Fatalf("stat configuration: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("configuration permissions = %o, want 600", info.Mode().Perm())
		}
	}

	show := environment.run(t, "profile", "show", "work")
	if show.exitCode != 0 {
		t.Fatalf("show profile: exit code = %d; stderr:\n%s", show.exitCode, show.stderr)
	}
	assertContains(t, show.stdout, `"timezone": "Asia/Taipei"`)
	assertContains(t, show.stdout, `"journal_destination": `)
	if strings.Contains(show.stdout, secret) || strings.Contains(show.stderr, secret) {
		t.Fatal("profile output leaked provider secret")
	}

	resolved := environment.run(t, "repo", "show")
	if resolved.exitCode != 0 {
		t.Fatalf("resolve repository: exit code = %d; stderr:\n%s", resolved.exitCode, resolved.stderr)
	}
	for _, expected := range []string{
		`"profile": "work"`,
		`"timezone": "Asia/Taipei"`,
		`"journal_destination": `,
	} {
		assertContains(t, resolved.stdout, expected)
	}
}

func TestSetupSupportsPersonalAndCustomProfiles(t *testing.T) {
	tests := []struct {
		name       string
		inputLines []string
		profile    string
	}{
		{name: "personal", inputLines: []string{"2"}, profile: "personal"},
		{name: "custom", inputLines: []string{"3", "client-a"}, profile: "client-a"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			environment := newEnvironment(t)
			environment.environment = cleanEnvironment(environment.environment, map[string]string{
				"TZ":          "Asia/Taipei",
				"LC_ALL":      "",
				"LC_MESSAGES": "",
				"LANG":        "zh_TW.UTF-8",
			})
			assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
			input := append([]string{}, test.inputLines...)
			input = append(input, "", environment.journalDestination, "", "y", "")

			result := environment.runWithInput(t, strings.Join(input, "\n"), "setup")

			assertSucceeded(t, result)
			assertContains(t, result.stdout, `Created profile "`+test.profile+`".`)
			show := environment.run(t, "profile", "show", test.profile)
			assertSucceeded(t, show)
		})
	}
}

func TestSetupAcceptsRepositoryAndJournalDefaults(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "Asia/Taipei",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "zh_TW.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
	nestedDirectory := filepath.Join(environment.repository, "nested", "directory")
	if err := os.MkdirAll(nestedDirectory, 0o700); err != nil {
		t.Fatalf("create nested working directory: %v", err)
	}
	input := strings.Join([]string{"", "", "", "", "y", ""}, "\n")

	result := environment.runWithInputAt(t, nestedDirectory, input, "setup")

	assertSucceeded(t, result)
	expectedJournal := filepath.Join(environment.home, "Documents", "worklog", "work")
	assertContains(t, result.stdout, "Repository ["+canonicalTestPath(t, environment.repository)+"]:")
	assertContains(t, result.stdout, "Journal folder ["+expectedJournal+"]:")
	show := environment.run(t, "profile", "show", "work")
	assertSucceeded(t, show)
	assertContains(t, show.stdout, `"journal_destination": "`+expectedJournal+`"`)
}

func TestSetupExpandsHomePaths(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	repository := filepath.Join(environment.home, "other-repository")
	initializeGitRepository(t, repository, environment.environment)
	git := exec.Command("git", "-C", repository, "config", "user.email", "other@example.invalid")
	git.Env = environment.environment
	if result := runCommand(t, git); result.exitCode != 0 {
		t.Fatalf("configure alternate repository: %s", result.stderr)
	}
	input := strings.Join([]string{"2", "~/other-repository", "~/journals/personal", "", "y", ""}, "\n")

	result := environment.runWithInput(t, input, "setup")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Author:         other@example.invalid")
	show := environment.run(t, "profile", "show", "personal")
	assertSucceeded(t, show)
	assertContains(t, show.stdout, `"journal_destination": "`+filepath.Join(environment.home, "journals", "personal")+`"`)
	repositories := environment.run(t, "repo", "list", "personal")
	assertSucceeded(t, repositories)
	assertContains(t, repositories.stdout, canonicalTestPath(t, repository))
	authors := environment.run(t, "author", "list", "personal")
	assertSucceeded(t, authors)
	if authors.stdout != "other@example.invalid\n" {
		t.Fatalf("author list = %q, want selected repository author", authors.stdout)
	}
}

func TestSetupPromptsOnlyWhenGitAuthorCannotBeDetected(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	input := strings.Join([]string{
		"1",
		"",
		"fallback@example.invalid",
		environment.journalDestination,
		"",
		"y",
		"",
	}, "\n")

	result := environment.runWithInput(t, input, "setup")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Author email (not available from Git config):")
	assertContains(t, result.stdout, "Language:       en-US")
}

func TestSetupPromptsForUndetectableTimezoneAndLanguage(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "Not/A-Timezone",
		"LC_ALL":      "C",
		"LC_MESSAGES": "C",
		"LANG":        "C",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
	input := strings.Join([]string{
		"1",
		"",
		"Asia/Taipei",
		"en_US",
		environment.journalDestination,
		"",
		"y",
		"",
	}, "\n")

	result := environment.runWithInput(t, input, "setup")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Timezone (could not be detected) [UTC]:")
	assertContains(t, result.stdout, "Language (could not be detected) [zh-TW]:")
	assertContains(t, result.stdout, "Timezone:       Asia/Taipei")
	assertContains(t, result.stdout, "Language:       en-US")
}

func TestSetupCancellationHasNoSideEffects(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
	journal := filepath.Join(environment.root, "cancelled-journal")
	input := strings.Join([]string{"1", "", journal, "", "n", ""}, "\n")

	result := environment.runWithInput(t, input, "setup")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Setup cancelled.")
	if count := strings.Count(result.stdout, "Create these settings?"); count != 1 {
		t.Fatalf("confirmation prompt count = %d, want 1", count)
	}
	for _, path := range []string{
		filepath.Join(environment.configRoot, "worklog"),
		filepath.Join(environment.stateRoot, "worklog"),
		journal,
	} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cancelled setup created %q: %v", path, err)
		}
	}
}

func TestGuidedProfileAddCreatesProfileAuthorAndRepository(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "Asia/Taipei",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "zh_TW.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "personal@example.invalid"))
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en-US",
		"--journal-destination", filepath.Join(environment.root, "work-journal"),
	))
	personalJournal := filepath.Join(environment.root, "personal-journal")
	input := strings.Join([]string{"", "", personalJournal, "", "y", ""}, "\n")

	result := environment.runWithInput(t, input, "profile", "add")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "Choose a profile:")
	assertContains(t, result.stdout, "1. Work      Already configured")
	assertContains(t, result.stdout, "Profile [2]:")
	assertContains(t, result.stdout, "Ready to add:")
	assertContains(t, result.stdout, `Added profile "personal" with its author and repository.`)
	if strings.Count(result.stdout, "Add this profile?") != 1 {
		t.Fatalf("guided add confirmation count is not one:\n%s", result.stdout)
	}
	show := environment.run(t, "profile", "show", "personal")
	assertSucceeded(t, show)
	for _, expected := range []string{
		`"personal@example.invalid"`,
		`"` + canonicalTestPath(t, environment.repository) + `"`,
		`"timezone": "Asia/Taipei"`,
		`"language": "zh-TW"`,
		`"journal_destination": "` + personalJournal + `"`,
	} {
		assertContains(t, show.stdout, expected)
	}
	if _, err := os.Stat(personalJournal); err != nil {
		t.Fatalf("guided add did not create journal directory: %v", err)
	}
}

func TestGuidedProfileAddWithNameCancellationHasNoSideEffects(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "client@example.invalid"))
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en-US",
		"--journal-destination", filepath.Join(environment.root, "work-journal"),
	))
	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration before guided add: %v", err)
	}
	journal := filepath.Join(environment.root, "cancelled-client-journal")
	input := strings.Join([]string{"", journal, "", "n", ""}, "\n")

	result := environment.runWithInput(t, input, "profile", "add", "client-a")

	assertSucceeded(t, result)
	if strings.Contains(result.stdout, "Choose a profile:") {
		t.Fatal("specified profile name unexpectedly showed the profile menu")
	}
	assertContains(t, result.stdout, "Profile not added.")
	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration after guided add: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("cancelled guided add changed config bytes")
	}
	if _, err := os.Stat(journal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled guided add created journal directory: %v", err)
	}
}

func TestGuidedProfileAddRejectsRepositoryConflictWithoutSideEffects(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.invalid"))
	setupInput := strings.Join([]string{"1", "", filepath.Join(environment.root, "work-journal"), "", "y", ""}, "\n")
	assertSucceeded(t, environment.runWithInput(t, setupInput, "setup"))
	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration before conflicting add: %v", err)
	}
	journal := filepath.Join(environment.root, "conflicting-personal-journal")
	input := strings.Join([]string{"", journal, "", "y", ""}, "\n")

	result := environment.runWithInput(t, input, "profile", "add", "personal")

	if result.exitCode == 0 {
		t.Fatal("conflicting guided add succeeded")
	}
	assertContains(t, result.stderr, "overlaps")
	assertContains(t, result.stderr, "profile \"work\"")
	assertContains(t, result.stderr, "profile \"personal\"")
	if strings.Contains(result.stdout, "Add this profile?") {
		t.Fatal("conflicting guided add reached confirmation")
	}
	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration after conflicting add: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("conflicting guided add changed config bytes")
	}
	if _, err := os.Stat(journal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflicting guided add created journal directory: %v", err)
	}
}

func TestSettingsUpdatesTheOnlyProfileWithoutMovingJournals(t *testing.T) {
	environment := newEnvironment(t)
	oldJournal := filepath.Join(environment.root, "old-journal")
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", oldJournal,
	))
	newJournal := filepath.Join(environment.home, "journals", "work")
	input := strings.Join([]string{"en_US.UTF-8", "Asia/Taipei", "~/journals/work", "", "y", ""}, "\n")

	result := environment.runWithInput(t, input, "settings")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "run worklog day <date> --refresh to regenerate one")
	assertContains(t, result.stdout, "Current settings for work:")
	assertContains(t, result.stdout, "Daily summary:  Built-in (no AI or network)")
	assertContains(t, result.stdout, "Existing journals will stay in their current folder")
	if strings.Count(result.stdout, "Save these settings?") != 1 {
		t.Fatalf("settings confirmation count is not one:\n%s", result.stdout)
	}
	show := environment.run(t, "profile", "show", "work")
	assertSucceeded(t, show)
	assertContains(t, show.stdout, `"language": "en-US"`)
	assertContains(t, show.stdout, `"timezone": "Asia/Taipei"`)
	assertContains(t, show.stdout, `"journal_destination": "`+newJournal+`"`)
	if _, err := os.Stat(oldJournal); err != nil {
		t.Fatalf("old journal directory was removed: %v", err)
	}
	if _, err := os.Stat(newJournal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("settings unexpectedly created new journal directory: %v", err)
	}
}

func TestSettingsChoosesAmongProfilesAndCancellationHasNoSideEffects(t *testing.T) {
	environment := newEnvironment(t)
	for _, name := range []string{"work", "personal"} {
		assertSucceeded(t, environment.run(t,
			"profile", "add", name,
			"--timezone", "UTC",
			"--language", "zh-TW",
			"--journal-destination", filepath.Join(environment.root, name+"-journal"),
		))
	}
	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration before settings: %v", err)
	}
	newJournal := filepath.Join(environment.root, "cancelled-settings-journal")
	input := strings.Join([]string{"2", "en-US", "Asia/Taipei", newJournal, "", "n", ""}, "\n")

	result := environment.runWithInput(t, input, "settings")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, "1. personal")
	assertContains(t, result.stdout, "2. work")
	assertContains(t, result.stdout, "Settings unchanged.")
	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration after settings: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("cancelled settings changed config bytes")
	}
	if _, err := os.Stat(newJournal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled settings created journal directory: %v", err)
	}
}

func TestProfileSetUpdatesOnlyTheNamedProfile(t *testing.T) {
	environment := newEnvironment(t)
	for _, name := range []string{"work", "personal"} {
		assertSucceeded(t, environment.run(t,
			"profile", "add", name,
			"--timezone", "UTC",
			"--language", "zh-TW",
			"--journal-destination", filepath.Join(environment.root, name+"-journal"),
		))
	}
	newJournal := filepath.Join(environment.home, "journals", "work")

	result := environment.run(t, "profile", "set", "work",
		"--language", "en_US.UTF-8",
		"--timezone", "Asia/Taipei",
		"--journal-destination", "~/journals/work",
	)

	assertSucceeded(t, result)
	work := environment.run(t, "profile", "show", "work")
	assertSucceeded(t, work)
	assertContains(t, work.stdout, `"language": "en-US"`)
	assertContains(t, work.stdout, `"timezone": "Asia/Taipei"`)
	assertContains(t, work.stdout, `"journal_destination": "`+newJournal+`"`)
	personal := environment.run(t, "profile", "show", "personal")
	assertSucceeded(t, personal)
	assertContains(t, personal.stdout, `"language": "zh-TW"`)
	assertContains(t, personal.stdout, `"timezone": "UTC"`)
	if _, err := os.Stat(newJournal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile set unexpectedly created new journal directory: %v", err)
	}
}

func TestProfileSetRejectsMissingOrInvalidUpdatesWithoutSaving(t *testing.T) {
	environment := newEnvironment(t)
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration before invalid updates: %v", err)
	}

	missing := environment.run(t, "profile", "set", "work")
	assertFailedWith(t, missing, "at least one profile setting is required")
	invalidTimezone := environment.run(t, "profile", "set", "work", "--timezone", "Not/A-Timezone")
	assertFailedWith(t, invalidTimezone, "invalid timezone")
	invalidLanguage := environment.run(t, "profile", "set", "work", "--language", "C")
	assertFailedWith(t, invalidLanguage, "language is required")

	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration after invalid updates: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid profile update changed config bytes")
	}
}

func TestProfileAuthorAndRepositoryCommands(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	personalJournal := filepath.Join(environment.root, "personal-journal")

	addWork := environment.run(t,
		"profile", "add", "work",
		"--timezone", "Asia/Taipei",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	)
	assertSucceeded(t, addWork)
	assertSucceeded(t, environment.run(t, "author", "add", "work", "developer@example.com"))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))

	addPersonal := environment.run(t,
		"profile", "add", "personal",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", personalJournal,
	)
	assertSucceeded(t, addPersonal)
	assertSucceeded(t, environment.run(t, "author", "add", "personal", "personal@example.net"))
	assertSucceeded(t, environment.run(t, "repo", "add", "personal", personalRepository))

	profiles := environment.run(t, "profile", "list")
	assertSucceeded(t, profiles)
	if profiles.stdout != "personal\nwork\n" {
		t.Fatalf("profile list = %q, want sorted profiles", profiles.stdout)
	}
	authors := environment.run(t, "author", "list", "work")
	assertSucceeded(t, authors)
	if authors.stdout != "developer@example.com\n" {
		t.Fatalf("author list = %q", authors.stdout)
	}
	repositories := environment.run(t, "repo", "list")
	assertSucceeded(t, repositories)
	assertContains(t, repositories.stdout, "work\t"+canonicalTestPath(t, environment.repository))
	assertContains(t, repositories.stdout, "personal\t"+canonicalTestPath(t, personalRepository))

	for _, arguments := range [][]string{
		{"profile", "validate"},
		{"profile", "validate", "work"},
		{"author", "validate", "work"},
		{"repo", "validate"},
		{"repo", "validate", personalRepository},
	} {
		assertSucceeded(t, environment.run(t, arguments...))
	}
}

func TestRepositoryMappingsRejectDuplicatesAndOverlaps(t *testing.T) {
	environment := newEnvironment(t)
	nestedRepository := filepath.Join(environment.repository, "nested")
	initializeGitRepository(t, nestedRepository, environment.environment)

	for _, profile := range []string{"work", "personal"} {
		journal := filepath.Join(environment.root, profile+"-journal")
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", journal,
		))
	}
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))

	duplicate := environment.run(t, "repo", "add", "personal", environment.repository)
	assertFailedWith(t, duplicate, "overlaps", "already owned by profile \"work\"")
	overlap := environment.run(t, "repo", "add", "personal", nestedRepository)
	assertFailedWith(t, overlap, "overlaps", "already owned by profile \"work\"")

	repositories := environment.run(t, "repo", "list")
	assertSucceeded(t, repositories)
	if strings.Contains(repositories.stdout, nestedRepository) {
		t.Fatal("conflicting nested repository was persisted")
	}
}

func TestConfigAndStateHomeEnvironmentOverrides(t *testing.T) {
	environment := newEnvironment(t)
	configHome := filepath.Join(environment.root, "custom-config-home")
	stateHome := filepath.Join(environment.root, "custom-state-home")
	environment.environment = append(environment.environment,
		"WORKLOG_CONFIG_HOME="+configHome,
		"WORKLOG_STATE_HOME="+stateHome,
	)

	result := environment.run(t,
		"profile", "add", "portable",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
	)
	assertSucceeded(t, result)
	if _, err := os.Stat(filepath.Join(configHome, "config.json")); err != nil {
		t.Fatalf("environment-controlled config home was not used: %v", err)
	}
	if _, err := os.Stat(stateHome); err != nil {
		t.Fatalf("environment-controlled state home was not used: %v", err)
	}
	if _, err := os.Stat(filepath.Join(environment.configRoot, "worklog", "config.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default config location unexpectedly used: %v", err)
	}
}

func TestSecretLikeSummarizerValueIsRejected(t *testing.T) {
	commands := []string{
		"summary --api-key=literal-secret",
		"API_KEY=literal-secret summary",
		"summary --api_key literal-secret",
		"summary --token literal-secret",
		"summary --password literal-secret",
		`summary --header "Authorization: Bearer literal-secret"`,
		"summary Authorization: Bearer literal-secret",
		`summary --api-key "$PROVIDER_API_KEY" --password literal-secret`,
	}

	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t,
				"profile", "add", "unsafe",
				"--timezone", "UTC",
				"--language", "en",
				"--journal-destination", environment.journalDestination,
				"--summarizer-command", command,
			)

			assertFailedWith(t, result, "appears to contain a secret")
			if strings.Contains(result.stdout, "literal-secret") || strings.Contains(result.stderr, "literal-secret") {
				t.Fatal("error output leaked rejected secret value")
			}
			configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
			if contents, err := os.ReadFile(configurationPath); err == nil && strings.Contains(string(contents), "literal-secret") {
				t.Fatal("rejected secret was stored")
			} else if err == nil {
				t.Fatal("rejected summarizer command created a configuration file")
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("inspect rejected configuration: %v", err)
			}
		})
	}
}

func TestSummarizerCommandAllowsIndirectEnvironmentConfiguration(t *testing.T) {
	commands := []string{
		"summary-adapter --credentials-from-env",
		"summary-adapter --api-key-env PROVIDER_API_KEY",
		`summary-adapter --api-key "$PROVIDER_API_KEY"`,
		`summary-adapter --header "Authorization: Bearer ${PROVIDER_TOKEN}"`,
	}

	for index, command := range commands {
		t.Run(command, func(t *testing.T) {
			environment := newEnvironment(t)
			profile := fmt.Sprintf("indirect-%d", index)
			result := environment.run(t,
				"profile", "add", profile,
				"--timezone", "UTC",
				"--language", "en",
				"--journal-destination", environment.journalDestination,
				"--summarizer-command", command,
			)

			assertSucceeded(t, result)
			contents, err := os.ReadFile(filepath.Join(environment.configRoot, "worklog", "config.json"))
			if err != nil {
				t.Fatalf("read configuration: %v", err)
			}
			var configuration struct {
				Profiles []struct {
					SummarizerCommand string `json:"summarizer_command"`
				} `json:"profiles"`
			}
			if err := json.Unmarshal(contents, &configuration); err != nil {
				t.Fatalf("decode configuration: %v", err)
			}
			if len(configuration.Profiles) != 1 || configuration.Profiles[0].SummarizerCommand != command {
				t.Fatalf("summarizer command = %q, want %q", configuration.Profiles[0].SummarizerCommand, command)
			}
		})
	}
}

func initializeGitRepository(t *testing.T, path string, environment []string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("create Git repository directory %q: %v", path, err)
	}
	git := exec.Command("git", "init", "--quiet", path)
	git.Env = environment
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("initialize isolated Git repository: %v\n%s", err, output)
	}
}

func canonicalTestPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve fixture path %q: %v", path, err)
	}
	return canonical
}

func assertSucceeded(t *testing.T, result processResult) {
	t.Helper()
	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", result.exitCode, result.stdout, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("stderr = %q, want empty", result.stderr)
	}
}

func assertFailedWith(t *testing.T, result processResult, expectedSubstrings ...string) {
	t.Helper()
	if result.exitCode == 0 {
		t.Fatalf("exit code = 0, want failure; stdout:\n%s", result.stdout)
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
	for _, expected := range expectedSubstrings {
		assertContains(t, result.stderr, expected)
	}
}

func TestGuidedPromptsCancelWhenInputCloses(t *testing.T) {
	environment := newEnvironment(t)
	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "en_US.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", "developer@example.com"))
	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")

	setup := environment.runWithInput(t, "1\n", "setup")
	assertInputClosedCancellation(t, setup)
	if _, err := os.Stat(configurationPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("setup with closed input created configuration: %v", err)
	}

	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en-US",
		"--journal-destination", filepath.Join(environment.root, "work-journal"),
	))
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	for _, arguments := range [][]string{{"settings"}, {"profile", "add", "personal"}} {
		result := environment.run(t, arguments...)
		assertInputClosedCancellation(t, result)
		after, err := os.ReadFile(configurationPath)
		if err != nil {
			t.Fatalf("read configuration after %v: %v", arguments, err)
		}
		if string(after) != string(before) {
			t.Fatalf("%v with closed input changed configuration", arguments)
		}
	}
}

func TestGuidedPromptAcceptsFinalAnswerWithoutNewline(t *testing.T) {
	environment := newEnvironment(t)
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en-US",
		"--journal-destination", filepath.Join(environment.root, "work-journal"),
	))

	result := environment.runWithInput(t, "zh-TW\n\n\n\ny", "settings")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, `Updated profile "work".`)
}

func assertInputClosedCancellation(t *testing.T, result processResult) {
	t.Helper()
	if result.exitCode == 0 {
		t.Fatalf("exit code = 0, want failure; stdout:\n%s", result.stdout)
	}
	assertContains(t, result.stderr, "no answer received (input closed); nothing was saved")
	if strings.Contains(result.stdout, "Created profile") || strings.Contains(result.stdout, "Updated profile") || strings.Contains(result.stdout, "Added profile") {
		t.Fatalf("closed input reported a saved profile:\n%s", result.stdout)
	}
}
