package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	srv := newConsulTestServer(t)

	testCases := []string{
		fmt.Sprintf("tf-unit/%s", time.Now().String()),
		fmt.Sprintf("tf-unit/%s/", time.Now().String()),
	}

	for _, path := range testCases {
		t.Run(path, func(*testing.T) {
			// Get the backend
			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
				"address": srv.HTTPAddr,
				"path":    path,
			}))

			// Grab the client
			state, err := b.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			// Test
			remote.TestClient(t, state.(*remote.State).Client)
		})
	}
}

// test the gzip functionality of the client
func TestRemoteClient_gzipUpgrade(t *testing.T) {
	srv := newConsulTestServer(t)

	statePath := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// Get the backend
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    statePath,
	}))

	// Grab the client
	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)

	// create a new backend with gzip
	b = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    statePath,
		"gzip":    true,
	}))

	// Grab the client
	state, err = b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test
	remote.TestClient(t, state.(*remote.State).Client)
}

// TestConsul_largeState tries to write a large payload using the Consul state
// manager, as there is a limit to the size of the values in the KV store it
// will need to be split up before being saved and put back together when read.
func TestConsul_largeState(t *testing.T) {
	srv := newConsulTestServer(t)

	path := "tf-unit/test-large-state"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}))

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)
	c.Path = path

	// testPaths fails the test if the keys found at the prefix don't match
	// what is expected
	testPaths := func(t *testing.T, expected []string) {
		kv := c.Client.KV()
		pairs, _, err := kv.List(c.Path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res := make([]string, 0)
		for _, p := range pairs {
			res = append(res, p.Key)
		}
		if !reflect.DeepEqual(res, expected) {
			t.Fatalf("Wrong keys: %#v", res)
		}
	}

	testPayload := func(t *testing.T, data map[string]string, keys []string) {
		payload, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		err = c.Put(payload)
		if err != nil {
			t.Fatal("could not put payload", err)
		}

		remote, err := c.Get()
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(payload, remote.Data) {
			t.Fatal("the data do not match")
		}

		testPaths(t, keys)
	}

	// The default limit for the size of the value in Consul is 524288 bytes
	testPayload(
		t,
		map[string]string{
			"foo": strings.Repeat("a", 524288+2),
		},
		[]string{
			"tf-unit/test-large-state",
			"tf-unit/test-large-state/tfstate.2cb96f52c9fff8e0b56cb786ec4d2bed/0",
			"tf-unit/test-large-state/tfstate.2cb96f52c9fff8e0b56cb786ec4d2bed/1",
		},
	)

	// This payload is just short enough to be stored but will be bigger when
	// going through the Transaction API as it will be base64 encoded
	testPayload(
		t,
		map[string]string{
			"foo": strings.Repeat("a", 524288-10),
		},
		[]string{
			"tf-unit/test-large-state",
			"tf-unit/test-large-state/tfstate.4f407ace136a86521fd0d366972fe5c7/0",
		},
	)

	// We try to replace the payload with a small one, the old chunks should be removed
	testPayload(
		t,
		map[string]string{"var": "a"},
		[]string{"tf-unit/test-large-state"},
	)

	// Test with gzip and chunks
	b = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
		"gzip":    true,
	}))

	s, err = b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	c = s.(*remote.State).Client.(*RemoteClient)
	c.Path = path

	// We need a long random string so it results in multiple chunks even after
	// being gziped

	// We use a fixed seed so the test can be reproductible
	rand.Seed(1234)
	RandStringRunes := func(n int) string {
		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		b := make([]rune, n)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		return string(b)
	}

	testPayload(
		t,
		map[string]string{
			"bar": RandStringRunes(5 * (524288 + 2)),
		},
		[]string{
			"tf-unit/test-large-state",
			"tf-unit/test-large-state/tfstate.58e8160335864b520b1cc7f2222a4019/0",
			"tf-unit/test-large-state/tfstate.58e8160335864b520b1cc7f2222a4019/1",
			"tf-unit/test-large-state/tfstate.58e8160335864b520b1cc7f2222a4019/2",
			"tf-unit/test-large-state/tfstate.58e8160335864b520b1cc7f2222a4019/3",
		},
	)

	// Deleting the state should remove all chunks
	err = c.Delete()
	if err != nil {
		t.Fatal(err)
	}
	testPaths(t, []string{})
}

