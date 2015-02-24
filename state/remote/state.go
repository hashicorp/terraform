package remote

import (
	"bytes"

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

	var state *terraform.State
	if payload != nil {
		state, err = terraform.ReadState(bytes.NewReader(payload.Data))
		if err != nil {
			return err
		}
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

	return s.Client.Put(buf.Bytes())
}
