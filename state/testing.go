package state

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

// TestStateInitial is the initial state that a State should have
// for TestState.
var TestStateInitial *terraform.State = &terraform.State{
	Modules: []*terraform.ModuleState{
		&terraform.ModuleState{
			Path: []string{"root", "child"},
			Outputs: map[string]string{
				"foo": "bar",
			},
		},
	},
}

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
	current := TestStateInitial

	// Check that the initial state is correct
	if !reflect.DeepEqual(reader.State(), current) {
		t.Fatalf("not initial: %#v", reader.State())
	}

	// Write a new state and verify that we have it
	if ws, ok := s.(StateWriter); ok {
		current.Modules = append(current.Modules, &terraform.ModuleState{
			Path: []string{"root"},
			Outputs: map[string]string{
				"bar": "baz",
			},
		})

		if err := ws.WriteState(current); err != nil {
			t.Fatalf("err: %s", err)
		}

		if actual := reader.State(); !reflect.DeepEqual(actual, current) {
			t.Fatalf("bad: %#v", actual)
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

		if actual := reader.State(); !reflect.DeepEqual(actual, current) {
			t.Fatalf("bad: %#v", actual)
		}
	}
}
