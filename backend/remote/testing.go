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
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
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
	obj := cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String),
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name":   cty.StringVal("prod"),
			"prefix": cty.NullVal(cty.String),
		}),
	})
	return testBackend(t, obj)
}

func testBackendNoDefault(t *testing.T) *Remote {
	obj := cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String),
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name":   cty.NullVal(cty.String),
			"prefix": cty.StringVal("my-app-"),
		}),
	})
	return testBackend(t, obj)
}

func testRemoteClient(t *testing.T) remote.Client {
	b := testBackendDefault(t)
	raw, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	s := raw.(*remote.State)
	return s.Client
}

func testBackend(t *testing.T, obj cty.Value) *Remote {
	s := testServer(t)
	b := New(testDisco(s))

	// Configure the backend so the client is created.
	valDiags := b.ValidateConfig(obj)
	if len(valDiags) != 0 {
		t.Fatal(valDiags.ErrWithWarnings())
	}

	confDiags := b.Configure(obj)
	if len(confDiags) != 0 {
		t.Fatal(confDiags.ErrWithWarnings())
	}

	// Get a new mock client.
	mc := newMockClient()

	// Replace the services we use with our mock services.
	b.CLI = cli.NewMockUi()
	b.client.Applies = mc.Applies
	b.client.ConfigurationVersions = mc.ConfigurationVersions
	b.client.Organizations = mc.Organizations
	b.client.Plans = mc.Plans
	b.client.PolicyChecks = mc.PolicyChecks
	b.client.Runs = mc.Runs
	b.client.StateVersions = mc.StateVersions
	b.client.Workspaces = mc.Workspaces

	b.ShowDiagnostics = func(vals ...interface{}) {
		var diags tfdiags.Diagnostics
		for _, diag := range diags.Append(vals...) {
			b.CLI.Error(diag.Description().Summary)
		}
	}

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

type unparsedVariableValue struct {
	value  string
	source terraform.ValueSourceType
}

func (v *unparsedVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	return &terraform.InputValue{
		Value:      cty.StringVal(v.value),
		SourceType: v.source,
	}, tfdiags.Diagnostics{}
}

// testVariable returns a backend.UnparsedVariableValue used for testing.
func testVariables(s terraform.ValueSourceType, vs ...string) map[string]backend.UnparsedVariableValue {
	vars := make(map[string]backend.UnparsedVariableValue, len(vs))
	for _, v := range vs {
		vars[v] = &unparsedVariableValue{
			value:  v,
			source: s,
		}
	}
	return vars
}
