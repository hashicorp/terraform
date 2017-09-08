package etcd

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	prepareEtcdv3(t)

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": os.Getenv("TF_ETCDV3_ENDPOINTS"),
		"prefix":    fmt.Sprintf("%s/%s", keyPrefix, time.Now().String()),
	})

	// Grab the client
	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("Error: %s.", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)
}

func TestEtcdv3_stateLock(t *testing.T) {
	prepareEtcdv3(t)

	key := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// Get the backend
	s1, err := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": os.Getenv("TF_ETCDV3_ENDPOINTS"),
		"prefix":    key,
	}).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": os.Getenv("TF_ETCDV3_ENDPOINTS"),
		"prefix":    key,
	}).State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

func TestEtcdv3_destroyLock(t *testing.T) {
	prepareEtcdv3(t)

	// Get the backend
	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"endpoints": os.Getenv("TF_ETCDV3_ENDPOINTS"),
		"prefix":    fmt.Sprintf("tf-unit/%s", time.Now().String()),
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

	if err := c.Unlock(id); err != nil {
		t.Fatal(err)
	}

	res, err := c.Client.KV.Get(context.TODO(), c.Key)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 0 {
		t.Fatalf("lock key not cleaned up at: %s", string(res.Kvs[0].Key))
	}
}
