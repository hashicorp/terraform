package state

import (
	"testing"
)

func TestInmemState(t *testing.T) {
	TestState(t, &InmemState{state: TestStateInitial()})
}

func TestInmemState_impl(t *testing.T) {
	var _ StateReader = new(InmemState)
	var _ StateWriter = new(InmemState)
	var _ StatePersister = new(InmemState)
	var _ StateRefresher = new(InmemState)
}
