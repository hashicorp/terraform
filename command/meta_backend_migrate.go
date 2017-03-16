package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
func (m *Meta) backendMigrateState(opts *backendMigrateOpts) error {
	one := opts.One.State()
	two := opts.Two.State()

	var confirmFunc func(opts *backendMigrateOpts) (bool, error)
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
	confirm, err := confirmFunc(opts)
	if err != nil {
		return err
	}
	if !confirm {
		return nil
	}

	// Confirmed! Write.
	if err := opts.Two.WriteState(one); err != nil {
		return fmt.Errorf(strings.TrimSpace(errBackendStateCopy),
			opts.OneType, opts.TwoType, err)
	}
	if err := opts.Two.PersistState(); err != nil {
		return fmt.Errorf(strings.TrimSpace(errBackendStateCopy),
			opts.OneType, opts.TwoType, err)
	}

	// And we're done.
	return nil
}

func (m *Meta) backendMigrateEmptyConfirm(opts *backendMigrateOpts) (bool, error) {
	inputOpts := &terraform.InputOpts{
		Id: "backend-migrate-to-backend",
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

func (m *Meta) backendMigrateNonEmptyConfirm(opts *backendMigrateOpts) (bool, error) {
	// We need to grab both states so we can write them to a file
	one := opts.One.State()
	two := opts.Two.State()

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
	One, Two         state.State
}

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
