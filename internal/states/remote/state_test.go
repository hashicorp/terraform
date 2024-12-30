// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"context"
	"log"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/version"
)

func TestState_impl(t *testing.T) {
	var _ statemgr.Reader = new(State)
	var _ statemgr.Writer = new(State)
	var _ statemgr.Persister = new(State)
	var _ statemgr.Refresher = new(State)
	var _ statemgr.OutputReader = new(State)
	var _ statemgr.Locker = new(State)
}

func TestStateRace(t *testing.T) {
	s := &State{
		Client: nilClient{},
	}

	current := states.NewState()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.WriteState(current)
			s.PersistState(nil)
			s.RefreshState()
		}()
	}
	wg.Wait()
}

// testCase encapsulates a test state test
type testCase struct {
	name string
	// A function to mutate state and return a cleanup function
	mutationFunc func(*State) (*states.State, func())
	// The expected requests to have taken place
	expectedRequests []mockClientRequest
	// Mark this case as not having a request
	noRequest bool
}

// isRequested ensures a test that is specified as not having
// a request doesn't have one by checking if a method exists
// on the expectedRequest.
func (tc testCase) isRequested(t *testing.T) bool {
	for _, expectedMethod := range tc.expectedRequests {
		hasMethod := expectedMethod.Method != ""
		if tc.noRequest && hasMethod {
			t.Fatalf("expected no content for %q but got: %v", tc.name, expectedMethod)
		}
	}
	return !tc.noRequest
}

