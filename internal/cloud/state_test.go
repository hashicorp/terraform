// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/zclconf/go-cty/cty"
)

func TestState_impl(t *testing.T) {
	var _ statemgr.Reader = new(State)
	var _ statemgr.Writer = new(State)
	var _ statemgr.Persister = new(State)
	var _ statemgr.Refresher = new(State)
	var _ statemgr.OutputReader = new(State)
	var _ statemgr.Locker = new(State)
}

type ExpectedOutput struct {
	Name      string
	Sensitive bool
	IsNull    bool
}

func TestState_GetRootOutputValues(t *testing.T) {
	b, bCleanup := testBackendWithOutputs(t)
	defer bCleanup()

	state := &State{tfeClient: b.client, organization: b.organization, workspace: &tfe.Workspace{
		ID: "ws-abcd",
	}}
	outputs, err := state.GetRootOutputValues()

	if err != nil {
		t.Fatalf("error returned from GetRootOutputValues: %s", err)
	}

	cases := []ExpectedOutput{
		{
			Name:      "sensitive_output",
			Sensitive: true,
			IsNull:    false,
		},
		{
			Name:      "nonsensitive_output",
			Sensitive: false,
			IsNull:    false,
		},
		{
			Name:      "object_output",
			Sensitive: false,
			IsNull:    false,
		},
		{
			Name:      "list_output",
			Sensitive: false,
			IsNull:    false,
		},
	}

	if len(outputs) != len(cases) {
		t.Errorf("Expected %d item but %d were returned", len(cases), len(outputs))
	}

	for _, testCase := range cases {
		so, ok := outputs[testCase.Name]
		if !ok {
			t.Fatalf("Expected key %s but it was not found", testCase.Name)
		}
		if so.Value.IsNull() != testCase.IsNull {
			t.Errorf("Key %s does not match null expectation %v", testCase.Name, testCase.IsNull)
		}
		if so.Sensitive != testCase.Sensitive {
			t.Errorf("Key %s does not match sensitive expectation %v", testCase.Name, testCase.Sensitive)
		}
	}
}

