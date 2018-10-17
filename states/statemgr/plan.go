package statemgr

import (
	"fmt"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

// PlannedStateUpdate is a special helper to obtain a statefile representation
// of a not-yet-written state snapshot that can be written later by a call
// to the companion function WritePlannedStateUpdate.
//
// The statefile object returned here has an unusual interpretation of its
// metadata that is understood only by WritePlannedStateUpdate, and so the
// returned object should not be used for any other purpose.
//
// If the state manager implements Locker then it is the caller's
// responsibility to hold the lock at least for the duration of this call.
// It is not safe to modify the given state concurrently while
// PlannedStateUpdate is running.
func PlannedStateUpdate(mgr Transient, planned *states.State) *statefile.File {
	ret := &statefile.File{
		State: planned.DeepCopy(),
	}

	// If the given manager uses snapshot metadata then we'll save that
	// in our file so we can check it again during WritePlannedStateUpdate.
	if mr, ok := mgr.(PersistentMeta); ok {
		m := mr.StateSnapshotMeta()
		ret.Lineage = m.Lineage
		ret.Serial = m.Serial
	}

	return ret
}

// WritePlannedStateUpdate is a companion to PlannedStateUpdate that attempts
// to apply a state update that was planned earlier to the given state
// manager.
//
// An error is returned if this function detects that a new state snapshot
// has been written to the backend since the update was planned, since that
// invalidates the plan. An error is returned also if the manager itself
// rejects the given state when asked to store it.
//
// If the returned error is nil, the given manager's transient state snapshot
// is updated to match what was planned. It is the caller's responsibility
// to then persist that state if the manager also implements Persistent and
// the snapshot should be written to the persistent store.
//
// If the state manager implements Locker then it is the caller's
// responsibility to hold the lock at least for the duration of this call.
func WritePlannedStateUpdate(mgr Transient, planned *statefile.File) error {
	// If the given manager uses snapshot metadata then we'll check to make
	// sure no new snapshots have been created since we planned to write
	// the given state file.
	if mr, ok := mgr.(PersistentMeta); ok {
		m := mr.StateSnapshotMeta()
		if planned.Lineage != "" {
			if planned.Lineage != m.Lineage {
				return fmt.Errorf("planned state update is from an unrelated state lineage than the current state")
			}
			if planned.Serial != m.Serial {
				return fmt.Errorf("stored state has been changed by another operation since the given update was planned")
			}
		}
	}

	return mgr.WriteState(planned.State)
}
