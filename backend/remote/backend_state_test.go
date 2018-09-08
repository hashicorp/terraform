package remote

import (
	"bytes"
	"os"
	"testing"

	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(remoteClient)
}

func TestRemoteClient(t *testing.T) {
	client := testRemoteClient(t)
	remote.TestClient(t, client)
}

func TestRemoteClient_withRunID(t *testing.T) {
	// Set the TFE_RUN_ID environment variable before creating the client!
	if err := os.Setenv("TFE_RUN_ID", generateID("run-")); err != nil {
		t.Fatalf("error setting env var TFE_RUN_ID: %v", err)
	}

	// Create a new test client.
	client := testRemoteClient(t)

	// Create a new empty state.
	state := bytes.NewBuffer(nil)
	if err := terraform.WriteState(terraform.NewState(), state); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Store the new state to verify (this will be done
	// by the mock that is used) that the run ID is set.
	if err := client.Put(state.Bytes()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
