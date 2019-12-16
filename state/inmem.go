package state

import (
	"sync"

	"github.com/hashicorp/terraform/terraform"
)

// InmemState is an in-memory state storage.
type InmemState struct {
	mu    sync.Mutex
	state *terraform.State
}

func (s *InmemState) State() *terraform.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

func (s *InmemState) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return nil
}

func (s *InmemState) WriteState(state *terraform.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state = state.DeepCopy()

	if s.state != nil {
		state.Serial = s.state.Serial

		if !s.state.MarshalEqual(state) {
			state.Serial++
		}
	}

	s.state = state

	return nil
}

func (s *InmemState) PersistState() error {
	return nil
}

func (s *InmemState) Lock(*LockInfo) (string, error) {
	return "", nil
}

func (s *InmemState) Unlock(string) error {
	return nil
}
