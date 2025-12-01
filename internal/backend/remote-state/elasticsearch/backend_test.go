// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package elasticsearch

// To run acceptance tests, start Elasticsearch locally and run:
// TF_ACC=1 go test -v -timeout=2m github.com/hashicorp/terraform/internal/backend/remote-state/elasticsearch

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// testACC skips the test unless TF_ACC is set
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == ""
	if skip {
		t.Log("elasticsearch backend acceptance tests require setting TF_ACC")
		t.Skip()
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	caData, err := os.ReadFile("testdata/certs/ca.cert.pem")
	if err != nil {
		t.Fatal("")
	}
	keyData, err := os.ReadFile("testdata/certs/client.key")
	if err != nil {
		t.Fatal("")
	}
	certData, err := os.ReadFile("testdata/certs/client.crt")
	if err != nil {
		t.Fatal("")
	}

	testCases := []struct {
		Name   string
		Config map[string]interface{}
	}{
		{
			Name: "skip-tls-verification",
			Config: map[string]interface{}{
				"endpoints":              []interface{}{endpoint},
				"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
				"username":               "elastic",
				"password":               "changeme",
				"skip_cert_verification": true,
			},
		},
		{
			Name: "with-ca-cert",
			Config: map[string]interface{}{
				"endpoints":          []interface{}{endpoint},
				"index":              fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
				"username":           "elastic",
				"password":           "changeme",
				"ca_certificate_pem": string(caData),
			},
		},
		{
			Name: "with-client-cert-key",
			Config: map[string]interface{}{
				"endpoints":              []interface{}{endpoint},
				"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
				"username":               "elastic",
				"password":               "changeme",
				"ca_certificate_pem":     string(caData),
				"client_certificate_pem": string(certData),
				"client_private_key_pem": string(keyData),
			},
		},
		{
			Name: "custom-index",
			Config: map[string]interface{}{
				"endpoints":          []interface{}{endpoint},
				"index":              fmt.Sprintf("my-custom-state-%s", strings.ToLower(t.Name())),
				"username":           "elastic",
				"password":           "changeme",
				"ca_certificate_pem": string(caData),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := backend.TestWrapConfig(tc.Config)
			b := backend.TestBackendConfig(t, New(), config).(*Backend)

			if b == nil {
				t.Fatal("Backend could not be configured")
			}

			// Test that we can get a state manager
			_, sDiags := b.StateMgr(backend.DefaultStateName)
			if sDiags.HasErrors() {
				t.Fatal(sDiags)
			}

			// Clean up - delete the index
			client := &RemoteClient{
				Client: b.client,
				Index:  b.index,
			}
			defer client.deleteIndex()
		})
	}
}

func TestBackendConfig_invalidEndpoint(t *testing.T) {
	testACC(t)

	config := backend.TestWrapConfig(map[string]interface{}{
		"endpoints":              []interface{}{"http://localhost:9201"},
		"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
		"username":               "elastic",
		"password":               "changeme",
		"skip_cert_verification": true,
	})

	b := New()

	// Decode the HCL body to cty.Value
	schema := b.ConfigSchema()
	spec := schema.DecoderSpec()
	obj, decDiags := hcldec.Decode(config, spec, nil)
	if decDiags.HasErrors() {
		t.Fatalf("Failed to decode config: %s", decDiags)
	}

	// Prepare and validate config
	newObj, valDiags := b.PrepareConfig(obj)
	if valDiags.HasErrors() {
		t.Fatalf("Failed to prepare config: %s", valDiags.Err())
	}

	// Configure should fail for unreachable endpoint
	diags := b.Configure(newObj)
	if !diags.HasErrors() {
		t.Fatal("Expected configuration to fail with unreachable endpoint, but it succeeded")
	}

	// Verify the error message mentions connection failure
	errMsg := diags.Err().Error()
	if !strings.Contains(errMsg, "failed to initialize Elasticsearch") {
		t.Fatalf("Expected error about Elasticsearch initialization, got: %s", errMsg)
	}
}

