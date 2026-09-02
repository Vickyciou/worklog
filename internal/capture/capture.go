package capture

import (
	"fmt"
	"io"

	"github.com/Vickyciou/worklog/internal/gitnotes"
	"github.com/Vickyciou/worklog/internal/worklogevent"
)

type Result struct {
	Commit  string
	Created bool
}

func Run(repository string, revision string, input io.Reader) (Result, error) {
	_, canonical, err := worklogevent.Decode(input)
	if err != nil {
		return Result{}, err
	}

	commit, err := gitnotes.ResolveCommit(repository, revision)
	if err != nil {
		return Result{}, err
	}

	writeResult, err := gitnotes.Add(repository, commit, canonical, worklogevent.Equivalent)
	if err != nil {
		return Result{}, fmt.Errorf("capture event for commit %s: %w", commit, err)
	}

	return Result{Commit: commit, Created: writeResult == gitnotes.Created}, nil
}
