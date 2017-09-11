package gcs

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestStateFile(t *testing.T) {
	cases := []struct {
		prefix           string
		defaultStateFile string
		name             string
		wantStateFile    string
		wantLockFile     string
	}{
		{"state", "", "default", "state/default.tfstate", "state/default.tflock"},
		{"state", "", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "legacy.tfstate", "default", "legacy.tfstate", "legacy.tflock"},
		{"state", "legacy.tfstate", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "legacy.state", "default", "legacy.state", "legacy.state.tflock"},
		{"state", "legacy.state", "test", "state/test.tfstate", "state/test.tflock"},
	}
	for _, c := range cases {
		b := &gcsBackend{
			prefix:           c.prefix,
			defaultStateFile: c.defaultStateFile,
		}

		if got := b.stateFile(c.name); got != c.wantStateFile {
			t.Errorf("stateFile(%q) = %q, want %q", c.name, got, c.wantStateFile)
		}

		if got := b.lockFile(c.name); got != c.wantLockFile {
			t.Errorf("lockFile(%q) = %q, want %q", c.name, got, c.wantLockFile)
		}
	}
}

func TestGCSBackend(t *testing.T) {
	// This test creates a bucket in GCS and populates it.
	// It may incur costs, so it will only run if the GOOGLE_PROJECT
	// environment variable is set.

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" {
		t.Skipf("skipping; set GOOGLE_PROJECT to activate")
	}

	const bucketName = "terraform_remote-state_test"
	t.Logf("using bucket %q in project %q", bucketName, projectID)

	config := map[string]interface{}{
		"bucket": bucketName,
		"prefix": "",
	}

	if creds := os.Getenv("GOOGLE_CREDENTIALS"); creds != "" {
		config["credentials"] = creds
		t.Logf("using credentials from %q", creds)
	} else {
		t.Log("using default credentials; set GOOGLE_CREDENTIALS for custom credentials")
	}

	be := backend.TestBackendConfig(t, New(), config)

	gcsBE, ok := be.(*gcsBackend)
	if !ok {
		t.Fatalf("backend: got %T, want *gcsBackend", be)
	}

	ctx := gcsBE.storageContext

	// create a new bucket and error out if we can't, e.g. because it already exists.
	if err := gcsBE.storageClient.Bucket(bucketName).Create(ctx, projectID, nil); err != nil {
		t.Fatalf("creating bucket failed: %v", err)
	}
	t.Log("bucket has been created")

	defer func() {
		if err := gcsBE.storageClient.Bucket(bucketName).Delete(ctx); err != nil {
			t.Errorf("deleting bucket failed: %v", err)
		} else {
			t.Log("bucket has been deleted")
		}
	}()

	// this should create a new state file
	_, err := be.State("TestGCSBackend")
	if err != nil {
		t.Fatalf("State(\"TestGCSBackend\"): %v", err)
	}

	states, err := be.States()
	if err != nil {
		t.Fatalf("States(): %v", err)
	}

	found := false
	for _, st := range states {
		if st == "TestGCSBackend" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("be.States() = %#v, missing \"TestGCSBackend\"", states)
	}

	// ensure state file exists
	if _, err := gcsBE.storageClient.Bucket(bucketName).Object("TestGCSBackend.tfstate").Attrs(ctx); err != nil {
		t.Fatalf("Attrs(\"TestGCSBackend.tfstate\"): %v", err)
	}

	c, err := gcsBE.client("TestGCSBackend_remote_TestClient")
	if err != nil {
		t.Fatal(err)
	}
	remote.TestClient(t, c)

	if err := be.DeleteState("TestGCSBackend"); err != nil {
		t.Errorf("DeleteState(\"TestGCSBackend\"): %v", err)
	}
}
