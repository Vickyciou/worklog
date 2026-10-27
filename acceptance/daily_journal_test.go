package acceptance_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDayPreviewsAndWritesDeterministicOfflineJournal(t *testing.T) {
	environment := newEnvironment(t)
	const profileEmail = "offline-journal@example.invalid"
	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)

	environment.environment = cleanEnvironment(environment.environment, map[string]string{
		"TZ":          "UTC",
		"LC_ALL":      "",
		"LC_MESSAGES": "",
		"LANG":        "zh_TW.UTF-8",
	})
	assertSucceeded(t, environment.git(t, "config", "user.email", profileEmail))

	setupInput := strings.Join([]string{
		"1",
		"",
		"",
		environment.journalDestination,
		"",
		"y",
		"",
	}, "\n")
	assertSucceeded(t, environment.runWithInput(t, setupInput, "setup"))

	commit := createCommitAt(t, environment, environment.repository, profileEmail,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "fallback must not appear", "")
	event := `{"version":1,"outcome":"完成離線日誌產生","approach":"沿用 DailyFacts 並由 core render","decisions":[{"decision":"由 profile 決定路徑","reason":"provider 不得控制 destination"}],"next_steps":["檢查今日預覽"],"technologies":[" go ","Go","GOLANG","Git","git","developer@example.invalid","/private/repository","private-project-identifier","Kubernetes","Docker","PostgreSQL","Redis","Terraform","GitHub Actions","TypeScript","Node.js","React","Swift","Rust","Python","Java"],"source":{"kind":"agent","name":"must-not-appear"}}`
	assertSucceeded(t, environment.runWithInput(t, event, "capture", environment.repository, commit))

	inspect := environment.run(t, "inspect", "work", date)
	assertSucceeded(t, inspect)
	digest := sha256.Sum256([]byte(inspect.stdout))
	digestText := hex.EncodeToString(digest[:])
	destination := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")
	expected := fmt.Sprintf(`---
date: %s
type: worklog-daily
tech: [Go, Git, Kubernetes, Docker, PostgreSQL, Redis, Terraform, GitHub Actions, TypeScript, Node.js]
tags: [工作日誌]
---

<!-- worklog:managed:start source-digest=sha256:%s -->
# 工作日誌 - %s

## 今日重點

- 完成離線日誌產生

## 完成事項

### 工作成果

- 完成離線日誌產生
  - 作法：沿用 DailyFacts 並由 core render

## 技術決策

- 由 profile 決定路徑（原因：provider 不得控制 destination）

## 後續行動

- 檢查今日預覽

<!-- worklog:managed:end -->
`, date, digestText, date)

	declined := environment.runWithInput(t, "no\n", "day")
	assertSucceeded(t, declined)
	if declined.stdout != fmt.Sprintf("work · %s · 1 commit · Built-in\nJournal preview: %s\n\n%s\nWrite this journal? [y/N]: \nNot written. Run worklog day again when you're ready.\n", date, destination, expected) {
		t.Fatalf("day preview mismatch\n--- got ---\n%s--- want ---\n%s", declined.stdout, expected)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unconfirmed day created destination: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(destination)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unconfirmed day created journal directories: %v", err)
	}

	confirmed := environment.runWithInput(t, "yes\n", "day")
	assertSucceeded(t, confirmed)
	assertContains(t, confirmed.stdout, fmt.Sprintf("Journal preview: %s\n\n%s", destination, expected))
	assertContains(t, confirmed.stdout, "Wrote journal "+destination+".")
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if string(contents) != expected {
		t.Fatalf("journal mismatch\n--- got ---\n%s--- want ---\n%s", contents, expected)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(destination)
		if err != nil {
			t.Fatalf("stat journal: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("journal permissions = %o, want 600", info.Mode().Perm())
		}
	}
	for _, forbidden := range []string{
		profileEmail, "/private/repository", "private-project-identifier", "must-not-appear", "React", "Swift", "Rust", "Python", "Java",
	} {
		if strings.Contains(string(contents), forbidden) {
			t.Fatalf("journal contains excluded value %q", forbidden)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatalf("read journal directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("atomic write left unexpected directory entries: %+v", entries)
	}
}

func TestDayStopsBeforeWritingWhenProfileMappingIsMissing(t *testing.T) {
	environment := newEnvironment(t)
	for _, profile := range []string{"personal", "work"} {
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile,
			"--timezone", "UTC",
			"--language", "zh-TW",
			"--journal-destination", filepath.Join(environment.root, profile+"-journal"),
		))
	}

	result := environment.runWithInput(t, "yes\n", "day")
	assertFailedWith(t, result, `profile "personal" has no registered repositories`, "worklog repo add personal <path>")
	if strings.Contains(result.stdout, "Journal preview:") {
		t.Fatal("incomplete profile produced a preview")
	}
}