func TestBackendConfig_invalidIndexNames(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	testCases := []struct {
		Name          string
		IndexName     string
		ExpectedError string
	}{
		{
			Name:          "uppercase-letters",
			IndexName:     "MyIndex",
			ExpectedError: "index name must be lowercase",
		},
		{
			Name:          "contains-space",
			IndexName:     "my index",
			ExpectedError: "index name cannot contain ' '",
		},
		{
			Name:          "starts-with-dash",
			IndexName:     "-myindex",
			ExpectedError: "index name cannot start with '-', '_', or '+'",
		},
		{
			Name:          "starts-with-underscore",
			IndexName:     "_myindex",
			ExpectedError: "index name cannot start with '-', '_', or '+'",
		},
		{
			Name:          "starts-with-plus",
			IndexName:     "+myindex",
			ExpectedError: "index name cannot start with '-', '_', or '+'",
		},
		{
			Name:          "contains-backslash",
			IndexName:     "my\\index",
			ExpectedError: "index name cannot contain '\\'",
		},
		{
			Name:          "contains-slash",
			IndexName:     "my/index",
			ExpectedError: "index name cannot contain '/'",
		},
		{
			Name:          "contains-asterisk",
			IndexName:     "my*index",
			ExpectedError: "index name cannot contain '*'",
		},
		{
			Name:          "contains-question",
			IndexName:     "my?index",
			ExpectedError: "index name cannot contain '?'",
		},
		{
			Name:          "contains-quote",
			IndexName:     "my\"index",
			ExpectedError: "index name cannot contain '\"'",
		},
		{
			Name:          "contains-less-than",
			IndexName:     "my<index",
			ExpectedError: "index name cannot contain '<'",
		},
		{
			Name:          "contains-greater-than",
			IndexName:     "my>index",
			ExpectedError: "index name cannot contain '>'",
		},
		{
			Name:          "contains-pipe",
			IndexName:     "my|index",
			ExpectedError: "index name cannot contain '|'",
		},
		{
			Name:          "contains-comma",
			IndexName:     "my,index",
			ExpectedError: "index name cannot contain ','",
		},
		{
			Name:          "contains-hash",
			IndexName:     "my#index",
			ExpectedError: "index name cannot contain '#'",
		},
		{
			Name:          "is-dot",
			IndexName:     ".",
			ExpectedError: "index name cannot be '.' or '..'",
		},
		{
			Name:          "is-double-dot",
			IndexName:     "..",
			ExpectedError: "index name cannot be '.' or '..'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := backend.TestWrapConfig(map[string]interface{}{
				"endpoints":              []interface{}{endpoint},
				"index":                  tc.IndexName,
				"username":               "elastic",
				"password":               "changeme",
				"skip_cert_verification": true,
			})

			b := New()

			// Decode the HCL body to cty.Value
			schema := b.ConfigSchema()
			spec := schema.DecoderSpec()
			obj, decDiags := hcldec.Decode(config, spec, nil)
			if decDiags.HasErrors() {
				t.Fatalf("Failed to decode config: %s", decDiags)
			}

			// Prepare and validate config
			newObj, valDiags := b.PrepareConfig(obj)
			if valDiags.HasErrors() {
				t.Fatalf("Failed to prepare config: %s", valDiags.Err())
			}

			// Configure should fail for invalid index name
			diags := b.Configure(newObj)
			if !diags.HasErrors() {
				t.Fatal("Expected configuration to fail with invalid index name, but it succeeded")
			}

			// Verify the error message contains expected text
			errMsg := diags.Err().Error()
			if !strings.Contains(errMsg, tc.ExpectedError) {
				t.Fatalf("Expected error containing %q, got: %s", tc.ExpectedError, errMsg)
			}
		})
	}
}

func TestBackendStates(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	config := backend.TestWrapConfig(map[string]interface{}{
		"endpoints":              []interface{}{endpoint},
		"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
		"username":               "elastic",
		"password":               "changeme",
		"skip_cert_verification": true,
	})

	b := backend.TestBackendConfig(t, New(), config).(*Backend)
	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	// Clean up - delete the index when done
	client := &RemoteClient{
		Client: b.client,
		Index:  b.index,
	}
	defer client.deleteIndex()

	backend.TestBackendStates(t, b)
}

func TestBackendStateLocks(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	config := backend.TestWrapConfig(map[string]interface{}{
		"endpoints":              []interface{}{endpoint},
		"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
		"username":               "elastic",
		"password":               "changeme",
		"skip_cert_verification": true,
	})

	b := backend.TestBackendConfig(t, New(), config).(*Backend)
	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	bb := backend.TestBackendConfig(t, New(), config).(*Backend)
	if bb == nil {
		t.Fatal("Second backend could not be configured")
	}

	// Clean up - delete the index when done
	client := &RemoteClient{
		Client: b.client,
		Index:  b.index,
	}
	defer client.deleteIndex()

	backend.TestBackendStateLocks(t, b, bb)
}

