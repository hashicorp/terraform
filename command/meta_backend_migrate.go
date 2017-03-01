package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/backend"
	clistate "github.com/hashicorp/terraform/command/state"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

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
	// We need to check what the named state status is. If we're converting
	// from multi-state to single-state for example, we need to handle that.
	var oneSingle, twoSingle bool
	oneStates, err := opts.One.States()
	if err == backend.ErrNamedStatesNotSupported {
		oneSingle = true
		err = nil
	}
	if err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateLoadStates), opts.OneType, err)
	}

	_, err = opts.Two.States()
	if err == backend.ErrNamedStatesNotSupported {
		twoSingle = true
		err = nil
	}
	if err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateLoadStates), opts.TwoType, err)
	}

	// Determine migration behavior based on whether the source/destionation
	// supports multi-state.
	switch {
	// Single-state to single-state. This is the easiest case: we just
	// copy the default state directly.
	case oneSingle && twoSingle:
		return m.backendMigrateState_s_s(opts)

	// Single-state to multi-state. This is easy since we just copy
	// the default state and ignore the rest in the destination.
	case oneSingle && !twoSingle:
		return m.backendMigrateState_s_s(opts)

	// Multi-state to single-state. If the source has more than the default
	// state this is complicated since we have to ask the user what to do.
	case !oneSingle && twoSingle:
		// If the source only has one state and it is the default,
		// treat it as if it doesn't support multi-state.
		if len(oneStates) == 1 && oneStates[0] == backend.DefaultStateName {
			panic("YO")
			return m.backendMigrateState_s_s(opts)
		}

		panic("unhandled")

	// Multi-state to multi-state. We merge the states together (migrating
	// each from the source to the destination one by one).
	case !oneSingle && !twoSingle:
		// If the source only has one state and it is the default,
		// treat it as if it doesn't support multi-state.
		if len(oneStates) == 1 && oneStates[0] == backend.DefaultStateName {
			return m.backendMigrateState_s_s(opts)
		}

		panic("unhandled")
	}

	return nil
}

//-------------------------------------------------------------------
// State Migration Scenarios
//
// The functions below cover handling all the various scenarios that
// can exist when migrating state. They are named in an immediately not
// obvious format but is simple:
//
// Format: backendMigrateState_s1_s2[_suffix]
//
// When s1 or s2 is lower case, it means that it is a single state backend.
// When either is uppercase, it means that state is a multi-state backend.
// The suffix is used to disambiguate multiple cases with the same type of
// states.
//
//-------------------------------------------------------------------

// Single state to single state, assumed default state name.
func (m *Meta) backendMigrateState_s_s(opts *backendMigrateOpts) error {
	stateOne, err := opts.One.State(backend.DefaultStateName)
	if err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateSingleLoadDefault), opts.OneType, err)
	}
	if err := stateOne.RefreshState(); err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateSingleLoadDefault), opts.OneType, err)
	}

	stateTwo, err := opts.Two.State(backend.DefaultStateName)
	if err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateSingleLoadDefault), opts.TwoType, err)
	}
	if err := stateTwo.RefreshState(); err != nil {
		return fmt.Errorf(strings.TrimSpace(
			errMigrateSingleLoadDefault), opts.TwoType, err)
	}

	lockInfoOne := state.NewLockInfo()
	lockInfoOne.Operation = "migration"
	lockInfoOne.Info = "source state"

	lockIDOne, err := clistate.Lock(stateOne, lockInfoOne, m.Ui, m.Colorize())
	if err != nil {
		return fmt.Errorf("Error locking source state: %s", err)
	}
	defer clistate.Unlock(stateOne, lockIDOne, m.Ui, m.Colorize())

	lockInfoTwo := state.NewLockInfo()
	lockInfoTwo.Operation = "migration"
	lockInfoTwo.Info = "destination state"

	lockIDTwo, err := clistate.Lock(stateTwo, lockInfoTwo, m.Ui, m.Colorize())
	if err != nil {
		return fmt.Errorf("Error locking destination state: %s", err)
	}
	defer clistate.Unlock(stateTwo, lockIDTwo, m.Ui, m.Colorize())

	one := stateOne.State()
	two := stateTwo.State()

	var confirmFunc func(state.State, state.State, *backendMigrateOpts) (bool, error)
	switch {
	// No migration necessary
	case one.Empty() && two.Empty():
		return nil

	// No migration necessary if we're inheriting state.
	case one.Empty() && !two.Empty():
		return nil

	// We have existing state moving into no state. Ask the user if
	// they'd like to do this.
	case !one.Empty() && two.Empty():
		confirmFunc = m.backendMigrateEmptyConfirm

	// Both states are non-empty, meaning we need to determine which
	// state should be used and update accordingly.
	case !one.Empty() && !two.Empty():
		confirmFunc = m.backendMigrateNonEmptyConfirm
	}

	if confirmFunc == nil {
		panic("confirmFunc must not be nil")
	}

	// Confirm with the user whether we want to copy state over
	confirm, err := confirmFunc(stateOne, stateTwo, opts)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	// Confirmed! Write.
	if err := stateTwo.WriteState(one); err != nil {
		return fmt.Errorf(strings.TrimSpace(errBackendStateCopy),
			opts.OneType, opts.TwoType, err)
	}
	if err := stateTwo.PersistState(); err != nil {
		return fmt.Errorf(strings.TrimSpace(errBackendStateCopy),
			opts.OneType, opts.TwoType, err)
	}

	// And we're done.
	return nil
}

