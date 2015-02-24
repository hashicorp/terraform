package remote

import (
	"testing"

	"github.com/hashicorp/terraform/state"
)

func TestState(t *testing.T) {
	s := &State{
		Client:    new(InmemClient),
		state:     state.TestStateInitial(),
		readState: state.TestStateInitial(),
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state.TestState(t, s)
}

func TestState_impl(t *testing.T) {
	var _ state.StateReader = new(State)
	var _ state.StateWriter = new(State)
	var _ state.StatePersister = new(State)
	var _ state.StateRefresher = new(State)
}
