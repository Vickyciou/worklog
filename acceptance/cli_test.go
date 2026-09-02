package acceptance_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var worklogBinary string

func TestMain(m *testing.M) {
	buildRoot, err := os.MkdirTemp("", "worklog-acceptance-build-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create isolated build root: %v\n", err)
		os.Exit(1)
	}
	worklogBinary = filepath.Join(buildRoot, "worklog")
	if runtime.GOOS == "windows" {
		worklogBinary += ".exe"
	}

	command := exec.Command("go", "build", "-o", worklogBinary, "./cmd/worklog")
	command.Dir = repositoryRoot()
	command.Env = isolatedGoEnvironment(buildRoot)
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "build worklog acceptance binary: %v\n%s", buildErr, output)
		if removeErr := os.RemoveAll(buildRoot); removeErr != nil {
			fmt.Fprintf(os.Stderr, "remove isolated build root: %v\n", removeErr)
		}
		os.Exit(1)
	}

	exitCode := m.Run()
	if removeErr := os.RemoveAll(buildRoot); removeErr != nil {
		fmt.Fprintf(os.Stderr, "remove isolated build root: %v\n", removeErr)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func TestRootHelp(t *testing.T) {
	const expected = `worklog records and renders portable work journals.

Usage:
  worklog <command>

Commands:
  help       Show help for worklog
  version    Print version information

Options:
  -h, --help       Show help for worklog
  -v, --version    Print version information
`
	for _, arguments := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		t.Run(strings.Join(arguments, "_"), func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, arguments...)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
			if result.stdout != expected {
				t.Fatalf("stdout mismatch\n--- got ---\n%s--- want ---\n%s", result.stdout, expected)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	for _, argument := range []string{"version", "--version", "-v"} {
		t.Run(argument, func(t *testing.T) {
			environment := newEnvironment(t)
			result := environment.run(t, argument)

			if result.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", result.exitCode, result.stderr)
			}
			if result.stdout != "worklog dev\n" {
				t.Fatalf("stdout = %q, want %q", result.stdout, "worklog dev\n")
			}
			if result.stderr != "" {
				t.Fatalf("stderr = %q, want empty", result.stderr)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	environment := newEnvironment(t)

	result := environment.run(t, "unknown")

	if result.exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
	assertContains(t, result.stderr, `unknown command "unknown"`)
	assertContains(t, result.stderr, "Run 'worklog --help' for usage.")
}

func TestIllegalArguments(t *testing.T) {
	environment := newEnvironment(t)

	result := environment.run(t, "version", "extra")

	if result.exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
	assertContains(t, result.stderr, `"version" does not accept arguments`)
}

type acceptanceEnvironment struct {
	root               string
	home               string
	configRoot         string
	stateRoot          string
	repository         string
	journalDestination string
	temporaryDirectory string
	environment        []string
}

func newEnvironment(t *testing.T) acceptanceEnvironment {
	t.Helper()

	root := t.TempDir()
	environment := acceptanceEnvironment{
		root:               root,
		home:               filepath.Join(root, "home"),
		configRoot:         filepath.Join(root, "config"),
		stateRoot:          filepath.Join(root, "state"),
		repository:         filepath.Join(root, "repository"),
		journalDestination: filepath.Join(root, "journal"),
		temporaryDirectory: filepath.Join(root, "tmp"),
	}

	for _, directory := range []string{
		environment.home,
		environment.configRoot,
		environment.stateRoot,
		environment.repository,
		environment.journalDestination,
		environment.temporaryDirectory,
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create isolated directory %q: %v", directory, err)
		}
	}

	environment.environment = cleanEnvironment(os.Environ(), map[string]string{
		"HOME":                environment.home,
		"USERPROFILE":         environment.home,
		"XDG_CONFIG_HOME":     environment.configRoot,
		"XDG_STATE_HOME":      environment.stateRoot,
		"XDG_CACHE_HOME":      filepath.Join(environment.stateRoot, "cache"),
		"XDG_DATA_HOME":       filepath.Join(environment.stateRoot, "data"),
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_GLOBAL":   filepath.Join(root, "gitconfig"),
		"GIT_TERMINAL_PROMPT": "0",
		"TMPDIR":              environment.temporaryDirectory,
	}, "GIT_", "WORKLOG_", "XDG_")

	git := exec.Command("git", "init", "--quiet", environment.repository)
	git.Env = environment.environment
	if output, err := git.CombinedOutput(); err != nil {
		t.Fatalf("initialize isolated Git repository: %v\n%s", err, output)
	}

	return environment
}

func (environment acceptanceEnvironment) run(t *testing.T, arguments ...string) processResult {
	t.Helper()

	command := exec.Command(worklogBinary, arguments...)
	command.Dir = environment.repository
	command.Env = environment.environment

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("execute worklog: %v", err)
		}
		exitCode = exitError.ExitCode()
	}

	return processResult{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}

type processResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func repositoryRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("locate acceptance test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
}

func isolatedGoEnvironment(root string) []string {
	return cleanEnvironment(os.Environ(), map[string]string{
		"HOME":           filepath.Join(root, "home"),
		"USERPROFILE":    filepath.Join(root, "home"),
		"GOCACHE":        filepath.Join(root, "go-cache"),
		"GOPATH":         filepath.Join(root, "go-path"),
		"GOMODCACHE":     filepath.Join(root, "go-mod-cache"),
		"XDG_CACHE_HOME": filepath.Join(root, "cache"),
		"GOENV":          "off",
		"GOTOOLCHAIN":    "local",
	}, "GO", "CGO_")
}

func cleanEnvironment(current []string, replacements map[string]string, removedPrefixes ...string) []string {
	result := make([]string, 0, len(current)+len(replacements))
	for _, variable := range current {
		name, _, found := strings.Cut(variable, "=")
		if _, replace := replacements[name]; found && replace {
			continue
		}
		remove := false
		for _, prefix := range removedPrefixes {
			if strings.HasPrefix(name, prefix) {
				remove = true
				break
			}
		}
		if remove {
			continue
		}
		result = append(result, variable)
	}
	for name, value := range replacements {
		result = append(result, name+"="+value)
	}
	return result
}

func assertContains(t *testing.T, value string, expectedSubstring string) {
	t.Helper()
	if !strings.Contains(value, expectedSubstring) {
		t.Fatalf("%q does not contain %q", value, expectedSubstring)
	}
}
