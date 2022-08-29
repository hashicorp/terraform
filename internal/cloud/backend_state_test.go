package cloud

import (
	"bytes"
	"os"
	"testing"

	tfe "github.com/hashicorp/go-tfe"

	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(remoteClient)
}

func TestRemoteClient(t *testing.T) {
	client := testRemoteClient(t)
	remote.TestClient(t, client)
}

func TestRemoteClient_stateVersionCreated(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	raw, err := b.StateMgr(testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	client := raw.(*State).Client

	err = client.Put(([]byte)(`
{
	"version": 4,
	"terraform_version": "1.3.0",
	"serial": 1,
	"lineage": "backend-change",
	"outputs": {
			"foo": {
					"type": "string",
					"value": "bar"
			}
	}
}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stateVersionsAPI := b.client.StateVersions.(*MockStateVersions)
	if got, want := len(stateVersionsAPI.stateVersions), 1; got != want {
		t.Fatalf("wrong number of state versions in the mock client %d; want %d", got, want)
	}

	var stateVersion *tfe.StateVersion
	for _, sv := range stateVersionsAPI.stateVersions {
		stateVersion = sv
	}

	if stateVersionsAPI.outputStates[stateVersion.ID] == nil || len(stateVersionsAPI.outputStates[stateVersion.ID]) == 0 {
		t.Fatal("no state version outputs in the mock client")
	}
}

func TestRemoteClient_TestRemoteLocks(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	s1, err := b.StateMgr(testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	s2, err := b.StateMgr(testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	remote.TestRemoteLocks(t, s1.(*State).Client, s2.(*State).Client)
}

func TestRemoteClient_withRunID(t *testing.T) {
	// Set the TFE_RUN_ID environment variable before creating the client!
	if err := os.Setenv("TFE_RUN_ID", GenerateID("run-")); err != nil {
		t.Fatalf("error setting env var TFE_RUN_ID: %v", err)
	}

	// Create a new test client.
	client := testRemoteClient(t)

	// Create a new empty state.
	sf := statefile.New(states.NewState(), "", 0)
	var buf bytes.Buffer
	statefile.Write(sf, &buf)

	// Store the new state to verify (this will be done
	// by the mock that is used) that the run ID is set.
	if err := client.Put(buf.Bytes()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