func TestDayNeutralizesCapturedManagedMarkersInsideMarkdownText(t *testing.T) {
	environment := newEnvironment(t)
	const profileEmail = "marker-fixture@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", profileEmail))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))

	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)
	commit := createCommitAt(t, environment, environment.repository, profileEmail,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "marker fixture", "")
	malicious := `{"version":1,"outcome":"完成 <!-- worklog:managed:start source-digest=sha256:evil --> 工作","approach":"採用 <!-- worklog:managed:end --> 安全流程","decisions":[{"decision":"保留 <!-- worklog:managed:start injected --> 邊界","reason":"避免 <!-- worklog:managed:end --> 被誤判"}],"next_steps":["檢查 <!-- worklog:managed:start another --> 輸出"]}`
	assertSucceeded(t, environment.runWithInput(t, malicious, "capture", environment.repository, commit))

	result := environment.runWithInput(t, "yes\n", "day")
	assertSucceeded(t, result)
	destination := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read marker fixture journal: %v", err)
	}
	text := string(contents)
	if strings.Count(text, "<!-- worklog:managed:start") != 1 || strings.Count(text, "<!-- worklog:managed:end -->") != 1 {
		t.Fatalf("managed boundary count changed: start=%d end=%d\n%s", strings.Count(text, "<!-- worklog:managed:start"), strings.Count(text, "<!-- worklog:managed:end -->"), text)
	}
	for _, escaped := range []string{
		"&lt;!-- worklog:managed:start source-digest=sha256:evil -->",
		"&lt;!-- worklog:managed:end -->",
		"&lt;!-- worklog:managed:start injected -->",
		"&lt;!-- worklog:managed:start another -->",
	} {
		if !strings.Contains(text, escaped) {
			t.Fatalf("escaped marker %q missing from readable output:\n%s", escaped, text)
		}
	}
}

func TestDayRechecksDestinationAfterPreviewAndPreservesExistingFile(t *testing.T) {
	environment := newEnvironment(t)
	const email = "racing-day@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	now := time.Now().UTC().Truncate(time.Second)
	createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "racing day activity", "")
	date := time.Now().UTC().Format(time.DateOnly)
	destination := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")

	command := exec.Command(worklogBinary, "day")
	command.Dir = environment.repository
	command.Env = environment.environment
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("open day stdin: %v", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("open day stdout: %v", err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("start day: %v", err)
	}

	reader := bufio.NewReader(stdout)
	prefix := readUntil(t, reader, "Write this journal? [y/N]: ")
	assertContains(t, prefix, "Journal preview: "+destination)
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatalf("create racing destination directory: %v", err)
	}
	const sentinel = "manual journal must remain\n"
	if err := os.WriteFile(destination, []byte(sentinel), 0o600); err != nil {
		t.Fatalf("create racing destination: %v", err)
	}
	if _, err := io.WriteString(stdin, "yes\n"); err != nil {
		t.Fatalf("confirm day: %v", err)
	}
	if err := stdin.Close(); err != nil {
		t.Fatalf("close day stdin: %v", err)
	}
	remainder, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read remaining day stdout: %v", err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("racing day succeeded, want refusal")
	} else {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("wait for day: %v", err)
		}
	}
	if !strings.Contains(stderr.String(), "journal was created after preview") {
		t.Fatalf("stderr = %q, want existing-journal refusal", stderr.String())
	}
	contents, err := os.ReadFile(destination)
	if err != nil || string(contents) != sentinel {
		t.Fatalf("existing journal changed: contents=%q error=%v stdout=%q", contents, err, remainder)
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatalf("read journal directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("refused write left temporary files: %+v", entries)
	}
}

