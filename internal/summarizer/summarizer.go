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

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/dailyfacts"
	"github.com/Vickyciou/worklog/internal/dailysummary"
)

const DefaultTimeout = 2 * time.Minute

const TimeoutEnv = "WORKLOG_SUMMARIZER_TIMEOUT"

const maxOutputBytes = 1024 * 1024

const presetPromptBase = "Summarize the post-redaction DailyFacts JSON from stdin. Return only one DailySummary JSON object matching the supplied schema. Do not return Markdown, choose paths, use tools, or add facts not supported by the input."

const dailySummarySchema = `{"type":"object","additionalProperties":false,"required":["version","focus","completed","decisions","next_steps","technologies"],"properties":{"version":{"const":1},"focus":{"type":"array","maxItems":3,"items":{"type":"string","minLength":1}},"completed":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["topic","items"],"properties":{"topic":{"type":"string","minLength":1},"items":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["outcome"],"properties":{"outcome":{"type":"string","minLength":1},"approach":{"type":"string","minLength":1}}}}}}},"decisions":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["decision"],"properties":{"decision":{"type":"string","minLength":1},"reason":{"type":"string","minLength":1}}}},"next_steps":{"type":"array","items":{"type":"string","minLength":1}},"technologies":{"type":"array","maxItems":10,"items":{"type":"string","minLength":1}}}}`

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
	return runArguments(ctx, arguments, facts, workingDirectory, timeout)
}

// Executable resolves only the executable portion of an advanced command. It is
// used by diagnostics and never invokes the command.
func Executable(command string) (string, error) {
	arguments, err := splitCommand(command)
	if err != nil || len(arguments) == 0 {
		return "", errors.New("summarizer command cannot be parsed")
	}
	return exec.LookPath(arguments[0])
}

func RunPreset(ctx context.Context, mode, language string, facts dailyfacts.Document, workingDirectory string, timeout time.Duration) (dailysummary.Document, Failure) {
	prompt := presetPrompt(language)
	switch mode {
	case config.SummaryCodex:
		executable, err := exec.LookPath("codex")
		if err != nil {
			return dailysummary.Document{}, FailureUnavailable
		}
		schema, err := os.CreateTemp(workingDirectory, ".daily-summary-schema-*.json")
		if err != nil {
			return dailysummary.Document{}, FailureUnavailable
		}
		schemaPath := schema.Name()
		defer os.Remove(schemaPath)
		if err := schema.Chmod(0o600); err != nil {
			schema.Close()
			return dailysummary.Document{}, FailureUnavailable
		}
		if _, err := schema.WriteString(dailySummarySchema); err != nil {
			schema.Close()
			return dailysummary.Document{}, FailureUnavailable
		}
		if err := schema.Close(); err != nil {
			return dailysummary.Document{}, FailureUnavailable
		}
		arguments := []string{
			executable, "exec", "--ephemeral", "--ignore-user-config", "--ignore-rules",
			"--sandbox", "read-only", "--skip-git-repo-check", "--color", "never",
			"--output-schema", schemaPath, prompt,
		}
		return runArguments(ctx, arguments, facts, workingDirectory, timeout)
	case config.SummaryClaudeCode:
		executable, err := exec.LookPath("claude")
		if err != nil {
			return dailysummary.Document{}, FailureUnavailable
		}
		arguments := []string{
			executable, "--print", "--safe-mode", "--tools", "", "--permission-prompts", "none",
			"--no-session-persistence", "--output-format", "text", "--json-schema", dailySummarySchema,
			"--system-prompt", prompt,
		}
		return runArguments(ctx, arguments, facts, workingDirectory, timeout)
	default:
		return dailysummary.Document{}, FailureUnavailable
	}
}

func presetPrompt(language string) string {
	return presetPromptBase + " Target language: " + strings.TrimSpace(language) + ". Write focus statements, topic names, completed outcomes and approaches, decisions and reasons, and next steps in the target language. Preserve public technology names and identifiers when translation would reduce accuracy."
}

func runArguments(ctx context.Context, arguments []string, facts dailyfacts.Document, workingDirectory string, timeout time.Duration) (dailysummary.Document, Failure) {
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
