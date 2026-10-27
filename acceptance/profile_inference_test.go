package acceptance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func addTestProfile(t *testing.T, environment acceptanceEnvironment, name string) {
	t.Helper()
	assertSucceeded(t, environment.run(t, "profile", "add", name,
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", filepath.Join(environment.root, name+"-journal"),
	))
}

func TestRepoAddUsesCurrentRepositoryAndOnlyProfile(t *testing.T) {
	environment := newEnvironment(t)
	addTestProfile(t, environment, "work")

	result := environment.run(t, "repo", "add")

	assertSucceeded(t, result)
	assertContains(t, result.stdout, `to profile "work"`)
	list := environment.run(t, "repo", "list", "work")
	assertSucceeded(t, list)
	assertContains(t, list.stdout, canonicalTestPath(t, environment.repository))
}

func TestRepoAddWithSeveralProfilesUsesFlagOrAsks(t *testing.T) {
	environment := newEnvironment(t)
	for _, name := range []string{"work", "personal"} {
		addTestProfile(t, environment, name)
	}
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)

	flagged := environment.run(t, "repo", "add", personalRepository, "--profile", "personal")
	assertSucceeded(t, flagged)
	assertContains(t, flagged.stdout, `to profile "personal"`)

	configurationPath := filepath.Join(environment.configRoot, "worklog", "config.json")
	before, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	assertInputClosedCancellation(t, environment.runWithInput(t, "", "repo", "add"))
	after, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("read configuration: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("repo add saved a mapping after input closed")
	}

	asked := environment.runWithInput(t, "work\n", "repo", "add")
	assertSucceeded(t, asked)
	assertContains(t, asked.stdout, "Choose a profile:")
	assertContains(t, asked.stdout, `to profile "work"`)
}

func TestRepoAddRejectsPositionalProfileWithReplacement(t *testing.T) {
	environment := newEnvironment(t)
	addTestProfile(t, environment, "work")

	result := environment.run(t, "repo", "add", "work", environment.repository)

	assertFailedWith(t, result, "Use: worklog repo add "+environment.repository+" --profile work")
	list := environment.run(t, "repo", "list", "work")
	assertSucceeded(t, list)
	if list.stdout != "" {
		t.Fatalf("rejected repo add registered a repository: %q", list.stdout)
	}
}

func TestInspectInfersProfileFromRepositoryOrSingleProfile(t *testing.T) {
	environment := newEnvironment(t)
	const email = "inspect-infer@example.invalid"
	addTestProfile(t, environment, "work")
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "--profile", "work"))
	now := time.Now().UTC().Truncate(time.Second)
	createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "Inferred profile change", "")
	date := now.Format(time.DateOnly)

	single := environment.runWithInputAt(t, environment.root, "", "inspect", date)
	assertSucceeded(t, single)
	assertContains(t, single.stdout, "Inferred profile change")

	addTestProfile(t, environment, "personal")
	owned := environment.run(t, "inspect", date)
	assertSucceeded(t, owned)
	assertContains(t, owned.stdout, "Inferred profile change")

	ambiguous := environment.runWithInputAt(t, environment.root, "", "inspect")
	assertFailedWith(t, ambiguous, "worklog inspect <profile>", "work", "personal")
	if strings.Contains(ambiguous.stderr, "Choose a profile:") {
		t.Fatal("inspect prompted instead of failing")
	}
}
