package redaction

import (
	"strings"
	"testing"

	"github.com/Vickyciou/worklog/internal/dailyfacts"
)

func TestApplyRedactsDeniedAndCommonSensitiveValues(t *testing.T) {
	approach := "Use /private/source with token=top-secret-value"
	reason := "Contact developer@example.invalid about AKIA1234567890123456"
	document := dailyfacts.Document{
		Version: 1,
		Profile: "client-a",
		Facts: []dailyfacts.Fact{{
			Outcome:  "Ship CLIENT-A at deadbeef",
			Approach: &approach,
			Decisions: []dailyfacts.Decision{{
				Decision: "Keep secrets local",
				Reason:   &reason,
			}},
			NextSteps:    []string{"Review ```raw code```"},
			Technologies: []string{"Go"},
		}},
	}

	result := Apply(document, []string{"client-a"})
	encoded := strings.Join([]string{
		result.Profile,
		result.Facts[0].Outcome,
		*result.Facts[0].Approach,
		result.Facts[0].Decisions[0].Decision,
		*result.Facts[0].Decisions[0].Reason,
		result.Facts[0].NextSteps[0],
	}, "\n")
	for _, forbidden := range []string{"client-a", "CLIENT-A", "deadbeef", "/private/source", "top-secret-value", "developer@example.invalid", "AKIA1234567890123456", "raw code"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("redacted output contains %q:\n%s", forbidden, encoded)
		}
	}
	if !strings.Contains(encoded, Placeholder) {
		t.Fatalf("redacted output contains no placeholder:\n%s", encoded)
	}
	if document.Facts[0].Outcome != "Ship CLIENT-A at deadbeef" {
		t.Fatal("Apply mutated its input")
	}
}
