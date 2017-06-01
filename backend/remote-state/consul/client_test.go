package consul

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
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

func TestConsul_destroyLock(t *testing.T) {
	srv := newConsulTestServer(t)
	defer srv.Stop()

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
	})

	// Grab the client
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)

	info := state.NewLockInfo()
	id, err := c.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	lockPath := c.Path + lockSuffix

	if err := c.Unlock(id); err != nil {
		t.Fatal(err)
	}

	// get the lock val
	pair, _, err := c.Client.KV().Get(lockPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pair != nil {
		t.Fatalf("lock key not cleaned up at: %s", pair.Key)
	}
}

func TestConsul_lostLock(t *testing.T) {
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
		"path":    path + "-not-used",
	}).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := state.NewLockInfo()
	info.Operation = "test-lost-lock"
	id, err := sA.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	reLocked := make(chan struct{})
	testLockHook = func() {
		close(reLocked)
	}

	// now we use the second client to break the lock
	kv := sB.(*remote.State).Client.(*RemoteClient).Client.KV()
	_, err = kv.Delete(path+lockSuffix, nil)
	if err != nil {
		t.Fatal(err)
	}

	<-reLocked

	if err := sA.Unlock(id); err != nil {
		t.Fatal(err)
	}
}
