package remote

import (
	"bytes"
	"fmt"
	"sync"

	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
)

// State implements the State interfaces in the state package to handle
// reading and writing the remote state. This State on its own does no
// local caching so every persist will go to the remote storage and local
// writes will go to memory.
type State struct {
	mu sync.Mutex

	Client Client

	lineage          string
	serial           uint64
	state, readState *states.State
	disableLocks     bool
}

var _ statemgr.Full = (*State)(nil)

// statemgr.Reader impl.
func (s *State) State() *states.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

// statemgr.Writer impl.
func (s *State) WriteState(state *states.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = state.DeepCopy()

	return nil
}

// statemgr.Refresher impl.
func (s *State) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.Client.Get()
	if err != nil {
		return err
	}

	// no remote state is OK
	if payload == nil {
		s.readState = nil
		s.state = nil
		s.lineage = ""
		s.serial = 0
		return nil
	}

	stateFile, err := statefile.Read(bytes.NewReader(payload.Data))
	if err != nil {
		return err
	}

	s.lineage = stateFile.Lineage
	s.serial = stateFile.Serial
	s.state = stateFile.State
	s.readState = s.state.DeepCopy() // our states must be separate instances so we can track changes
	return nil
}

// statemgr.Persister impl.
func (s *State) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.readState != nil {
		if !statefile.StatesMarshalEqual(s.state, s.readState) {
			s.serial++
		}
	} else {
		// We might be writing a new state altogether, but before we do that
		// we'll check to make sure there isn't already a snapshot present
		// that we ought to be updating.
		err := s.RefreshState()
		if err != nil {
			return fmt.Errorf("failed checking for existing remote state: %s", err)
		}
		if s.lineage == "" { // indicates that no state snapshot is present yet
			lineage, err := uuid.GenerateUUID()
			if err != nil {
				return fmt.Errorf("failed to generate initial lineage: %v", err)
			}
			s.lineage = lineage
			s.serial = 0
		}
	}

	f := statefile.New(s.state, s.lineage, s.serial)

	var buf bytes.Buffer
	err := statefile.Write(f, &buf)
	if err != nil {
		return err
	}

	err = s.Client.Put(buf.Bytes())
	if err != nil {
		return err
	}

	// After we've successfully persisted, what we just wrote is our new
	// reference state until someone calls RefreshState again.
	s.readState = s.state.DeepCopy()
	return nil
}

// Lock calls the Client's Lock method if it's implemented.
func (s *State) Lock(info *state.LockInfo) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disableLocks {
		return "", nil
	}

	if c, ok := s.Client.(ClientLocker); ok {
		return c.Lock(info)
	}
	return "", nil
}

// Unlock calls the Client's Unlock method if it's implemented.
func (s *State) Unlock(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disableLocks {
		return nil
	}

	if c, ok := s.Client.(ClientLocker); ok {
		return c.Unlock(id)
	}
	return nil
}

// DisableLocks turns the Lock and Unlock methods into no-ops. This is intended
// to be called during initialization of a state manager and should not be
// called after any of the statemgr.Full interface methods have been called.
func (s *State) DisableLocks() {
	s.disableLocks = true
}

// StateSnapshotMeta returns the metadata from the most recently persisted
// or refreshed persistent state snapshot.
//
// This is an implementation of statemgr.PersistentMeta.
func (s *State) StateSnapshotMeta() statemgr.SnapshotMeta {
	return statemgr.SnapshotMeta{
		Lineage: s.lineage,
		Serial:  s.serial,
	}
}
