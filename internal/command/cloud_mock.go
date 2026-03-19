// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func cloudTestServerWithVars(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	initialPath, _ := os.Getwd()

	mux.HandleFunc("/api/cloudplugin/v1/manifest", func(w http.ResponseWriter, r *http.Request) {
		fileToSend, _ := os.Open(filepath.Join(initialPath, "testdata/cloud-archives/manifest.json"))
		defer fileToSend.Close()
		io.Copy(w, fileToSend)
	})

	// Respond to service version constraints calls.
	mux.HandleFunc("/v1/versions/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`{
  "service": "%s",
  "product": "terraform",
  "minimum": "0.1.0",
  "maximum": "10.0.0"
}`, filepath.Base(r.URL.Path)))
	})

	// Respond to pings to get the API version header.
	mux.HandleFunc("/api/v2/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("TFP-API-Version", "2.5")
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

	// Respond to our "test" workspace
	mux.HandleFunc("/api/v2/organizations/hashicorp/workspaces/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": {
    "id": "ws-GExadygjSbKP8hsY",
    "type": "workspaces",
    "attributes": {
      "name": "test",
      "terraform-version": "1.15.0",
      "execution-mode": "remote"
    }
  }
}`)
	})

	// Respond to the "default" workspace request, will always be requested by Terraform
	mux.HandleFunc("/api/v2/organizations/hashicorp/workspaces/default", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": {
    "id": "ws-GExadygjSbKP8hsX",
    "type": "workspaces",
    "attributes": {
      "name": "default",
      "terraform-version": "1.15.0",
      "execution-mode": "remote"
    }
  }
}`)
	})

	// Respond to the variables for the "test" workspace:
	// module_name -> example
	mux.HandleFunc("/api/v2/workspaces/ws-GExadygjSbKP8hsY/all-vars", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": [
    {
      "id": "var-1234",
      "type": "vars",
      "attributes": {
        "key": "module_name",
        "value": "example",
        "category": "terraform",
        "hcl": false,
        "sensitive": false
      }
    }
  ]
}`)
	})

	return httptest.NewServer(mux)
}
