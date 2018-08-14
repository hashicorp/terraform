package command

import (
	"errors"
	"fmt"
	"time"

	backendLocal "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
)

// StateMeta is the meta struct that should be embedded in state subcommands.
type StateMeta struct {
	Meta
}

// State returns the state for this meta. This gets the appropriate state from
// the backend, but changes the way that backups are done. This configures
// backups to be timestamped rather than just the original state path plus a
// backup path.
func (c *StateMeta) State() (state.State, error) {
	var realState state.State
	backupPath := c.backupPath
	stateOutPath := c.statePath

	// use the specified state
	if c.statePath != "" {
		realState = statemgr.NewFilesystem(c.statePath)
	} else {
		// Load the backend
		b, backendDiags := c.Backend(nil)
		if backendDiags.HasErrors() {
			return nil, backendDiags.Err()
		}

		workspace := c.Workspace()
		// Get the state
		s, err := b.StateMgr(workspace)
		if err != nil {
			return nil, err
		}

		// Get a local backend
		localRaw, backendDiags := c.Backend(&BackendOpts{ForceLocal: true})
		if backendDiags.HasErrors() {
			// This should never fail
			panic(backendDiags.Err())
		}
		localB := localRaw.(*backendlocal.Local)
		_, stateOutPath, _ = localB.StatePaths(workspace)
		if err != nil {
			return nil, err
		}

		realState = s
	}

	// We always backup state commands, so set the back if none was specified
	// (the default is "-", but some tests bypass the flag parsing).
	if backupPath == "-" || backupPath == "" {
		// Determine the backup path. stateOutPath is set to the resulting
		// file where state is written (cached in the case of remote state)
		backupPath = fmt.Sprintf(
			"%s.%d%s",
			stateOutPath,
			time.Now().UTC().Unix(),
			DefaultBackupExtension)
	}

	// If the backend is local (which it should always be, given our asserting
	// of it above) we can now enable backups for it.
	if lb, ok := realState.(*statemgr.Filesystem); ok {
		lb.SetBackupPath(backupPath)
	}

	return realState, nil
}

// filterInstance filters a single instance out of filter results.
func (c *StateMeta) filterInstance(rs []*terraform.StateFilterResult) (*terraform.StateFilterResult, error) {
	var result *terraform.StateFilterResult
	for _, r := range rs {
		if _, ok := r.Value.(*terraform.InstanceState); !ok {
			continue
		}

		if result != nil {
			return nil, errors.New(errStateMultiple)
		}

		result = r
	}

	return result, nil
}

const errStateMultiple = `Multiple instances found for the given pattern!

This command requires that the pattern match exactly one instance
of a resource. To view the matched instances, use "terraform state list".
Please modify the pattern to match only a single instance.`
