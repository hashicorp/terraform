package consul

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	srv := newConsulTestServer(t)
	defer srv.Stop()

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})

	// Grab the client
	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)
}

// test the gzip functionality of the client
func TestRemoteClient_gzipUpgrade(t *testing.T) {
	srv := newConsulTestServer(t)
	defer srv.Stop()

	statePath := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    statePath,
	})

	// Grab the client
	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)

	// create a new backend with gzip
	b = backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    statePath,
		"gzip":    true,
	})

	// Grab the client
	state, err = b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)
}

func TestConsul_stateLock(t *testing.T) {
	srv := newConsulTestServer(t)
	defer srv.Stop()

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// create 2 instances to get 2 remote.Clients
	sA, err := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	sB, err := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, sA.(*remote.State).Client, sB.(*remote.State).Client)
}
