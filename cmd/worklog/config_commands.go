package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/dailyfacts"
	"github.com/Vickyciou/worklog/internal/dailysummary"
	"github.com/Vickyciou/worklog/internal/journal"
	"github.com/Vickyciou/worklog/internal/redaction"
	"github.com/Vickyciou/worklog/internal/summarizer"
)

type cli struct {
	stdin  io.Reader
	stdout io.Writer
	paths  config.Paths
}

func (application cli) execute(command string, arguments []string) error {
	switch command {
	case "setup":
		return application.setup(arguments)
	case "settings":
		return application.settings(arguments)
	case "profile":
		return application.profile(arguments)
	case "author":
		return application.author(arguments)
	case "repo":
		return application.repository(arguments)
	case "inspect":
		return application.inspect(arguments)
	case "day":
		return application.day(arguments)
	case "doctor":
		return application.doctor(arguments)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func (application cli) day(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, dayHelp)
		return nil
	}
	requestedDate, refresh, err := parseDayArguments(arguments)
	if err != nil {
		return err
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	plans, err := application.planActiveProfiles(configuration, requestedDate, refresh)
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Fprintln(application.stdout, "No profile activity found; no journals to write.")
		return nil
	}
	reader := bufio.NewReader(application.stdin)
	for index := range plans {
		if index > 0 {
			fmt.Fprintln(application.stdout)
		}
		if err := application.runDayPlan(reader, plans[index], refresh, len(plans) > 1); err != nil {
			return err
		}
	}
	return nil
}

type dayPlan struct {
	profile     config.Profile
	date        string
	facts       dailyfacts.Document
	destination string
	existing    []byte
}

