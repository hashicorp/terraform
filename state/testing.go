package state

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// TestState is a helper for testing state implementations. It is expected
// that the given implementation is pre-loaded with the TestStateInitial
// state.
func TestState(t *testing.T, s State) {
	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check that the initial state is correct.
	// These do have different Lineages, but we will replace current below.
	initial := TestStateInitial()
	if state := s.State(); !state.Equal(initial) {
		t.Fatalf("state does not match expected initial state:\n%#v\n\n%#v", state, initial)
	}

	// Now we've proven that the state we're starting with is an initial
	// state, we'll complete our work here with that state, since otherwise
	// further writes would violate the invariant that we only try to write
	// states that share the same lineage as what was initially written.
	current := s.State()

	// Write a new state and verify that we have it
	current.AddModuleState(&terraform.ModuleState{
		Path: []string{"root"},
		Outputs: map[string]*terraform.OutputState{
			"bar": &terraform.OutputState{
				Type:      "string",
				Sensitive: false,
				Value:     "baz",
			},
		},
	})

	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual := s.State(); !actual.Equal(current) {
		t.Fatalf("bad:\n%#v\n\n%#v", actual, current)
	}

	// Test persistence
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Refresh if we got it
	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if s.State().Lineage != current.Lineage {
		t.Fatalf("Lineage changed from %s to %s", s.State().Lineage, current.Lineage)
	}

	// Just set the serials the same... Then compare.
	actual := s.State()
	if !actual.Equal(current) {
		t.Fatalf("bad: %#v\n\n%#v", actual, current)
	}

	// Same serial
	serial := s.State().Serial
	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if s.State().Serial != serial {
		t.Fatalf("serial changed after persisting with no changes: got %d, want %d", s.State().Serial, serial)
	}

	// Change the serial
	current = current.DeepCopy()
	current.Modules = []*terraform.ModuleState{
		&terraform.ModuleState{
			Path: []string{"root", "somewhere"},
			Outputs: map[string]*terraform.OutputState{
				"serialCheck": &terraform.OutputState{
					Type:      "string",
					Sensitive: false,
					Value:     "true",
				},
			},
		},
	}
	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if s.State().Serial <= serial {
		t.Fatalf("serial incorrect after persisting with changes: got %d, want > %d", s.State().Serial, serial)
	}

	if s.State().Version != current.Version {
		t.Fatalf("Version changed from %d to %d", s.State().Version, current.Version)
	}

	if s.State().TFVersion != current.TFVersion {
		t.Fatalf("TFVersion changed from %s to %s", s.State().TFVersion, current.TFVersion)
	}

	// verify that Lineage doesn't change along with Serial, or during copying.
	if s.State().Lineage != current.Lineage {
		t.Fatalf("Lineage changed from %s to %s", s.State().Lineage, current.Lineage)
	}

	// Check that State() returns a copy by modifying the copy and comparing
	// to the current state.
	stateCopy := s.State()
	stateCopy.Serial++
	if reflect.DeepEqual(stateCopy, s.State()) {
		t.Fatal("State() should return a copy")
	}

	// our current expected state should also marhsal identically to the persisted state
	if current.MarshalEqual(s.State()) {
		t.Fatalf("Persisted state altered unexpectedly. Expected: %#v\b Got: %#v", current, s.State())
	}
}

// TestStateInitial is the initial state that a State should have
// for TestState.
func TestStateInitial() *terraform.State {
	initial := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root", "child"},
				Outputs: map[string]*terraform.OutputState{
					"foo": &terraform.OutputState{
						Type:      "string",
						Sensitive: false,
						Value:     "bar",
					},
				},
			},
		},
	}

	initial.Init()

	return initial
}
