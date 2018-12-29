// Package state exposes common helpers for working with state from the CLI.
//
// This is a separate package so that backends can use this for consistent
// messaging without creating a circular reference to the command package.
package clistate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/slowmessage"
	"github.com/hashicorp/terraform/state"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

const (
	LockThreshold    = 400 * time.Millisecond
	LockMessage      = "Acquiring state lock. This may take a few moments..."
	LockErrorMessage = `Error acquiring the state lock: {{err}}

Terraform acquires a state lock to protect the state from being written
by multiple users at the same time. Please resolve the issue above and try
again. For most commands, you can disable locking with the "-lock=false"
flag, but this is not recommended.`

	UnlockMessage      = "Releasing state lock. This may take a few moments..."
	UnlockErrorMessage = `
[reset][bold][red]Error releasing the state lock![reset][red]

Error message: %s

Terraform acquires a lock when accessing your state to prevent others
running Terraform to potentially modify the state at the same time. An
error occurred while releasing this lock. This could mean that the lock
did or did not release properly. If the lock didn't release properly,
Terraform may not be able to run future commands since it'll appear as if
the lock is held.

In this scenario, please call the "force-unlock" command to unlock the
state manually. This is a very dangerous operation since if it is done
erroneously it could result in two people modifying state at the same time.
Only call this command if you're certain that the unlock above failed and
that no one else is holding a lock.
`
)

// Lock locks the given state and outputs to the user if locking
// is taking longer than the threshold.  The lock is retried until the context
// is cancelled.
func Lock(ctx context.Context, s state.State, info *state.LockInfo, ui cli.Ui, color *colorstring.Colorize) (string, error) {
	var lockID string

	err := slowmessage.Do(LockThreshold, func() error {
		id, err := state.LockWithContext(ctx, s, info)
		lockID = id
		return err
	}, func() {
		if ui != nil {
			ui.Output(color.Color(LockMessage))
		}
	})

	if err != nil {
		err = errwrap.Wrapf(strings.TrimSpace(LockErrorMessage), err)
	}

	return lockID, err
}

// Unlock unlocks the given state and outputs to the user if the
// unlock fails what can be done.
func Unlock(s state.State, id string, ui cli.Ui, color *colorstring.Colorize) error {
	err := slowmessage.Do(LockThreshold, func() error {
		return s.Unlock(id)
	}, func() {
		if ui != nil {
			ui.Output(color.Color(UnlockMessage))
		}
	})

	if err != nil {
		ui.Output(color.Color(fmt.Sprintf(
			"\n"+strings.TrimSpace(UnlockErrorMessage)+"\n", err)))

		err = fmt.Errorf(
			"Error releasing the state lock. Please see the longer error message above.")
	}

	return err
}
