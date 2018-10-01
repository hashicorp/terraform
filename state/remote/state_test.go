package remote

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform/state"
)

func TestState_impl(t *testing.T) {
	var _ state.StateReader = new(State)
	var _ state.StateWriter = new(State)
	var _ state.StatePersister = new(State)
	var _ state.StateRefresher = new(State)
	var _ state.Locker = new(State)
}

func TestStateRace(t *testing.T) {
	t.Fatal("FIXME: this test is either hanging or getting into an infinite loop")
	s := &State{
		Client: nilClient{},
	}

	current := state.TestStateInitial()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.WriteState(current)
			s.PersistState()
			s.RefreshState()
		}()
	}
	wg.Wait()
}