func (m *Meta) backendMigrateEmptyConfirm(one, two state.State, opts *backendMigrateOpts) (bool, error) {
	inputOpts := &terraform.InputOpts{
		Id: "backend-migrate-copy-to-empty",
		Query: fmt.Sprintf(
			"Do you want to copy state from %q to %q?",
			opts.OneType, opts.TwoType),
		Description: fmt.Sprintf(
			strings.TrimSpace(inputBackendMigrateEmpty),
			opts.OneType, opts.TwoType),
	}

	// Confirm with the user that the copy should occur
	for {
		v, err := m.UIInput().Input(inputOpts)
		if err != nil {
			return false, fmt.Errorf(
				"Error asking for state copy action: %s", err)
		}

		switch strings.ToLower(v) {
		case "no":
			return false, nil

		case "yes":
			return true, nil
		}
	}
}

func (m *Meta) backendMigrateNonEmptyConfirm(
	stateOne, stateTwo state.State, opts *backendMigrateOpts) (bool, error) {
	// We need to grab both states so we can write them to a file
	one := stateOne.State()
	two := stateTwo.State()

	// Save both to a temporary
	td, err := ioutil.TempDir("", "terraform")
	if err != nil {
		return false, fmt.Errorf("Error creating temporary directory: %s", err)
	}
	defer os.RemoveAll(td)

	// Helper to write the state
	saveHelper := func(n, path string, s *terraform.State) error {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return terraform.WriteState(s, f)
	}

	// Write the states
	onePath := filepath.Join(td, fmt.Sprintf("1-%s.tfstate", opts.OneType))
	twoPath := filepath.Join(td, fmt.Sprintf("2-%s.tfstate", opts.TwoType))
	if err := saveHelper(opts.OneType, onePath, one); err != nil {
		return false, fmt.Errorf("Error saving temporary state: %s", err)
	}
	if err := saveHelper(opts.TwoType, twoPath, two); err != nil {
		return false, fmt.Errorf("Error saving temporary state: %s", err)
	}

	// Ask for confirmation
	inputOpts := &terraform.InputOpts{
		Id: "backend-migrate-to-backend",
		Query: fmt.Sprintf(
			"Do you want to copy state from %q to %q?",
			opts.OneType, opts.TwoType),
		Description: fmt.Sprintf(
			strings.TrimSpace(inputBackendMigrateNonEmpty),
			opts.OneType, opts.TwoType, onePath, twoPath),
	}

	// Confirm with the user that the copy should occur
	for {
		v, err := m.UIInput().Input(inputOpts)
		if err != nil {
			return false, fmt.Errorf(
				"Error asking for state copy action: %s", err)
		}

		switch strings.ToLower(v) {
		case "no":
			return false, nil

		case "yes":
			return true, nil
		}
	}
}

type backendMigrateOpts struct {
	OneType, TwoType string
	One, Two         backend.Backend
}

const errMigrateLoadStates = `
Error inspecting state in %q: %s

Prior to changing backends, Terraform inspects the source and destionation
states to determine what kind of migration steps need to be taken, if any.
Terraform failed to load the states. The data in both the source and the
destination remain unmodified. Please resolve the above error and try again.
`

const errMigrateSingleLoadDefault = `
Error loading state from %q: %s

Terraform failed to load the default state from %[1]q.
State migration cannot occur unless the state can be loaded. Backend
modification and state migration has been aborted. The state in both the
source and the destination remain unmodified. Please resolve the
above error and try again.
`

const errBackendStateCopy = `
Error copying state from %q to %q: %s

The state in %[1]q remains intact and unmodified. Please resolve the
error above and try again.
`

const inputBackendMigrateEmpty = `
Pre-existing state was found in %q while migrating to %q. No existing
state was found in %[2]q. Do you want to copy the state from %[1]q to
%[2]q? Enter "yes" to copy and "no" to start with an empty state.
`

const inputBackendMigrateNonEmpty = `
Pre-existing state was found in %q while migrating to %q. An existing
non-empty state exists in %[2]q. The two states have been saved to temporary
files that will be removed after responding to this query.

One (%[1]q): %[3]s
Two (%[2]q): %[4]s

Do you want to copy the state from %[1]q to %[2]q? Enter "yes" to copy
and "no" to start with the existing state in %[2]q.
`
