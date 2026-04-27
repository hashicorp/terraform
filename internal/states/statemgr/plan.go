// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
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
// stateContentHash computes a SHA-256 hash of the serialized state content,
// using a version-neutral serialization to ensure the hash is stable across
// Terraform upgrades between plan and apply operations.
func stateContentHash(s *states.State) string {
	if s == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := statefile.WriteForTest(&statefile.File{State: s}, &buf); err != nil {
		return ""
	}
	h := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(h[:])
}

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

	// Compute a cryptographic hash of the prior state content so that
	// WritePlannedStateUpdate can detect backend state tampering even when
	// an attacker preserves the original lineage UUID and serial number.
	ret.ContentHash = stateContentHash(planned)

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

	// Verify the cryptographic content hash to detect backend state tampering
	// even when an attacker has preserved the original lineage UUID and serial.
	if planned.ContentHash != "" {
		currentHash := stateContentHash(mgr.State())
		if currentHash != "" && planned.ContentHash != currentHash {
			return fmt.Errorf("state file integrity check failed: backend state content has been modified since the plan was created")
		}
	}

	return mgr.WriteState(planned.State)
}
