package remote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/mitchellh/cli"
)

const (
	testCred = "test-auth-token"
)

var (
	tfeHost  = svchost.Hostname(defaultHostname)
	credsSrc = auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
		tfeHost: {"token": testCred},
	})
)

func testInput(t *testing.T, answers map[string]string) *mockInput {
	return &mockInput{answers: answers}
}

func testBackendDefault(t *testing.T) *Remote {
	c := map[string]interface{}{
		"organization": "hashicorp",
		"workspaces": []interface{}{
			map[string]interface{}{
				"name": "prod",
			},
		},
	}
	return testBackend(t, c)
}

func testBackendNoDefault(t *testing.T) *Remote {
	c := map[string]interface{}{
		"organization": "hashicorp",
		"workspaces": []interface{}{
			map[string]interface{}{
				"prefix": "my-app-",
			},
		},
	}
	return testBackend(t, c)
}

func testRemoteClient(t *testing.T) remote.Client {
	b := testBackendDefault(t)
	raw, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	s := raw.(*remote.State)
	return s.Client
}

func testBackend(t *testing.T, c map[string]interface{}) *Remote {
	s := testServer(t)
	b := New(testDisco(s))

	// Configure the backend so the client is created.
	backend.TestBackendConfig(t, b, c)

	// Get a new mock client.
	mc := newMockClient()

	// Replace the services we use with our mock services.
	b.CLI = cli.NewMockUi()
	b.client.Applies = mc.Applies
	b.client.ConfigurationVersions = mc.ConfigurationVersions
	b.client.Organizations = mc.Organizations
	b.client.Plans = mc.Plans
	b.client.Runs = mc.Runs
	b.client.StateVersions = mc.StateVersions
	b.client.Workspaces = mc.Workspaces

	ctx := context.Background()

	// Create the organization.
	_, err := b.client.Organizations.Create(ctx, tfe.OrganizationCreateOptions{
		Name: tfe.String(b.organization),
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Create the default workspace if required.
	if b.workspace != "" {
		_, err = b.client.Workspaces.Create(ctx, b.organization, tfe.WorkspaceCreateOptions{
			Name: tfe.String(b.workspace),
		})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}

	return b
}

// testServer returns a *httptest.Server used for local testing.
func testServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Respond to service discovery calls.
	mux.HandleFunc("/well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"tfe.v2":"/api/v2/"}`)
	})

	return httptest.NewServer(mux)
}

// testDisco returns a *disco.Disco mapping app.terraform.io and
// localhost to a local test server.
func testDisco(s *httptest.Server) *disco.Disco {
	services := map[string]interface{}{
		"tfe.v2": fmt.Sprintf("%s/api/v2/", s.URL),
	}
	d := disco.NewWithCredentialsSource(credsSrc)

	d.ForceHostServices(svchost.Hostname(defaultHostname), services)
	d.ForceHostServices(svchost.Hostname("localhost"), services)
	return d
}
