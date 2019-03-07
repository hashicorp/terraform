package oss

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
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-oss-test-%x", time.Now().Unix())
	path := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"path":    path,
		"encrypt": true,
	})).(*Backend)

	createOSSBucket(t, b.ossClient, bucketName)
	defer deleteOSSBucket(t, b.ossClient, bucketName)

	state, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestOSS_stateLock(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-oss-test-%x", time.Now().Unix())
	path := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"path":    path,
		"encrypt": true,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"path":    path,
		"encrypt": true,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

// verify that we can unlock a state with an existing lock
func TestOSS_destroyLock(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-oss-test-%x", time.Now().Unix())
	path := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"path":    path,
		"encrypt": true,
	})).(*Backend)

	createOSSBucket(t, b.ossClient, bucketName)
	defer deleteOSSBucket(t, b.ossClient, bucketName)

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)

	info := state.NewLockInfo()
	id, err := c.Lock(info)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := c.Unlock(id); err != nil {
		t.Fatalf("err: %s", err)
	}

	res, err := c.getObj(c.lockFile)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if res != nil && res.String() != "" {
		t.Fatalf("lock key not cleaned up at: %s", string(c.stateFile))
	}
}