func TestDayGeneratesAndRefreshesHistoricalManagedBlock(t *testing.T) {
	environment := newEnvironment(t)
	const profileEmail = "historical-refresh@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", profileEmail))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))

	historical := time.Now().UTC().AddDate(0, 0, -7).Truncate(time.Second)
	date := historical.Format(time.DateOnly)
	firstCommit := createCommitAt(t, environment, environment.repository, profileEmail,
		historical.Format(time.RFC3339), historical.Format(time.RFC3339), "historical first", "")
	assertSucceeded(t, environment.runWithInput(t,
		`{"version":1,"outcome":"Historical first outcome","technologies":["Go"]}`,
		"capture", environment.repository, firstCommit,
	))

	generated := environment.runWithInput(t, "yes\n", "day", date)
	assertSucceeded(t, generated)
	destination := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")
	original, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read historical journal: %v", err)
	}
	start := bytes.Index(original, []byte("<!-- worklog:managed:start"))
	if start < 0 {
		t.Fatal("generated journal has no managed start")
	}
	withManual := make([]byte, 0, len(original)+64)
	withManual = append(withManual, original[:start]...)
	withManual = append(withManual, []byte("manual before block\r\n")...)
	withManual = append(withManual, original[start:]...)
	withManual = append(withManual, []byte("manual after block\x00\n")...)
	if err := os.WriteFile(destination, withManual, 0o600); err != nil {
		t.Fatalf("add manual journal content: %v", err)
	}
	wantUnmanaged := unmanagedJournalBytes(t, withManual)

	secondTime := historical.Add(time.Minute)
	secondCommit := createCommitAt(t, environment, environment.repository, profileEmail,
		secondTime.Format(time.RFC3339), secondTime.Format(time.RFC3339), "historical second", "")
	assertSucceeded(t, environment.runWithInput(t,
		`{"version":1,"outcome":"Historical refreshed outcome","technologies":["Git"]}`,
		"capture", environment.repository, secondCommit,
	))

	declined := environment.runWithInput(t, "no\n", "day", date)
	assertSucceeded(t, declined)
	assertContains(t, declined.stdout, "Not updated. Run worklog day again when you're ready.")
	assertContains(t, declined.stdout, "Journal changes: "+destination)
	assertContains(t, declined.stdout, "Update this journal? [y/N]: ")
	assertContains(t, declined.stdout, "+ - Historical refreshed outcome")
	if strings.Contains(declined.stdout, "source-digest") {
		t.Fatalf("update diff shows the source digest marker:\n%s", declined.stdout)
	}
	afterDecline, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(afterDecline, withManual) {
		t.Fatalf("declined refresh changed journal: error=%v", err)
	}

	refreshed := environment.runWithInput(t, "yes\n", "day", date)
	assertSucceeded(t, refreshed)
	assertContains(t, refreshed.stdout, "Updated journal "+destination+".")
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read refreshed journal: %v", err)
	}
	if !bytes.Equal(unmanagedJournalBytes(t, contents), wantUnmanaged) {
		t.Fatalf("refresh changed bytes outside managed block\n--- got ---\n%q\n--- want ---\n%q", unmanagedJournalBytes(t, contents), wantUnmanaged)
	}
	for _, expected := range []string{"Historical first outcome", "Historical refreshed outcome"} {
		if !strings.Contains(string(contents), expected) {
			t.Fatalf("refreshed managed block missing %q:\n%s", expected, contents)
		}
	}
}

func TestDayRefreshRefusesJournalWithoutValidManagedBoundaries(t *testing.T) {
	environment := newEnvironment(t)
	const email = "invalid-refresh@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	commitTime := time.Now().UTC().AddDate(0, 0, -2).Truncate(time.Second)
	date := commitTime.Format(time.DateOnly)
	createCommitAt(t, environment, environment.repository, email,
		commitTime.Format(time.RFC3339), commitTime.Format(time.RFC3339), "invalid refresh activity", "")
	destination := filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md")
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatalf("create historical journal directory: %v", err)
	}
	original := []byte("manual journal without managed boundaries\n")
	if err := os.WriteFile(destination, original, 0o600); err != nil {
		t.Fatalf("write manual journal: %v", err)
	}

	for _, arguments := range [][]string{{"day", date}, {"day", date, "--refresh"}} {
		result := environment.runWithInput(t, "yes\n", arguments...)
		assertFailedWith(t, result, "exactly one valid managed block")
	}
	contents, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(contents, original) {
		t.Fatalf("invalid journal changed: contents=%q error=%v", contents, err)
	}
}

