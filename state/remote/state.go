package remote

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// State implements the State interfaces in the state package to handle
// reading and writing the remote state. This State on its own does no
// local caching so every persist will go to the remote storage and local
// writes will go to memory.
type State struct {
	Client Client

	state, readState *terraform.State
}

// StateReader impl.
func (s *State) State() *terraform.State {
	return s.state.DeepCopy()
}

// StateWriter impl.
func (s *State) WriteState(state *terraform.State) error {
	s.state = state
	return nil
}

// StateRefresher impl.
func (s *State) RefreshState() error {
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
	s.state.IncrementSerialMaybe(s.readState)

	var buf bytes.Buffer
	if err := terraform.WriteState(s.state, &buf); err != nil {
		return err
	}

	if err := s.Client.Put(buf.Bytes()); err != nil {
		// Fallback to write the state to a local file if pushing to remote backend fails.
		var l state.State = &state.LocalState{Path: "remote.tfstate"}
		if err := l.WriteState(s.state); err == nil {
			fmt.Printf("\n" +
				"*** Local copy of the state-out file is saved to `remote.tfstate`. ***\n" +
				"*** You can upload it manually to the remote backend.              ***\n\n")
		}
		// Return previous (remote) error.
		return err
	}
	return nil
}

// Lock calls the Client's Lock method if it's implemented.
func (s *State) Lock(info *state.LockInfo) (string, error) {
	if c, ok := s.Client.(ClientLocker); ok {
		return c.Lock(info)
	}
	return "", nil
}

// Unlock calls the Client's Unlock method if it's implemented.
func (s *State) Unlock(id string) error {
	if c, ok := s.Client.(ClientLocker); ok {
		return c.Unlock(id)
	}
	return nil
}
