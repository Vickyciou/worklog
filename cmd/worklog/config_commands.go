package main

import (
	"bufio"
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
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func (application cli) day(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, dayHelp)
		return nil
	}
	if len(arguments) != 0 {
		return errors.New("usage: worklog day")
	}
	configuration, err := config.Load(application.paths)
	if err != nil {
		return err
	}
	if len(configuration.Profiles) != 1 {
		return fmt.Errorf("day requires exactly one configured profile in M1; found %d", len(configuration.Profiles))
	}
	profile := configuration.Profiles[0]
	date, err := dailyfacts.CurrentDate(profile, time.Now())
	if err != nil {
		return err
	}
	facts, err := dailyfacts.Collect(profile, date)
	if err != nil {
		return fmt.Errorf("collect profile %q: %w", profile.Name, err)
	}
	summary := dailysummary.Fallback(facts)
	contents, err := journal.Render(profile, facts, summary)
	if err != nil {
		return fmt.Errorf("render profile %q: %w", profile.Name, err)
	}
	destination, err := journal.Path(profile, date)
	if err != nil {
		return err
	}
	if err := journal.CheckAvailable(destination); err != nil {
		return fmt.Errorf("refuse journal destination %q: %w", destination, err)
	}

	fmt.Fprintf(application.stdout, "Journal preview: %s\n\n", destination)
	if _, err := application.stdout.Write(contents); err != nil {
		return err
	}
	fmt.Fprint(application.stdout, "\nWrite this journal? [y/N]: ")
	answer, err := bufio.NewReader(application.stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(application.stdout)
		return errors.New("journal write not confirmed")
	}
	if err := journal.WriteAtomic(destination, contents); err != nil {
		fmt.Fprintln(application.stdout)
		return fmt.Errorf("write journal %q: %w", destination, err)
	}
	fmt.Fprintf(application.stdout, "\nWrote journal %s.\n", destination)
	return nil
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
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	repositoryDefault := workingDirectory
	if root, rootErr := gitRepositoryRoot(workingDirectory); rootErr == nil {
		repositoryDefault = root
	}
	repository, err := application.prompt(reader, "Repository", repositoryDefault, true)
	if err != nil {
		return err
	}
	repository, err = expandUserPath(repository)
	if err != nil {
		return err
	}
	repositoryRoot, err := gitRepositoryRoot(repository)
	if err != nil {
		return err
	}
	author, err := gitAuthorEmail(repositoryRoot)
	if err != nil {
		return err
	}
	if author == "" || config.ValidateAuthor(author) != nil {
		author, err = application.prompt(reader, "Author email (not available from Git config)", "", true)
		if err != nil {
			return err
		}
	}
	author = strings.TrimSpace(author)
	if err := config.ValidateAuthor(author); err != nil {
		return err
	}
	timezone, detected := detectTimezone()
	if !detected {
		timezone, err = application.prompt(reader, "Timezone (could not be detected)", "UTC", true)
		if err != nil {
			return err
		}
	}
	if normalized, valid := validTimezone(timezone); valid {
		timezone = normalized
	} else {
		return fmt.Errorf("invalid timezone %q; use an IANA timezone such as Asia/Taipei", timezone)
	}
	language, detected := detectLanguage()
	if !detected {
		language, err = application.prompt(reader, "Language (could not be detected)", "zh-TW", true)
		if err != nil {
			return err
		}
	}
	language = normalizeLanguage(language)
	if language == "" {
		return errors.New("language is required; use a language such as zh-TW or en-US")
	}
	journalDefault, err := defaultJournalDestination(name)
	if err != nil {
		return err
	}
	destination, err := application.prompt(reader, "Journal folder", journalDefault, true)
	if err != nil {
		return err
	}
	destination, err = expandUserPath(destination)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve journal folder: %w", err)
	}

	fmt.Fprintf(application.stdout, `
Ready:
  Profile:        %s
  Repository:     %s
  Author:         %s
  Timezone:       %s
  Language:       %s
  Journal folder: %s
  Daily summary:  Built-in (no AI or network)

`, name, repositoryRoot, author, timezone, language, destination)
	confirmed, err := application.confirm(reader, "Create these settings?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(application.stdout, "Setup cancelled.")
		return nil
	}

	profile, err := config.NewProfile(name, timezone, language, destination, "")
	if err != nil {
		return err
	}
	configuration := config.Config{Version: config.Version, Profiles: []config.Profile{profile}}
	if err := configuration.AddAuthor(profile.Name, strings.TrimSpace(author)); err != nil {
		return err
	}
	if err := configuration.AddRepository(profile.Name, repositoryRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(profile.JournalDestination, 0o700); err != nil {
		return fmt.Errorf("create journal destination: %w", err)
	}
	if err := config.Save(application.paths, configuration); err != nil {
		return err
	}
	fmt.Fprintf(application.stdout, "Created profile %q.\n", profile.Name)
	return nil
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
		return errors.New("usage: worklog profile <add|list|show|validate>; run 'worklog profile --help' for details")
	}
	switch arguments[0] {
	case "add":
		if len(arguments) == 2 && (arguments[1] == "-h" || arguments[1] == "--help") {
			fmt.Fprint(application.stdout, profileAddHelp)
			return nil
		}
		return application.addProfile(arguments[1:])
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

func (application cli) addProfile(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: worklog profile add <name> --timezone <zone> --language <language> --journal-destination <path> [--summarizer-command <command>]")
	}
	name := arguments[0]
	flags := flag.NewFlagSet("profile add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	timezone := flags.String("timezone", "", "IANA timezone")
	language := flags.String("language", "", "journal language")
	destination := flags.String("journal-destination", "", "journal destination")
	summarizer := flags.String("summarizer-command", "", "optional summarizer command")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 {
		return errors.New("usage: worklog profile add <name> --timezone <zone> --language <language> --journal-destination <path> [--summarizer-command <command>]")
	}
	if *timezone == "" || *language == "" || *destination == "" {
		return errors.New("timezone, language, and journal destination are required")
	}
	profile, err := config.NewProfile(name, *timezone, *language, *destination, *summarizer)
	if err != nil {
		return err
	}
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
