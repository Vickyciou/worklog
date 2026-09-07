package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestRefreshReplacesOnlyManagedBlock(t *testing.T) {
	existing := []byte("prefix\r\n" + managedFixture("old") + "\nmanual suffix\x00\n")
	rendered := []byte("different generated prefix\n" + managedFixture("new") + "\n")

	refreshed, difference, err := Refresh(existing, rendered)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	want := []byte("prefix\r\n" + managedFixture("new") + "\nmanual suffix\x00\n")
	if string(refreshed) != string(want) {
		t.Fatalf("refreshed journal = %q, want %q", refreshed, want)
	}
	for _, value := range []string{"--- existing managed block", "-old", "+new"} {
		if !strings.Contains(string(difference), value) {
			t.Fatalf("diff missing %q:\n%s", value, difference)
		}
	}
}

func TestValidateManagedRejectsAmbiguousOrMalformedBoundaries(t *testing.T) {
	tests := []string{
		"plain manual journal\n",
		"<!-- worklog:managed:start source-digest=sha256:short -->\nold\n<!-- worklog:managed:end -->",
		managedFixture("one") + "\n" + managedFixture("two"),
		"<!-- worklog:managed:start source-digest=sha256:" + testDigest + " -->\nold",
	}
	for _, contents := range tests {
		if err := ValidateManaged([]byte(contents)); !errors.Is(err, ErrInvalidManagedBlock) {
			t.Errorf("ValidateManaged(%q) = %v, want ErrInvalidManagedBlock", contents, err)
		}
	}
}

func TestReplaceAtomicFailureBeforeRenamePreservesJournalAndCleansTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "journal.md")
	existing := []byte("original journal\n")
	if err := os.WriteFile(path, existing, 0o640); err != nil {
		t.Fatalf("write existing journal: %v", err)
	}
	injected := errors.New("injected write failure")

	err := replaceAtomic(path, existing, []byte("replacement\n"), func() error { return injected })
	if !errors.Is(err, injected) {
		t.Fatalf("replaceAtomic error = %v, want injected failure", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != string(existing) {
		t.Fatalf("existing journal changed: contents=%q error=%v", contents, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("replacement failure left partial files: %+v", entries)
	}
}

func TestReplaceAtomicRejectsJournalChangedAfterPreview(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "journal.md")
	if err := os.WriteFile(path, []byte("changed\n"), 0o600); err != nil {
		t.Fatalf("write journal: %v", err)
	}
	if err := ReplaceAtomic(path, []byte("previewed\n"), []byte("replacement\n")); !errors.Is(err, ErrChanged) {
		t.Fatalf("ReplaceAtomic = %v, want ErrChanged", err)
	}
}

func managedFixture(body string) string {
	return ManagedStartPrefix + testDigest + " -->\n" + body + "\n" + ManagedEnd
}
