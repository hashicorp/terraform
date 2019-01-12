package state

import (
	"testing"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
)

// TestState is a helper for testing state implementations. It is expected
// that the given implementation is pre-loaded with the TestStateInitial
// state.
func TestState(t *testing.T, s State) {
	t.Helper()
	statemgr.TestFull(t, s)
}

// TestStateInitial is the initial state that a State should have
// for TestState.
func TestStateInitial() *states.State {
	return statemgr.TestFullInitialState()
}
