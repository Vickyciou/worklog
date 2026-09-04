package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	Version           = 1
	ConfigHomeEnv     = "WORKLOG_CONFIG_HOME"
	StateHomeEnv      = "WORKLOG_STATE_HOME"
	configurationFile = "config.json"
)

var ErrNotConfigured = errors.New("worklog is not configured; run 'worklog setup'")

var credentialPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:^|\s)--?(?:api[-_]?key|apikey|access[-_]?token|auth[-_]?token|token|password|passwd|client[-_]?secret|secret)(?:\s*=\s*|\s+)(?:"([^"]*)"|'([^']*)'|([^\s]+))`),
	regexp.MustCompile(`(?i)(?:^|\s)(?:[a-z0-9]+[-_])*(?:api[-_]?key|access[-_]?token|auth[-_]?token|token|password|passwd|client[-_]?secret|secret)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s]+))`),
	regexp.MustCompile(`(?i)\bauthorization\s*:\s*(?:bearer\s+)?(?:"([^"]*)"|'([^']*)'|([^\s'"]+))`),
	regexp.MustCompile(`(?i)\bbearer\s+(?:"([^"]*)"|'([^']*)'|([^\s'"]+))`),
}

type Paths struct {
	ConfigHome string
	StateHome  string
	ConfigFile string
}

type Config struct {
	Version  int       `json:"version"`
	Profiles []Profile `json:"profiles"`
}

type Profile struct {
	Name               string   `json:"name"`
	Authors            []string `json:"authors"`
	Repositories       []string `json:"repositories"`
	Timezone           string   `json:"timezone"`
	Language           string   `json:"language"`
	JournalDestination string   `json:"journal_destination"`
	DeniedLiterals     []string `json:"denied_literals,omitempty"`
	SummarizerCommand  string   `json:"summarizer_command,omitempty"`
}

func DefaultPaths() (Paths, error) {
	configHome, err := applicationHome(ConfigHomeEnv, configBase)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve config home: %w", err)
	}
	stateHome, err := applicationHome(StateHomeEnv, stateBase)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve state home: %w", err)
	}
	return Paths{
		ConfigHome: configHome,
		StateHome:  stateHome,
		ConfigFile: filepath.Join(configHome, configurationFile),
	}, nil
}

func applicationHome(environmentName string, fallback func() (string, error)) (string, error) {
	if value := strings.TrimSpace(os.Getenv(environmentName)); value != "" {
		return filepath.Abs(value)
	}
	base, err := fallback()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "worklog"), nil
}

func configBase() (string, error) {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return filepath.Abs(value)
	}
	return os.UserConfigDir()
}

func stateBase() (string, error) {
	if value := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); value != "" {
		return filepath.Abs(value)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		if value := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); value != "" {
			return filepath.Abs(value)
		}
	}
	return filepath.Join(home, ".local", "state"), nil
}

func Load(paths Paths) (Config, error) {
	contents, err := os.ReadFile(paths.ConfigFile)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, ErrNotConfigured
	}
	if err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}
	var configuration Config
	if err := json.Unmarshal(contents, &configuration); err != nil {
		return Config{}, fmt.Errorf("decode configuration: %w", err)
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}
	return configuration, nil
}

func Save(paths Paths, configuration Config) error {
	if err := configuration.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(paths.ConfigHome, 0o700); err != nil {
		return fmt.Errorf("create config home: %w", err)
	}
	if err := os.MkdirAll(paths.StateHome, 0o700); err != nil {
		return fmt.Errorf("create state home: %w", err)
	}
	contents, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return fmt.Errorf("encode configuration: %w", err)
	}
	contents = append(contents, '\n')
	temporary, err := os.CreateTemp(paths.ConfigHome, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary configuration: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("protect temporary configuration: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary configuration: %w", err)
	}
	if err := os.Rename(temporaryName, paths.ConfigFile); err != nil {
		return fmt.Errorf("replace configuration: %w", err)
	}
	return nil
}

