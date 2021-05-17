// Package state exposes common helpers for working with state from the CLI.
//
// This is a separate package so that backends can use this for consistent
// messaging without creating a circular reference to the command package.
package clistate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/helper/slowmessage"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	LockThreshold    = 400 * time.Millisecond
	LockErrorMessage = `Error message: %s

Terraform acquires a state lock to protect the state from being written
by multiple users at the same time. Please resolve the issue above and try
again. For most commands, you can disable locking with the "-lock=false"
flag, but this is not recommended.`

	UnlockErrorMessage = `Error message: %s

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
that no one else is holding a lock.`
)

// Locker allows for more convenient usage of the lower-level statemgr.Locker
// implementations.
// The statemgr.Locker API requires passing in a statemgr.LockInfo struct. Locker
// implementations are expected to create the required LockInfo struct when
// Lock is called, populate the Operation field with the "reason" string
// provided, and pass that on to the underlying statemgr.Locker.
// Locker implementations are also expected to store any state required to call
// Unlock, which is at a minimum the LockID string returned by the
// statemgr.Locker.
type Locker interface {
	// Returns a shallow copy of the locker with its context changed to ctx.
	WithContext(ctx context.Context) Locker

	// Lock the provided state manager, storing the reason string in the LockInfo.
	Lock(s statemgr.Locker, reason string) tfdiags.Diagnostics

	// Unlock the previously locked state.
	Unlock() tfdiags.Diagnostics

	// Timeout returns the configured timeout duration
	Timeout() time.Duration
}

type locker struct {
	mu      sync.Mutex
	ctx     context.Context
	timeout time.Duration
	state   statemgr.Locker
	view    views.StateLocker
	lockID  string
}

var _ Locker = (*locker)(nil)

// Create a new Locker.
// This Locker uses state.LockWithContext to retry the lock until the provided
// timeout is reached, or the context is canceled. Lock progress will be be
// reported to the user through the provided UI.
func NewLocker(timeout time.Duration, view views.StateLocker) Locker {
	return &locker{
		ctx:     context.Background(),
		timeout: timeout,
		view:    view,
	}
}

// WithContext returns a new Locker with the specified context, copying the
// timeout and view parameters from the original Locker.
func (l *locker) WithContext(ctx context.Context) Locker {
	if ctx == nil {
		panic("nil context")
	}
	return &locker{
		ctx:     ctx,
		timeout: l.timeout,
		view:    l.view,
	}
}

// Locker locks the given state and outputs to the user if locking is taking
// longer than the threshold. The lock is retried until the context is
// cancelled.
func (l *locker) Lock(s statemgr.Locker, reason string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	l.mu.Lock()
	defer l.mu.Unlock()

	l.state = s

	ctx, cancel := context.WithTimeout(l.ctx, l.timeout)
	defer cancel()

	lockInfo := statemgr.NewLockInfo()
	lockInfo.Operation = reason

	err := slowmessage.Do(LockThreshold, func() error {
		id, err := statemgr.LockWithContext(ctx, s, lockInfo)
		l.lockID = id
		return err
	}, l.view.Locking)

	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error acquiring the state lock",
			fmt.Sprintf(LockErrorMessage, err),
		))
	}

	return diags
}

func (l *locker) Unlock() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lockID == "" {
		return diags
	}

	err := slowmessage.Do(LockThreshold, func() error {
		return l.state.Unlock(l.lockID)
	}, l.view.Unlocking)

	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error releasing the state lock",
			fmt.Sprintf(UnlockErrorMessage, err),
		))
	}

	return diags

}

func (l *locker) Timeout() time.Duration {
	return l.timeout
}

type noopLocker struct{}

// NewNoopLocker returns a valid Locker that does nothing.
func NewNoopLocker() Locker {
	return noopLocker{}
}

var _ Locker = noopLocker{}

func (l noopLocker) WithContext(ctx context.Context) Locker {
	return l
}

func (l noopLocker) Lock(statemgr.Locker, string) tfdiags.Diagnostics {
	return nil
}

func (l noopLocker) Unlock() tfdiags.Diagnostics {
	return nil
}

func (l noopLocker) Timeout() time.Duration {
	return 0
}