func TestBackendConcurrentLock(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	getStateMgr := func(indexName string) (statemgr.Full, *statemgr.LockInfo) {
		config := backend.TestWrapConfig(map[string]interface{}{
			"endpoints":              []interface{}{endpoint},
			"index":                  indexName,
			"username":               "elastic",
			"password":               "changeme",
			"skip_cert_verification": true,
		})
		b := backend.TestBackendConfig(t, New(), config).(*Backend)

		if b == nil {
			t.Fatal("Backend could not be configured")
		}

		// Clean up - delete the index when done
		defer func() {
			client := &RemoteClient{
				Client: b.client,
				Index:  b.index,
			}
			client.deleteIndex()
		}()

		stateMgr, sDiags := b.StateMgr(backend.DefaultStateName)
		if sDiags.HasErrors() {
			t.Fatalf("Failed to get the state manager: %v", sDiags)
		}

		info := statemgr.NewLockInfo()
		info.Operation = "test"
		info.Who = indexName

		return stateMgr, info
	}

	s1, i1 := getStateMgr(fmt.Sprintf("terraform-test-%s-1", strings.ToLower(t.Name())))
	s2, i2 := getStateMgr(fmt.Sprintf("terraform-test-%s-2", strings.ToLower(t.Name())))

	// First we need to create the workspace as the lock for creating them is
	// global
	lockID1, err := s1.Lock(i1)
	if err != nil {
		t.Fatalf("failed to lock first state: %v", err)
	}

	if err = s1.PersistState(nil); err != nil {
		t.Fatalf("failed to persist state: %v", err)
	}

	if err := s1.Unlock(lockID1); err != nil {
		t.Fatalf("failed to unlock first state: %v", err)
	}

	lockID2, err := s2.Lock(i2)
	if err != nil {
		t.Fatalf("failed to lock second state: %v", err)
	}

	if err = s2.PersistState(nil); err != nil {
		t.Fatalf("failed to persist state: %v", err)
	}

	if err := s2.Unlock(lockID2); err != nil {
		t.Fatalf("failed to unlock second state: %v", err)
	}

	// Now we can test concurrent lock - different indices should be able to lock simultaneously
	lockID1, err = s1.Lock(i1)
	if err != nil {
		t.Fatalf("failed to lock first state: %v", err)
	}

	lockID2, err = s2.Lock(i2)
	if err != nil {
		t.Fatalf("failed to lock second state: %v", err)
	}

	if err := s1.Unlock(lockID1); err != nil {
		t.Fatalf("failed to unlock first state: %v", err)
	}

	if err := s2.Unlock(lockID2); err != nil {
		t.Fatalf("failed to unlock second state: %v", err)
	}
}

func TestBackendConcurrentLock_SameIndexShouldFail(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()
	indexName := fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name()))

	// Create two backends pointing to the SAME index
	getStateMgr := func() (statemgr.Full, *statemgr.LockInfo) {
		config := backend.TestWrapConfig(map[string]interface{}{
			"endpoints":              []interface{}{endpoint},
			"index":                  indexName, // Same index for both!
			"username":               "elastic",
			"password":               "changeme",
			"skip_cert_verification": true,
		})
		b := backend.TestBackendConfig(t, New(), config).(*Backend)

		if b == nil {
			t.Fatal("Backend could not be configured")
		}

		stateMgr, sDiags := b.StateMgr(backend.DefaultStateName)
		if sDiags.HasErrors() {
			t.Fatalf("Failed to get the state manager: %v", sDiags)
		}

		info := statemgr.NewLockInfo()
		info.Operation = "test"
		info.Who = "test-client"

		return stateMgr, info
	}

	// Clean up at the end
	defer func() {
		config := backend.TestWrapConfig(map[string]interface{}{
			"endpoints":              []interface{}{endpoint},
			"index":                  indexName,
			"username":               "elastic",
			"password":               "changeme",
			"skip_cert_verification": true,
		})
		b := backend.TestBackendConfig(t, New(), config).(*Backend)
		client := &RemoteClient{
			Client: b.client,
			Index:  b.index,
		}
		client.deleteIndex()
	}()

	s1, i1 := getStateMgr()
	s2, i2 := getStateMgr()

	// First state manager acquires lock
	lockID1, err := s1.Lock(i1)
	if err != nil {
		t.Fatalf("failed to lock first state: %v", err)
	}
	defer s1.Unlock(lockID1)

	// Second state manager should FAIL to acquire lock on same index
	_, err = s2.Lock(i2)
	if err == nil {
		s2.Unlock("") // Clean up if it somehow succeeded
		t.Fatal("second state manager was able to acquire lock on same index - locks are not working!")
	}

	// Verify it's a lock error, not some other error
	if _, ok := err.(*statemgr.LockError); !ok {
		t.Fatalf("expected LockError, got: %T - %v", err, err)
	}

	t.Logf("Good! Second lock attempt correctly failed with: %v", err)
}

