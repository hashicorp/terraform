package s3

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
	})).(*Backend)

	createS3Bucket(t, b.s3Client, bucketName)
	defer deleteS3Bucket(t, b.s3Client, bucketName)

	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func TestRemoteClientLocks(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(t, b1.s3Client, bucketName)
	defer deleteS3Bucket(t, b1.s3Client, bucketName)
	createDynamoDBTable(t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b1.dynClient, bucketName)

	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

// verify that we can unlock a state with an existing lock
func TestForceUnlock(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-remote-s3-test-force-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"encrypt":        true,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(t, b1.s3Client, bucketName)
	defer deleteS3Bucket(t, b1.s3Client, bucketName)
	createDynamoDBTable(t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b1.dynClient, bucketName)

	// first test with default
	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	info := state.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err := s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal("failed to get default state to force unlock:", err)
	}

	if err := s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock default state")
	}

	// now try the same thing with a named state
	// first test with default
	s1, err = b1.State("test")
	if err != nil {
		t.Fatal(err)
	}

	info = state.NewLockInfo()
	info.Operation = "test"
	info.Who = "clientA"

	lockID, err = s1.Lock(info)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// s1 is now locked, get the same state through s2 and unlock it
	s2, err = b2.State("test")
	if err != nil {
		t.Fatal("failed to get named state to force unlock:", err)
	}

	if err = s2.Unlock(lockID); err != nil {
		t.Fatal("failed to force-unlock named state")
	}
}

func TestRemoteClient_clientMD5(t *testing.T) {
	testACC(t)

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(t, b.s3Client, bucketName)
	defer deleteS3Bucket(t, b.s3Client, bucketName)
	createDynamoDBTable(t, b.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b.dynClient, bucketName)

	s, err := b.State(backend.DefaultStateName)
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

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":         bucketName,
		"key":            keyName,
		"dynamodb_table": bucketName,
	})).(*Backend)

	createS3Bucket(t, b1.s3Client, bucketName)
	defer deleteS3Bucket(t, b1.s3Client, bucketName)
	createDynamoDBTable(t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b1.dynClient, bucketName)

	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	client1 := s1.(*remote.State).Client

	// create a old and new state version to persist
	s := state.TestStateInitial()
	var oldState bytes.Buffer
	if err := terraform.WriteState(s, &oldState); err != nil {
		t.Fatal(err)
	}
	s.Serial++
	var newState bytes.Buffer
	if err := terraform.WriteState(s, &newState); err != nil {
		t.Fatal(err)
	}

	// Use b2 without a dynamodb_table to bypass the lock table to write the state directly.
	// client2 will write the "incorrect" state, simulating s3 eventually consistency delays
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"key":    keyName,
	})).(*Backend)
	s2, err := b2.State(backend.DefaultStateName)
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
