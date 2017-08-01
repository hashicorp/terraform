package inmem

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

// reset the states and locks between tests
func reset() {
	states = stateMap{
		m: map[string]*remote.State{},
	}

	locks = lockMap{
		m: map[string]*state.LockInfo{},
	}
}

func TestBackendConfig(t *testing.T) {
	defer reset()
	testID := "test_lock_id"

	config := map[string]interface{}{
		"lock_id": testID,
	}

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)
	if c.Name != backend.DefaultStateName {
		t.Fatal("client name is not configured")
	}

	if err := locks.unlock(backend.DefaultStateName, testID); err != nil {
		t.Fatalf("default state should have been locked: %s", err)
	}
}

func TestBackend(t *testing.T) {
	defer reset()
	b := backend.TestBackendConfig(t, New(), nil).(*Backend)
	backend.TestBackend(t, b, nil)
}

func TestBackendLocked(t *testing.T) {
	defer reset()
	b1 := backend.TestBackendConfig(t, New(), nil).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), nil).(*Backend)

	backend.TestBackend(t, b1, b2)
}