func TestConsul_stateLock(t *testing.T) {
	srv := newConsulTestServer(t)

	testCases := []string{
		fmt.Sprintf("tf-unit/%s", time.Now().String()),
		fmt.Sprintf("tf-unit/%s/", time.Now().String()),
	}

	for _, path := range testCases {
		t.Run(path, func(*testing.T) {
			// create 2 instances to get 2 remote.Clients
			sA, err := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
				"address": srv.HTTPAddr,
				"path":    path,
			})).StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatal(err)
			}

			sB, err := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
				"address": srv.HTTPAddr,
				"path":    path,
			})).StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatal(err)
			}

			remote.TestRemoteLocks(t, sA.(*remote.State).Client, sB.(*remote.State).Client)
		})
	}
}

func TestConsul_destroyLock(t *testing.T) {
	srv := newConsulTestServer(t)

	testCases := []string{
		fmt.Sprintf("tf-unit/%s", time.Now().String()),
		fmt.Sprintf("tf-unit/%s/", time.Now().String()),
	}

	testLock := func(client *RemoteClient, lockPath string) {
		// get the lock val
		pair, _, err := client.Client.KV().Get(lockPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		if pair != nil {
			t.Fatalf("lock key not cleaned up at: %s", pair.Key)
		}
	}

	for _, path := range testCases {
		t.Run(path, func(*testing.T) {
			// Get the backend
			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
				"address": srv.HTTPAddr,
				"path":    path,
			}))

			// Grab the client
			s, err := b.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			clientA := s.(*remote.State).Client.(*RemoteClient)

			info := statemgr.NewLockInfo()
			id, err := clientA.Lock(info)
			if err != nil {
				t.Fatal(err)
			}

			lockPath := clientA.Path + lockSuffix

			if err := clientA.Unlock(id); err != nil {
				t.Fatal(err)
			}

			testLock(clientA, lockPath)

			// The release the lock from a second client to test the
			// `terraform force-unlock <lock_id>` functionnality
			s, err = b.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatalf("err: %s", err)
			}

			clientB := s.(*remote.State).Client.(*RemoteClient)

			info = statemgr.NewLockInfo()
			id, err = clientA.Lock(info)
			if err != nil {
				t.Fatal(err)
			}

			if err := clientB.Unlock(id); err != nil {
				t.Fatal(err)
			}

			testLock(clientA, lockPath)

			err = clientA.Unlock(id)

			if err == nil {
				t.Fatal("consul lock should have been lost")
			}
			if err.Error() != "consul lock was lost" {
				t.Fatal("got wrong error", err)
			}
		})
	}
}

func TestConsul_lostLock(t *testing.T) {
	srv := newConsulTestServer(t)

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	// create 2 instances to get 2 remote.Clients
	sA, err := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	})).StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	sB, err := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path + "-not-used",
	})).StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := statemgr.NewLockInfo()
	info.Operation = "test-lost-lock"
	id, err := sA.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	reLocked := make(chan struct{})
	testLockHook = func() {
		close(reLocked)
		testLockHook = nil
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

	// create an "unreliable" network by closing all the consul client's
	// network connections
	conns := &unreliableConns{}
	origDialFn := dialContext
	defer func() {
		dialContext = origDialFn
	}()
	dialContext = conns.DialContext

	path := fmt.Sprintf("tf-unit/%s", time.Now().String())

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"address": srv.HTTPAddr,
		"path":    path,
	}))

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := statemgr.NewLockInfo()
	info.Operation = "test-lost-lock-connection"
	id, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	// kill the connection a few times
	for i := 0; i < 3; i++ {
		dialed := conns.dialedDone()
		// kill any open connections
		conns.Kill()
		// wait for a new connection to be dialed, and kill it again
		<-dialed
	}

	if err := s.Unlock(id); err != nil {
		t.Fatal("unlock error:", err)
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

func (u *unreliableConns) dialedDone() chan struct{} {
	u.Lock()
	defer u.Unlock()
	dialed := make(chan struct{})
	u.dialCallback = func() {
		defer close(dialed)
		u.dialCallback = nil
	}

	return dialed
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