func (application cli) planActiveProfiles(configuration config.Config, requestedDate string, refresh bool) ([]dayPlan, error) {
	plans := make([]dayPlan, 0, len(configuration.Profiles))
	destinations := make(map[string]string)
	for _, profile := range configuration.Profiles {
		if len(profile.Repositories) == 0 {
			return nil, fmt.Errorf("profile %q has no registered repositories; run 'worklog repo add %s <path>'", profile.Name, profile.Name)
		}
		if len(profile.Authors) == 0 {
			return nil, fmt.Errorf("profile %q has no author identities; run 'worklog author add %s <email>'", profile.Name, profile.Name)
		}
		date := requestedDate
		if date == "" {
			var err error
			date, err = dailyfacts.CurrentDate(profile, time.Now())
			if err != nil {
				return nil, err
			}
		}
		facts, err := dailyfacts.Collect(profile, date)
		if err != nil {
			return nil, fmt.Errorf("collect profile %q: %w; verify mappings with 'worklog repo validate'", profile.Name, err)
		}
		if len(facts.Facts) == 0 {
			continue
		}
		facts = redaction.Apply(facts, profile.DeniedLiterals)
		destination, err := journal.Path(profile, date)
		if err != nil {
			return nil, err
		}
		if owner, exists := destinations[destination]; exists {
			return nil, fmt.Errorf("journal destination %q is ambiguous for active profiles %q and %q; update one with 'worklog profile set <name> --journal-destination <path>'", destination, owner, profile.Name)
		}
		destinations[destination] = profile.Name

		plan := dayPlan{profile: profile, date: date, facts: facts, destination: destination}
		if refresh {
			plan.existing, err = os.ReadFile(destination)
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("refresh journal %q: journal does not exist; omit --refresh to generate it", destination)
			}
			if err != nil {
				return nil, fmt.Errorf("read journal %q for refresh: %w", destination, err)
			}
			if err := journal.ValidateManaged(plan.existing); err != nil {
				return nil, fmt.Errorf("refuse journal refresh %q: %w", destination, err)
			}
		} else if err := journal.CheckAvailable(destination); err != nil {
			return nil, fmt.Errorf("refuse journal destination %q: %w", destination, err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func (application cli) runDayPlan(reader *bufio.Reader, plan dayPlan, refresh, showProfile bool) error {
	summary := dailysummary.Fallback(plan.facts)
	if plan.profile.EffectiveSummaryMode() != config.SummaryBuiltIn {
		if err := os.MkdirAll(application.paths.StateHome, 0o700); err != nil {
			fmt.Fprintln(application.stdout, "Summarizer unavailable (state directory unavailable); using Built-in.")
		} else {
			var providerSummary dailysummary.Document
			var failure summarizer.Failure
			if plan.profile.EffectiveSummaryMode() == config.SummaryCustom {
				providerSummary, failure = summarizer.Run(context.Background(), plan.profile.SummarizerCommand, plan.facts, application.paths.StateHome, summarizer.ConfiguredTimeout())
			} else {
				providerSummary, failure = summarizer.RunPreset(context.Background(), plan.profile.EffectiveSummaryMode(), plan.profile.Language, plan.facts, application.paths.StateHome, summarizer.ConfiguredTimeout())
			}
			if failure != "" {
				label := "Summarizer"
				if plan.profile.EffectiveSummaryMode() != config.SummaryCustom {
					label = dailySummaryLabel(plan.profile.EffectiveSummaryMode())
				}
				fmt.Fprintf(application.stdout, "%s unavailable (%s); using Built-in.\n", label, failure)
			} else {
				summary = providerSummary
			}
		}
	}
	contents, err := journal.Render(plan.profile, plan.facts, summary)
	if err != nil {
		return fmt.Errorf("render profile %q: %w", plan.profile.Name, err)
	}

	prompt := "Write this journal? [y/N]: "
	if refresh {
		var difference []byte
		contents, difference, err = journal.Refresh(plan.existing, contents)
		if err != nil {
			return fmt.Errorf("prepare journal refresh %q: %w", plan.destination, err)
		}
		if showProfile {
			fmt.Fprintf(application.stdout, "Journal refresh diff [%s]: %s\n\n", plan.profile.Name, plan.destination)
		} else {
			fmt.Fprintf(application.stdout, "Journal refresh diff: %s\n\n", plan.destination)
		}
		if _, err := application.stdout.Write(difference); err != nil {
			return err
		}
		prompt = "Refresh this journal? [y/N]: "
	} else {
		if showProfile {
			fmt.Fprintf(application.stdout, "Journal preview [%s]: %s\n\n", plan.profile.Name, plan.destination)
		} else {
			fmt.Fprintf(application.stdout, "Journal preview: %s\n\n", plan.destination)
		}
		if _, err := application.stdout.Write(contents); err != nil {
			return err
		}
	}
	fmt.Fprint(application.stdout, "\n"+prompt)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(application.stdout)
		if refresh {
			return errors.New("journal refresh not confirmed")
		}
		return errors.New("journal write not confirmed")
	}
	if refresh {
		err = journal.ReplaceAtomic(plan.destination, plan.existing, contents)
	} else {
		err = journal.WriteAtomic(plan.destination, contents)
	}
	if err != nil {
		fmt.Fprintln(application.stdout)
		return fmt.Errorf("write journal %q: %w", plan.destination, err)
	}
	verb := "Wrote"
	if refresh {
		verb = "Refreshed"
	}
	fmt.Fprintf(application.stdout, "\n%s journal %s.\n", verb, plan.destination)
	return nil
}

func parseDayArguments(arguments []string) (string, bool, error) {
	date := ""
	refresh := false
	for _, argument := range arguments {
		switch {
		case argument == "--refresh":
			if refresh {
				return "", false, errors.New("usage: worklog day [YYYY-MM-DD] [--refresh]")
			}
			refresh = true
		case strings.HasPrefix(argument, "-") || date != "":
			return "", false, errors.New("usage: worklog day [YYYY-MM-DD] [--refresh]")
		default:
			date = argument
		}
	}
	return date, refresh, nil
}

func (application cli) inspect(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, inspectHelp)
		return nil
	}
	if len(arguments) < 1 || len(arguments) > 2 {
		return errors.New("usage: worklog inspect <profile> [YYYY-MM-DD]")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	profile, found := configuration.Profile(arguments[0])
	if !found {
		return fmt.Errorf("profile %q does not exist", arguments[0])
	}
	date := ""
	if len(arguments) == 2 {
		date = arguments[1]
	} else {
		date, err = dailyfacts.CurrentDate(*profile, time.Now())
		if err != nil {
			return err
		}
	}
	document, err := dailyfacts.Collect(*profile, date)
	if err != nil {
		return fmt.Errorf("inspect profile %q: %w", profile.Name, err)
	}
	document = redaction.Apply(document, profile.DeniedLiterals)
	contents, err := dailyfacts.Encode(document)
	if err != nil {
		return err
	}
	_, err = application.stdout.Write(contents)
	return err
}

func (application cli) setup(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, setupHelp)
		return nil
	}
	if len(arguments) != 0 {
		return errors.New("usage: worklog setup; run 'worklog setup --help' for details")
	}
	if _, err := config.Load(application.paths); err == nil {
		return errors.New("configuration already exists; use profile, author, or repo commands to update it")
	} else if !errors.Is(err, config.ErrNotConfigured) {
		return err
	}

	reader := bufio.NewReader(application.stdin)
	name, err := application.chooseInitialProfile(reader)
	if err != nil {
		return err
	}
	profile, repositoryRoot, author, err := application.collectGuidedProfile(reader, name)
	if err != nil {
		return err
	}

	fmt.Fprintf(application.stdout, `
Ready:
  Profile:        %s
  Repository:     %s
  Author:         %s
  Timezone:       %s
  Language:       %s
  Journal folder: %s
  Daily summary:  %s

`, profile.Name, repositoryRoot, author, profile.Timezone, profile.Language, profile.JournalDestination, dailySummaryLabel(profile.EffectiveSummaryMode()))
	confirmed, err := application.confirm(reader, "Create these settings?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(application.stdout, "Setup cancelled.")
		return nil
	}

	configuration := config.Config{Version: config.Version, Profiles: []config.Profile{profile}}
	if err := os.MkdirAll(profile.JournalDestination, 0o700); err != nil {
		return fmt.Errorf("create journal destination: %w", err)
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Created profile %q.\n", profile.Name)
	return nil
}

