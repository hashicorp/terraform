package inmem

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	defer Reset()
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
	defer Reset()
	b := backend.TestBackendConfig(t, New(), nil).(*Backend)
	backend.TestBackend(t, b, nil)
}

func TestBackendLocked(t *testing.T) {
	defer Reset()
	b1 := backend.TestBackendConfig(t, New(), nil).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), nil).(*Backend)

	backend.TestBackend(t, b1, b2)
}

// use the this backen to test the remote.State implementation
func TestRemoteState(t *testing.T) {
	defer Reset()
	b := backend.TestBackendConfig(t, New(), nil)

	workspace := "workspace"

	// create a new workspace in this backend
	s, err := b.State(workspace)
	if err != nil {
		t.Fatal(err)
	}

	// force overwriting the remote state
	newState := terraform.NewState()

	if err := s.WriteState(newState); err != nil {
		t.Fatal(err)
	}

	if err := s.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}

	savedState := s.State()
	if err != nil {
		t.Fatal(err)
	}

	if savedState.Lineage != newState.Lineage {
		t.Fatal("saved state has incorrect lineage")
	}
}
