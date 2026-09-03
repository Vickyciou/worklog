package gitnotes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const Ref = "refs/notes/worklog"

var ErrConflict = errors.New("commit already has a different worklog event")

type Result int

const (
	Created Result = iota
	Unchanged
)

func ResolveCommit(repository string, revision string) (string, error) {
	info, err := os.Stat(repository)
	if err != nil {
		return "", fmt.Errorf("access repository %q: %w", repository, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("access repository %q: not a directory", repository)
	}

	if _, err := run(repository, nil, "rev-parse", "--git-dir"); err != nil {
		return "", fmt.Errorf("open repository %q: %w", repository, err)
	}

	object, err := run(repository, nil, "rev-parse", "--verify", "--end-of-options", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve commit %q: %w", revision, err)
	}

	return strings.TrimSpace(string(object)), nil
}

func Add(repository string, commit string, note []byte, equivalent func([]byte, []byte) bool) (Result, error) {
	existing, found, err := Read(repository, commit)
	if err != nil {
		return Created, err
	}
	if found {
		if equivalent(existing, note) {
			return Unchanged, nil
		}
		return Created, ErrConflict
	}

	_, writeErr := run(repository, note, "notes", "--ref="+Ref, "add", "-F", "-", commit)
	if writeErr == nil {
		return Created, nil
	}

	// Another writer can win after the initial read. Re-read so an identical
	// concurrent write remains idempotent and a different write stays intact.
	existing, found, readErr := Read(repository, commit)
	if readErr != nil {
		return Created, readErr
	}
	if found {
		if equivalent(existing, note) {
			return Unchanged, nil
		}
		return Created, ErrConflict
	}

	return Created, fmt.Errorf("write Git note: %w", writeErr)
}

// Read returns the worklog note for a commit without modifying the notes ref.
func Read(repository string, commit string) ([]byte, bool, error) {
	listed, err := run(repository, nil, "notes", "--ref="+Ref, "list", commit)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("inspect Git note: %w", err)
	}
	if len(bytes.TrimSpace(listed)) == 0 {
		return nil, false, nil
	}

	note, err := run(repository, nil, "notes", "--ref="+Ref, "show", commit)
	if err != nil {
		return nil, false, fmt.Errorf("read Git note: %w", err)
	}
	return note, true, nil
}

func run(repository string, stdin []byte, arguments ...string) ([]byte, error) {
	commandArguments := append([]string{"-C", repository}, arguments...)
	command := exec.Command("git", commandArguments...)
	if stdin != nil {
		command.Stdin = bytes.NewReader(stdin)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, message)
	}
	return stdout.Bytes(), nil
}
