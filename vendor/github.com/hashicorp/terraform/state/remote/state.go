package remote

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// State implements the State interfaces in the state package to handle
// reading and writing the remote state. This State on its own does no
// local caching so every persist will go to the remote storage and local
// writes will go to memory.
type State struct {
	mu sync.Mutex

	Client Client

	state, readState *terraform.State
}

// StateReader impl.
func (s *State) State() *terraform.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

// StateWriter impl.
func (s *State) WriteState(state *terraform.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.readState != nil && !state.SameLineage(s.readState) {
		return fmt.Errorf("incompatible state lineage; given %s but want %s", state.Lineage, s.readState.Lineage)
	}

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = state.DeepCopy()

	// Force our new state to have the same serial as our read state. We'll
	// update this if PersistState is called later. (We don't require nor trust
	// the caller to properly maintain serial for transient state objects since
	// the rest of Terraform treats state as an openly mutable object.)
	//
	// If we have no read state then we assume we're either writing a new
	// state for the first time or we're migrating a state from elsewhere,
	// and in both cases we wish to retain the lineage and serial from
	// the given state.
	if s.readState != nil {
		s.state.Serial = s.readState.Serial
	}

	return nil
}

// StateRefresher impl.
func (s *State) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.Client.Get()
	if err != nil {
		return err
	}

	// no remote state is OK
	if payload == nil {
		return nil
	}

	state, err := terraform.ReadState(bytes.NewReader(payload.Data))
	if err != nil {
		return err
	}

	s.state = state
	s.readState = s.state.DeepCopy() // our states must be separate instances so we can track changes
	return nil
}

// StatePersister impl.
func (s *State) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.state.MarshalEqual(s.readState) {
		// Our new state does not marshal as byte-for-byte identical to
		// the old, so we need to increment the serial.
		// Note that in WriteState we force the serial to match that of
		// s.readState, if we have a readState.
		s.state.Serial++
	}

	var buf bytes.Buffer
	if err := terraform.WriteState(s.state, &buf); err != nil {
		return err
	}

	err := s.Client.Put(buf.Bytes())
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

	if c, ok := s.Client.(ClientLocker); ok {
		return c.Lock(info)
	}
	return "", nil
}

// Unlock calls the Client's Unlock method if it's implemented.
func (s *State) Unlock(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.Client.(ClientLocker); ok {
		return c.Unlock(id)
	}
	return nil
}
