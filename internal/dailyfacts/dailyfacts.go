package dailyfacts

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/gitnotes"
	"github.com/Vickyciou/worklog/internal/worklogevent"
)

const Version = 1

type Document struct {
	Version  int    `json:"version"`
	Profile  string `json:"profile"`
	Date     string `json:"date"`
	Timezone string `json:"timezone"`
	Facts    []Fact `json:"facts"`
}

type Fact struct {
	CommittedAt  string     `json:"committed_at"`
	Outcome      string     `json:"outcome"`
	Approach     *string    `json:"approach,omitempty"`
	Decisions    []Decision `json:"decisions,omitempty"`
	NextSteps    []string   `json:"next_steps,omitempty"`
	Technologies []string   `json:"technologies,omitempty"`
}

type Decision struct {
	Decision string  `json:"decision"`
	Reason   *string `json:"reason,omitempty"`
}

type commit struct {
	hash      string
	author    string
	timestamp time.Time
	subject   string
}

func CurrentDate(profile config.Profile, now time.Time) (string, error) {
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return "", fmt.Errorf("load profile timezone: %w", err)
	}
	return now.In(location).Format(time.DateOnly), nil
}

func Collect(profile config.Profile, date string) (Document, error) {
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return Document{}, fmt.Errorf("load profile timezone: %w", err)
	}
	day, err := time.ParseInLocation(time.DateOnly, date, location)
	if err != nil {
		return Document{}, fmt.Errorf("parse date %q: use YYYY-MM-DD", date)
	}
	end := day.AddDate(0, 0, 1)
	authors := make(map[string]struct{}, len(profile.Authors))
	for _, author := range profile.Authors {
		authors[strings.ToLower(author)] = struct{}{}
	}

	document := Document{
		Version:  Version,
		Profile:  profile.Name,
		Date:     date,
		Timezone: profile.Timezone,
		Facts:    []Fact{},
	}
	for _, repository := range profile.Repositories {
		commits, err := listCommits(repository)
		if err != nil {
			return Document{}, fmt.Errorf("collect repository: %w", err)
		}
		for _, candidate := range commits {
			if _, included := authors[strings.ToLower(candidate.author)]; !included {
				continue
			}
			localTime := candidate.timestamp.In(location)
			if localTime.Before(day) || !localTime.Before(end) {
				continue
			}
			fact, err := factForCommit(repository, candidate)
			if err != nil {
				return Document{}, err
			}
			fact.CommittedAt = localTime.Format(time.RFC3339)
			document.Facts = append(document.Facts, fact)
		}
	}

	sort.SliceStable(document.Facts, func(left, right int) bool {
		if document.Facts[left].CommittedAt != document.Facts[right].CommittedAt {
			return document.Facts[left].CommittedAt < document.Facts[right].CommittedAt
		}
		return document.Facts[left].Outcome < document.Facts[right].Outcome
	})
	return document, nil
}

func listCommits(repository string) ([]commit, error) {
	command := exec.Command(
		"git", "-C", repository, "log", "-z",
		"--format=%H%x00%ae%x00%ct%x00%s",
		"--no-merges", "--exclude="+gitnotes.Ref, "--all",
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, err
		}
		return nil, errors.New(message)
	}
	if len(output) == 0 {
		return nil, nil
	}

	fields := bytes.Split(output, []byte{0})
	if len(fields) > 0 && len(fields[len(fields)-1]) == 0 {
		fields = fields[:len(fields)-1]
	}
	if len(fields)%4 != 0 {
		return nil, errors.New("Git returned malformed commit metadata")
	}
	commits := make([]commit, 0, len(fields)/4)
	for index := 0; index < len(fields); index += 4 {
		seconds, err := strconv.ParseInt(string(fields[index+2]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse commit timestamp: %w", err)
		}
		commits = append(commits, commit{
			hash:      string(fields[index]),
			author:    string(fields[index+1]),
			timestamp: time.Unix(seconds, 0),
			subject:   string(fields[index+3]),
		})
	}
	return commits, nil
}

func factForCommit(repository string, candidate commit) (Fact, error) {
	note, found, err := gitnotes.Read(repository, candidate.hash)
	if err != nil {
		return Fact{}, fmt.Errorf("read worklog note: %w", err)
	}
	if !found {
		return Fact{Outcome: candidate.subject}, nil
	}
	if len(bytes.TrimSpace(note)) > 0 && bytes.TrimSpace(note)[0] == '{' {
		event, _, err := worklogevent.Decode(bytes.NewReader(note))
		if err != nil {
			return Fact{}, fmt.Errorf("decode JSON worklog note: %w", err)
		}
		return factFromEvent(event), nil
	}
	fact, err := decodeLineNote(note)
	if err != nil {
		return Fact{}, fmt.Errorf("decode line-oriented worklog note: %w", err)
	}
	return fact, nil
}

func factFromEvent(event worklogevent.Event) Fact {
	fact := Fact{
		Outcome:      event.Outcome,
		Approach:     event.Approach,
		NextSteps:    append([]string(nil), event.NextSteps...),
		Technologies: append([]string(nil), event.Technologies...),
	}
	for _, decision := range event.Decisions {
		reason := decision.Reason
		fact.Decisions = append(fact.Decisions, Decision{Decision: decision.Decision, Reason: &reason})
	}
	return fact
}

func decodeLineNote(note []byte) (Fact, error) {
	var fact Fact
	versionSeen := false
	scanner := bufio.NewScanner(bytes.NewReader(note))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			return Fact{}, fmt.Errorf("invalid line %q", line)
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "Worklog-Version":
			if value != "1" {
				return Fact{}, fmt.Errorf("unsupported Worklog-Version %q", value)
			}
			versionSeen = true
		case "Outcome":
			fact.Outcome = value
		case "Approach":
			approach := value
			fact.Approach = &approach
		case "Decision":
			fact.Decisions = append(fact.Decisions, Decision{Decision: value})
		case "Next":
			fact.NextSteps = append(fact.NextSteps, value)
		case "Tech":
			for _, technology := range strings.Split(value, ",") {
				if technology = strings.TrimSpace(technology); technology != "" {
					fact.Technologies = append(fact.Technologies, technology)
				}
			}
		default:
			return Fact{}, fmt.Errorf("unknown field %q", strings.TrimSpace(key))
		}
	}
	if err := scanner.Err(); err != nil {
		return Fact{}, err
	}
	if !versionSeen {
		return Fact{}, errors.New("Worklog-Version is required")
	}
	if strings.TrimSpace(fact.Outcome) == "" {
		return Fact{}, errors.New("Outcome is required")
	}
	return fact, nil
}

func Encode(document Document) ([]byte, error) {
	contents, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode DailyFacts: %w", err)
	}
	return append(contents, '\n'), nil
}
