package remote

import (
	"bytes"
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

	s.state = state
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
	s.readState = state
	return nil
}

// StatePersister impl.
func (s *State) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state.IncrementSerialMaybe(s.readState)

	var buf bytes.Buffer
	if err := terraform.WriteState(s.state, &buf); err != nil {
		return err
	}

	return s.Client.Put(buf.Bytes())
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
