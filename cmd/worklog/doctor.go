package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Vickyciou/worklog/internal/config"
	"github.com/Vickyciou/worklog/internal/gitnotes"
	"github.com/Vickyciou/worklog/internal/summarizer"
)

type doctorReport struct {
	writer   io.Writer
	failures int
}

func (application cli) doctor(arguments []string) error {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprint(application.stdout, doctorHelp)
		return nil
	}
	if len(arguments) != 0 {
		return errors.New("usage: worklog doctor")
	}

	report := doctorReport{writer: application.stdout}
	_, gitError := exec.LookPath("git")
	report.check("Git executable is available", gitError, "Install Git and make sure it is available on PATH.")
	report.check("Config root exists and is writable", checkWritableDirectory(application.paths.ConfigHome), "Create the worklog config root and make it writable.")
	report.check("State root exists and is writable", checkWritableDirectory(application.paths.StateHome), "Create the worklog state root and make it writable.")

	configuration, err := config.Load(application.paths)
	if err != nil {
		fix := "Repair config.json, or move it aside and run 'worklog setup'."
		if errors.Is(err, config.ErrNotConfigured) {
			fix = "Run 'worklog setup'."
		} else if strings.Contains(err.Error(), "overlaps") {
			fix = "Remove overlapping repository mappings, then run 'worklog repo validate'."
		}
		report.check("Configuration is present and valid", err, fix)
		return report.finish()
	}
	report.check("Configuration is present and valid", nil, "")
	report.check("Repository mappings are non-overlapping", nil, "")

	for _, profile := range configuration.Profiles {
		profileLabel := fmt.Sprintf("Profile %q", profile.Name)
		report.check(profileLabel+" required settings are valid", config.ValidateProfile(profile), "Run 'worklog settings "+profile.Name+"' and correct the reported values.")
		if len(profile.Authors) == 0 {
			report.check(profileLabel+" has author identities", errors.New("missing authors"), "Run 'worklog author add "+profile.Name+" <email>'.")
		} else {
			report.check(profileLabel+" has valid author identities", validateAuthors(profile), "Replace invalid entries with 'worklog author add "+profile.Name+" <email>'.")
		}
		if len(profile.Repositories) == 0 {
			report.check(profileLabel+" has registered repositories", errors.New("missing repositories"), "Run 'worklog repo add <path> --profile "+profile.Name+"'.")
		}
		for index, repository := range profile.Repositories {
			label := fmt.Sprintf("Repository %d for profile %q", index+1, profile.Name)
			report.check(label+" is the registered Git root", checkRepository(repository), "Repair the repository or its mapping, then run 'worklog repo validate'.")
			report.check(label+" supports the worklog Git notes ref", checkNotes(repository), "Repair Git repository access, then run 'worklog repo validate'.")
		}
		report.check(profileLabel+" journal destination is writable", checkWritableDirectory(profile.JournalDestination), "Create the journal folder or update it with 'worklog profile set "+profile.Name+" --journal-destination <path>'.")
		report.check(profileLabel+" daily summary executable is available", checkSummaryExecutable(profile), "Choose Built-in with 'worklog profile set "+profile.Name+" --daily-summary built-in' or install the configured CLI.")
	}
	return report.finish()
}

func (report *doctorReport) check(label string, err error, fix string) {
	if err == nil {
		fmt.Fprintf(report.writer, "[OK] %s\n", label)
		return
	}
	report.failures++
	fmt.Fprintf(report.writer, "[FAIL] %s\n", label)
	if fix != "" {
		fmt.Fprintf(report.writer, "       Fix: %s\n", fix)
	}
}

func (report doctorReport) finish() error {
	if report.failures == 0 {
		fmt.Fprintln(report.writer, "Doctor found no problems.")
		return nil
	}
	return fmt.Errorf("doctor found %d problem(s)", report.failures)
}

func checkWritableDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	probe, err := os.CreateTemp(path, ".worklog-doctor-*.tmp")
	if err != nil {
		return err
	}
	probePath := probe.Name()
	if err := probe.Close(); err != nil {
		os.Remove(probePath)
		return err
	}
	if err := os.Remove(probePath); err != nil {
		return err
	}
	return nil
}

func validateAuthors(profile config.Profile) error {
	for _, author := range profile.Authors {
		if err := config.ValidateAuthor(author); err != nil {
			return err
		}
	}
	return nil
}

func checkRepository(repository string) error {
	actual, err := gitRepositoryRoot(repository)
	if err != nil {
		return err
	}
	registered, err := config.CanonicalPath(repository)
	if err != nil {
		return err
	}
	if filepath.Clean(actual) != filepath.Clean(registered) {
		return errors.New("registered path is not the repository root")
	}
	return nil
}

func checkNotes(repository string) error {
	command := exec.Command("git", "-C", repository, "notes", "--ref="+gitnotes.Ref, "list")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}

func checkSummaryExecutable(profile config.Profile) error {
	switch profile.EffectiveSummaryMode() {
	case config.SummaryBuiltIn:
		return nil
	case config.SummaryCodex:
		_, err := exec.LookPath("codex")
		return err
	case config.SummaryClaudeCode:
		_, err := exec.LookPath("claude")
		return err
	case config.SummaryCustom:
		_, err := summarizer.Executable(profile.SummarizerCommand)
		return err
	default:
		return errors.New("unsupported daily summary mode")
	}
}
