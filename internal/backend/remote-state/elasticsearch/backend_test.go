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

	"github.com/hashicorp/terraform/internal/backend"
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
		"endpoints":              []interface{}{"localhost:9201"},
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
