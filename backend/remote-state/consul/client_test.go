package consul

import (
	"context"
	"fmt"
	"net"
	"sync"
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

func TestConsul_lostLockConnection(t *testing.T) {
	srv := newConsulTestServer(t)
	defer srv.Stop()

	// create an "unreliable" network by closing all the consul client's
	// network connections
	conns := &unreliableConns{}
	origDialFn := dialContext
	defer func() {
		dialContext = origDialFn
	}()
	dialContext = conns.DialContext

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	})

	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := state.NewLockInfo()
	info.Operation = "test-lost-lock-connection"
	id, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	// set a callback to know when the monitor loop re-connects
	dialed := make(chan struct{})
	conns.dialCallback = func() {
		close(dialed)
		conns.dialCallback = nil
	}

	// kill any open connections
	// once the consul client is fixed, we should loop over this a few time to
	// be sure, since we can't hook into the client's internal lock monitor
	// loop.
	conns.Kill()
	// wait for a new connection to be dialed, and kill it again
	<-dialed
	conns.Kill()

	// since the lock monitor loop is hidden in the consul api client, we can
	// only wait a bit to make sure we were notified of the failure
	time.Sleep(time.Second)

	// once the consul client can reconnect properly, there will no longer be
	// an error here
	//if err := s.Unlock(id); err != nil {
	if err := s.Unlock(id); err != lostLockErr {
		t.Fatalf("expected lost lock error, got %v", err)
	}
}

type unreliableConns struct {
	sync.Mutex
	conns        []net.Conn
	dialCallback func()
}

func (u *unreliableConns) DialContext(ctx context.Context, netw, addr string) (net.Conn, error) {
	u.Lock()
	defer u.Unlock()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, netw, addr)
	if err != nil {
		return nil, err
	}

	u.conns = append(u.conns, conn)

	if u.dialCallback != nil {
		u.dialCallback()
	}

	return conn, nil
}

// Kill these with a deadline, just to make sure we don't end up with any EOFs
// that get ignored.
func (u *unreliableConns) Kill() {
	u.Lock()
	defer u.Unlock()

	for _, conn := range u.conns {
		conn.(*net.TCPConn).SetDeadline(time.Now())
	}
	u.conns = nil
}
