package command

import (
	"fmt"

	"github.com/hashicorp/terraform/backend"
)

type backendMigrateOpts struct {
	OneType, TwoType string
	One, Two         backend.Backend

	// Fields below are set internally when migrate is called

	oneEnv string // source env
	twoEnv string // dest env
	force  bool   // if true, won't ask for confirmation
}

// backendMigrateState handles migrating (copying) state from one backend
// to another. This function handles asking the user for confirmation
// as well as the copy itself.
//
// This function can handle all scenarios of state migration regardless
// of the existence of state in either backend.
//
// After migrating the state, the existing state in the first backend
// remains untouched.
//
// This will attempt to lock both states for the migration.
func (m *Meta) backendMigrateState(opts *backendMigrateOpts) error {
	return fmt.Errorf("state migration is not supported in workspaces2 prototype")
}
