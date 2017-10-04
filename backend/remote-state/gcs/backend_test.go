package gcs

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestStateFile(t *testing.T) {
	t.Parallel()

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

func TestRemoteClient(t *testing.T) {
	t.Parallel()

	be := setupBackend(t)
	defer teardownBackend(t, be)

	ss, err := be.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("be.State(%q) = %v", backend.DefaultStateName, err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("be.State(): got a %T, want a *remote.State", ss)
	}

	remote.TestClient(t, rs.Client)
}

func TestRemoteLocks(t *testing.T) {
	t.Parallel()

	be := setupBackend(t)
	defer teardownBackend(t, be)

	remoteClient := func() (remote.Client, error) {
		ss, err := be.State(backend.DefaultStateName)
		if err != nil {
			return nil, err
		}

		rs, ok := ss.(*remote.State)
		if !ok {
			return nil, fmt.Errorf("be.State(): got a %T, want a *remote.State", ss)
		}

		return rs.Client, nil
	}

	c0, err := remoteClient()
	if err != nil {
		t.Fatalf("remoteClient(0) = %v", err)
	}
	c1, err := remoteClient()
	if err != nil {
		t.Fatalf("remoteClient(1) = %v", err)
	}

	remote.TestRemoteLocks(t, c0, c1)
}

func TestBackend(t *testing.T) {
	t.Parallel()

	be0 := setupBackend(t)
	defer teardownBackend(t, be0)

	be1 := setupBackend(t)

	backend.TestBackend(t, be0, be1)
}

// setupBackend returns a new GCS backend.
func setupBackend(t *testing.T) backend.Backend {
	t.Helper()

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" || os.Getenv("TF_ACC") == "" {
		t.Skip("This test creates a bucket in GCS and populates it. " +
			"Since this may incur costs, it will only run if " +
			"the TF_ACC and GOOGLE_PROJECT environment variables are set.")
	}

	config := map[string]interface{}{
		"project": projectID,
		"bucket":  strings.ToLower(t.Name()),
		"prefix":  "",
	}

	if creds := os.Getenv("GOOGLE_CREDENTIALS"); creds != "" {
		config["credentials"] = creds
		t.Logf("using credentials from %q", creds)
	} else {
		t.Log("using default credentials; set GOOGLE_CREDENTIALS for custom credentials")
	}

	return backend.TestBackendConfig(t, New(), config)
}

// teardownBackend deletes all states from be except the default state.
func teardownBackend(t *testing.T, be backend.Backend) {
	t.Helper()

	// Delete all states. The bucket must be empty before it can be deleted.
	states, err := be.States()
	if err != nil {
		t.Fatalf("be.States() = %v; manual clean-up may be required", err)
	}
	for _, st := range states {
		if st == backend.DefaultStateName {
			continue
		}
		if err := be.DeleteState(st); err != nil {
			t.Fatalf("be.DeleteState(%q) = %v; manual clean-up may be required", st, err)
		}
	}

	gcsBE, ok := be.(*gcsBackend)
	if !ok {
		t.Fatalf("be is a %T, want a *gcsBackend", be)
	}
	ctx := gcsBE.storageContext

	// Delete the default state, which DeleteState() will refuse to do.
	// It's okay if this fails, not all tests create a default state.
	if err := gcsBE.storageClient.Bucket(gcsBE.bucketName).Object("default.tfstate").Delete(ctx); err != nil {
		t.Logf("deleting \"default.tfstate\": %v; manual clean-up may be required", err)
	}

	// Delete the bucket itself.
	if err := gcsBE.storageClient.Bucket(gcsBE.bucketName).Delete(ctx); err != nil {
		t.Fatalf("deleting bucket failed: %v; manual cleanup may be required, though later test runs will happily reuse an existing bucket", err)
	}
}
