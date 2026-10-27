package acceptance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectEmitsProfileScopedPostRedactionPayload(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	const (
		workEmail     = "work-redaction@example.invalid"
		personalEmail = "personal-redaction@example.invalid"
		targetDate    = "2026-09-03"
	)
	profiles := []struct {
		name       string
		repository string
		email      string
		denied     string
	}{
		{name: "work", repository: environment.repository, email: workEmail, denied: "work-policy-marker"},
		{name: "personal", repository: personalRepository, email: personalEmail, denied: "personal-policy-marker"},
	}
	for _, profile := range profiles {
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", filepath.Join(environment.root, profile.name+"-journal"),
			"--deny-literal", profile.denied,
		))
		assertSucceeded(t, environment.run(t, "author", "add", profile.name, profile.email))
		assertSucceeded(t, environment.run(t, "repo", "add", profile.repository, "--profile", profile.name))
	}

	workCommit := createCommitAt(t, environment, environment.repository, workEmail,
		"2026-09-03T09:00:00Z", "2026-09-03T09:00:00Z", "work metadata", "")
	workEvent := fmt.Sprintf(`{"version":1,"outcome":"work-only-outcome work-policy-marker personal-policy-marker %s","approach":"Use %s with token=fixture-secret-token and developer@example.invalid","next_steps":["diff --git a/private.go b/private.go\n@@ -1 +1 @@\n-secret\n+secret"],"technologies":["Go"]}`,
		workCommit, environment.repository)
	assertSucceeded(t, environment.runWithInput(t, workEvent, "capture", environment.repository, workCommit))

	personalCommit := createCommitAt(t, environment, personalRepository, personalEmail,
		"2026-09-03T10:00:00Z", "2026-09-03T10:00:00Z", "personal metadata", "")
	personalEvent := `{"version":1,"outcome":"personal-only-outcome personal-policy-marker work-policy-marker","approach":"Authorization: Bearer fixture-bearer-secret","technologies":["Git"]}`
	assertSucceeded(t, environment.runWithInput(t, personalEvent, "capture", personalRepository, personalCommit))

	work := environment.run(t, "inspect", "work", targetDate)
	assertSucceeded(t, work)
	personal := environment.run(t, "inspect", "personal", targetDate)
	assertSucceeded(t, personal)

	for _, forbidden := range []string{workCommit, environment.repository, workEmail, "work-policy-marker", "fixture-secret-token", "developer@example.invalid", "diff --git", "private.go"} {
		if strings.Contains(work.stdout, forbidden) {
			t.Fatalf("work payload contains %q:\n%s", forbidden, work.stdout)
		}
	}
	for _, forbidden := range []string{"personal-policy-marker", "fixture-bearer-secret"} {
		if strings.Contains(personal.stdout, forbidden) {
			t.Fatalf("personal payload contains %q:\n%s", forbidden, personal.stdout)
		}
	}
	assertContains(t, work.stdout, "personal-policy-marker")
	assertContains(t, personal.stdout, "work-policy-marker")
	assertContains(t, work.stdout, "work-only-outcome")
	assertContains(t, personal.stdout, "personal-only-outcome")
	if strings.Contains(work.stdout, "personal-only-outcome") || strings.Contains(personal.stdout, "work-only-outcome") {
		t.Fatal("inspect mixed profile activity")
	}
}