func (application cli) collectGuidedProfile(reader *bufio.Reader, name string) (config.Profile, string, string, error) {
	if err := config.ValidateProfileName(name); err != nil {
		return config.Profile{}, "", "", err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return config.Profile{}, "", "", fmt.Errorf("get current directory: %w", err)
	}
	repositoryDefault := workingDirectory
	if root, rootErr := gitRepositoryRoot(workingDirectory); rootErr == nil {
		repositoryDefault = root
	}
	repository, err := application.prompt(reader, "Repository", repositoryDefault, true)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	repository, err = expandUserPath(repository)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	repositoryRoot, err := gitRepositoryRoot(repository)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	author, err := gitAuthorEmail(repositoryRoot)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	if author == "" || config.ValidateAuthor(author) != nil {
		author, err = application.prompt(reader, "Author email (not available from Git config)", "", true)
		if err != nil {
			return config.Profile{}, "", "", err
		}
	}
	author = strings.TrimSpace(author)
	if err := config.ValidateAuthor(author); err != nil {
		return config.Profile{}, "", "", err
	}
	timezone, detected := detectTimezone()
	if !detected {
		timezone, err = application.prompt(reader, "Timezone (could not be detected)", "UTC", true)
		if err != nil {
			return config.Profile{}, "", "", err
		}
	}
	if normalized, valid := validTimezone(timezone); valid {
		timezone = normalized
	} else {
		return config.Profile{}, "", "", fmt.Errorf("invalid timezone %q; use an IANA timezone such as Asia/Taipei", timezone)
	}
	language, detected := detectLanguage()
	if !detected {
		language, err = application.prompt(reader, "Language (could not be detected)", "zh-TW", true)
		if err != nil {
			return config.Profile{}, "", "", err
		}
	}
	language = normalizeLanguage(language)
	if language == "" {
		return config.Profile{}, "", "", errors.New("language is required; use a language such as zh-TW or en-US")
	}
	journalDefault, err := defaultJournalDestination(name)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	destination, err := application.prompt(reader, "Journal folder", journalDefault, true)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	destination, err = expandUserPath(destination)
	if err != nil {
		return config.Profile{}, "", "", err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return config.Profile{}, "", "", fmt.Errorf("resolve journal folder: %w", err)
	}
	summaryMode, err := application.chooseDailySummary(reader, config.SummaryBuiltIn)
	if err != nil {
		return config.Profile{}, "", "", err
	}

	profile, err := config.NewProfile(name, timezone, language, destination, "")
	if err != nil {
		return config.Profile{}, "", "", err
	}
	profile.SummaryMode = summaryMode
	profile.Authors = []string{author}
	profile.Repositories = []string{repositoryRoot}
	if err := config.ValidateProfile(profile); err != nil {
		return config.Profile{}, "", "", err
	}
	return profile, repositoryRoot, author, nil
}

func (application cli) chooseInitialProfile(reader *bufio.Reader) (string, error) {
	fmt.Fprint(application.stdout, `Choose a profile:
  1. Work      Work repositories and journals
  2. Personal  Personal repositories and journals
  3. Custom    Choose your own profile name

`)
	choice, err := application.prompt(reader, "Profile", "1", true)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "1", "work":
		return "work", nil
	case "2", "personal":
		return "personal", nil
	case "3", "custom":
		name, err := application.prompt(reader, "Profile name (for example client-a)", "", true)
		if err != nil {
			return "", err
		}
		if err := config.ValidateProfileName(name); err != nil {
			return "", err
		}
		return name, nil
	default:
		return "", errors.New("profile must be 1 (Work), 2 (Personal), or 3 (Custom)")
	}
}

func (application cli) chooseDailySummary(reader *bufio.Reader, current string) (string, error) {
	fmt.Fprintln(application.stdout, "Daily summary (AI is optional):")
	fmt.Fprintln(application.stdout, "  1. Built-in    Deterministic; no AI or network")
	for index, preset := range []struct {
		mode       string
		label      string
		executable string
	}{
		{mode: config.SummaryCodex, label: "Codex", executable: "codex"},
		{mode: config.SummaryClaudeCode, label: "Claude Code", executable: "claude"},
	} {
		availability := "available; uses the CLI's current session"
		if _, err := exec.LookPath(preset.executable); err != nil {
			availability = "unavailable; executable not found"
		}
		fmt.Fprintf(application.stdout, "  %d. %-11s %s\n", index+2, preset.label, availability)
	}
	fmt.Fprintln(application.stdout)
	defaultValue := map[string]string{config.SummaryBuiltIn: "1", config.SummaryCodex: "2", config.SummaryClaudeCode: "3"}[current]
	if defaultValue == "" {
		defaultValue = "1"
	}
	choice, err := application.prompt(reader, "Daily summary", defaultValue, true)
	if err != nil {
		return "", err
	}
	var mode string
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "1", "built-in", "builtin":
		mode = config.SummaryBuiltIn
	case "2", "codex":
		mode = config.SummaryCodex
	case "3", "claude", "claude-code", "claude code":
		mode = config.SummaryClaudeCode
	default:
		return "", errors.New("daily summary must be 1 (Built-in), 2 (Codex), or 3 (Claude Code)")
	}
	if err := validateSummaryModeAvailable(mode); err != nil {
		return "", err
	}
	return mode, nil
}