func TestStatePersist(t *testing.T) {
	testCases := []testCase{
		{
			name: "first state persistence",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				mgr.state = &states.State{
					Modules: map[string]*states.Module{"": {}},
				}
				s := mgr.State()
				s.RootModule().SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Name: "myfile",
						Type: "local_file",
					}.Instance(addrs.NoKey),
					&states.ResourceInstanceObjectSrc{
						AttrsFlat: map[string]string{
							"filename": "file.txt",
						},
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Provider: tfaddr.Provider{Namespace: "local"},
					},
				)
				return s, func() {}
			},
			expectedRequests: []mockClientRequest{
				// Expect an initial refresh, which returns nothing since there is no remote state.
				{
					Method:  "Get",
					Content: nil,
				},
				// Expect a second refresh, since the read state is nil
				{
					Method:  "Get",
					Content: nil,
				},
				// Expect an initial push with values and a serial of 1
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "some meaningless value",
						"serial":            1.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs":           map[string]interface{}{},
						"resources": []interface{}{
							map[string]interface{}{
								"instances": []interface{}{
									map[string]interface{}{
										"attributes_flat": map[string]interface{}{
											"filename": "file.txt",
										},
										"schema_version":       0.0,
										"sensitive_attributes": []interface{}{},
									},
								},
								"mode":     "managed",
								"name":     "myfile",
								"provider": `provider["/local/"]`,
								"type":     "local_file",
							},
						},
						"check_results": nil,
					},
				},
			},
		},
		// If lineage changes, expect the serial to increment
		{
			name: "change lineage",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				mgr.lineage = "mock-lineage"
				return mgr.State(), func() {}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            2.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs":           map[string]interface{}{},
						"resources": []interface{}{
							map[string]interface{}{
								"instances": []interface{}{
									map[string]interface{}{
										"attributes_flat": map[string]interface{}{
											"filename": "file.txt",
										},
										"schema_version":       0.0,
										"sensitive_attributes": []interface{}{},
									},
								},
								"mode":     "managed",
								"name":     "myfile",
								"provider": `provider["/local/"]`,
								"type":     "local_file",
							},
						},
						"check_results": nil,
					},
				},
			},
		},
		// removing resources should increment the serial
		{
			name: "remove resources",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				mgr.state.RootModule().Resources = map[string]*states.Resource{}
				return mgr.State(), func() {}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            3.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs":           map[string]interface{}{},
						"resources":         []interface{}{},
						"check_results":     nil,
					},
				},
			},
		},
		// If the remote serial is incremented, then we increment it once more.
		{
			name: "change serial",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				originalSerial := mgr.serial
				mgr.serial++
				return mgr.State(), func() {
					mgr.serial = originalSerial
				}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            5.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs":           map[string]interface{}{},
						"resources":         []interface{}{},
						"check_results":     nil,
					},
				},
			},
		},
		// Adding an output should cause the serial to increment as well.
		{
			name: "add output to state",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				s := mgr.State()
				s.SetOutputValue(
					addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
					cty.StringVal("bar"), false,
				)
				return s, func() {}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            4.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs": map[string]interface{}{
							"foo": map[string]interface{}{
								"type":  "string",
								"value": "bar",
							},
						},
						"resources":     []interface{}{},
						"check_results": nil,
					},
				},
			},
		},
		// ...as should changing an output
		{
			name: "mutate state bar -> baz",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				s := mgr.State()
				s.SetOutputValue(
					addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
					cty.StringVal("baz"), false,
				)
				return s, func() {}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            5.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs": map[string]interface{}{
							"foo": map[string]interface{}{
								"type":  "string",
								"value": "baz",
							},
						},
						"resources":     []interface{}{},
						"check_results": nil,
					},
				},
			},
		},
		{
			name: "nothing changed",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				s := mgr.State()
				return s, func() {}
			},
			noRequest: true,
		},
		// If the remote state's serial is less (force push), then we
		// increment it once from there.
		{
			name: "reset serial (force push style)",
			mutationFunc: func(mgr *State) (*states.State, func()) {
				mgr.serial = 2
				return mgr.State(), func() {}
			},
			expectedRequests: []mockClientRequest{
				{
					Method: "Put",
					Content: map[string]interface{}{
						"version":           4.0, // encoding/json decodes this as float64 by default
						"lineage":           "mock-lineage",
						"serial":            3.0, // encoding/json decodes this as float64 by default
						"terraform_version": version.Version,
						"outputs": map[string]interface{}{
							"foo": map[string]interface{}{
								"type":  "string",
								"value": "baz",
							},
						},
						"resources":     []interface{}{},
						"check_results": nil,
					},
				},
			},
		},
	}

	// Initial setup of state just to give us a fixed starting point for our
	// test assertions below, or else we'd need to deal with
	// random lineage.
	mgr := &State{
		Client: &mockClient{},
	}

	// In normal use (during a Terraform operation) we always refresh and read
	// before any writes would happen, so we'll mimic that here for realism.
	// NB This causes a GET to be logged so the first item in the test cases
	// must account for this
	if err := mgr.RefreshState(); err != nil {
		t.Fatalf("failed to RefreshState: %s", err)
	}

	// Our client is a mockClient which has a log we
	// use to check that operations generate expected requests
	mockClient := mgr.Client.(*mockClient)

	// logIdx tracks the current index of the log separate from
	// the loop iteration so we can check operations that don't
	// cause any requests to be generated
	logIdx := 0

	// Run tests in order.
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, cleanup := tc.mutationFunc(mgr)

			if err := mgr.WriteState(s); err != nil {
				t.Fatalf("failed to WriteState for %q: %s", tc.name, err)
			}
			if err := mgr.PersistState(nil); err != nil {
				t.Fatalf("failed to PersistState for %q: %s", tc.name, err)
			}

			if tc.isRequested(t) {
				// Get captured request from the mock client log
				// based on the index of the current test
				if logIdx >= len(mockClient.log) {
					t.Fatalf("request lock and index are out of sync on %q: idx=%d len=%d", tc.name, logIdx, len(mockClient.log))
				}
				for expectedRequestIdx := 0; expectedRequestIdx < len(tc.expectedRequests); expectedRequestIdx++ {
					loggedRequest := mockClient.log[logIdx]
					logIdx++
					if diff := cmp.Diff(tc.expectedRequests[expectedRequestIdx], loggedRequest, cmpopts.IgnoreMapEntries(func(key string, value interface{}) bool {
						// This is required since the initial state creation causes the lineage to be a UUID that is not known at test time.
						return tc.name == "first state persistence" && key == "lineage"
					})); len(diff) > 0 {
						t.Logf("incorrect client requests for %q:\n%s", tc.name, diff)
						t.Fail()
					}
				}
			}
			cleanup()
		})
	}
	logCnt := len(mockClient.log)
	if logIdx != logCnt {
		t.Fatalf("not all requests were read. Expected logIdx to be %d but got %d", logCnt, logIdx)
	}
}

func TestState_GetRootOutputValues(t *testing.T) {
	// Initial setup of state with outputs already defined
	mgr := &State{
		Client: &mockClient{
			current: []byte(`
				{
					"version": 4,
					"lineage": "mock-lineage",
					"serial": 1,
					"terraform_version":"0.0.0",
					"outputs": {"foo": {"value":"bar", "type": "string"}},
					"resources": []
				}
			`),
		},
	}

	outputs, err := mgr.GetRootOutputValues(context.Background())
	if err != nil {
		t.Errorf("Expected GetRootOutputValues to not return an error, but it returned %v", err)
	}

	if len(outputs) != 1 {
		t.Errorf("Expected %d outputs, but received %d", 1, len(outputs))
	}
}

