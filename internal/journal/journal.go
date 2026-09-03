package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/dailyfacts"
	"github.com/Vickyciou/worklog/internal/dailysummary"
)

const (
	ManagedStartPrefix = "<!-- worklog:managed:start source-digest=sha256:"
	ManagedEnd         = "<!-- worklog:managed:end -->"
)

var ErrExists = errors.New("journal already exists; refresh is required to replace it")

func Path(profile config.Profile, date string) (string, error) {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil || parsed.Format(time.DateOnly) != date {
		return "", fmt.Errorf("parse journal date %q: use YYYY-MM-DD", date)
	}
	if !filepath.IsAbs(profile.JournalDestination) {
		return "", errors.New("journal destination must be an absolute path")
	}
	return filepath.Join(
		filepath.Clean(profile.JournalDestination),
		parsed.Format("2006"),
		parsed.Format("01"),
		date+".md",
	), nil
}

func Render(profile config.Profile, facts dailyfacts.Document, summary dailysummary.Document) ([]byte, error) {
	encodedFacts, err := dailyfacts.Encode(facts)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encodedFacts)

	var output bytes.Buffer
	output.WriteString("---\n")
	output.WriteString("date: ")
	output.WriteString(facts.Date)
	output.WriteByte('\n')
	output.WriteString("profile: ")
	output.WriteString(yamlString(profile.Name))
	output.WriteByte('\n')
	output.WriteString("tags:\n  - \"worklog\"\n")
	output.WriteString("technologies:")
	if len(summary.Technologies) == 0 {
		output.WriteString(" []\n")
	} else {
		output.WriteByte('\n')
		for _, technology := range summary.Technologies {
			output.WriteString("  - ")
			output.WriteString(yamlString(technology))
			output.WriteByte('\n')
		}
	}
	output.WriteString("---\n\n")
	output.WriteString(ManagedStartPrefix)
	output.WriteString(hex.EncodeToString(digest[:]))
	output.WriteString(" -->\n")
	output.WriteString("# ")
	output.WriteString(facts.Date)
	output.WriteString(" 工作日誌\n\n")

	writeListSection(&output, "今日重點", summary.Focus)
	output.WriteString("## 完成事項\n\n")
	if len(summary.Completed) == 0 {
		output.WriteString("- 無\n\n")
	} else {
		for _, topic := range summary.Completed {
			output.WriteString("### ")
			output.WriteString(markdownText(topic.Topic))
			output.WriteString("\n\n")
			for _, item := range topic.Items {
				output.WriteString("- ")
				output.WriteString(markdownText(item.Outcome))
				output.WriteByte('\n')
				if item.Approach != nil {
					output.WriteString("  - 作法：")
					output.WriteString(markdownText(*item.Approach))
					output.WriteByte('\n')
				}
			}
			output.WriteByte('\n')
		}
	}

	output.WriteString("## 技術決策\n\n")
	if len(summary.Decisions) == 0 {
		output.WriteString("- 無\n\n")
	} else {
		for _, decision := range summary.Decisions {
			output.WriteString("- ")
			output.WriteString(markdownText(decision.Decision))
			if decision.Reason != nil {
				output.WriteString("（原因：")
				output.WriteString(markdownText(*decision.Reason))
				output.WriteString("）")
			}
			output.WriteByte('\n')
		}
		output.WriteByte('\n')
	}

	writeListSection(&output, "後續行動", summary.NextSteps)
	output.WriteString(ManagedEnd)
	output.WriteByte('\n')
	return output.Bytes(), nil
}

func WriteAtomic(path string, contents []byte) error {
	if err := CheckAvailable(path); err != nil {
		return err
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create journal directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".worklog-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary journal: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect temporary journal: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary journal: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary journal: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary journal: %w", err)
	}
	if _, err := os.Lstat(path); err == nil {
		return ErrExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("recheck journal destination: %w", err)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrExists
		}
		return fmt.Errorf("install journal atomically: %w", err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("remove temporary journal after install: %w", err)
	}

	if runtime.GOOS == "windows" {
		return nil
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open journal directory for sync: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync journal directory: %w", err)
	}
	return nil
}

func CheckAvailable(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return ErrExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect journal destination: %w", err)
	}
	return nil
}

func writeListSection(output *bytes.Buffer, heading string, values []string) {
	output.WriteString("## ")
	output.WriteString(heading)
	output.WriteString("\n\n")
	if len(values) == 0 {
		output.WriteString("- 無\n\n")
		return
	}
	for _, value := range values {
		output.WriteString("- ")
		output.WriteString(markdownText(value))
		output.WriteByte('\n')
	}
	output.WriteByte('\n')
}

func yamlString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func markdownText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	// Keep the marker text readable while ensuring captured facts cannot create
	// another managed boundary in the rendered document.
	value = strings.ReplaceAll(value, "<!-- worklog:managed:start", "&lt;!-- worklog:managed:start")
	value = strings.ReplaceAll(value, "<!-- worklog:managed:end -->", "&lt;!-- worklog:managed:end -->")
	return value
}
