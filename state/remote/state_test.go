package remote

import (
	"testing"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
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

func TestState_conflictUpdate(t *testing.T) {
	s := &State{
		Client:    new(InmemClient),
		state:     state.TestStateInitial(),
		readState: state.TestStateInitial(),
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := s.State()
	state.EnsureHasLineage()

	// Writing an unrelated (different lineage) state should work
	// as long as there are no resources in the existing state.
	{
		emptyState := state.DeepCopy()
		emptyState.Modules = []*terraform.ModuleState{
			{
				Path:      []string{"root"},
				Resources: map[string]*terraform.ResourceState{},
			},
		}

		if err := s.WriteState(emptyState); err != nil {
			t.Fatalf("error %#v while writing empty state; want success", err)
		}

		unrelatedState := emptyState.DeepCopy()
		unrelatedState.Lineage = "--unrelated--"

		if err := s.WriteState(unrelatedState); err != nil {
			t.Fatalf("error %#v while writing unrelated, empty state; want success", err)
		}
	}

	// On the other hand, writing un unrelated state that *has* resources
	// *should* fail.
	{
		initialState := state.DeepCopy()
		initialState.Modules = []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": {},
				},
			},
		}

		if err := s.WriteState(initialState); err != nil {
			t.Fatalf("error %#v while writing initial state; want success", err)
		}

		unrelatedState := initialState.DeepCopy()
		unrelatedState.Lineage = "--unrelated--"

		if err := s.WriteState(unrelatedState); err == nil {
			t.Fatalf("success writing unrelated state; want error")
		}
	}
}

func TestState_impl(t *testing.T) {
	var _ state.StateReader = new(State)
	var _ state.StateWriter = new(State)
	var _ state.StatePersister = new(State)
	var _ state.StateRefresher = new(State)
}