func unmanagedJournalBytes(t *testing.T, contents []byte) []byte {
	t.Helper()
	start := bytes.Index(contents, []byte("<!-- worklog:managed:start"))
	end := bytes.Index(contents, []byte("<!-- worklog:managed:end -->"))
	if start < 0 || end < start {
		t.Fatalf("journal has invalid managed block:\n%s", contents)
	}
	end += len("<!-- worklog:managed:end -->")
	unmanaged := append([]byte(nil), contents[:start]...)
	unmanaged = append(unmanaged, contents[end:]...)
	return unmanaged
}

func readUntil(t *testing.T, reader *bufio.Reader, marker string) string {
	t.Helper()
	var output strings.Builder
	for !strings.Contains(output.String(), marker) {
		value, err := reader.ReadByte()
		if err != nil {
			t.Fatalf("read process output before %q: %v\n%s", marker, err, output.String())
		}
		output.WriteByte(value)
	}
	return output.String()
}

func TestDayRendersJournalLabelsInProfileLanguage(t *testing.T) {
	environment := newEnvironment(t)
	const profileEmail = "english-labels@example.invalid"
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "en-US",
		"--journal-destination", environment.journalDestination,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", profileEmail))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)
	commit := createCommitAt(t, environment, environment.repository, profileEmail,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "label fixture", "")
	event := `{"version":1,"outcome":"Ship retry","approach":"Bounded backoff","decisions":[{"decision":"Cap retries","reason":"Avoid storms"}]}`
	assertSucceeded(t, environment.runWithInput(t, event, "capture", environment.repository, commit))

	result := environment.runWithInput(t, "yes\n", "day")

	assertSucceeded(t, result)
	contents, err := os.ReadFile(filepath.Join(environment.journalDestination, date[:4], date[:7], date+".md"))
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	text := string(contents)
	for _, expected := range []string{
		"tags: [工作日誌]\n",
		"# Work Log - " + date + "\n",
		"## Highlights\n\n- Ship retry\n",
		"## Completed\n\n### Work\n\n- Ship retry\n  - Approach: Bounded backoff\n",
		"## Decisions\n\n- Cap retries (reason: Avoid storms)\n",
		"## Next Steps\n\n- None\n",
	} {
		assertContains(t, text, expected)
	}
	for _, chinese := range []string{"今日重點", "完成事項", "工作成果", "技術決策", "後續行動", "無"} {
		if strings.Contains(text, chinese) {
			t.Fatalf("en-US journal contains Chinese label %q:\n%s", chinese, text)
		}
	}
}

