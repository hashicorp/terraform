package cloud

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestCloudState_impl(t *testing.T) {
	var _ remote.Client = new(remoteClient)
}

func TestCloudState(t *testing.T) {
	state := testCloudState(t)
	TestState(t, state)
}

func TestCloudState_stateLock(t *testing.T) {
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

	TestCloudLocks(t, s1, s2)
}

func TestCloudState_withRunID(t *testing.T) {
	// Set the TFE_RUN_ID environment variable before creating the client!
	if err := os.Setenv("TFE_RUN_ID", GenerateID("run-")); err != nil {
		t.Fatalf("error setting env var TFE_RUN_ID: %v", err)
	}

	// Create a new test client.
	state := testCloudState(t)

	// Create a new empty state.
	sf := statefile.New(states.NewState(), "", 0)
	var buf bytes.Buffer
	statefile.Write(sf, &buf)

	jsonState, err := ioutil.ReadFile("../command/testdata/show-json-state/sensitive-variables/output.json")

	if err != nil {
		t.Fatal(err)
	}

	if err := state.uploadState(state.lineage, state.serial, state.forcePush, buf.Bytes(), jsonState); err != nil {
		t.Fatalf("put: %s", err)
	}
}
