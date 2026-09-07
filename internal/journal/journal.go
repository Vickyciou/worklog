package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

var (
	ErrInvalidManagedBlock = errors.New("journal does not contain exactly one valid managed block")
	ErrChanged             = errors.New("journal changed after preview; refresh again")
)

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
		parsed.Format("2006-01"),
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
	output.WriteString("type: worklog-daily\n")
	output.WriteString("tech: [")
	if len(summary.Technologies) == 0 {
		output.WriteString("]\n")
	} else {
		output.WriteString(strings.Join(summary.Technologies, ", "))
		output.WriteString("]\n")
	}
	output.WriteString("tags: [工作日誌]\n")
	output.WriteString("---\n\n")
	output.WriteString(ManagedStartPrefix)
	output.WriteString(hex.EncodeToString(digest[:]))
	output.WriteString(" -->\n")
	output.WriteString("# 工作日誌 - ")
	output.WriteString(facts.Date)
	output.WriteString("\n\n")

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

// Refresh replaces only the managed block from rendered and returns a displayable
// diff of that block. Bytes before and after the existing block are retained
// exactly as they were.
func Refresh(existing, rendered []byte) ([]byte, []byte, error) {
	existingStart, existingEnd, err := managedBlockBounds(existing)
	if err != nil {
		return nil, nil, err
	}
	renderedStart, renderedEnd, err := managedBlockBounds(rendered)
	if err != nil {
		return nil, nil, fmt.Errorf("rendered journal: %w", err)
	}

	existingBlock := existing[existingStart:existingEnd]
	renderedBlock := rendered[renderedStart:renderedEnd]
	replacement := make([]byte, 0, len(existing)-len(existingBlock)+len(renderedBlock))
	replacement = append(replacement, existing[:existingStart]...)
	replacement = append(replacement, renderedBlock...)
	replacement = append(replacement, existing[existingEnd:]...)
	return replacement, managedDiff(existingBlock, renderedBlock), nil
}

// ValidateManaged verifies that contents can be refreshed without guessing at
// boundaries.
func ValidateManaged(contents []byte) error {
	_, _, err := managedBlockBounds(contents)
	return err
}

// ReplaceAtomic replaces an existing regular journal only if it still matches the
// bytes that were previewed.
func ReplaceAtomic(path string, expected, contents []byte) error {
	return replaceAtomic(path, expected, contents, nil)
}

func replaceAtomic(path string, expected, contents []byte, beforeReplace func() error) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect journal for replacement: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("journal replacement requires an existing regular file")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read journal for replacement: %w", err)
	}
	if !bytes.Equal(current, expected) {
		return ErrChanged
	}

	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".worklog-*.tmp")
	if err != nil {
		return fmt.Errorf("create replacement journal: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		temporary.Close()
		return fmt.Errorf("protect replacement journal: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write replacement journal: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync replacement journal: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close replacement journal: %w", err)
	}
	current, err = os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("recheck journal for replacement: %w", err)
	}
	if !bytes.Equal(current, expected) {
		return ErrChanged
	}
	if beforeReplace != nil {
		if err := beforeReplace(); err != nil {
			return fmt.Errorf("before journal replacement: %w", err)
		}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace journal atomically: %w", err)
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

func managedBlockBounds(contents []byte) (int, int, error) {
	startMarker := []byte("<!-- worklog:managed:start")
	start := bytes.Index(contents, startMarker)
	if start < 0 || bytes.Index(contents[start+len(startMarker):], startMarker) >= 0 {
		return 0, 0, ErrInvalidManagedBlock
	}
	lineEnd := bytes.IndexByte(contents[start:], '\n')
	if lineEnd < 0 {
		return 0, 0, ErrInvalidManagedBlock
	}
	lineEnd += start
	startLine := strings.TrimSuffix(string(contents[start:lineEnd]), "\r")
	digest := strings.TrimSuffix(strings.TrimPrefix(startLine, ManagedStartPrefix), " -->")
	if !strings.HasPrefix(startLine, ManagedStartPrefix) || !strings.HasSuffix(startLine, " -->") || len(digest) != sha256.Size*2 {
		return 0, 0, ErrInvalidManagedBlock
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || hex.EncodeToString(decoded) != digest {
		return 0, 0, ErrInvalidManagedBlock
	}

	endMarker := []byte(ManagedEnd)
	endRelative := bytes.Index(contents[lineEnd+1:], endMarker)
	if endRelative < 0 {
		return 0, 0, ErrInvalidManagedBlock
	}
	end := lineEnd + 1 + endRelative
	if bytes.Index(contents[end+len(endMarker):], endMarker) >= 0 {
		return 0, 0, ErrInvalidManagedBlock
	}
	return start, end + len(endMarker), nil
}

func managedDiff(existing, refreshed []byte) []byte {
	var output bytes.Buffer
	output.WriteString("--- existing managed block\n")
	output.WriteString("+++ refreshed managed block\n")
	output.WriteString("@@\n")
	if bytes.Equal(existing, refreshed) {
		output.WriteString(" (no managed changes)\n")
		return output.Bytes()
	}
	writeDiffLines(&output, '-', existing)
	writeDiffLines(&output, '+', refreshed)
	return output.Bytes()
}

func writeDiffLines(output *bytes.Buffer, prefix byte, contents []byte) {
	for len(contents) > 0 {
		lineEnd := bytes.IndexByte(contents, '\n')
		output.WriteByte(prefix)
		if lineEnd < 0 {
			output.Write(contents)
			output.WriteByte('\n')
			return
		}
		output.Write(contents[:lineEnd+1])
		contents = contents[lineEnd+1:]
	}
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

func markdownText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	// Keep the marker text readable while ensuring captured facts cannot create
	// another managed boundary in the rendered document.
	value = strings.ReplaceAll(value, "<!-- worklog:managed:start", "&lt;!-- worklog:managed:start")
	value = strings.ReplaceAll(value, "<!-- worklog:managed:end -->", "&lt;!-- worklog:managed:end -->")
	return value
}
