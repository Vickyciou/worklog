package redaction

import (
	"regexp"
	"strings"

	"github.com/Vickyciou/worklog/internal/dailyfacts"
)

const Placeholder = "[REDACTED]"

var commonPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:api[-_]?key|access[-_]?token|auth[-_]?token|token|password|passwd|client[-_]?secret|secret)\s*[:=]\s*["']?[^\s"']+`),
	regexp.MustCompile(`(?i)\bauthorization\s*:\s*(?:bearer\s+)?[^\s"']+`),
	regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{8,}|gh[pousr]_[A-Za-z0-9_]{8,}|github_pat_[A-Za-z0-9_]{8,}|AKIA[0-9A-Z]{16})\b`),
	regexp.MustCompile(`(?i)\b[0-9a-f]{7,64}\b`),
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`),
	regexp.MustCompile(`(?:^|[\s("'=])/(?:[^\s"'<>]+)`),
	regexp.MustCompile(`(?i)(?:^|[\s("'=])[A-Z]:\\(?:[^\s"'<>]+)`),
}

var rawContentPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^diff --git `),
	regexp.MustCompile(`(?m)^@@ -\d`),
	regexp.MustCompile("```"),
}

func Apply(document dailyfacts.Document, deniedLiterals []string) dailyfacts.Document {
	result := document
	result.Profile = redact(result.Profile, deniedLiterals)
	result.Facts = make([]dailyfacts.Fact, len(document.Facts))
	for index, fact := range document.Facts {
		result.Facts[index] = redactFact(fact, deniedLiterals)
	}
	return result
}

func redactFact(fact dailyfacts.Fact, deniedLiterals []string) dailyfacts.Fact {
	result := fact
	result.Outcome = redact(fact.Outcome, deniedLiterals)
	result.Approach = redactPointer(fact.Approach, deniedLiterals)
	result.Decisions = make([]dailyfacts.Decision, len(fact.Decisions))
	for index, decision := range fact.Decisions {
		result.Decisions[index] = dailyfacts.Decision{
			Decision: redact(decision.Decision, deniedLiterals),
			Reason:   redactPointer(decision.Reason, deniedLiterals),
		}
	}
	result.NextSteps = redactStrings(fact.NextSteps, deniedLiterals)
	result.Technologies = redactStrings(fact.Technologies, deniedLiterals)
	return result
}

func redactPointer(value *string, deniedLiterals []string) *string {
	if value == nil {
		return nil
	}
	redacted := redact(*value, deniedLiterals)
	return &redacted
}

func redactStrings(values []string, deniedLiterals []string) []string {
	if values == nil {
		return nil
	}
	redacted := make([]string, len(values))
	for index, value := range values {
		redacted[index] = redact(value, deniedLiterals)
	}
	return redacted
}

func redact(value string, deniedLiterals []string) string {
	for _, pattern := range rawContentPatterns {
		if pattern.MatchString(value) {
			return Placeholder
		}
	}
	for _, literal := range deniedLiterals {
		literal = strings.TrimSpace(literal)
		if literal == "" {
			continue
		}
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(literal))
		value = pattern.ReplaceAllString(value, Placeholder)
	}
	for _, pattern := range commonPatterns {
		value = pattern.ReplaceAllString(value, Placeholder)
	}
	return value
}
