package command

import (
	"errors"
	"fmt"
	"time"

	backendlocal "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// StateMeta is the meta struct that should be embedded in state subcommands.
type StateMeta struct{}

// State returns the state for this meta. This gets the appropriate state from
// the backend, but changes the way that backups are done. This configures
// backups to be timestamped rather than just the original state path plus a
// backup path.
func (c *StateMeta) State(m *Meta) (state.State, error) {
	// Load the backend
	b, err := m.Backend(nil)
	if err != nil {
		return nil, err
	}

	env := m.Env()
	// Get the state
	s, err := b.State(env)
	if err != nil {
		return nil, err
	}

	// Get a local backend
	localRaw, err := m.Backend(&BackendOpts{ForceLocal: true})
	if err != nil {
		// This should never fail
		panic(err)
	}
	localB := localRaw.(*backendlocal.Local)
	_, stateOutPath, _ := localB.StatePaths(env)
	if err != nil {
		return nil, err
	}

	// Determine the backup path. stateOutPath is set to the resulting
	// file where state is written (cached in the case of remote state)
	backupPath := fmt.Sprintf(
		"%s.%d%s",
		stateOutPath,
		time.Now().UTC().Unix(),
		DefaultBackupExtension)

	// Wrap it for backups
	s = &state.BackupState{
		Real: s,
		Path: backupPath,
	}

	return s, nil
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