func TestState(t *testing.T) {
	var buf bytes.Buffer
	s := statemgr.TestFullInitialState()
	sf := statefile.New(s, "stub-lineage", 2)
	err := statefile.Write(sf, &buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	data := buf.Bytes()

	state := testCloudState(t)

	jsonState, err := ioutil.ReadFile("../command/testdata/show-json-state/sensitive-variables/output.json")
	if err != nil {
		t.Fatal(err)
	}

	jsonStateOutputs := []byte(`
{
	"outputs": {
			"foo": {
					"type": "string",
					"value": "bar"
			}
	}
}`)

	if err := state.uploadState(state.lineage, state.serial, state.forcePush, data, jsonState, jsonStateOutputs); err != nil {
		t.Fatalf("put: %s", err)
	}

	payload, err := state.getStatePayload()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if !bytes.Equal(payload.Data, data) {
		t.Fatalf("expected full state %q\n\ngot: %q", string(payload.Data), string(data))
	}

	if err := state.Delete(true); err != nil {
		t.Fatalf("delete: %s", err)
	}

	p, err := state.getStatePayload()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if p != nil {
		t.Fatalf("expected empty state, got: %q", string(p.Data))
	}
}

func TestCloudLocks(t *testing.T) {
	back, bCleanup := testBackendWithName(t)
	defer bCleanup()

	a, err := back.StateMgr(testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	b, err := back.StateMgr(testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lockerA, ok := a.(statemgr.Locker)
	if !ok {
		t.Fatal("client A not a statemgr.Locker")
	}

	lockerB, ok := b.(statemgr.Locker)
	if !ok {
		t.Fatal("client B not a statemgr.Locker")
	}

	infoA := statemgr.NewLockInfo()
	infoA.Operation = "test"
	infoA.Who = "clientA"

	infoB := statemgr.NewLockInfo()
	infoB.Operation = "test"
	infoB.Who = "clientB"

	lockIDA, err := lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	_, err = lockerB.Lock(infoB)
	if err == nil {
		lockerA.Unlock(lockIDA)
		t.Fatal("client B obtained lock while held by client A")
	}
	if _, ok := err.(*statemgr.LockError); !ok {
		t.Errorf("expected a LockError, but was %t: %s", err, err)
	}

	if err := lockerA.Unlock(lockIDA); err != nil {
		t.Fatal("error unlocking client A", err)
	}

	lockIDB, err := lockerB.Lock(infoB)
	if err != nil {
		t.Fatal("unable to obtain lock from client B")
	}

	if lockIDB == lockIDA {
		t.Fatalf("duplicate lock IDs: %q", lockIDB)
	}

	if err = lockerB.Unlock(lockIDB); err != nil {
		t.Fatal("error unlocking client B:", err)
	}
}

func TestDelete_SafeDeleteNotSupported(t *testing.T) {
	state := testCloudState(t)
	workspaceId := state.workspace.ID
	state.workspace.Permissions.CanForceDelete = nil
	state.workspace.ResourceCount = 5

	// Typically delete(false) should safe-delete a cloud workspace, which should fail on this workspace with resources
	// However, since we have set the workspace canForceDelete permission to nil, we should fall back to force delete
	if err := state.Delete(false); err != nil {
		t.Fatalf("delete: %s", err)
	}
	workspace, err := state.tfeClient.Workspaces.ReadByID(context.Background(), workspaceId)
	if workspace != nil || err != tfe.ErrResourceNotFound {
		t.Fatalf("workspace %s not deleted", workspaceId)
	}
}

func TestDelete_ForceDelete(t *testing.T) {
	state := testCloudState(t)
	workspaceId := state.workspace.ID
	state.workspace.Permissions.CanForceDelete = tfe.Bool(true)
	state.workspace.ResourceCount = 5

	if err := state.Delete(true); err != nil {
		t.Fatalf("delete: %s", err)
	}
	workspace, err := state.tfeClient.Workspaces.ReadByID(context.Background(), workspaceId)
	if workspace != nil || err != tfe.ErrResourceNotFound {
		t.Fatalf("workspace %s not deleted", workspaceId)
	}
}

func TestDelete_SafeDelete(t *testing.T) {
	state := testCloudState(t)
	workspaceId := state.workspace.ID
	state.workspace.Permissions.CanForceDelete = tfe.Bool(false)
	state.workspace.ResourceCount = 5

	// safe-deleting a workspace with resources should fail
	err := state.Delete(false)
	if err == nil {
		t.Fatalf("workspace should have failed to safe delete")
	}

	// safe-deleting a workspace with resources should succeed once it has no resources
	state.workspace.ResourceCount = 0
	if err = state.Delete(false); err != nil {
		t.Fatalf("workspace safe-delete err: %s", err)
	}

	workspace, err := state.tfeClient.Workspaces.ReadByID(context.Background(), workspaceId)
	if workspace != nil || err != tfe.ErrResourceNotFound {
		t.Fatalf("workspace %s not deleted", workspaceId)
	}
}

func TestState_PersistState(t *testing.T) {
	t.Run("Initial PersistState", func(t *testing.T) {
		cloudState := testCloudState(t)

		if cloudState.readState != nil {
			t.Fatal("expected nil initial readState")
		}

		err := cloudState.PersistState(nil)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		var expectedSerial uint64 = 1
		if cloudState.readSerial != expectedSerial {
			t.Fatalf("expected initial state readSerial to be %d, got %d", expectedSerial, cloudState.readSerial)
		}
	})

	t.Run("Snapshot Interval Backpressure Header", func(t *testing.T) {
		// The "Create a State Version" API is allowed to return a special
		// HTTP response header X-Terraform-Snapshot-Interval, in which case
		// we should remember the number of seconds it specifies and delay
		// creating any more intermediate state snapshots for that many seconds.

		cloudState := testCloudState(t)

		if cloudState.stateSnapshotInterval != 0 {
			t.Error("state manager already has a nonzero snapshot interval")
		}

		// For this test we'll use a real client talking to a fake server,
		// since HTTP-level concerns like headers are out of scope for the
		// mock client we typically use in other tests in this package, which
		// aim to abstract away HTTP altogether.
		var serverURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Log(r.Method, r.URL.String())

			if r.URL.Path == "/state-json" {
				t.Log("pretending to be Archivist")
				fakeState := states.NewState()
				fakeStateFile := statefile.New(fakeState, "boop", 1)
				var buf bytes.Buffer
				statefile.Write(fakeStateFile, &buf)
				respBody := buf.Bytes()
				w.Header().Set("content-type", "application/json")
				w.Header().Set("content-length", strconv.FormatInt(int64(len(respBody)), 10))
				w.WriteHeader(http.StatusOK)
				w.Write(respBody)
				return
			}
			if r.URL.Path == "/api/ping" {
				t.Log("pretending to be Ping")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			fakeBody := map[string]any{
				"data": map[string]any{
					"type": "state-versions",
					"attributes": map[string]any{
						"hosted-state-download-url": serverURL + "/state-json",
					},
				},
			}
			fakeBodyRaw, err := json.Marshal(fakeBody)
			if err != nil {
				t.Fatal(err)
			}

			w.Header().Set("content-type", "application/json")
			w.Header().Set("content-length", strconv.FormatInt(int64(len(fakeBodyRaw)), 10))

			switch r.Method {
			case "POST":
				t.Log("pretending to be Create a State Version")
				w.Header().Set("x-terraform-snapshot-interval", "300")
				w.WriteHeader(http.StatusAccepted)
			case "GET":
				t.Log("pretending to be Fetch the Current State Version for a Workspace")
				w.WriteHeader(http.StatusOK)
			default:
				t.Fatal("don't know what API operation this was supposed to be")
			}

			w.WriteHeader(http.StatusOK)
			w.Write(fakeBodyRaw)
		}))
		defer server.Close()
		serverURL = server.URL
		cfg := &tfe.Config{
			Address:  server.URL,
			BasePath: "api",
			Token:    "placeholder",
		}
		client, err := tfe.NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		cloudState.tfeClient = client

		err = cloudState.RefreshState()
		if err != nil {
			t.Fatal(err)
		}
		cloudState.WriteState(states.BuildState(func(s *states.SyncState) {
			s.SetOutputValue(
				addrs.OutputValue{Name: "boop"}.Absolute(addrs.RootModuleInstance),
				cty.StringVal("beep"), false,
			)
		}))

		err = cloudState.PersistState(nil)
		if err != nil {
			t.Fatal(err)
		}

		// The PersistState call above should have sent a request to the test
		// server and got back the x-terraform-snapshot-interval header, whose
		// value should therefore now be recorded in the relevant field.
		if got, want := cloudState.stateSnapshotInterval, 300*time.Second; got != want {
			t.Errorf("wrong state snapshot interval after PersistState\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestState_ShouldPersistIntermediateState(t *testing.T) {
	cloudState := testCloudState(t)

	// We'll specify a normal interval and a server-supplied interval that
	// have enough time between them that we can be confident that the
	// fake timestamps we'll pass into ShouldPersistIntermediateState are
	// either too soon for normal, long enough for normal but not for server,
	// or too long for server.
	shortServerInterval := 5 * time.Second
	normalInterval := 60 * time.Second
	longServerInterval := 120 * time.Second
	beforeNormalInterval := 20 * time.Second
	betweenInterval := 90 * time.Second
	afterLongServerInterval := 200 * time.Second

	// Before making any requests the state manager should just respect the
	// normal interval, because it hasn't yet heard a request from the server.
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-beforeNormalInterval),
		})
		if should {
			t.Errorf("indicated that should persist before normal interval")
		}
	}
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-betweenInterval),
		})
		if !should {
			t.Errorf("indicated that should not persist after normal interval")
		}
	}

	// After making a request to the "Create a State Version" operation, the
	// server might return a header that causes us to set this field:
	cloudState.stateSnapshotInterval = shortServerInterval

	// The short server interval is shorter than the normal interval, so the
	// normal interval takes priority here.
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-beforeNormalInterval),
		})
		if should {
			t.Errorf("indicated that should persist before normal interval")
		}
	}
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-betweenInterval),
		})
		if !should {
			t.Errorf("indicated that should not persist after normal interval")
		}
	}

	// The server might instead request a longer interval.
	cloudState.stateSnapshotInterval = longServerInterval
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-beforeNormalInterval),
		})
		if should {
			t.Errorf("indicated that should persist before server interval")
		}
	}
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-betweenInterval),
		})
		if should {
			t.Errorf("indicated that should persist before server interval")
		}
	}
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-afterLongServerInterval),
		})
		if !should {
			t.Errorf("indicated that should not persist after server interval")
		}
	}

	// The "force" mode should always win, regardless of how much time has passed.
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-beforeNormalInterval),
			ForcePersist:             true,
		})
		if !should {
			t.Errorf("ignored ForcePersist")
		}
	}
	{
		should := cloudState.ShouldPersistIntermediateState(&local.IntermediateStatePersistInfo{
			RequestedPersistInterval: normalInterval,
			LastPersist:              time.Now().Add(-betweenInterval),
			ForcePersist:             true,
		})
		if !should {
			t.Errorf("ignored ForcePersist")
		}
	}
}
