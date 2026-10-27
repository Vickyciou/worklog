package dailysummary

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Vickyciou/worklog/internal/dailyfacts"
)

const (
	Version            = 1
	MaxFocusStatements = 3
	MaxTechnologies    = 10
)

var (
	hexIdentifier    = regexp.MustCompile(`(?i)^[0-9a-f]{7,64}$`)
	opaqueIdentifier = regexp.MustCompile(`^[A-Za-z0-9_-]{24,}$`)
	publicName       = regexp.MustCompile(`^[\pL\pN][\pL\pN .+#()_-]*$|^\.[\pL\pN][\pL\pN .+#()_-]*$`)
	sensitiveWords   = regexp.MustCompile(`(?i)(?:^|[^\pL\pN])(secret|token|password|passwd|credential|authorization|bearer|private|confidential|internal)(?:$|[^\pL\pN])`)
)

var canonicalTechnologies = map[string]string{
	"aws":            "AWS",
	"c#":             "C#",
	"c++":            "C++",
	"ci/cd":          "CI/CD",
	"docker":         "Docker",
	"git":            "Git",
	"github actions": "GitHub Actions",
	"go":             "Go",
	"golang":         "Go",
	"javascript":     "JavaScript",
	"kubernetes":     "Kubernetes",
	"node.js":        "Node.js",
	"postgres":       "PostgreSQL",
	"postgresql":     "PostgreSQL",
	"typescript":     "TypeScript",
}

type Document struct {
	Version      int              `json:"version"`
	Focus        []string         `json:"focus"`
	Completed    []CompletedTopic `json:"completed"`
	Decisions    []Decision       `json:"decisions"`
	NextSteps    []string         `json:"next_steps"`
	Technologies []string         `json:"technologies"`
}

type CompletedTopic struct {
	Topic string          `json:"topic"`
	Items []CompletedItem `json:"items"`
}

type CompletedItem struct {
	Outcome  string  `json:"outcome"`
	Approach *string `json:"approach,omitempty"`
}

type Decision struct {
	Decision string  `json:"decision"`
	Reason   *string `json:"reason,omitempty"`
}

func Fallback(facts dailyfacts.Document, completedTopic string) Document {
	summary := Document{
		Version:      Version,
		Focus:        []string{},
		Completed:    []CompletedTopic{},
		Decisions:    []Decision{},
		NextSteps:    []string{},
		Technologies: []string{},
	}

	completed := CompletedTopic{Topic: completedTopic, Items: []CompletedItem{}}
	seenFocus := make(map[string]struct{})
	seenDecisions := make(map[string]struct{})
	seenNextSteps := make(map[string]struct{})
	var technologies []string

	for _, fact := range facts.Facts {
		outcome := cleanText(fact.Outcome)
		if len(summary.Focus) < MaxFocusStatements {
			appendUnique(&summary.Focus, seenFocus, outcome)
		}

		item := CompletedItem{Outcome: outcome}
		if fact.Approach != nil {
			approach := cleanText(*fact.Approach)
			if approach != "" {
				item.Approach = &approach
			}
		}
		completed.Items = append(completed.Items, item)

		for _, candidate := range fact.Decisions {
			decision := cleanText(candidate.Decision)
			var reason *string
			if candidate.Reason != nil {
				cleaned := cleanText(*candidate.Reason)
				if cleaned != "" {
					reason = &cleaned
				}
			}
			key := strings.ToLower(decision) + "\x00"
			if reason != nil {
				key += strings.ToLower(*reason)
			}
			if decision != "" {
				if _, found := seenDecisions[key]; !found {
					seenDecisions[key] = struct{}{}
					summary.Decisions = append(summary.Decisions, Decision{Decision: decision, Reason: reason})
				}
			}
		}

		for _, candidate := range fact.NextSteps {
			appendUnique(&summary.NextSteps, seenNextSteps, cleanText(candidate))
		}
		technologies = append(technologies, fact.Technologies...)
	}

	if len(completed.Items) > 0 {
		summary.Completed = append(summary.Completed, completed)
	}
	summary.Technologies = NormalizeTechnologies(technologies)
	return summary
}

func NormalizeTechnologies(candidates []string) []string {
	result := make([]string, 0, min(len(candidates), MaxTechnologies))
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		normalized := normalizeTechnology(candidate)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, normalized)
		if len(result) == MaxTechnologies {
			break
		}
	}
	return result
}

func normalizeTechnology(value string) string {
	value = cleanText(value)
	if canonical, found := canonicalTechnologies[strings.ToLower(value)]; found {
		return canonical
	}
	if value == "" || utf8.RuneCountInString(value) > 40 {
		return ""
	}
	if strings.ContainsAny(value, `/\\@:=`) || strings.Contains(value, "..") {
		return ""
	}
	if sensitiveWords.MatchString(value) || hexIdentifier.MatchString(value) || opaqueIdentifier.MatchString(value) {
		return ""
	}
	if !publicName.MatchString(value) {
		return ""
	}
	return value
}

func appendUnique(values *[]string, seen map[string]struct{}, value string) {
	if value == "" {
		return
	}
	key := strings.ToLower(value)
	if _, found := seen[key]; found {
		return
	}
	seen[key] = struct{}{}
	*values = append(*values, value)
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
