package state

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// TestState is a helper for testing state implementations. It is expected
// that the given implementation is pre-loaded with the TestStateInitial
// state.
func TestState(t *testing.T, s interface{}) {
	reader, ok := s.(StateReader)
	if !ok {
		t.Fatalf("must at least be a StateReader")
	}

	// If it implements refresh, refresh
	if rs, ok := s.(StateRefresher); ok {
		if err := rs.RefreshState(); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// current will track our current state
	current := TestStateInitial()

	// Check that the initial state is correct
	if state := reader.State(); !current.Equal(state) {
		t.Fatalf("not initial:\n%#v\n\n%#v", state, current)
	}

	// Write a new state and verify that we have it
	if ws, ok := s.(StateWriter); ok {
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

		if err := ws.WriteState(current); err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual := reader.State(); !actual.Equal(current) {
			t.Fatalf("bad:\n%#v\n\n%#v", actual, current)
		}
	}

	// Test persistence
	if ps, ok := s.(StatePersister); ok {
		if err := ps.PersistState(); err != nil {
			t.Fatalf("err: %s", err)
		}

		// Refresh if we got it
		if rs, ok := s.(StateRefresher); ok {
			if err := rs.RefreshState(); err != nil {
				t.Fatalf("err: %s", err)
			}
		}

		// Just set the serials the same... Then compare.
		actual := reader.State()
		if !actual.Equal(current) {
			t.Fatalf("bad: %#v\n\n%#v", actual, current)
		}
	}

	// If we can write and persist then verify that the serial
	// is only implemented on change.
	writer, writeOk := s.(StateWriter)
	persister, persistOk := s.(StatePersister)
	if writeOk && persistOk {
		// Same serial
		serial := current.Serial
		if err := writer.WriteState(current); err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := persister.PersistState(); err != nil {
			t.Fatalf("err: %s", err)
		}

		if reader.State().Serial != serial {
			t.Fatalf("bad: expected %d, got %d", serial, reader.State().Serial)
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
		if err := writer.WriteState(current); err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := persister.PersistState(); err != nil {
			t.Fatalf("err: %s", err)
		}

		if reader.State().Serial <= serial {
			t.Fatalf("bad: expected %d, got %d", serial, reader.State().Serial)
		}

		// Check that State() returns a copy by modifying the copy and comparing
		// to the current state.
		stateCopy := reader.State()
		stateCopy.Serial++
		if reflect.DeepEqual(stateCopy, current) {
			t.Fatal("State() should return a copy")
		}
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
