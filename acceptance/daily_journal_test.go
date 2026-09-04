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
		environment.journalDestination,
		"",
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
	destination := filepath.Join(environment.journalDestination, date[:4], date[5:7], date+".md")
	expected := fmt.Sprintf(`---
date: %s
profile: "work"
tags:
  - "worklog"
technologies:
  - "Go"
  - "Git"
  - "Kubernetes"
  - "Docker"
  - "PostgreSQL"
  - "Redis"
  - "Terraform"
  - "GitHub Actions"
  - "TypeScript"
  - "Node.js"
---

<!-- worklog:managed:start source-digest=sha256:%s -->
# %s 工作日誌

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
	if declined.exitCode == 0 {
		t.Fatal("unconfirmed day exit code = 0, want non-zero")
	}
	assertContains(t, declined.stderr, "journal write not confirmed")
	if declined.stdout != fmt.Sprintf("Journal preview: %s\n\n%s\nWrite this journal? [y/N]: \n", destination, expected) {
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

func TestDayRejectsAmbiguousProfilesWithoutWriting(t *testing.T) {
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
	assertFailedWith(t, result, "day requires exactly one configured profile in M1; found 2")
	if strings.Contains(result.stdout, "Journal preview:") {
		t.Fatal("ambiguous day produced a preview")
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
	destination := filepath.Join(environment.journalDestination, date[:4], date[5:7], date+".md")
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
	assertSucceeded(t, environment.run(t,
		"profile", "add", "work",
		"--timezone", "UTC",
		"--language", "zh-TW",
		"--journal-destination", environment.journalDestination,
	))
	date := time.Now().UTC().Format(time.DateOnly)
	destination := filepath.Join(environment.journalDestination, date[:4], date[5:7], date+".md")

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
	if !strings.Contains(stderr.String(), "journal already exists") {
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
