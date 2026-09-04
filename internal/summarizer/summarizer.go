package summarizer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/Vickyciou/worklog/internal/dailyfacts"
	"github.com/Vickyciou/worklog/internal/dailysummary"
)

const DefaultTimeout = 2 * time.Minute

const TimeoutEnv = "WORKLOG_SUMMARIZER_TIMEOUT"

const maxOutputBytes = 1024 * 1024

type Failure string

const (
	FailureUnavailable Failure = "command unavailable"
	FailureTimeout     Failure = "command timed out"
	FailureNonzero     Failure = "command exited unsuccessfully"
	FailureEmpty       Failure = "command returned empty output"
	FailureInvalid     Failure = "command returned invalid DailySummary JSON"
)

func ConfiguredTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv(TimeoutEnv))
	if value == "" {
		return DefaultTimeout
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return DefaultTimeout
	}
	return timeout
}

func Run(ctx context.Context, command string, facts dailyfacts.Document, workingDirectory string, timeout time.Duration) (dailysummary.Document, Failure) {
	arguments, err := splitCommand(command)
	if err != nil || len(arguments) == 0 {
		return dailysummary.Document{}, FailureUnavailable
	}
	payload, err := dailyfacts.Encode(facts)
	if err != nil {
		return dailysummary.Document{}, FailureInvalid
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	invocationContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	invocation := exec.CommandContext(invocationContext, arguments[0], arguments[1:]...)
	invocation.Dir = workingDirectory
	invocation.Env = os.Environ()
	invocation.Stdin = bytes.NewReader(payload)
	stdout := cappedBuffer{limit: maxOutputBytes}
	invocation.Stdout = &stdout
	invocation.Stderr = ioDiscard{}
	if err := invocation.Run(); err != nil {
		if errors.Is(invocationContext.Err(), context.DeadlineExceeded) {
			return dailysummary.Document{}, FailureTimeout
		}
		var pathError *exec.Error
		if errors.As(err, &pathError) {
			return dailysummary.Document{}, FailureUnavailable
		}
		return dailysummary.Document{}, FailureNonzero
	}
	if stdout.exceeded {
		return dailysummary.Document{}, FailureInvalid
	}
	if len(bytes.TrimSpace(stdout.Bytes())) == 0 {
		return dailysummary.Document{}, FailureEmpty
	}
	summary, err := dailysummary.Decode(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return dailysummary.Document{}, FailureInvalid
	}
	return summary, ""
}

type cappedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *cappedBuffer) Write(contents []byte) (int, error) {
	originalLength := len(contents)
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return originalLength, nil
	}
	if len(contents) > remaining {
		contents = contents[:remaining]
		buffer.exceeded = true
	}
	_, _ = buffer.Buffer.Write(contents)
	return originalLength, nil
}

type ioDiscard struct{}

func (ioDiscard) Write(contents []byte) (int, error) {
	return len(contents), nil
}

func splitCommand(command string) ([]string, error) {
	var arguments []string
	var current strings.Builder
	var quote rune
	escaped := false
	started := false
	flush := func() {
		if started {
			arguments = append(arguments, current.String())
			current.Reset()
			started = false
		}
	}
	for _, character := range command {
		if escaped {
			current.WriteRune(character)
			escaped = false
			started = true
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			started = true
			continue
		}
		switch {
		case character == '\'' || character == '"':
			quote = character
			started = true
		case unicode.IsSpace(character):
			flush()
		default:
			current.WriteRune(character)
			started = true
		}
	}
	if escaped {
		return nil, fmt.Errorf("summarizer command ends with an escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("summarizer command has an unterminated quote")
	}
	flush()
	return arguments, nil
}