type migrationTestCase struct {
	name string
	// A function to generate a statefile
	stateFile func(*State) *statefile.File
	// The expected request to have taken place
	expectedRequest mockClientRequest
	// Mark this case as not having a request
	expectedError string
	// force flag passed to client
	force bool
}

func TestWriteStateForMigration(t *testing.T) {
	mgr := &State{
		Client: &mockClient{
			current: []byte(`
				{
					"version": 4,
					"lineage": "mock-lineage",
					"serial": 3,
					"terraform_version":"0.0.0",
					"outputs": {"foo": {"value":"bar", "type": "string"}},
					"resources": []
				}
			`),
		},
	}

	testCases := []migrationTestCase{
		// Refreshing state before we run the test loop causes a GET
		{
			name: "refresh state",
			stateFile: func(mgr *State) *statefile.File {
				return mgr.StateForMigration()
			},
			expectedRequest: mockClientRequest{
				Method: "Get",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "mock-lineage",
					"serial":            3.0,
					"terraform_version": "0.0.0",
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
				},
			},
		},
		{
			name: "cannot import lesser serial without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, mgr.lineage, 1)
			},
			expectedError: "cannot import state with serial 1 over newer state with serial 3",
		},
		{
			name: "cannot import differing lineage without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, "different-lineage", mgr.serial)
			},
			expectedError: `cannot import state with lineage "different-lineage" over unrelated state with lineage "mock-lineage"`,
		},
		{
			name: "can import lesser serial with force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, mgr.lineage, 1)
			},
			expectedRequest: mockClientRequest{
				Method: "Put",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "mock-lineage",
					"serial":            2.0,
					"terraform_version": version.Version,
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
					"check_results":     nil,
				},
			},
			force: true,
		},
		{
			name: "cannot import differing lineage without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, "different-lineage", mgr.serial)
			},
			expectedRequest: mockClientRequest{
				Method: "Put",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "different-lineage",
					"serial":            3.0,
					"terraform_version": version.Version,
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
					"check_results":     nil,
				},
			},
			force: true,
		},
	}

	// In normal use (during a Terraform operation) we always refresh and read
	// before any writes would happen, so we'll mimic that here for realism.
	// NB This causes a GET to be logged so the first item in the test cases
	// must account for this
	if err := mgr.RefreshState(); err != nil {
		t.Fatalf("failed to RefreshState: %s", err)
	}

	if err := mgr.WriteState(mgr.State()); err != nil {
		t.Fatalf("failed to write initial state: %s", err)
	}

	// Our client is a mockClient which has a log we
	// use to check that operations generate expected requests
	mockClient := mgr.Client.(*mockClient)

	// logIdx tracks the current index of the log separate from
	// the loop iteration so we can check operations that don't
	// cause any requests to be generated
	logIdx := 0

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sf := tc.stateFile(mgr)
			err := mgr.WriteStateForMigration(sf, tc.force)
			shouldError := tc.expectedError != ""

			// If we are expecting and error check it and move on
			if shouldError {
				if err == nil {
					t.Fatalf("test case %q should have failed with error %q", tc.name, tc.expectedError)
				} else if err.Error() != tc.expectedError {
					t.Fatalf("test case %q expected error %q but got %q", tc.name, tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("test case %q failed: %v", tc.name, err)
			}

			// At this point we should just do a normal write and persist
			// as would happen from the CLI
			mgr.WriteState(mgr.State())
			mgr.PersistState(nil)

			if logIdx >= len(mockClient.log) {
				t.Fatalf("request lock and index are out of sync on %q: idx=%d len=%d", tc.name, logIdx, len(mockClient.log))
			}
			loggedRequest := mockClient.log[logIdx]
			logIdx++
			if diff := cmp.Diff(tc.expectedRequest, loggedRequest); len(diff) > 0 {
				t.Fatalf("incorrect client requests for %q:\n%s", tc.name, diff)
			}
		})
	}

	logCnt := len(mockClient.log)
	if logIdx != logCnt {
		log.Fatalf("not all requests were read. Expected logIdx to be %d but got %d", logCnt, logIdx)
	}
}