func validateSummaryModeAvailable(mode string) error {
	executable := map[string]string{config.SummaryCodex: "codex", config.SummaryClaudeCode: "claude"}[mode]
	if executable == "" {
		return nil
	}
	if _, err := exec.LookPath(executable); err != nil {
		return fmt.Errorf("%s daily summary is unavailable because %q was not found; choose Built-in or install the CLI", dailySummaryLabel(mode), executable)
	}
	return nil
}

func dailySummaryLabel(mode string) string {
	switch mode {
	case config.SummaryCodex:
		return "Codex (uses the CLI's current session)"
	case config.SummaryClaudeCode:
		return "Claude Code (uses the CLI's current session)"
	case config.SummaryCustom:
		return "Custom command (advanced configuration)"
	default:
		return "Built-in (no AI or network)"
	}
}

func (application cli) prompt(reader *bufio.Reader, label, defaultValue string, required bool) (string, error) {
	if defaultValue == "" {
		fmt.Fprintf(application.stdout, "%s: ", label)
	} else {
		fmt.Fprintf(application.stdout, "%s [%s]: ", label, defaultValue)
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	if value == "" && required {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}

func (application cli) confirm(reader *bufio.Reader, label string) (bool, error) {
	fmt.Fprintf(application.stdout, "%s [Y/n]: ", label)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, errors.New("confirmation must be y, yes, n, or no")
	}
}

func (application cli) profile(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, profileHelp)
		return nil
	}
	if len(arguments) == 0 {
		return errors.New("usage: worklog profile <add|set|list|show|validate>; run 'worklog profile --help' for details")
	}
	switch arguments[0] {
	case "add":
		if len(arguments) == 2 && (arguments[1] == "-h" || arguments[1] == "--help") {
			fmt.Fprint(application.stdout, profileAddHelp)
			return nil
		}
		return application.addProfile(arguments[1:])
	case "set":
		if len(arguments) == 2 && (arguments[1] == "-h" || arguments[1] == "--help") {
			fmt.Fprint(application.stdout, profileSetHelp)
			return nil
		}
		return application.setProfile(arguments[1:])
	case "list":
		return application.listProfiles(arguments[1:])
	case "show":
		return application.showProfile(arguments[1:])
	case "validate":
		return application.validateProfiles(arguments[1:])
	default:
		return fmt.Errorf("unknown profile command %q", arguments[0])
	}
}