func TestBackendConcurrentLock_DifferentWorkspaces(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()
	indexName := fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name()))

	// Create two state managers for DIFFERENT workspaces in the SAME index
	getStateMgr := func(workspace string) (statemgr.Full, *statemgr.LockInfo) {
		config := backend.TestWrapConfig(map[string]interface{}{
			"endpoints":              []interface{}{endpoint},
			"index":                  indexName, // Same index!
			"username":               "elastic",
			"password":               "changeme",
			"skip_cert_verification": true,
		})
		b := backend.TestBackendConfig(t, New(), config).(*Backend)

		if b == nil {
			t.Fatal("Backend could not be configured")
		}

		stateMgr, sDiags := b.StateMgr(workspace) // Different workspace!
		if sDiags.HasErrors() {
			t.Fatalf("Failed to get the state manager: %v", sDiags)
		}

		info := statemgr.NewLockInfo()
		info.Operation = "test"
		info.Who = fmt.Sprintf("client-%s", workspace)

		return stateMgr, info
	}

	// Clean up at the end
	defer func() {
		config := backend.TestWrapConfig(map[string]interface{}{
			"endpoints":              []interface{}{endpoint},
			"index":                  indexName,
			"username":               "elastic",
			"password":               "changeme",
			"skip_cert_verification": true,
		})
		b := backend.TestBackendConfig(t, New(), config).(*Backend)
		client := &RemoteClient{
			Client: b.client,
			Index:  b.index,
		}
		client.deleteIndex()
	}()

	// Get state managers for different workspaces
	s1, i1 := getStateMgr("production")
	s2, i2 := getStateMgr("staging")

	// Both should be able to acquire locks simultaneously since they're different workspaces
	lockID1, err := s1.Lock(i1)
	if err != nil {
		t.Fatalf("failed to lock production workspace: %v", err)
	}
	t.Logf("Successfully locked production workspace with ID: %s", lockID1)

	lockID2, err := s2.Lock(i2)
	if err != nil {
		s1.Unlock(lockID1) // Clean up first lock
		t.Fatalf("failed to lock staging workspace while production is locked: %v", err)
	}
	t.Logf("Successfully locked staging workspace with ID: %s (while production is still locked)", lockID2)

	// Clean up - unlock both
	if err := s1.Unlock(lockID1); err != nil {
		t.Fatalf("failed to unlock production workspace: %v", err)
	}

	if err := s2.Unlock(lockID2); err != nil {
		t.Fatalf("failed to unlock staging workspace: %v", err)
	}

	t.Log("SUCCESS: Different workspaces in the same index can be locked concurrently")
}

func TestRemoteClient_documentID(t *testing.T) {
	client := &RemoteClient{
		Workspace: "default",
	}

	docID := client.documentID()
	expectedID := "state-default"

	if docID != expectedID {
		t.Fatalf("Expected document ID \"%s\", got \"%s\"", expectedID, docID)
	}

	client.Workspace = "production"
	docID = client.documentID()
	expectedID = "state-production"

	if docID != expectedID {
		t.Fatalf("Expected document ID \"%s\", got \"%s\"", expectedID, docID)
	}
}

func TestRemoteClient_lockDocumentID(t *testing.T) {
	client := &RemoteClient{
		Workspace: "default",
	}

	lockID := client.lockDocumentID()
	expectedID := "lock-default"

	if lockID != expectedID {
		t.Fatalf("Expected lock document ID \"%s\", got \"%s\"", expectedID, lockID)
	}

	client.Workspace = "staging"
	lockID = client.lockDocumentID()
	expectedID = "lock-staging"

	if lockID != expectedID {
		t.Fatalf("Expected lock document ID \"%s\", got \"%s\"", expectedID, lockID)
	}
}

func getEndpoint() string {
	return os.Getenv("ELASTICSEARCH_URL")
}
