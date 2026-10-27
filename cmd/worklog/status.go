package main

import (
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/dailyfacts"
	"github.com/Vickyciou/worklog/internal/journal"
	"github.com/Vickyciou/worklog/internal/redaction"
)

type profileStatus int

const (
	statusNoActivity profileStatus = iota
	statusNotWritten
	statusUpToDate
	statusChanged
	statusNeedsAttention
)

var profileStatusLabels = map[profileStatus]string{
	statusNoActivity:     "no activity",
	statusNotWritten:     "not written yet",
	statusUpToDate:       "up to date",
	statusChanged:        "has new changes",
	statusNeedsAttention: "needs attention",
}

func (application cli) status(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, statusHelp)
		return nil
	}
	if len(arguments) != 0 {
		return errors.New("usage: worklog status")
	}
	configuration, err := config.Load(application.paths)
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Fprintln(application.stdout, "worklog is not set up yet.")
		fmt.Fprintln(application.stdout, "Next: run worklog setup inside a Git repository.")
		return nil
	}
	if err != nil {
		return err
	}

	counts := make(map[profileStatus]int)
	table := tabwriter.NewWriter(application.stdout, 0, 0, 3, ' ', 0)
	for _, profile := range configuration.Profiles {
		date, commits, state := todayStatus(profile, time.Now())
		counts[state]++
		activity := commitCount(commits)
		if state == statusNeedsAttention {
			activity = "-"
		}
		fmt.Fprintf(table, "  %s\t%s\t%s\t%s\n", profile.Name, date, activity, profileStatusLabels[state])
	}
	if err := table.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(application.stdout)
	// day stops before writing when any profile is misconfigured, so repairs come first.
	switch {
	case counts[statusNeedsAttention] > 0:
		fmt.Fprintln(application.stdout, "Next: worklog doctor")
	case counts[statusNotWritten]+counts[statusChanged] > 0:
		fmt.Fprintln(application.stdout, "Next: worklog day")
	case counts[statusUpToDate] > 0:
		fmt.Fprintln(application.stdout, "Today's journals are up to date.")
	default:
		fmt.Fprintln(application.stdout, "No activity yet today.")
	}
	return nil
}

func todayStatus(profile config.Profile, now time.Time) (string, int, profileStatus) {
	date, err := dailyfacts.CurrentDate(profile, now)
	if err != nil {
		return "-", 0, statusNeedsAttention
	}
	if len(profile.Repositories) == 0 || len(profile.Authors) == 0 {
		return date, 0, statusNeedsAttention
	}
	facts, err := dailyfacts.Collect(profile, date)
	if err != nil {
		return date, 0, statusNeedsAttention
	}
	if len(facts.Facts) == 0 {
		return date, 0, statusNoActivity
	}
	destination, err := journal.Path(profile, date)
	if err != nil {
		return date, len(facts.Facts), statusNeedsAttention
	}
	state, _, err := classifyJournal(destination, redaction.Apply(facts, profile.DeniedLiterals), false)
	if err != nil {
		return date, len(facts.Facts), statusNeedsAttention
	}
	switch state {
	case journalCurrent:
		return date, len(facts.Facts), statusUpToDate
	case journalChanged:
		return date, len(facts.Facts), statusChanged
	default:
		return date, len(facts.Facts), statusNotWritten
	}
}