func (application cli) settings(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, settingsHelp)
		return nil
	}
	if len(arguments) > 1 {
		return errors.New("usage: worklog settings [profile]")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(application.stdin)
	profileName := ""
	if len(arguments) == 1 {
		profileName = arguments[0]
	} else if len(configuration.Profiles) == 1 {
		profileName = configuration.Profiles[0].Name
	} else {
		profileName, err = application.chooseExistingProfile(reader, configuration.Profiles)
		if err != nil {
			return err
		}
	}
	profile, found := configuration.Profile(profileName)
	if !found {
		return fmt.Errorf("profile %q does not exist", profileName)
	}

	fmt.Fprintf(application.stdout, `Current settings for %s:
  Language:       %s
  Timezone:       %s
  Journal folder: %s
  Daily summary:  %s

`, profile.Name, profile.Language, profile.Timezone, profile.JournalDestination, dailySummaryLabel(profile.EffectiveSummaryMode()))
	language, err := application.prompt(reader, "Language", profile.Language, true)
	if err != nil {
		return err
	}
	timezone, err := application.prompt(reader, "Timezone", profile.Timezone, true)
	if err != nil {
		return err
	}
	destination, err := application.prompt(reader, "Journal folder", profile.JournalDestination, true)
	if err != nil {
		return err
	}
	summaryMode := profile.EffectiveSummaryMode()
	if summaryMode == config.SummaryCustom {
		fmt.Fprintln(application.stdout, "Daily summary remains Custom command (advanced configuration).")
	} else {
		summaryMode, err = application.chooseDailySummary(reader, summaryMode)
		if err != nil {
			return err
		}
	}
	candidate, err := updatedProfile(*profile, language, timezone, destination)
	if err != nil {
		return err
	}
	if summaryMode != config.SummaryCustom {
		candidate.SummaryMode = summaryMode
		candidate.SummarizerCommand = ""
	}
	fmt.Fprintf(application.stdout, `
Ready:
  Profile:        %s
  Language:       %s
  Timezone:       %s
  Journal folder: %s
  Daily summary:  %s

`, candidate.Name, candidate.Language, candidate.Timezone, candidate.JournalDestination, dailySummaryLabel(candidate.EffectiveSummaryMode()))
	if candidate.JournalDestination != profile.JournalDestination {
		fmt.Fprintln(application.stdout, "Existing journals will stay in their current folder; future output uses the new folder.")
	}
	confirmed, err := application.confirm(reader, "Save these settings?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(application.stdout, "Settings unchanged.")
		return nil
	}
	if err := configuration.ReplaceProfile(candidate); err != nil {
		return err
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Updated profile %q.\n", candidate.Name)
	return nil
}

func (application cli) chooseExistingProfile(reader *bufio.Reader, profiles []config.Profile) (string, error) {
	fmt.Fprintln(application.stdout, "Choose a profile:")
	for index, profile := range profiles {
		fmt.Fprintf(application.stdout, "  %d. %s\n", index+1, profile.Name)
	}
	fmt.Fprintln(application.stdout)
	choice, err := application.prompt(reader, "Profile", "", true)
	if err != nil {
		return "", err
	}
	for index, profile := range profiles {
		if choice == profile.Name || choice == fmt.Sprint(index+1) {
			return profile.Name, nil
		}
	}
	return "", fmt.Errorf("profile %q does not exist", choice)
}

func (application cli) setProfile(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: worklog profile set <name> [--language <language>] [--timezone <zone>] [--journal-destination <path>] [--daily-summary <built-in|codex|claude-code>] [--deny-literal <value> ...]")
	}
	name := arguments[0]
	flags := flag.NewFlagSet("profile set", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	language := flags.String("language", "", "journal language")
	timezone := flags.String("timezone", "", "IANA timezone")
	destination := flags.String("journal-destination", "", "journal destination")
	dailySummary := flags.String("daily-summary", "", "daily summary preset")
	var deniedLiterals stringListFlag
	flags.Var(&deniedLiterals, "deny-literal", "literal to redact before summarization; repeatable")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 {
		return errors.New("usage: worklog profile set <name> [--language <language>] [--timezone <zone>] [--journal-destination <path>] [--daily-summary <built-in|codex|claude-code>] [--deny-literal <value> ...]")
	}
	specified := map[string]bool{}
	flags.Visit(func(current *flag.Flag) { specified[current.Name] = true })
	if len(specified) == 0 {
		return errors.New("at least one profile setting is required")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	profile, found := configuration.Profile(name)
	if !found {
		return fmt.Errorf("profile %q does not exist", name)
	}
	candidate := *profile
	if specified["language"] {
		candidate.Language = *language
	}
	if specified["timezone"] {
		candidate.Timezone = *timezone
	}
	if specified["journal-destination"] {
		candidate.JournalDestination = *destination
	}
	if specified["deny-literal"] {
		candidate.DeniedLiterals = deniedLiterals.values()
	}
	if specified["daily-summary"] {
		mode, err := normalizeSummaryMode(*dailySummary)
		if err != nil {
			return err
		}
		if err := validateSummaryModeAvailable(mode); err != nil {
			return err
		}
		candidate.SummaryMode = mode
		candidate.SummarizerCommand = ""
	}
	candidate, err = updatedProfile(candidate, candidate.Language, candidate.Timezone, candidate.JournalDestination)
	if err != nil {
		return err
	}
	if err := configuration.ReplaceProfile(candidate); err != nil {
		return err
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Updated profile %q.\n", candidate.Name)
	return nil
}

func updatedProfile(profile config.Profile, language, timezone, destination string) (config.Profile, error) {
	language = normalizeLanguage(language)
	if language == "" {
		return config.Profile{}, errors.New("language is required; use a language such as zh-TW or en-US")
	}
	timezone, valid := validTimezone(strings.TrimSpace(timezone))
	if !valid {
		return config.Profile{}, fmt.Errorf("invalid timezone %q; use an IANA timezone such as Asia/Taipei", strings.TrimSpace(timezone))
	}
	destination, err := expandUserPath(destination)
	if err != nil {
		return config.Profile{}, err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return config.Profile{}, fmt.Errorf("resolve journal folder: %w", err)
	}
	profile.Language = language
	profile.Timezone = timezone
	profile.JournalDestination = filepath.Clean(destination)
	if err := config.ValidateProfile(profile); err != nil {
		return config.Profile{}, err
	}
	return profile, nil
}

func normalizeSummaryMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "built-in", "builtin":
		return config.SummaryBuiltIn, nil
	case "codex":
		return config.SummaryCodex, nil
	case "claude", "claude-code", "claude code":
		return config.SummaryClaudeCode, nil
	default:
		return "", errors.New("daily summary must be built-in, codex, or claude-code")
	}
}

type stringListFlag struct {
	items []string
}

func (values *stringListFlag) String() string {
	return strings.Join(values.items, ",")
}

func (values *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("literal must not be empty")
	}
	values.items = append(values.items, value)
	return nil
}

