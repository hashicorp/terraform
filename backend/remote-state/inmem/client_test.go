package inmem

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	remotestate "github.com/hashicorp/terraform/backend/remote-state"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	b := backend.TestBackendConfig(t, New(), nil)
	remotestate.TestClient(t, b)
}

func TestInmemLocks(t *testing.T) {
	s, err := backend.TestBackendConfig(t, New(), nil).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s.(*remote.State).Client, s.(*remote.State).Client)
}