// This test runs the same test cases as above, but with
// a client that implements EnableForcePush -- this allows
// us to test that -force continues to work for backends without
// this interface, but that this interface works for those that do.
func TestWriteStateForMigrationWithForcePushClient(t *testing.T) {
	mgr := &State{
		Client: &mockClientForcePusher{
			current: []byte(`
				{
					"version": 4,
					"lineage": "mock-lineage",
					"serial": 3,
					"terraform_version":"0.0.0",
					"outputs": {"foo": {"value":"bar", "type": "string"}},
					"resources": []
				}
			`),
		},
	}

	testCases := []migrationTestCase{
		// Refreshing state before we run the test loop causes a GET
		{
			name: "refresh state",
			stateFile: func(mgr *State) *statefile.File {
				return mgr.StateForMigration()
			},
			expectedRequest: mockClientRequest{
				Method: "Get",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "mock-lineage",
					"serial":            3.0,
					"terraform_version": "0.0.0",
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
				},
			},
		},
		{
			name: "cannot import lesser serial without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, mgr.lineage, 1)
			},
			expectedError: "cannot import state with serial 1 over newer state with serial 3",
		},
		{
			name: "cannot import differing lineage without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, "different-lineage", mgr.serial)
			},
			expectedError: `cannot import state with lineage "different-lineage" over unrelated state with lineage "mock-lineage"`,
		},
		{
			name: "can import lesser serial with force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, mgr.lineage, 1)
			},
			expectedRequest: mockClientRequest{
				Method: "Force Put",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "mock-lineage",
					"serial":            2.0,
					"terraform_version": version.Version,
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
					"check_results":     nil,
				},
			},
			force: true,
		},
		{
			name: "cannot import differing lineage without force",
			stateFile: func(mgr *State) *statefile.File {
				return statefile.New(mgr.state, "different-lineage", mgr.serial)
			},
			expectedRequest: mockClientRequest{
				Method: "Force Put",
				Content: map[string]interface{}{
					"version":           4.0,
					"lineage":           "different-lineage",
					"serial":            3.0,
					"terraform_version": version.Version,
					"outputs":           map[string]interface{}{"foo": map[string]interface{}{"type": string("string"), "value": string("bar")}},
					"resources":         []interface{}{},
					"check_results":     nil,
				},
			},
			force: true,
		},
	}

	// In normal use (during a Terraform operation) we always refresh and read
	// before any writes would happen, so we'll mimic that here for realism.
	// NB This causes a GET to be logged so the first item in the test cases
	// must account for this
	if err := mgr.RefreshState(); err != nil {
		t.Fatalf("failed to RefreshState: %s", err)
	}

	if err := mgr.WriteState(mgr.State()); err != nil {
		t.Fatalf("failed to write initial state: %s", err)
	}

	// Our client is a mockClientForcePusher which has a log we
	// use to check that operations generate expected requests
	mockClient := mgr.Client.(*mockClientForcePusher)

	if mockClient.force {
		t.Fatalf("client should not default to force")
	}

	// logIdx tracks the current index of the log separate from
	// the loop iteration so we can check operations that don't
	// cause any requests to be generated
	logIdx := 0

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Always reset client to not be force pushing
			mockClient.force = false
			sf := tc.stateFile(mgr)
			err := mgr.WriteStateForMigration(sf, tc.force)
			shouldError := tc.expectedError != ""

			// If we are expecting and error check it and move on
			if shouldError {
				if err == nil {
					t.Fatalf("test case %q should have failed with error %q", tc.name, tc.expectedError)
				} else if err.Error() != tc.expectedError {
					t.Fatalf("test case %q expected error %q but got %q", tc.name, tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("test case %q failed: %v", tc.name, err)
			}

			if tc.force && !mockClient.force {
				t.Fatalf("test case %q should have enabled force push", tc.name)
			}

			// At this point we should just do a normal write and persist
			// as would happen from the CLI
			mgr.WriteState(mgr.State())
			mgr.PersistState(nil)

			if logIdx >= len(mockClient.log) {
				t.Fatalf("request lock and index are out of sync on %q: idx=%d len=%d", tc.name, logIdx, len(mockClient.log))
			}
			loggedRequest := mockClient.log[logIdx]
			logIdx++
			if diff := cmp.Diff(tc.expectedRequest, loggedRequest); len(diff) > 0 {
				t.Fatalf("incorrect client requests for %q:\n%s", tc.name, diff)
			}
		})
	}

	logCnt := len(mockClient.log)
	if logIdx != logCnt {
		log.Fatalf("not all requests were read. Expected logIdx to be %d but got %d", logCnt, logIdx)
	}
}
