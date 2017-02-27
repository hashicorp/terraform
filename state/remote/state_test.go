package remote

import (
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
