package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDayKeepsActiveProfileAdapterRequestsAndJournalsSeparate(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	adapter := filepath.Join(environment.root, "profile-adapter")
	requestDirectory := filepath.Join(environment.root, "requests")
	counter := filepath.Join(environment.root, "request-counter")
	if err := os.MkdirAll(requestDirectory, 0o700); err != nil {
		t.Fatalf("create request directory: %v", err)
	}
	writeProfileAdapter(t, adapter)
	environment.environment = append(environment.environment,
		"WORKLOG_MULTI_REQUESTS="+requestDirectory,
		"WORKLOG_MULTI_COUNTER="+counter,
	)

	dateTime := time.Now().UTC().AddDate(0, 0, -5).Truncate(time.Second)
	date := dateTime.Format(time.DateOnly)
	profiles := []struct {
		name       string
		repository string
		email      string
		denied     string
		visible    string
	}{
		{name: "work", repository: environment.repository, email: "work-day@example.invalid", denied: "work-private", visible: "work-visible"},
		{name: "personal", repository: personalRepository, email: "personal-day@example.invalid", denied: "personal-private", visible: "personal-visible"},
	}
	for index, profile := range profiles {
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", filepath.Join(environment.root, profile.name+"-journal"),
			"--deny-literal", profile.denied,
			"--summarizer-command", adapter,
		))
		assertSucceeded(t, environment.run(t, "author", "add", profile.name, profile.email))
		assertSucceeded(t, environment.run(t, "repo", "add", profile.name, profile.repository))
		commitTime := dateTime.Add(time.Duration(index) * time.Minute)
		commit := createCommitAt(t, environment, profile.repository, profile.email,
			commitTime.Format(time.RFC3339), commitTime.Format(time.RFC3339), profile.name+" metadata", "")
		event := `{"version":1,"outcome":"` + profile.name + `-only ` + profile.denied + ` ` + profile.visible + `"}`
		assertSucceeded(t, environment.runWithInput(t, event, "capture", profile.repository, commit))
	}

	result := environment.runWithInput(t, "yes\nyes\n", "day", date)
	assertSucceeded(t, result)
	for _, profile := range []string{"personal", "work"} {
		assertContains(t, result.stdout, "Journal preview ["+profile+"]:")
	}

	countBytes, err := os.ReadFile(counter)
	if err != nil {
		t.Fatalf("read adapter counter: %v", err)
	}
	if strings.TrimSpace(string(countBytes)) != "2" {
		t.Fatalf("adapter invocation count = %q, want 2", countBytes)
	}
	requests := make(map[string]string)
	for index := 1; index <= 2; index++ {
		payload, err := os.ReadFile(filepath.Join(requestDirectory, "request-"+strconv.Itoa(index)+".json"))
		if err != nil {
			t.Fatalf("read adapter request %d: %v", index, err)
		}
		var envelope struct {
			Profile string `json:"profile"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("decode adapter request %d: %v\n%s", index, err, payload)
		}
		requests[envelope.Profile] = string(payload)
	}
	if len(requests) != 2 {
		t.Fatalf("adapter requests were not profile-scoped: %+v", requests)
	}
	for _, profile := range profiles {
		payload := requests[profile.name]
		if !strings.Contains(payload, profile.visible) || strings.Contains(payload, profile.denied) {
			t.Fatalf("%s request did not apply its own policy:\n%s", profile.name, payload)
		}
		for _, other := range profiles {
			if other.name != profile.name && strings.Contains(payload, other.visible) {
				t.Fatalf("%s request contains %s activity:\n%s", profile.name, other.name, payload)
			}
		}
		journalPath := filepath.Join(environment.root, profile.name+"-journal", date[:4], date[:7], date+".md")
		journalBytes, err := os.ReadFile(journalPath)
		if err != nil {
			t.Fatalf("read %s journal: %v", profile.name, err)
		}
		journal := string(journalBytes)
		if !strings.Contains(journal, profile.name+" adapter output") {
			t.Fatalf("%s journal missing its adapter output:\n%s", profile.name, journal)
		}
		for _, other := range profiles {
			if other.name != profile.name && strings.Contains(journal, other.name+" adapter output") {
				t.Fatalf("%s journal contains %s output:\n%s", profile.name, other.name, journal)
			}
		}
	}
}

func TestDayPreflightsAmbiguousActiveJournalDestinationsBeforeWriting(t *testing.T) {
	environment := newEnvironment(t)
	secondRepository := filepath.Join(environment.root, "second-repository")
	initializeGitRepository(t, secondRepository, environment.environment)
	dateTime := time.Now().UTC().AddDate(0, 0, -4).Truncate(time.Second)
	date := dateTime.Format(time.DateOnly)
	for index, fixture := range []struct {
		name       string
		repository string
		email      string
	}{
		{name: "first", repository: environment.repository, email: "first@example.invalid"},
		{name: "second", repository: secondRepository, email: "second@example.invalid"},
	} {
		assertSucceeded(t, environment.run(t,
			"profile", "add", fixture.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", environment.journalDestination,
		))
		assertSucceeded(t, environment.run(t, "author", "add", fixture.name, fixture.email))
		assertSucceeded(t, environment.run(t, "repo", "add", fixture.name, fixture.repository))
		commitTime := dateTime.Add(time.Duration(index) * time.Minute)
		createCommitAt(t, environment, fixture.repository, fixture.email,
			commitTime.Format(time.RFC3339), commitTime.Format(time.RFC3339), fixture.name+" activity", "")
	}

	result := environment.runWithInput(t, "yes\nyes\n", "day", date)
	assertFailedWith(t, result, "journal destination", "ambiguous", "worklog profile set")
	journalPath := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")
	if _, err := os.Stat(journalPath); !os.IsNotExist(err) {
		t.Fatalf("ambiguous destination wrote journal state: %v", err)
	}
}

func TestDaySkipsProfilesWithoutActivity(t *testing.T) {
	environment := newEnvironment(t)
	assertSucceeded(t, environment.run(t,
		"profile", "add", "quiet",
		"--timezone", "UTC",
		"--language", "en",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "quiet", "quiet@example.invalid"))
	assertSucceeded(t, environment.run(t, "repo", "add", "quiet", environment.repository))

	result := environment.runWithInput(t, "yes\n", "day", "2001-01-01")
	assertSucceeded(t, result)
	if result.stdout != "No profile activity found; no journals to write.\n" {
		t.Fatalf("quiet day stdout = %q", result.stdout)
	}
	journalPath := filepath.Join(environment.journalDestination, "2001", "2001-01", "2001-01-01.md")
	if _, err := os.Stat(journalPath); !os.IsNotExist(err) {
		t.Fatalf("quiet day created journal destination: %v", err)
	}
}

func writeProfileAdapter(t *testing.T, path string) {
	t.Helper()
	contents := `#!/bin/sh
set -eu
payload=$(/bin/cat)
count=0
if [ -f "$WORKLOG_MULTI_COUNTER" ]; then
  count=$(/bin/cat "$WORKLOG_MULTI_COUNTER")
fi
count=$((count + 1))
printf '%s\n' "$count" > "$WORKLOG_MULTI_COUNTER"
printf '%s\n' "$payload" > "$WORKLOG_MULTI_REQUESTS/request-$count.json"
case "$payload" in
  *'"profile": "personal"'*) outcome='personal adapter output' ;;
  *'"profile": "work"'*) outcome='work adapter output' ;;
  *) exit 9 ;;