func (values stringListFlag) values() []string {
	return append([]string(nil), values.items...)
}

func (application cli) addProfile(arguments []string) error {
	if len(arguments) == 0 || len(arguments) == 1 && !strings.HasPrefix(arguments[0], "-") {
		name := ""
		if len(arguments) == 1 {
			name = arguments[0]
		}
		return application.addProfileGuided(name)
	}
	return application.addProfileAdvanced(arguments)
}

func (application cli) addProfileGuided(name string) error {
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(application.stdin)
	if name == "" {
		name, err = application.chooseAdditionalProfile(reader, configuration)
		if err != nil {
			return err
		}
	}
	name = strings.TrimSpace(name)
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	if _, found := configuration.Profile(name); found {
		return fmt.Errorf("profile %q already exists", name)
	}
	profile, repositoryRoot, author, err := application.collectGuidedProfile(reader, name)
	if err != nil {
		return err
	}
	if err := configuration.AddProfile(profile); err != nil {
		return err
	}

	fmt.Fprintf(application.stdout, `
Ready to add:
  Profile:        %s
  Repository:     %s
  Author:         %s
  Timezone:       %s
  Language:       %s
  Journal folder: %s
  Daily summary:  %s

`, profile.Name, repositoryRoot, author, profile.Timezone, profile.Language, profile.JournalDestination, dailySummaryLabel(profile.EffectiveSummaryMode()))
	confirmed, err := application.confirm(reader, "Add this profile?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(application.stdout, "Profile not added.")
		return nil
	}
	if err := os.MkdirAll(profile.JournalDestination, 0o700); err != nil {
		return fmt.Errorf("create journal destination: %w", err)
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Added profile %q with its author and repository.\n", profile.Name)
	return nil
}

func (application cli) chooseAdditionalProfile(reader *bufio.Reader, configuration config.Config) (string, error) {
	_, hasWork := configuration.Profile("work")
	_, hasPersonal := configuration.Profile("personal")
	workDescription := "Work repositories and journals"
	if hasWork {
		workDescription = "Already configured"
	}
	personalDescription := "Personal repositories and journals"
	if hasPersonal {
		personalDescription = "Already configured"
	}
	defaultChoice := "3"
	if !hasWork {
		defaultChoice = "1"
	} else if !hasPersonal {
		defaultChoice = "2"
	}
	fmt.Fprintf(application.stdout, `Choose a profile:
  1. Work      %s
  2. Personal  %s
  3. Custom    Choose your own profile name

`, workDescription, personalDescription)
	choice, err := application.prompt(reader, "Profile", defaultChoice, true)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "1", "work":
		if hasWork {
			return "", errors.New("profile \"work\" already exists; choose Personal or Custom")
		}
		return "work", nil
	case "2", "personal":
		if hasPersonal {
			return "", errors.New("profile \"personal\" already exists; choose Work or Custom")
		}
		return "personal", nil
	case "3", "custom":
		name, err := application.prompt(reader, "Profile name (for example client-a)", "", true)
		if err != nil {
			return "", err
		}
		if err := config.ValidateProfileName(name); err != nil {
			return "", err
		}
		return name, nil
	default:
		return "", errors.New("profile must be 1 (Work), 2 (Personal), or 3 (Custom)")
	}
}