func (configuration Config) Validate() error {
	if configuration.Version != Version {
		return fmt.Errorf("unsupported config version %d", configuration.Version)
	}
	if len(configuration.Profiles) == 0 {
		return errors.New("at least one profile is required")
	}
	names := make(map[string]struct{}, len(configuration.Profiles))
	var repositories []ownedRepository
	for index := range configuration.Profiles {
		profile := configuration.Profiles[index]
		if err := ValidateProfile(profile); err != nil {
			return fmt.Errorf("profile %q: %w", profile.Name, err)
		}
		if _, exists := names[profile.Name]; exists {
			return fmt.Errorf("duplicate profile %q", profile.Name)
		}
		names[profile.Name] = struct{}{}
		for _, repository := range profile.Repositories {
			for _, existing := range repositories {
				if pathsOverlap(repository, existing.path) {
					return fmt.Errorf("repository %q for profile %q overlaps %q for profile %q", repository, profile.Name, existing.path, existing.profile)
				}
			}
			repositories = append(repositories, ownedRepository{path: repository, profile: profile.Name})
		}
	}
	return nil
}

func ValidateProfile(profile Profile) error {
	if err := ValidateProfileName(profile.Name); err != nil {
		return err
	}
	if _, err := time.LoadLocation(profile.Timezone); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", profile.Timezone, err)
	}
	if strings.TrimSpace(profile.Language) == "" {
		return errors.New("language is required")
	}
	if !filepath.IsAbs(profile.JournalDestination) {
		return errors.New("journal destination must be an absolute path")
	}
	if err := ValidateSummarizerCommand(profile.SummarizerCommand); err != nil {
		return err
	}
	deniedLiterals := make(map[string]struct{}, len(profile.DeniedLiterals))
	for _, literal := range profile.DeniedLiterals {
		literal = strings.TrimSpace(literal)
		if literal == "" {
			return errors.New("denied literals must not be empty")
		}
		if _, exists := deniedLiterals[literal]; exists {
			return errors.New("denied literals must be unique")
		}
		deniedLiterals[literal] = struct{}{}
	}
	authors := make(map[string]struct{}, len(profile.Authors))
	for _, author := range profile.Authors {
		if err := ValidateAuthor(author); err != nil {
			return err
		}
		key := strings.ToLower(author)
		if _, exists := authors[key]; exists {
			return fmt.Errorf("duplicate author identity %q", author)
		}
		authors[key] = struct{}{}
	}
	for _, repository := range profile.Repositories {
		if !filepath.IsAbs(repository) {
			return fmt.Errorf("repository %q must be an absolute path", repository)
		}
		if filepath.Clean(repository) != repository {
			return fmt.Errorf("repository %q must be a clean path", repository)
		}
	}
	return nil
}

func ValidateProfileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("profile name is required")
	}
	if strings.ContainsAny(name, "\\/\t\r\n") {
		return fmt.Errorf("invalid profile name %q", name)
	}
	return nil
}

func ValidateAuthor(author string) error {
	address, err := mail.ParseAddress(author)
	if err != nil || !strings.EqualFold(address.Address, author) {
		return fmt.Errorf("invalid author identity %q; use an email address", author)
	}
	return nil
}

func ValidateSummarizerCommand(command string) error {
	for _, pattern := range credentialPatterns {
		for _, match := range pattern.FindAllStringSubmatch(command, -1) {
			for _, value := range match[1:] {
				if value != "" && !isIndirectCredential(value) {
					return errors.New("summarizer command appears to contain a secret; use environment variables or credential storage")
				}
			}
		}
	}
	return nil
}

func isIndirectCredential(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "$") || strings.HasPrefix(strings.ToLower(value), "env:") {
		return true
	}
	return len(value) > 2 && strings.HasPrefix(value, "%") && strings.HasSuffix(value, "%")
}