esac
printf '{"version":1,"focus":["%s"],"completed":[{"topic":"Adapter","items":[{"outcome":"%s"}]}],"decisions":[],"next_steps":[],"technologies":[]}\n' "$outcome" "$outcome"
`
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatalf("write profile adapter: %v", err)
	}
}

func TestDayContinuesToNextProfileAfterDecline(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	dateTime := time.Now().UTC().AddDate(0, 0, -3).Truncate(time.Second)
	date := dateTime.Format(time.DateOnly)
	destinations := make(map[string]string)
	for _, profile := range []struct{ name, repository string }{
		{name: "work", repository: environment.repository},
		{name: "personal", repository: personalRepository},
	} {
		journalRoot := filepath.Join(environment.root, profile.name+"-journal")
		destinations[profile.name] = filepath.Join(journalRoot, date[:4], date[:7], date+".md")
		email := profile.name + "-decline@example.invalid"
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", journalRoot,
		))
		assertSucceeded(t, environment.run(t, "author", "add", profile.name, email))
		assertSucceeded(t, environment.run(t, "repo", "add", profile.name, profile.repository))
		createCommitAt(t, environment, profile.repository, email,
			dateTime.Format(time.RFC3339), dateTime.Format(time.RFC3339), profile.name+" change", "")
	}

	result := environment.runWithInput(t, "no\nyes\n", "day", date)

	assertSucceeded(t, result)
	declinedProfile, writtenProfile := "work", "personal"
	if strings.Index(result.stdout, "Journal preview [personal]:") < strings.Index(result.stdout, "Journal preview [work]:") {
		declinedProfile, writtenProfile = "personal", "work"
	}
	assertContains(t, result.stdout, "Not written. Run worklog day again when you're ready.")
	if _, err := os.Stat(destinations[declinedProfile]); !os.IsNotExist(err) {
		t.Fatalf("declined %s journal exists: %v", declinedProfile, err)
	}
	if _, err := os.Stat(destinations[writtenProfile]); err != nil {
		t.Fatalf("confirmed %s journal after a declined profile: %v", writtenProfile, err)
	}
}
