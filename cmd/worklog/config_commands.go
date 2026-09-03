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
	default:
		return fmt.Errorf("unknown command %q", command)
	}
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
	if len(arguments) != 0 {
		return errors.New("usage: worklog setup")
	}
	if _, err := config.Load(application.paths); err == nil {
		return errors.New("configuration already exists; use profile, author, or repo commands to update it")
	} else if !errors.Is(err, config.ErrNotConfigured) {
		return err
	}

	reader := bufio.NewReader(application.stdin)
	name, err := application.prompt(reader, "Profile name", "", true)
	if err != nil {
		return err
	}
	author, err := application.prompt(reader, "Author email", "", true)
	if err != nil {
		return err
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	repository, err := application.prompt(reader, "Repository", workingDirectory, true)
	if err != nil {
		return err
	}
	timezone, err := application.prompt(reader, "Timezone", defaultTimezone(), true)
	if err != nil {
		return err
	}
	language, err := application.prompt(reader, "Language", "zh-TW", true)
	if err != nil {
		return err
	}
	journalDefault, err := defaultJournalDestination(name)
	if err != nil {
		return err
	}
	destination, err := application.prompt(reader, "Journal destination", journalDefault, true)
	if err != nil {
		return err
	}
	summarizer, err := application.prompt(reader, "Summarizer command (optional; credentials must use the environment)", "", false)
	if err != nil {
		return err
	}

	profile, err := config.NewProfile(name, timezone, language, destination, summarizer)
	if err != nil {
		return err
	}
	configuration := config.Config{Version: config.Version, Profiles: []config.Profile{profile}}
	if err := configuration.AddAuthor(profile.Name, strings.TrimSpace(author)); err != nil {
		return err
	}
	repositoryRoot, err := gitRepositoryRoot(repository)
	if err != nil {
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

func (application cli) profile(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: worklog profile <add|list|show|validate>")
	}
	switch arguments[0] {
	case "add":
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
	if len(arguments) == 0 {
		return errors.New("usage: worklog author <add|list|validate>")
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
	if len(arguments) == 0 {
		return errors.New("usage: worklog repo <add|list|show|validate>")
	}
	switch arguments[0] {
	case "add":
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

func defaultTimezone() string {
	if timezone := strings.TrimSpace(os.Getenv("TZ")); timezone != "" {
		return timezone
	}
	return "UTC"
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
