package datastore

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = &RemoteClient{}
	var _ remote.ClientLocker = &RemoteClient{}
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	config := configFromEnv(t)
	defer cleanupTestNamespace(t, config)

	b1 := backend.TestBackendConfig(t, New(), config).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), config).(*Backend)

	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("b1.State(%v): %v", backend.DefaultStateName, err)
	}
	s2, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("b2.State(%v): %v", backend.DefaultStateName, err)
	}
	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}
