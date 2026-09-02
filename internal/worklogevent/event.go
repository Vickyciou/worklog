package worklogevent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const Version = 1

type Event struct {
	Version      int        `json:"version"`
	Outcome      string     `json:"outcome"`
	Approach     *string    `json:"approach,omitempty"`
	Decisions    []Decision `json:"decisions,omitempty"`
	NextSteps    []string   `json:"next_steps,omitempty"`
	Technologies []string   `json:"technologies,omitempty"`
	Source       *Source    `json:"source,omitempty"`
}

type Decision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type Source struct {
	Kind string  `json:"kind"`
	Name *string `json:"name,omitempty"`
}

func Decode(input io.Reader) (Event, []byte, error) {
	document, err := io.ReadAll(input)
	if err != nil {
		return Event{}, nil, fmt.Errorf("read WorklogEvent JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()

	var event Event
	if err := decoder.Decode(&event); err != nil {
		return Event{}, nil, fmt.Errorf("decode WorklogEvent JSON: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Event{}, nil, fmt.Errorf("decode WorklogEvent JSON: expected exactly one JSON object")
		}
		return Event{}, nil, fmt.Errorf("decode WorklogEvent JSON: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(document, &fields); err != nil {
		return Event{}, nil, fmt.Errorf("decode WorklogEvent JSON: %w", err)
	}
	if _, found := fields["version"]; !found {
		return Event{}, nil, fmt.Errorf("validate WorklogEvent: version is required")
	}
	for _, field := range []string{"version", "outcome", "approach", "decisions", "next_steps", "technologies", "source"} {
		if value, found := fields[field]; found && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Event{}, nil, fmt.Errorf("validate WorklogEvent: %s must not be null", field)
		}
	}
	if rawSource, found := fields["source"]; found {
		var sourceFields map[string]json.RawMessage
		if err := json.Unmarshal(rawSource, &sourceFields); err == nil {
			if name, hasName := sourceFields["name"]; hasName && bytes.Equal(bytes.TrimSpace(name), []byte("null")) {
				return Event{}, nil, fmt.Errorf("validate WorklogEvent: source.name must not be null")
			}
		}
	}

	if err := event.Validate(); err != nil {
		return Event{}, nil, err
	}

	canonical, err := json.Marshal(event)
	if err != nil {
		return Event{}, nil, fmt.Errorf("encode WorklogEvent JSON: %w", err)
	}
	canonical = append(canonical, '\n')

	return event, canonical, nil
}

func (event Event) Validate() error {
	if event.Version != Version {
		return fmt.Errorf("validate WorklogEvent: version must be %d", Version)
	}
	if strings.TrimSpace(event.Outcome) == "" {
		return fmt.Errorf("validate WorklogEvent: outcome is required and must be non-empty")
	}
	if event.Approach != nil && strings.TrimSpace(*event.Approach) == "" {
		return fmt.Errorf("validate WorklogEvent: approach must be non-empty when provided")
	}
	for index, decision := range event.Decisions {
		if strings.TrimSpace(decision.Decision) == "" {
			return fmt.Errorf("validate WorklogEvent: decisions[%d].decision must be non-empty", index)
		}
		if strings.TrimSpace(decision.Reason) == "" {
			return fmt.Errorf("validate WorklogEvent: decisions[%d].reason must be non-empty", index)
		}
	}
	if err := validateStrings("next_steps", event.NextSteps); err != nil {
		return err
	}
	if err := validateStrings("technologies", event.Technologies); err != nil {
		return err
	}
	if event.Source != nil {
		if strings.TrimSpace(event.Source.Kind) == "" {
			return fmt.Errorf("validate WorklogEvent: source.kind must be non-empty")
		}
		if event.Source.Name != nil && strings.TrimSpace(*event.Source.Name) == "" {
			return fmt.Errorf("validate WorklogEvent: source.name must be non-empty when provided")
		}
	}

	return nil
}

func Equivalent(note []byte, canonical []byte) bool {
	_, existingCanonical, err := Decode(bytes.NewReader(note))
	return err == nil && bytes.Equal(existingCanonical, canonical)
}

func validateStrings(field string, values []string) error {
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("validate WorklogEvent: %s[%d] must be non-empty", field, index)
		}
	}
	return nil
}