func (application cli) addProfileAdvanced(arguments []string) error {
	if len(arguments) == 0 || strings.HasPrefix(arguments[0], "-") {
		return errors.New("usage: worklog profile add [<name>] or worklog profile add <name> --timezone <zone> --language <language> --journal-destination <path> [--daily-summary <built-in|codex|claude-code>] [--deny-literal <value> ...] [--summarizer-command <command>]")
	}
	name := arguments[0]
	flags := flag.NewFlagSet("profile add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	timezone := flags.String("timezone", "", "IANA timezone")
	language := flags.String("language", "", "journal language")
	destination := flags.String("journal-destination", "", "journal destination")
	summarizer := flags.String("summarizer-command", "", "optional summarizer command")
	dailySummary := flags.String("daily-summary", "", "daily summary preset")
	var deniedLiterals stringListFlag
	flags.Var(&deniedLiterals, "deny-literal", "literal to redact before summarization; repeatable")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 {
		return errors.New("usage: worklog profile add <name> --timezone <zone> --language <language> --journal-destination <path> [--daily-summary <built-in|codex|claude-code>] [--deny-literal <value> ...] [--summarizer-command <command>]")
	}
	if *timezone == "" || *language == "" || *destination == "" {
		return errors.New("timezone, language, and journal destination are required")
	}
	profile, err := config.NewProfile(name, *timezone, *language, *destination, *summarizer)
	if err != nil {
		return err
	}
	if *dailySummary != "" {
		if *summarizer != "" {
			return errors.New("daily summary preset and summarizer command cannot both be configured")
		}
		mode, err := normalizeSummaryMode(*dailySummary)
		if err != nil {
			return err
		}
		if err := validateSummaryModeAvailable(mode); err != nil {
			return err
		}
		profile.SummaryMode = mode
	}
	profile.DeniedLiterals = deniedLiterals.values()
	configuration, err := application.loadOrInitialize()
	if err != nil {
		return err
	}
	if err := configuration.AddProfile(profile); err != nil {
		return err
	}
	if err := os.MkdirAll(profile.JournalDestination, 0o700); err != nil {
		return fmt.Errorf("create journal destination: %w", err)
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Added profile %q.\n", profile.Name)
	return nil
}

func (application cli) listProfiles(arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: worklog profile list")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	for _, profile := range configuration.Profiles {
		fmt.Fprintln(application.stdout, profile.Name)
	}
	return nil
}

func (application cli) showProfile(arguments []string) error {
	if len(arguments) != 1 {
		return errors.New("usage: worklog profile show <name>")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	profile, found := configuration.Profile(arguments[0])
	if !found {
		return fmt.Errorf("profile %q does not exist", arguments[0])
	}
	return writeJSON(application.stdout, profile)
}

func (application cli) validateProfiles(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("usage: worklog profile validate [name]")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	if len(arguments) == 1 {
		profile, found := configuration.Profile(arguments[0])
		if !found {
			return fmt.Errorf("profile %q does not exist", arguments[0])
		}
		if err := config.ValidateProfile(*profile); err != nil {
			return err
		}
		fmt.Fprintf(application.stdout, "Profile %q is valid.\n", profile.Name)
		return nil
	}
	fmt.Fprintln(application.stdout, "All profiles are valid.")
	return nil
}

func (application cli) author(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, authorHelp)
		return nil
	}
	if len(arguments) == 0 {
		return errors.New("usage: worklog author <add|list|validate>; run 'worklog author --help' for details")
	}
	switch arguments[0] {
	case "add":
		if len(arguments) != 3 {
			return errors.New("usage: worklog author add <profile> <email>")
		}
		configuration, err := config.Load(application.paths)
		if err != nil {
			return err
		}
		if err := configuration.AddAuthor(arguments[1], arguments[2]); err != nil {
			return err
		}
		if err := config.Save(application.paths, configuration); err != nil {
			return err
		}
		fmt.Fprintf(application.stdout, "Added author identity to profile %q.\n", arguments[1])
		return nil
	case "list":
		if len(arguments) != 2 {
			return errors.New("usage: worklog author list <profile>")
		}
		configuration, err := config.Load(application.paths)
		if err != nil {
			return err
		}
		profile, found := configuration.Profile(arguments[1])
		if !found {
			return fmt.Errorf("profile %q does not exist", arguments[1])
		}
		for _, author := range profile.Authors {
			fmt.Fprintln(application.stdout, author)
		}
		return nil
	case "validate":
		if len(arguments) != 2 {
			return errors.New("usage: worklog author validate <profile>")
		}
		configuration, err := config.Load(application.paths)
		if err != nil {
			return err
		}
		profile, found := configuration.Profile(arguments[1])
		if !found {
			return fmt.Errorf("profile %q does not exist", arguments[1])
		}
		for _, author := range profile.Authors {
			if err := config.ValidateAuthor(author); err != nil {
				return err
			}
		}
		fmt.Fprintf(application.stdout, "Author identities for profile %q are valid.\n", profile.Name)
		return nil
	default:
		return fmt.Errorf("unknown author command %q", arguments[0])
	}
}

func (application cli) repository(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, repoHelp)
		return nil
	}
	if len(arguments) == 0 {
		return errors.New("usage: worklog repo <add|list|show|validate>; run 'worklog repo --help' for details")
	}
	switch arguments[0] {
	case "add":
		if len(arguments) == 2 && (arguments[1] == "-h" || arguments[1] == "--help") {
			fmt.Fprint(application.stdout, repoAddHelp)
			return nil
		}
		return application.addRepository(arguments[1:])
	case "list":
		return application.listRepositories(arguments[1:])
	case "show":
		return application.showRepository(arguments[1:])
	case "validate":
		return application.validateRepositories(arguments[1:])
	default:
		return fmt.Errorf("unknown repo command %q", arguments[0])
	}
}

