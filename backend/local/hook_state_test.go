package local

import (
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

func TestStateHook_impl(t *testing.T) {
	var _ terraform.Hook = new(StateHook)
}

func TestStateHook(t *testing.T) {
	is := &state.InmemState{}
	var hook terraform.Hook = &StateHook{State: is}

	s := state.TestStateInitial()
	action, err := hook.PostStateUpdate(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("bad: %v", action)
	}
	if !is.State().Equal(s) {
		t.Fatalf("bad state: %#v", is.State())
	}
}

// testPersistState stores the state on WriteState, and
type testPersistState struct {
	*state.InmemState

	mu        sync.Mutex
	persisted bool
}

func (s *testPersistState) WriteState(state *terraform.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.persisted = false
	return s.InmemState.WriteState(state)
}

func (s *testPersistState) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.persisted = true
	return nil
}

// verify that StateHook calls PersistState if the last call was more than
// persistStateHookInterval
func TestStateHookPersist(t *testing.T) {
	is := &testPersistState{
		InmemState: &state.InmemState{},
	}
	hook := &StateHook{State: is}

	s := state.TestStateInitial()
	hook.PostStateUpdate(s)

	// the first call should persist, since the last time was zero
	if !is.persisted {
		t.Fatal("PersistState not called")
	}

	s.Serial++
	hook.PostStateUpdate(s)

	// this call should not have persisted
	if is.persisted {
		t.Fatal("PostStateUpdate called PersistState early")
	}

	if !is.State().Equal(s) {
		t.Fatalf("bad state: %#v", is.State())
	}

	// set the last call back to before our interval
	hook.lastPersist = time.Now().Add(-2 * persistStateHookInterval)

	s.Serial++
	hook.PostStateUpdate(s)

	if !is.persisted {
		t.Fatal("PersistState not called")
	}

	if !is.State().Equal(s) {
		t.Fatalf("bad state: %#v", is.State())
	}
}

// verify that the satet hook is safe for concurrent use
func TestStateHookRace(t *testing.T) {
	is := &state.InmemState{}
	var hook terraform.Hook = &StateHook{State: is}

	s := state.TestStateInitial()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			action, err := hook.PostStateUpdate(s)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if action != terraform.HookActionContinue {
				t.Fatalf("bad: %v", action)
			}
			if !is.State().Equal(s) {
				t.Fatalf("bad state: %#v", is.State())
			}
		}()
	}
	wg.Wait()
}
