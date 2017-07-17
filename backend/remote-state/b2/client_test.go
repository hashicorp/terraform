package b2

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"gopkg.in/kothar/go-backblaze.v0"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)

	bucketName := fmt.Sprintf("terraform-state-b2-test-%x", time.Now().Unix())
	config := getB2Config(t, bucketName)

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	createB2Bucket(t, b.b2, bucketName)
	defer deleteB2Bucket(t, b.b2, bucketName)

	state, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Error(err)
	}

	remote.TestClient(t, state.(*remote.State).Client)
}

func createB2Bucket(t *testing.T, b2 *backblaze.B2, name string) {
	t.Logf("creating B2 bucket %s", name)

	_, err := b2.CreateBucket(name, backblaze.AllPrivate)
	if err != nil {
		t.Fatal("failed to create B2 bucket:", err)
	}
}

func deleteB2Bucket(t *testing.T, b2 *backblaze.B2, name string) {
	warning := "WARNING: Failed to delete the test B2 bucket %s. It may have been left in your Backblaze account and may incur storage charges. (error was %s)"
	t.Logf("deleting B2 bucket %s", name)

	bucket, err := b2.Bucket(name)
	if err != nil {
		t.Fatalf(warning, name, err)
	}

	// We hide files rather than delete in normal operation,
	// so need to clean up test files manually
	for {
		resp, err := bucket.ListFileVersions("", "", 1000)
		if err != nil {
			t.Fatalf(warning, name, err)
		}

		for _, file := range resp.Files {
			_, err := bucket.DeleteFileVersion(file.Name, file.ID)
			if err != nil {
				t.Fatalf(warning, name, err)
			}
		}

		if resp.NextFileName == "" {
			break
		}
	}

	err = bucket.Delete()
	if err != nil {
		t.Fatalf(warning, name, err)
	}
}
