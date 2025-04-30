// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
)

func TestLoad_Local(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := TestStateFile(t, state)
	loader := &Loader{}
	loadedState, diags := loader.LoadState(strings.TrimSuffix(statePath, "/terraform.tfstate"))
	if diags.HasErrors() {
		t.Fatalf("failed to load state: %s", diags.Err())
	}

	if !statefile.StatesMarshalEqual(state, loadedState) {
		t.Fatalf("loaded state does not match original state")
	}
}

func TestLoad(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON:    []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:       states.ObjectReady,
				Dependencies: []addrs.ConfigResource{mustResourceAddr("test_instance.foo")},
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
	statePath := TestStateFile(t, state)

	s := testServer(t, statePath)
	backendStatePath := testBackendStateFile(t, cty.ObjectVal(map[string]cty.Value{
		"organization": cty.StringVal("hashicorp"),
		"hostname":     cty.StringVal("localhost"),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name":   cty.NullVal(cty.String),
			"prefix": cty.StringVal("my-app-"),
		}),
	}))
	dir := strings.TrimSuffix(backendStatePath, ".terraform/.terraform.tfstate")
	defer s.Close()
	loader := Loader{Discovery: testDisco(s)}
	t.Setenv(WorkspaceNameEnvVar, "test")
	loadedState, diags := loader.LoadState(dir)
	if diags.HasErrors() {
		t.Fatalf("failed to load state: %s", diags.Err())
	}

	if !statefile.StatesMarshalEqual(state, loadedState) {
		t.Fatalf("loaded state does not match original state")
	}
}

func mustResourceAddr(s string) addrs.ConfigResource {
	addr, diags := addrs.ParseAbsResourceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr.Config()
}

func testBackendStateFile(t *testing.T, value cty.Value) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".terraform", ".terraform.tfstate")

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("failed to create directories for temporary state file %s: %s", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create temporary state file %s: %s", path, err)
	}

	fmt.Fprintf(f, `{
	  "version": 3,
	  "terraform_version": "1.9.4",
	  "backend": {
	    "type": "remote",
	    "config": {
	      "hostname": %q,
	      "organization": %q,
	      "token": "foo",
	      "workspaces": {
	        "name": null,
	        "prefix": %q
	      }
	    },
	    "hash": 2143736989
	  }
	}`, value.GetAttr("hostname").AsString(),
		value.GetAttr("organization").AsString(),
		value.GetAttr("workspaces").GetAttr("prefix").AsString())

	f.Close()
	return path
}

func createTempFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return filePath
}

// testDisco returns a *disco.Disco mapping app.terraform.io and
// localhost to a local test server.
func testDisco(s *httptest.Server) *disco.Disco {
	services := map[string]interface{}{
		"state.v2":    fmt.Sprintf("%s/api/v2/", s.URL),
		"tfe.v2.1":    fmt.Sprintf("%s/api/v2/", s.URL),
		"versions.v1": fmt.Sprintf("%s/v1/versions/", s.URL),
	}
	d := disco.NewWithCredentialsSource(auth.NoCredentials)
	d.SetUserAgent(httpclient.TerraformUserAgent(version.String()))

	d.ForceHostServices(svchost.Hostname("localhost"), services)
	d.ForceHostServices(svchost.Hostname("app.terraform.io"), services)
	return d
}