func (application cli) addRepository(arguments []string) error {
	if len(arguments) != 2 {
		return errors.New("usage: worklog repo add <profile> <path>")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	repositoryRoot, err := gitRepositoryRoot(arguments[1])
	if err != nil {
		return err
	}
	if err := configuration.AddRepository(arguments[0], repositoryRoot); err != nil {
		return err
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Registered repository %q to profile %q.\n", repositoryRoot, arguments[0])
	return nil
}

func (application cli) listRepositories(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("usage: worklog repo list [profile]")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	for _, profile := range configuration.Profiles {
		if len(arguments) == 1 && profile.Name != arguments[0] {
			continue
		}
		for _, repository := range profile.Repositories {
			fmt.Fprintf(application.stdout, "%s\t%s\n", profile.Name, repository)
		}
	}
	if len(arguments) == 1 {
		if _, found := configuration.Profile(arguments[0]); !found {
			return fmt.Errorf("profile %q does not exist", arguments[0])
		}
	}
	return nil
}

type resolvedRepository struct {
	Profile            string `json:"profile"`
	Repository         string `json:"repository"`
	Timezone           string `json:"timezone"`
	Language           string `json:"language"`
	JournalDestination string `json:"journal_destination"`
}

func (application cli) showRepository(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("usage: worklog repo show [path]")
	}
	path := "."
	if len(arguments) == 1 {
		path = arguments[0]
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	profile, repository, err := configuration.ResolveRepository(path)
	if err != nil {
		return err
	}
	return writeJSON(application.stdout, resolvedRepository{
		Profile:            profile.Name,
		Repository:         repository,
		Timezone:           profile.Timezone,
		Language:           profile.Language,
		JournalDestination: profile.JournalDestination,
	})
}

func (application cli) validateRepositories(arguments []string) error {
	if len(arguments) > 1 {
		return errors.New("usage: worklog repo validate [path]")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	if len(arguments) == 1 {
		profile, registered, err := configuration.ResolveRepository(arguments[0])
		if err != nil {
			return err
		}
		actual, err := gitRepositoryRoot(arguments[0])
		if err != nil {
			return err
		}
		if actual != registered {
			return fmt.Errorf("path belongs to nested repository %q, not registered repository %q", actual, registered)
		}
		fmt.Fprintf(application.stdout, "Repository is valid and owned by profile %q.\n", profile.Name)
		return nil
	}
	for _, profile := range configuration.Profiles {
		for _, repository := range profile.Repositories {
			actual, err := gitRepositoryRoot(repository)
			if err != nil {
				return fmt.Errorf("repository %q for profile %q: %w", repository, profile.Name, err)
			}
			if actual != repository {
				return fmt.Errorf("repository %q for profile %q resolves to Git root %q", repository, profile.Name, actual)
			}
		}
	}
	fmt.Fprintln(application.stdout, "All repository mappings are valid.")
	return nil
}

func (application cli) loadOrInitialize() (config.Config, error) {
	configuration, err := config.Load(application.paths)
	if errors.Is(err, config.ErrNotConfigured) {
		return config.Config{Version: config.Version}, nil
	}
	return configuration, err
}

func gitRepositoryRoot(path string) (string, error) {
	command := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("%q is not a Git repository", path)
	}
	root := strings.TrimSpace(string(output))
	canonical, err := config.CanonicalPath(root)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

func gitAuthorEmail(repository string) (string, error) {
	command := exec.Command("git", "-C", repository, "config", "--get", "user.email")
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("read Git author email: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func detectTimezone() (string, bool) {
	if value, configured := os.LookupEnv("TZ"); configured && strings.TrimSpace(value) != "" {
		return validTimezone(strings.TrimPrefix(strings.TrimSpace(value), ":"))
	}
	if timezone, valid := validTimezone(time.Now().Location().String()); valid && timezone != "Local" {
		return timezone, true
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		for _, marker := range []string{"/zoneinfo/", "/zoneinfo.default/"} {
			if index := strings.LastIndex(target, marker); index >= 0 {
				if timezone, valid := validTimezone(target[index+len(marker):]); valid {
					return timezone, true
				}
			}
		}
	}
	if contents, err := os.ReadFile("/etc/timezone"); err == nil {
		if timezone, valid := validTimezone(strings.TrimSpace(string(contents))); valid {
			return timezone, true
		}
	}
	return "", false
}

func validTimezone(value string) (string, bool) {
	if value == "" || value == "Local" {
		return "", false
	}
	if _, err := time.LoadLocation(value); err != nil {
		return "", false
	}
	return value, true
}

func detectLanguage() (string, bool) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.TrimSpace(os.Getenv(name))
		if value == "" {
			continue
		}
		language := normalizeLanguage(value)
		return language, language != ""
	}
	return "", false
}

func normalizeLanguage(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexAny(value, ".@"); index >= 0 {
		value = value[:index]
	}
	if value == "" || value == "C" || value == "POSIX" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return character == '_' || character == '-'
	})
	if len(parts) == 0 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	if len(parts) > 1 {
		parts[1] = strings.ToUpper(parts[1])
	}
	return strings.Join(parts, "-")
}

func expandUserPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func defaultJournalDestination(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "Documents", "worklog", strings.TrimSpace(profile)), nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
