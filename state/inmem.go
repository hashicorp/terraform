package state

import (
	"github.com/hashicorp/terraform/terraform"
)

// InmemState is an in-memory state storage.
type InmemState struct {
	state *terraform.State
}

func (s *InmemState) State() *terraform.State {
	return s.state
}

func (s *InmemState) RefreshState() error {
	return nil
}

func (s *InmemState) WriteState(state *terraform.State) error {
	s.state = state
	return nil
}

func (s *InmemState) PersistState() error {
	return nil
}
