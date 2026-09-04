package dailysummary

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func Decode(reader io.Reader) (Document, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode DailySummary: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Document{}, errors.New("decode DailySummary: expected exactly one JSON object")
		}
		return Document{}, fmt.Errorf("decode DailySummary: %w", err)
	}
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func Validate(document Document) error {
	if document.Version != Version {
		return fmt.Errorf("DailySummary version must be %d", Version)
	}
	if document.Focus == nil || document.Completed == nil || document.Decisions == nil || document.NextSteps == nil || document.Technologies == nil {
		return errors.New("DailySummary focus, completed, decisions, next_steps, and technologies arrays are required")
	}
	if len(document.Focus) > MaxFocusStatements {
		return fmt.Errorf("DailySummary focus must contain at most %d items", MaxFocusStatements)
	}
	if err := validateStrings("focus", document.Focus); err != nil {
		return err
	}
	for _, topic := range document.Completed {
		if strings.TrimSpace(topic.Topic) == "" {
			return errors.New("DailySummary completed topic must not be empty")
		}
		if topic.Items == nil || len(topic.Items) == 0 {
			return errors.New("DailySummary completed topic must contain items")
		}
		for _, item := range topic.Items {
			if strings.TrimSpace(item.Outcome) == "" {
				return errors.New("DailySummary completed outcome must not be empty")
			}
			if item.Approach != nil && strings.TrimSpace(*item.Approach) == "" {
				return errors.New("DailySummary completed approach must not be empty")
			}
		}
	}
	for _, decision := range document.Decisions {
		if strings.TrimSpace(decision.Decision) == "" {
			return errors.New("DailySummary decision must not be empty")
		}
		if decision.Reason != nil && strings.TrimSpace(*decision.Reason) == "" {
			return errors.New("DailySummary decision reason must not be empty")
		}
	}
	if err := validateStrings("next_steps", document.NextSteps); err != nil {
		return err
	}
	if err := validateStrings("technologies", document.Technologies); err != nil {
		return err
	}
	if normalized := NormalizeTechnologies(document.Technologies); !reflect.DeepEqual(normalized, document.Technologies) {
		return fmt.Errorf("DailySummary technologies must contain at most %d unique public names in canonical form", MaxTechnologies)
	}
	return nil
}

func validateStrings(field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("DailySummary %s items must not be empty", field)
		}
	}
	return nil
}