// testServer returns a *httptest.Server used for local testing.
// This server simulates the APIs needed to load a remote state.
func testServer(t *testing.T, statePath string) *httptest.Server {
	mux := http.NewServeMux()

	f, err := os.Open(statePath)
	if err != nil {
		t.Fatalf("failed to open state file: %s", err)
	}

	// Respond to service discovery calls.
	mux.HandleFunc("/well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
  "state.v2": "/api/v2/",
  "tfe.v2.1": "/api/v2/",
  "versions.v1": "/v1/versions/"
	}`)
	})

	// Respond to service version constraints calls.
	mux.HandleFunc("/v1/versions/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`{
  "service": "%s",
  "product": "terraform",
  "minimum": "0.1.0",
  "maximum": "10.0.0"
}`, path.Base(r.URL.Path)))
	})

	// Respond to pings to get the API version header.
	mux.HandleFunc("/api/v2/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("TFP-API-Version", "2.4")
	})

	// Respond to the initial query to read the hashicorp org entitlements.
	mux.HandleFunc("/api/v2/organizations/hashicorp/entitlement-set", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": {
    "id": "org-GExadygjSbKP8hsY",
    "type": "entitlement-sets",
    "attributes": {
      "operations": true,
      "private-module-registry": true,
      "sentinel": true,
      "state-storage": true,
      "teams": true,
      "vcs-integrations": true
    }
  }
}`)
	})

	// Respond to the initial query to read the no-operations org entitlements.
	mux.HandleFunc("/api/v2/organizations/no-operations/entitlement-set", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": {
    "id": "org-ufxa3y8jSbKP8hsT",
    "type": "entitlement-sets",
    "attributes": {
      "operations": false,
      "private-module-registry": true,
      "sentinel": true,
      "state-storage": true,
      "teams": true,
      "vcs-integrations": true
    }
  }
}`)
	})

	mux.HandleFunc("/api/v2/organizations/hashicorp/workspaces/my-app-test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{
	"data": {
		"id": "ws-EUht4zmoJaZTZMv8",
		"type": "workspaces",
		"attributes": {
			"locked": false,
			"name": "my-app-test",
			"queue-all-runs": false,
			"speculative-enabled": true,
			"structured-run-output-enabled": true,
			"terraform-version": "1.9.4",
			"operations": true,
			"execution-mode": "remote",
			"file-triggers-enabled": true,
			"locked-reason": "",
			"source": "terraform"
		}
	}
}`)
	})

	mux.HandleFunc("/api/v2/workspaces/ws-EUht4zmoJaZTZMv8/actions/lock", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{
    "data": {
        "id": "ws-EUht4zmoJaZTZMv8",
        "type": "workspaces",
        "attributes": {
            "locked": true,
            "name": "my-app-test",
            "queue-all-runs": false,
            "speculative-enabled": true,
            "structured-run-output-enabled": true,
            "terraform-version": "1.9.4",
            "source": "terraform",
            "source-name": null,
            "source-url": null,
            "tag-names": []
        }
    }
}`)
	})

	mux.HandleFunc("/api/v2/workspaces/ws-EUht4zmoJaZTZMv8/current-state-version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `
		{
			"data": {
            "id": "sv-XJmHFY12zJFmwkWN",
            "type": "state-versions",
            "attributes": {
                "created-at": "2025-02-12T14:16:43.541Z",
                "size": 878,
                "hosted-state-download-url": "/api/state-versions/sv-XJmHFY12zJFmwkWN/hosted_state",
                "hosted-json-state-download-url": "/api/state-versions/sv-XJmHFY12zJFmwkWN/hosted_json_state",
                "serial": 1,
                "state-version": 4,
                "status": "finalized",
                "terraform-version": "1.9.4"
            }
        }
		}
		`)
	})

	mux.HandleFunc("/api/state-versions/sv-XJmHFY12zJFmwkWN/hosted_state", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.Copy(w, f)
	})

	mux.HandleFunc("/api/v2/workspaces/ws-EUht4zmoJaZTZMv8/actions/unlock", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{
	"data": {
		"id": "ws-EUht4zmoJaZTZMv8",
		"type": "workspaces",
		"attributes": {
			"locked": false,
			"name": "my-app-test",
			"queue-all-runs": false,
			"speculative-enabled": true,
			"structured-run-output-enabled": true,
			"terraform-version": "1.9.4",
			"source": "terraform",
			"source-name": null,
			"source-url": null,
			"tag-names": []
		}
	}
}`)
	})

	return httptest.NewServer(mux)
}
