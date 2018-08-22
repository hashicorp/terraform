// Package state exposes common helpers for working with state from the CLI.
//
// This is a separate package so that backends can use this for consistent
// messaging without creating a circular reference to the command package.
package clistate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/errwrap"
	multierror "github.com/hashicorp/go-multierror"
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

// Locker allows for more convenient usage of the lower-level state.Locker
// implementations.
// The state.Locker API requires passing in a state.LockInfo struct. Locker
// implementations are expected to create the required LockInfo struct when
// Lock is called, populate the Operation field with the "reason" string
// provided, and pass that on to the underlying state.Locker.
// Locker implementations are also expected to store any state required to call
// Unlock, which is at a minimum the LockID string returned by the
// state.Locker.
type Locker interface {
	// Lock the provided state, storing the reason string in the LockInfo.
	Lock(s state.State, reason string) error
	// Unlock the previously locked state.
	// An optional error can be passed in, and will be combined with any error
	// from the Unlock operation.
	Unlock(error) error
}

type locker struct {
	mu      sync.Mutex
	ctx     context.Context
	timeout time.Duration
	state   state.State
	ui      cli.Ui
	color   *colorstring.Colorize
	lockID  string
}

// Create a new Locker.
// This Locker uses state.LockWithContext to retry the lock until the provided
// timeout is reached, or the context is canceled. Lock progress will be be
// reported to the user through the provided UI.
func NewLocker(
	ctx context.Context,
	timeout time.Duration,
	ui cli.Ui,
	color *colorstring.Colorize) Locker {

	l := &locker{
		ctx:     ctx,
		timeout: timeout,
		ui:      ui,
		color:   color,
	}
	return l
}

// Locker locks the given state and outputs to the user if locking is taking
// longer than the threshold. The lock is retried until the context is
// cancelled.
func (l *locker) Lock(s state.State, reason string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.state = s

	ctx, cancel := context.WithTimeout(l.ctx, l.timeout)
	defer cancel()

	lockInfo := state.NewLockInfo()
	lockInfo.Operation = reason

	err := slowmessage.Do(LockThreshold, func() error {
		id, err := state.LockWithContext(ctx, s, lockInfo)
		l.lockID = id
		return err
	}, func() {
		if l.ui != nil {
			l.ui.Output(l.color.Color(LockMessage))
		}
	})

	if err != nil {
		return errwrap.Wrapf(strings.TrimSpace(LockErrorMessage), err)
	}

	return nil
}

func (l *locker) Unlock(parentErr error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lockID == "" {
		return parentErr
	}

	err := slowmessage.Do(LockThreshold, func() error {
		return l.state.Unlock(l.lockID)
	}, func() {
		if l.ui != nil {
			l.ui.Output(l.color.Color(UnlockMessage))
		}
	})

	if err != nil {
		l.ui.Output(l.color.Color(fmt.Sprintf(
			"\n"+strings.TrimSpace(UnlockErrorMessage)+"\n", err)))

		if parentErr != nil {
			parentErr = multierror.Append(parentErr, err)
		}
	}

	return parentErr

}

type noopLocker struct{}

// NewNoopLocker returns a valid Locker that does nothing.
func NewNoopLocker() Locker {
	return noopLocker{}
}

func (l noopLocker) Lock(state.State, string) error {
	return nil
}

func (l noopLocker) Unlock(err error) error {
	return err
}