func TestInspectCollectsOnlyTheProfileDailyFactsAtLocalDateBoundaries(t *testing.T) {
	environment := newEnvironment(t)
	const (
		profileEmail = "daily-facts@example.invalid"
		targetDate   = "2026-09-02"
	)

	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "Asia/Taipei",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
		"--summarizer-command", "summarizer-that-must-not-run",
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", profileEmail))
	assertSucceeded(t, environment.run(t, "repo", "add", environment.repository, "--profile", "work"))

	beforeBoundary := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-01T15:59:59Z", "2026-09-01T15:59:59Z", "before local boundary", "")
	jsonCommit := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-01T16:00:00Z", "2026-09-01T16:00:00Z", "JSON metadata must be replaced", "private-body-marker")
	lineCommit := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-02T05:00:00Z", "2026-09-02T05:00:00Z", "line metadata must be replaced", "")
	if err := os.WriteFile(filepath.Join(environment.repository, "private.go"), []byte("package fixture // private-source-code-marker\n"), 0o600); err != nil {
		t.Fatalf("write private source fixture: %v", err)
	}
	assertSucceeded(t, environment.git(t, "add", "private.go"))
	fallbackCommit := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-02T15:59:59Z", "2026-09-02T15:59:59Z", "Fallback metadata only", "private-body-marker")
	afterBoundary := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-02T16:00:00Z", "2026-09-02T16:00:00Z", "after local boundary", "")
	wrongAuthor := createCommitAt(t, environment, environment.repository, "other@example.invalid",
		"2026-09-02T06:00:00Z", "2026-09-02T06:00:00Z", "wrong author marker", "")
	commitTimestampWins := createCommitAt(t, environment, environment.repository, profileEmail,
		"2026-09-02T06:30:00Z", "2026-09-02T16:30:00Z", "author date must not select commit", "")

	jsonEvent := `{"version":1,"outcome":"Recorded JSON outcome","approach":"Used structured facts","decisions":[{"decision":"Keep the contract minimal","reason":"Protect private metadata"}],"next_steps":["Render the daily journal"],"technologies":["Go","Git"],"source":{"kind":"agent","name":"private-source-marker"}}`
	assertSucceeded(t, environment.runWithInput(t, jsonEvent, "capture", environment.repository, jsonCommit))
	lineNote := strings.Join([]string{
		"Worklog-Version: 1",
		"Outcome: Read a version-one note",
		"Approach: Parsed labeled lines",
		"Decision: Preserve old notes without rewriting them",
		"Next: Extend collection in the day command",
		"Tech: Go, Git",
	}, "\n")
	assertSucceeded(t, gitAt(t, environment, environment.repository, profileEmail,
		"2026-09-02T07:00:00Z", "notes", "--ref=refs/notes/worklog", "add", "-m", lineNote, lineCommit))

	assertSucceeded(t, environment.git(t, "branch", "duplicate-ref", jsonCommit))
	assertSucceeded(t, environment.git(t, "tag", "duplicate-tag", jsonCommit))
	mergeTree := environment.git(t, "rev-parse", fallbackCommit+"^{tree}")
	assertSucceeded(t, mergeTree)
	merge := gitAt(t, environment, environment.repository, profileEmail, "2026-09-02T08:00:00Z",
		"commit-tree", strings.TrimSpace(mergeTree.stdout), "-p", fallbackCommit, "-p", jsonCommit, "-m", "merge marker")
	assertSucceeded(t, merge)
	assertSucceeded(t, environment.git(t, "update-ref", "refs/heads/merge-fixture", strings.TrimSpace(merge.stdout)))

	unregistered := filepath.Join(environment.root, "unregistered")
	initializeGitRepository(t, unregistered, environment.environment)
	createCommitAt(t, environment, unregistered, profileEmail,
		"2026-09-02T09:00:00Z", "2026-09-02T09:00:00Z", "unregistered repository marker", "")

	journalSentinel := filepath.Join(environment.journalDestination, "manual-note.md")
	if err := os.WriteFile(journalSentinel, []byte("manual journal content\n"), 0o600); err != nil {
		t.Fatalf("write journal sentinel: %v", err)
	}
	notesRefBefore := environment.git(t, "rev-parse", "refs/notes/worklog")
	assertSucceeded(t, notesRefBefore)

	result := environment.run(t, "inspect", "work", targetDate)
	assertSucceeded(t, result)

	var document struct {
		Version  int    `json:"version"`
		Profile  string `json:"profile"`
		Date     string `json:"date"`
		Timezone string `json:"timezone"`
		Facts    []struct {
			CommittedAt string `json:"committed_at"`
			Outcome     string `json:"outcome"`
			Approach    string `json:"approach"`
			Decisions   []struct {
				Decision string  `json:"decision"`
				Reason   *string `json:"reason"`
			} `json:"decisions"`
			NextSteps    []string `json:"next_steps"`
			Technologies []string `json:"technologies"`
		} `json:"facts"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
		t.Fatalf("decode inspect output: %v\n%s", err, result.stdout)
	}
	if document.Version != 1 || document.Profile != "work" || document.Date != targetDate || document.Timezone != "Asia/Taipei" {
		t.Fatalf("unexpected DailyFacts envelope: %+v", document)
	}
	if len(document.Facts) != 3 {
		t.Fatalf("facts count = %d, want 3; output:\n%s", len(document.Facts), result.stdout)
	}
	if document.Facts[0].CommittedAt != "2026-09-02T00:00:00+08:00" || document.Facts[0].Outcome != "Recorded JSON outcome" {
		t.Fatalf("JSON fact = %+v", document.Facts[0])
	}
	if len(document.Facts[0].Decisions) != 1 || document.Facts[0].Decisions[0].Reason == nil || *document.Facts[0].Decisions[0].Reason != "Protect private metadata" {
		t.Fatalf("JSON decision was not preserved: %+v", document.Facts[0].Decisions)
	}
	if document.Facts[1].CommittedAt != "2026-09-02T13:00:00+08:00" || document.Facts[1].Outcome != "Read a version-one note" {
		t.Fatalf("line-oriented fact = %+v", document.Facts[1])
	}
	if len(document.Facts[1].Decisions) != 1 || document.Facts[1].Decisions[0].Reason != nil {
		t.Fatalf("line-oriented decision invented a reason: %+v", document.Facts[1].Decisions)
	}
	if document.Facts[2].CommittedAt != "2026-09-02T23:59:59+08:00" || document.Facts[2].Outcome != "Fallback metadata only" {
		t.Fatalf("metadata fallback fact = %+v", document.Facts[2])
	}
	if document.Facts[2].Approach != "" || len(document.Facts[2].Decisions) != 0 || len(document.Facts[2].NextSteps) != 0 {
		t.Fatalf("metadata fallback invented details: %+v", document.Facts[2])
	}

	for _, forbidden := range []string{
		jsonCommit, lineCommit, fallbackCommit, beforeBoundary, afterBoundary, wrongAuthor, commitTimestampWins,
		environment.repository, unregistered, profileEmail, "other@example.invalid", "private-body-marker",
		"private-source-code-marker", "private-source-marker", "merge marker", "unregistered repository marker",
		"before local boundary", "after local boundary",
	} {
		if strings.Contains(result.stdout, forbidden) {
			t.Fatalf("inspect output contains forbidden value %q:\n%s", forbidden, result.stdout)
		}
	}
	contents, err := os.ReadFile(journalSentinel)
	if err != nil || string(contents) != "manual journal content\n" {
		t.Fatalf("inspect changed journal sentinel: contents=%q error=%v", contents, err)
	}
	notesRefAfter := environment.git(t, "rev-parse", "refs/notes/worklog")
	if notesRefAfter.exitCode != 0 || notesRefAfter.stdout != notesRefBefore.stdout {
		t.Fatalf("inspect rewrote worklog notes: before=%q after=%q error=%s", notesRefBefore.stdout, notesRefAfter.stdout, notesRefAfter.stderr)
	}
}

func TestInspectDefaultsToTheProfilesCurrentLocalDate(t *testing.T) {
	environment := newEnvironment(t)
	const profileEmail = "local-date@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "local",
		"--timezone", "Pacific/Kiritimati",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "local", profileEmail))
	assertSucceeded(t, environment.run(t, "repo", "add", environment.repository, "--profile", "local"))

	location, err := time.LoadLocation("Pacific/Kiritimati")
	if err != nil {
		t.Fatalf("load fixture timezone: %v", err)
	}
	dateBefore := time.Now().In(location).Format(time.DateOnly)
	result := environment.run(t, "inspect", "local")
	dateAfter := time.Now().In(location).Format(time.DateOnly)
	assertSucceeded(t, result)
	var envelope struct {
		Date     string `json:"date"`
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal([]byte(result.stdout), &envelope); err != nil {
		t.Fatalf("decode inspect output: %v", err)
	}
	if envelope.Timezone != "Pacific/Kiritimati" || envelope.Date != dateBefore && envelope.Date != dateAfter {
		t.Fatalf("inspect output did not use profile-local current date: %+v", envelope)
	}
}

func createCommitAt(t *testing.T, environment acceptanceEnvironment, repository, email, authorDate, commitDate, subject, body string) string {
	t.Helper()
	arguments := []string{"commit", "--quiet", "--allow-empty", "-m", subject}
	if body != "" {
		arguments = append(arguments, "-m", body)
	}
	result := gitAtWithDates(t, environment, repository, email, authorDate, commitDate, arguments...)
	assertSucceeded(t, result)
	hash := gitInRepository(t, environment, repository, "rev-parse", "HEAD")
	assertSucceeded(t, hash)
	return strings.TrimSpace(hash.stdout)
}

func gitAt(t *testing.T, environment acceptanceEnvironment, repository, email, date string, arguments ...string) processResult {
	t.Helper()
	return gitAtWithDates(t, environment, repository, email, date, date, arguments...)
}

func gitAtWithDates(t *testing.T, environment acceptanceEnvironment, repository, email, authorDate, commitDate string, arguments ...string) processResult {
	t.Helper()
	commandArguments := append([]string{"-C", repository}, arguments...)
	command := exec.Command("git", commandArguments...)
	command.Env = append(environment.environment,
		"GIT_AUTHOR_NAME=Daily Facts Fixture",
		"GIT_AUTHOR_EMAIL="+email,
		"GIT_AUTHOR_DATE="+authorDate,
		"GIT_COMMITTER_NAME=Daily Facts Fixture",
		"GIT_COMMITTER_EMAIL="+email,
		"GIT_COMMITTER_DATE="+commitDate,
	)
	return runCommand(t, command)
}

func gitInRepository(t *testing.T, environment acceptanceEnvironment, repository string, arguments ...string) processResult {
	t.Helper()
	commandArguments := append([]string{"-C", repository}, arguments...)
	command := exec.Command("git", commandArguments...)
	command.Env = environment.environment
	return runCommand(t, command)
}
