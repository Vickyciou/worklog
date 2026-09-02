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
	environment.environment = append(environment.environment, "PROVIDER_API_TOKEN="+secret)
	input := strings.Join([]string{
		"work",
		"developer@example.com",
		environment.repository,
		"Asia/Taipei",
		"zh-TW",
		environment.journalDestination,
		"summary-adapter --format json",
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
		`"summarizer_command": "summary-adapter --format json"`,
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
