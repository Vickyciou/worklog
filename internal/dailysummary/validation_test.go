package dailysummary

import (
	"strings"
	"testing"
)

func TestDecodeValidatesStrictDailySummarySchema(t *testing.T) {
	valid := `{"version":1,"focus":["Ship adapter"],"completed":[{"topic":"Summarizer","items":[{"outcome":"Validated output"}]}],"decisions":[],"next_steps":[],"technologies":["Go"]}`
	if _, err := Decode(strings.NewReader(valid)); err != nil {
		t.Fatalf("Decode(valid) error = %v", err)
	}
	invalid := []string{
		`{"version":1,"focus":["Ship adapter"],"completed":[],"decisions":[],"next_steps":[]}`,
		`{"version":2,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":[]}`,
		`{"version":1,"focus":[],"completed":[{"topic":"","items":[]}],"decisions":[],"next_steps":[],"technologies":[]}`,
		`{"version":1,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":["developer@example.invalid"]}`,
		`{"version":1,"focus":[],"completed":[],"decisions":[],"next_steps":[],"technologies":[],"markdown":"# unsafe"}`,
	}
	for _, input := range invalid {
		if _, err := Decode(strings.NewReader(input)); err == nil {
			t.Errorf("Decode(%s) succeeded, want failure", input)
		}
	}
}
