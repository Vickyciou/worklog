package acceptance_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusBeforeSetupPointsToSetup(t *testing.T) {
	environment := newEnvironment(t)

	for _, arguments := range [][]string{nil, {"status"}} {
		result := environment.run(t, arguments...)

		assertSucceeded(t, result)
		assertContains(t, result.stdout, "worklog is not set up yet.")
		assertContains(t, result.stdout, "Next: run worklog setup")
	}
}

func TestStatusReportsEachProfilesJournalStateForToday(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	const email = "status-work@example.invalid"
	for _, profile := range []struct{ name, repository, email string }{
		{name: "work", repository: environment.repository, email: email},
		{name: "personal", repository: personalRepository, email: "status-personal@example.invalid"},
	} {
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", filepath.Join(environment.root, profile.name+"-journal"),
		))
		assertSucceeded(t, environment.run(t, "author", "add", profile.name, profile.email))
		assertSucceeded(t, environment.run(t, "repo", "add", profile.repository, "--profile", profile.name))
	}
	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)
	createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "Private status subject", "")

	notWritten := environment.run(t)
	assertSucceeded(t, notWritten)
	assertStatusLine(t, notWritten.stdout, "work", date, "1 commit", "not written yet")
	assertStatusLine(t, notWritten.stdout, "personal", date, "0 commits", "no activity")
	assertContains(t, notWritten.stdout, "Next: worklog day")
	for _, private := range []string{"Private status subject", email} {
		if strings.Contains(notWritten.stdout, private) {
			t.Fatalf("status printed private value %q:\n%s", private, notWritten.stdout)
		}
	}

	assertSucceeded(t, environment.runWithInput(t, "y\n", "day"))
	upToDate := environment.run(t, "status")
	assertSucceeded(t, upToDate)
	assertStatusLine(t, upToDate.stdout, "work", date, "1 commit", "up to date")
	assertContains(t, upToDate.stdout, "Today's journals are up to date.")

	later := now.Add(time.Minute)
	createCommitAt(t, environment, environment.repository, email,
		later.Format(time.RFC3339), later.Format(time.RFC3339), "Second status subject", "")
	changed := environment.run(t, "status")
	assertSucceeded(t, changed)
	assertStatusLine(t, changed.stdout, "work", date, "2 commits", "has new changes")
	assertContains(t, changed.stdout, "Next: worklog day")

	assertSucceeded(t, environment.run(t,
		"profile", "add", "client",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", filepath.Join(environment.root, "client-journal"),
	))
	attention := environment.run(t, "status")
	assertSucceeded(t, attention)
	assertStatusLine(t, attention.stdout, "client", date, "-", "needs attention")
	assertStatusLine(t, attention.stdout, "work", date, "2 commits", "has new changes")
	assertContains(t, attention.stdout, "Next: worklog doctor")
}

func assertStatusLine(t *testing.T, output string, fields ...string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Join(strings.Fields(line), " ") == strings.Join(fields, " ") {
			return
		}
	}
	t.Fatalf("status output missing line %q:\n%s", strings.Join(fields, " "), output)
}
