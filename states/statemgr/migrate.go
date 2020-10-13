package statemgr

import (
	"fmt"

	"github.com/hashicorp/terraform/states/statefile"
)

// Migrator is an optional interface implemented by state managers that
// are capable of direct migration of state snapshots with their associated
// metadata unchanged.
//
// This interface is used when available by function Migrate. See that
// function for more information on how it is used.
type Migrator interface {
	PersistentMeta

	// StateForMigration returns a full statefile representing the latest
	// snapshot (as would be returned by Reader.State) and the associated
	// snapshot metadata (as would be returned by
	// PersistentMeta.StateSnapshotMeta).
	//
	// Just as with Reader.State, this must not fail.
	StateForMigration() *statefile.File

	// WriteStateForMigration accepts a full statefile including associated
	// snapshot metadata, and atomically updates the stored file (as with
	// Writer.WriteState) and the metadata.
	//
	// If "force" is not set, the manager must call CheckValidImport with
	// the given file and the current file and complete the update only if
	// that function returns nil. If force is set this may override such
	// checks, but some backends do not support forcing and so will act
	// as if force is always false.
	WriteStateForMigration(f *statefile.File, force bool) error
}

// Migrate writes the latest transient state snapshot from src into dest,
// preserving snapshot metadata (serial and lineage) where possible.
//
// If both managers implement the optional interface Migrator then it will
// be used to copy the snapshot and its associated metadata. Otherwise,
// the normal Reader and Writer interfaces will be used instead.
//
// If the destination manager refuses the new state or fails to write it then
// its error is returned directly.
//
// For state managers that also implement Persistent, it is the caller's
// responsibility to persist the newly-written state after a successful result,
// just as with calls to Writer.WriteState.
//
// This function doesn't do any locking of its own, so if the state managers
// also implement Locker the caller should hold a lock on both managers
// for the duration of this call.
func Migrate(dst, src Transient) error {
	if dstM, ok := dst.(Migrator); ok {
		if srcM, ok := src.(Migrator); ok {
			// Full-fidelity migration, them.
			s := srcM.StateForMigration()
			return dstM.WriteStateForMigration(s, true)
		}
	}

	// Managers to not support full-fidelity migration, so migration will not
	// preserve serial/lineage.
	s := src.State()
	return dst.WriteState(s)
}

// Import loads the given state snapshot into the given manager, preserving
// its metadata (serial and lineage) if the target manager supports metadata.
//
// A state manager must implement the optional interface Migrator to get
// access to the full metadata.
//
// Unless "force" is true, Import will check first that the metadata given
// in the file matches the current snapshot metadata for the manager, if the
// manager supports metadata. Some managers do not support forcing, so a
// write with an unsuitable lineage or serial may still be rejected even if
// "force" is set. "force" has no effect for managers that do not support
// snapshot metadata.
//
// For state managers that also implement Persistent, it is the caller's
// responsibility to persist the newly-written state after a successful result,
// just as with calls to Writer.WriteState.
//
// This function doesn't do any locking of its own, so if the state manager
// also implements Locker the caller should hold a lock on it for the
// duration of this call.
func Import(f *statefile.File, mgr Transient, force bool) error {
	if mgrM, ok := mgr.(Migrator); ok {
		return mgrM.WriteStateForMigration(f, force)
	}

	// For managers that don't implement Migrator, this is just a normal write
	// of the state contained in the given file.
	return mgr.WriteState(f.State)
}

// Export retrieves the latest state snapshot from the given manager, including
// its metadata (serial and lineage) where possible.
//
// A state manager must also implement either Migrator or PersistentMeta
// for the metadata to be included. Otherwise, the relevant fields will have
// zero value in the returned object.
//
// For state managers that also implement Persistent, it is the caller's
// responsibility to refresh from persistent storage first if needed.
//
// This function doesn't do any locking of its own, so if the state manager
// also implements Locker the caller should hold a lock on it for the
// duration of this call.
func Export(mgr Reader) *statefile.File {
	switch mgrT := mgr.(type) {
	case Migrator:
		return mgrT.StateForMigration()
	case PersistentMeta:
		s := mgr.State()
		meta := mgrT.StateSnapshotMeta()
		return statefile.New(s, meta.Lineage, meta.Serial)
	default:
		s := mgr.State()
		return statefile.New(s, "", 0)
	}
}

// SnapshotMetaRel describes a relationship between two SnapshotMeta values,
// returned from the SnapshotMeta.Compare method where the "first" value
// is the receiver of that method and the "second" is the given argument.
type SnapshotMetaRel rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=SnapshotMetaRel

const (
	// SnapshotOlder indicates that two snapshots have a common lineage and
	// that the first has a lower serial value.
	SnapshotOlder SnapshotMetaRel = '<'

	// SnapshotNewer indicates that two snapshots have a common lineage and
	// that the first has a higher serial value.
	SnapshotNewer SnapshotMetaRel = '>'

	// SnapshotEqual indicates that two snapshots have a common lineage and
	// the same serial value.
	SnapshotEqual SnapshotMetaRel = '='

	// SnapshotUnrelated indicates that two snapshots have different lineage
	// and thus cannot be meaningfully compared.
	SnapshotUnrelated SnapshotMetaRel = '!'

	// SnapshotLegacy indicates that one or both of the snapshots
	// does not have a lineage at all, and thus no comparison is possible.
	SnapshotLegacy SnapshotMetaRel = '?'
)

// Compare determines the relationship, if any, between the given existing
// SnapshotMeta and the potential "new" SnapshotMeta that is the receiver.
func (m SnapshotMeta) Compare(existing SnapshotMeta) SnapshotMetaRel {
	switch {
	case m.Lineage == "" || existing.Lineage == "":
		return SnapshotLegacy
	case m.Lineage != existing.Lineage:
		return SnapshotUnrelated
	case m.Serial > existing.Serial:
		return SnapshotNewer
	case m.Serial < existing.Serial:
		return SnapshotOlder
	default:
		// both serials are equal, by elimination
		return SnapshotEqual
	}
}

// CheckValidImport returns nil if the "new" snapshot can be imported as a
// successor of the "existing" snapshot without forcing.
//
// If not, an error is returned describing why.
func CheckValidImport(newFile, existingFile *statefile.File) error {
	if existingFile == nil || existingFile.State.Empty() {
		// It's always okay to overwrite an empty state, regardless of
		// its lineage/serial.
		return nil
	}
	new := SnapshotMeta{
		Lineage: newFile.Lineage,
		Serial:  newFile.Serial,
	}
	existing := SnapshotMeta{
		Lineage: existingFile.Lineage,
		Serial:  existingFile.Serial,
	}
	rel := new.Compare(existing)
	switch rel {
	case SnapshotNewer:
		return nil // a newer snapshot is fine
	case SnapshotLegacy:
		return nil // anything goes for a legacy state
	case SnapshotUnrelated:
		return fmt.Errorf("cannot import state with lineage %q over unrelated state with lineage %q", new.Lineage, existing.Lineage)
	case SnapshotEqual:
		if statefile.StatesMarshalEqual(newFile.State, existingFile.State) {
			// If lineage, serial, and state all match then this is fine.
			return nil
		}
		return fmt.Errorf("cannot overwrite existing state with serial %d with a different state that has the same serial", new.Serial)
	case SnapshotOlder:
		return fmt.Errorf("cannot import state with serial %d over newer state with serial %d", new.Serial, existing.Serial)
	default:
		// Should never happen, but we'll check to make sure for safety
		return fmt.Errorf("unsupported state snapshot relationship %s", rel)
	}
}
