package s3

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
	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
	}).(*Backend)

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

	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

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

	b1 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":     bucketName,
		"key":        keyName,
		"encrypt":    true,
		"lock_table": bucketName,
	}).(*Backend)

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
