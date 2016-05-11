package command

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// StateMeta is the meta struct that should be embedded in state subcommands.
type StateMeta struct{}

// State returns the state for this meta. This is different then Meta.State
// in the way that backups are done. This configures backups to be timestamped
// rather than just the original state path plus a backup path.
func (c *StateMeta) State(m *Meta) (state.State, error) {
	// Disable backups since we wrap it manually below
	m.backupPath = "-"

	// Get the state (shouldn't be wrapped in a backup)
	s, err := m.State()
	if err != nil {
		return nil, err
	}

	// Determine the backup path. stateOutPath is set to the resulting
	// file where state is written (cached in the case of remote state)
	backupPath := fmt.Sprintf(
		"%s.%d.%s",
		m.stateOutPath,
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
