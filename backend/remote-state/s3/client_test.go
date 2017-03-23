package s3

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
	testACC(t)

	bucketName := fmt.Sprintf("terraform-remote-s3-test-%x", time.Now().Unix())
	keyName := "testState"

	b := backend.TestBackendConfig(t, New(), map[string]interface{}{
		"bucket":  bucketName,
		"key":     keyName,
		"encrypt": true,
	}).(*Backend)

	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	createS3Bucket(t, b.s3Client, bucketName)
	defer deleteS3Bucket(t, b.s3Client, bucketName)

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

	s1, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	createS3Bucket(t, b1.s3Client, bucketName)
	defer deleteS3Bucket(t, b1.s3Client, bucketName)
	createDynamoDBTable(t, b1.dynClient, bucketName)
	defer deleteDynamoDBTable(t, b1.dynClient, bucketName)

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}