func NewProfile(name, timezone, language, destination, summarizer string) (Profile, error) {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return Profile{}, fmt.Errorf("resolve journal destination: %w", err)
	}
	profile := Profile{
		Name:               strings.TrimSpace(name),
		Timezone:           strings.TrimSpace(timezone),
		Language:           strings.TrimSpace(language),
		JournalDestination: filepath.Clean(destination),
		SummarizerCommand:  strings.TrimSpace(summarizer),
	}
	if err := ValidateProfile(profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (configuration *Config) AddProfile(profile Profile) error {
	if err := ValidateProfile(profile); err != nil {
		return err
	}
	if _, found := configuration.Profile(profile.Name); found {
		return fmt.Errorf("profile %q already exists", profile.Name)
	}
	configuration.Profiles = append(configuration.Profiles, profile)
	sort.Slice(configuration.Profiles, func(i, j int) bool {
		return configuration.Profiles[i].Name < configuration.Profiles[j].Name
	})
	return configuration.Validate()
}

func (configuration *Config) ReplaceProfile(profile Profile) error {
	if err := ValidateProfile(profile); err != nil {
		return err
	}
	index := -1
	for candidateIndex := range configuration.Profiles {
		if configuration.Profiles[candidateIndex].Name == profile.Name {
			index = candidateIndex
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("profile %q does not exist", profile.Name)
	}
	candidate := *configuration
	candidate.Profiles = append([]Profile(nil), configuration.Profiles...)
	candidate.Profiles[index] = profile
	if err := candidate.Validate(); err != nil {
		return err
	}
	*configuration = candidate
	return nil
}

func (configuration *Config) AddAuthor(profileName, author string) error {
	if err := ValidateAuthor(author); err != nil {
		return err
	}
	profile, found := configuration.Profile(profileName)
	if !found {
		return fmt.Errorf("profile %q does not exist", profileName)
	}
	for _, existing := range profile.Authors {
		if strings.EqualFold(existing, author) {
			return fmt.Errorf("author identity %q already exists in profile %q", author, profileName)
		}
	}
	profile.Authors = append(profile.Authors, author)
	sort.Slice(profile.Authors, func(i, j int) bool {
		return strings.ToLower(profile.Authors[i]) < strings.ToLower(profile.Authors[j])
	})
	return configuration.Validate()
}

func (configuration *Config) AddRepository(profileName, repository string) error {
	profile, found := configuration.Profile(profileName)
	if !found {
		return fmt.Errorf("profile %q does not exist", profileName)
	}
	for _, candidate := range configuration.Profiles {
		for _, existing := range candidate.Repositories {
			if pathsOverlap(repository, existing) {
				return fmt.Errorf("repository %q for profile %q overlaps %q already owned by profile %q", repository, profileName, existing, candidate.Name)
			}
		}
	}
	profile.Repositories = append(profile.Repositories, repository)
	sort.Strings(profile.Repositories)
	return configuration.Validate()
}

func (configuration *Config) Profile(name string) (*Profile, bool) {
	for index := range configuration.Profiles {
		if configuration.Profiles[index].Name == name {
			return &configuration.Profiles[index], true
		}
	}
	return nil, false
}

func (configuration Config) ResolveRepository(path string) (Profile, string, error) {
	canonical, err := CanonicalPath(path)
	if err != nil {
		return Profile{}, "", err
	}
	for _, profile := range configuration.Profiles {
		for _, repository := range profile.Repositories {
			if pathContains(repository, canonical) {
				return profile, repository, nil
			}
		}
	}
	return Profile{}, "", fmt.Errorf("repository %q is not registered", canonical)
}

func CanonicalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	return filepath.Clean(canonical), nil
}

type ownedRepository struct {
	path    string
	profile string
}

func pathsOverlap(left, right string) bool {
	return pathContains(left, right) || pathContains(right, left)
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
