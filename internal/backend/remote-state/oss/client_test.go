// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oss

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"bytes"
	"crypto/md5"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// NOTE: Before running this testcase, please create a OTS instance called 'tf-oss-remote'
var RemoteTestUsedOTSEndpoint = "https://tf-oss-remote.cn-hangzhou.ots.aliyuncs.com"

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("tf-remote-oss-test-%x", time.Now().Unix())
	path := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"prefix":  path,
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

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("tf-remote-oss-test-%x", time.Now().Unix())
	tableName := fmt.Sprintf("tfRemoteTestForce%x", time.Now().Unix())
	path := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)
	createTablestoreTable(t, b1.otsClient, tableName)
	defer deleteTablestoreTable(t, b1.otsClient, tableName)

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

// verify that the backend can handle more than one state in the same table
func TestRemoteClientLocks_multipleStates(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("tf-remote-oss-test-force-%x", time.Now().Unix())
	tableName := fmt.Sprintf("tfRemoteTestForce%x", time.Now().Unix())
	path := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)
	createTablestoreTable(t, b1.otsClient, tableName)
	defer deleteTablestoreTable(t, b1.otsClient, tableName)

	s1, err := b1.StateMgr("s1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatal("failed to get lock for s1:", err)
	}

	// s1 is now locked, s2 should not be locked as it's a different state file
	s2, err := b2.StateMgr("s2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatal("failed to get lock for s2:", err)
	}
}

// verify that we can unlock a state with an existing lock
func TestRemoteForceUnlock(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("tf-remote-oss-test-force-%x", time.Now().Unix())
	tableName := fmt.Sprintf("tfRemoteTestForce%x", time.Now().Unix())
	path := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"encrypt":             true,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)
	createTablestoreTable(t, b1.otsClient, tableName)
	defer deleteTablestoreTable(t, b1.otsClient, tableName)

	// first test with default
	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err := s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal("failed to get default state to force unlock:", err)
	}

	if err := s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock default state")
	}

	// now try the same thing with a named state
	// first test with default
	s1, err = b1.StateMgr("test")
	if err != nil {
		t.Fatal(err)
	}

	info = statemgr.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err = s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err = b2.StateMgr("test")
	if err != nil {
		t.Fatal("failed to get named state to force unlock:", err)
	}

	if err = s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock named state")
	}
}

func TestRemoteClient_clientMD5(t *testing.T) {
	testACC(t)

	bucketName := fmt.Sprintf("tf-remote-oss-test-%x", time.Now().Unix())
	tableName := fmt.Sprintf("tfRemoteTestForce%x", time.Now().Unix())
	path := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	createOSSBucket(t, b.ossClient, bucketName)
	defer deleteOSSBucket(t, b.ossClient, bucketName)
	createTablestoreTable(t, b.otsClient, tableName)
	defer deleteTablestoreTable(t, b.otsClient, tableName)

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client := s.(*remote.State).Client.(*RemoteClient)

	sum := md5.Sum([]byte("test"))

	if err := client.putMD5(sum[:]); err != nil {
		t.Fatal(err)
	}

	getSum, err := client.getMD5()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(getSum, sum[:]) {
		t.Fatalf("getMD5 returned the wrong checksum: expected %x, got %x", sum[:], getSum)
	}

	if err := client.deleteMD5(); err != nil {
		t.Fatal(err)
	}

	if getSum, err := client.getMD5(); err == nil {
		t.Fatalf("expected getMD5 error, got none. checksum: %x", getSum)
	}
}

// verify that a client won't return a state with an incorrect checksum.
func TestRemoteClient_stateChecksum(t *testing.T) {
	testACC(t)

	bucketName := fmt.Sprintf("tf-remote-oss-test-%x", time.Now().Unix())
	tableName := fmt.Sprintf("tfRemoteTestForce%x", time.Now().Unix())
	path := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":              bucketName,
		"prefix":              path,
		"tablestore_table":    tableName,
		"tablestore_endpoint": RemoteTestUsedOTSEndpoint,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)
	createTablestoreTable(t, b1.otsClient, tableName)
	defer deleteTablestoreTable(t, b1.otsClient, tableName)

	s1, err := b1.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client1 := s1.(*remote.State).Client

	// create an old and new state version to persist
	s := statemgr.TestFullInitialState()
	sf := &statefile.File{State: s}
	var oldState bytes.Buffer
	if err := statefile.Write(sf, &oldState); err != nil {
		t.Fatal(err)
	}
	sf.Serial++
	var newState bytes.Buffer
	if err := statefile.Write(sf, &newState); err != nil {
		t.Fatal(err)
	}

	// Use b2 without a tablestore_table to bypass the lock table to write the state directly.
	// client2 will write the "incorrect" state, simulating oss eventually consistency delays
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"prefix": path,
	})).(*Backend)
	s2, err := b2.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client2 := s2.(*remote.State).Client

	// write the new state through client2 so that there is no checksum yet
	if err := client2.Put(newState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// verify that we can pull a state without a checksum
	if _, err := client1.Get(); err != nil {
		t.Fatal(err)
	}

	// write the new state back with its checksum
	if err := client1.Put(newState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// put an empty state in place to check for panics during get
	if err := client2.Put([]byte{}); err != nil {
		t.Fatal(err)
	}

	// remove the timeouts so we can fail immediately
	origTimeout := consistencyRetryTimeout
	origInterval := consistencyRetryPollInterval
	defer func() {
		consistencyRetryTimeout = origTimeout
		consistencyRetryPollInterval = origInterval
	}()
	consistencyRetryTimeout = 0
	consistencyRetryPollInterval = 0

	// fetching an empty state through client1 should now error out due to a
	// mismatched checksum.
	if _, err := client1.Get(); !strings.HasPrefix(err.Error(), errBadChecksumFmt[:80]) {
		t.Fatalf("expected state checksum error: got %s", err)
	}

	// put the old state in place of the new, without updating the checksum
	if err := client2.Put(oldState.Bytes()); err != nil {
		t.Fatal(err)
	}

	// fetching the wrong state through client1 should now error out due to a
	// mismatched checksum.
	if _, err := client1.Get(); !strings.HasPrefix(err.Error(), errBadChecksumFmt[:80]) {
		t.Fatalf("expected state checksum error: got %s", err)
	}

	// update the state with the correct one after we Get again
	testChecksumHook = func() {
		if err := client2.Put(newState.Bytes()); err != nil {
			t.Fatal(err)
		}
		testChecksumHook = nil
	}

	consistencyRetryTimeout = origTimeout

	// this final Get will fail to fail the checksum verification, the above
	// callback will update the state with the correct version, and Get should
	// retry automatically.
	if _, err := client1.Get(); err != nil {
		t.Fatal(err)
	}
}