func TestDayUpdatesTodaysJournalWithoutRefreshFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures are covered by portable summarizer tests on Windows")
	}
	environment := newEnvironment(t)
	const email = "rerun-day@example.invalid"
	counter := filepath.Join(environment.root, "summarizer-calls")
	summarizerPath := filepath.Join(environment.root, "counting-summarizer")
	script := "#!/bin/sh\nprintf 'x' >> \"$WORKLOG_TEST_CALLS\"\nexit 1\n"
	if err := os.WriteFile(summarizerPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write counting summarizer: %v", err)
	}
	environment.environment = append(environment.environment, "WORKLOG_TEST_CALLS="+counter)
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", filepath.Join(environment.home, "journal"),
		"--summarizer-command", summarizerPath,
	))
	assertSucceeded(t, environment.run(t, "author", "add", "work", email))
	assertSucceeded(t, environment.run(t, "repo", "add", "work", environment.repository))
	now := time.Now().UTC().Truncate(time.Second)
	date := now.Format(time.DateOnly)
	shown := "~/journal/" + date[:4] + "/" + date[:7] + "/" + date + ".md"
	destination := filepath.Join(environment.home, "journal", date[:4], date[:7], date+".md")
	calls := func() int {
		contents, err := os.ReadFile(counter)
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}
		if err != nil {
			t.Fatalf("read summarizer counter: %v", err)
		}
		return len(contents)
	}
	createCommitAt(t, environment, environment.repository, email,
		now.Format(time.RFC3339), now.Format(time.RFC3339), "Morning change", "")

	written := environment.runWithInput(t, "y\n", "day")
	assertSucceeded(t, written)
	assertContains(t, written.stdout, "Journal preview: "+shown)
	assertContains(t, written.stdout, "Wrote journal "+shown+".")
	if err := appendFile(destination, "manual note\n"); err != nil {
		t.Fatalf("add manual note: %v", err)
	}

	current := environment.run(t, "day")
	assertSucceeded(t, current)
	assertContains(t, current.stdout, "work: journal for "+date+" is already up to date.")
	if strings.Contains(current.stdout, "[y/N]") {
		t.Fatalf("up-to-date journal asked for confirmation:\n%s", current.stdout)
	}
	if got := calls(); got != 1 {
		t.Fatalf("summarizer calls after up-to-date run = %d, want 1", got)
	}

	later := now.Add(time.Minute)
	createCommitAt(t, environment, environment.repository, email,
		later.Format(time.RFC3339), later.Format(time.RFC3339), "Afternoon change", "")
	updated := environment.runWithInput(t, "y\n", "day")
	assertSucceeded(t, updated)
	assertContains(t, updated.stdout, "Journal changes: "+shown)
	assertContains(t, updated.stdout, "  - Morning change\n+ - Afternoon change\n")
	if strings.Contains(updated.stdout, "- - Morning change") {
		t.Fatalf("update diff removed an unchanged line:\n%s", updated.stdout)
	}
	assertContains(t, updated.stdout, "Updated journal "+shown+".")
	contents, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read updated journal: %v", err)
	}
	for _, expected := range []string{"Morning change", "Afternoon change", "manual note\n"} {
		assertContains(t, string(contents), expected)
	}

	forced := environment.runWithInput(t, "n\n", "day", "--refresh")
	assertSucceeded(t, forced)
	assertContains(t, forced.stdout, "Journal changes: "+shown)
	assertContains(t, forced.stdout, "Not updated.")
	if got := calls(); got != 3 {
		t.Fatalf("summarizer calls after forced refresh = %d, want 3", got)
	}
}

func TestDayHandlesEachProfileJournalStateIndependently(t *testing.T) {
	environment := newEnvironment(t)
	personalRepository := filepath.Join(environment.root, "personal-repository")
	initializeGitRepository(t, personalRepository, environment.environment)
	commitTime := time.Now().UTC().AddDate(0, 0, -2).Truncate(time.Second)
	date := commitTime.Format(time.DateOnly)
	for _, profile := range []struct{ name, repository string }{
		{name: "work", repository: environment.repository},
		{name: "personal", repository: personalRepository},
	} {
		email := profile.name + "-state@example.invalid"
		assertSucceeded(t, environment.run(t,
			"profile", "add", profile.name,
			"--timezone", "UTC",
			"--language", "en",
			"--journal-destination", filepath.Join(environment.root, profile.name+"-journal"),
		))
		assertSucceeded(t, environment.run(t, "author", "add", profile.name, email))
		assertSucceeded(t, environment.run(t, "repo", "add", profile.name, profile.repository))
		createCommitAt(t, environment, profile.repository, email,
			commitTime.Format(time.RFC3339), commitTime.Format(time.RFC3339), profile.name+" change", "")
	}
	assertSucceeded(t, environment.runWithInput(t, "y\nn\n", "day", date))

	result := environment.runWithInput(t, "y\n", "day", date)

	assertSucceeded(t, result)
	if strings.Count(result.stdout, "is already up to date.") != 1 || strings.Count(result.stdout, "Journal preview: ") != 1 {
		t.Fatalf("want one up-to-date profile and one new preview:\n%s", result.stdout)
	}
	for _, profile := range []string{"work", "personal"} {
		if _, err := os.Stat(filepath.Join(environment.root, profile+"-journal", date[:4], date[:7], date+".md")); err != nil {
			t.Fatalf("%s journal missing: %v", profile, err)
		}
	}
}

func appendFile(path, text string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(text); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
