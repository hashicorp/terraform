// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	backendCloud "github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
	"google.golang.org/grpc/metadata"
)

func newCloudPluginManifestHTTPTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	initialPath, _ := os.Getwd()

	mux.HandleFunc("/api/cloudplugin/v1/manifest", func(w http.ResponseWriter, r *http.Request) {
		fileToSend, _ := os.Open(path.Join(initialPath, "testdata/cloud-archives/manifest.json"))
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
}`, path.Base(r.URL.Path)))
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

	mux.HandleFunc("/api/v2/organizations/hashicorp/workspaces/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		io.WriteString(w, `{
  "data": {
    "id": "ws-GExadygjSbKP8hsY",
    "type": "workspaces",
    "attributes": {
			"name": "test",
      "terraform-version": "1.5.4"
    }
  }
}`)
	})

	return httptest.NewServer(mux)
}

// testDisco returns a *disco.Disco mapping app.terraform.io and
// localhost to a local test server.
func testDisco(s *httptest.Server) *disco.Disco {
	host, _ := url.Parse(s.URL)
	defaultHostname := "app.terraform.io"
	tfeHost := svchost.Hostname(defaultHostname)
	services := map[string]interface{}{
		"cloudplugin.v1": fmt.Sprintf("%s/api/cloudplugin/v1/", s.URL),
		"tfe.v2":         fmt.Sprintf("%s/api/v2/", s.URL),
	}

	credsSrc := auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
		tfeHost: {"token": "test-auth-token"},
	})

	d := disco.NewWithCredentialsSource(credsSrc)
	d.SetUserAgent(httpclient.TerraformUserAgent(version.String()))
	d.ForceHostServices(tfeHost, services)
	d.ForceHostServices(svchost.Hostname(host.Host), services)

	return d
}

func TestCloud_withBackendConfig(t *testing.T) {
	t.Skip("To be converted to an e2e test")

	server := newCloudPluginManifestHTTPTestServer(t)
	disco := testDisco(server)

	wd := tempWorkingDirFixture(t, "cloud-config")
	defer testChdir(t, wd.RootModuleDir())()

	// Overwrite the cloud backend with the test disco
	previousBackend := backendInit.Backend("cloud")
	backendInit.Set("cloud", func() backend.Backend { return backendCloud.New(disco) })
	defer backendInit.Set("cloud", previousBackend)

	ui := cli.NewMockUi()
	view, _ := testView(t)

	// Initialize the backend
	ic := &InitCommand{
		Meta{
			Ui:               ui,
			View:             view,
			testingOverrides: metaOverridesForProvider(testProvider()),
			Services:         disco,
		},
	}

	log.Print("[TRACE] TestCloud_withBackendConfig running: terraform init")
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	// Run the cloud command
	ui = cli.NewMockUi()
	c := &CloudCommand{
		Meta: Meta{
			Ui:               ui,
			testingOverrides: metaOverridesForProvider(testProvider()),
			Services:         disco,
			WorkingDir:       wd,
		},
	}

	args := []string{"version"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("expected exit 0, got %d: \n%s", code, ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	expected := "HCP Terraform Plugin v0.1.0\n\n"
	if output != expected {
		t.Fatalf("the output did not equal the expected string:\n%s", cmp.Diff(expected, output))
	}
}

func TestCloud_withENVConfig(t *testing.T) {
	t.Skip("To be converted to an e2e test")

	server := newCloudPluginManifestHTTPTestServer(t)
	disco := testDisco(server)

	wd := tempWorkingDir(t)
	defer testChdir(t, wd.RootModuleDir())()

	serverURL, _ := url.Parse(server.URL)

	os.Setenv("TF_CLOUD_HOSTNAME", serverURL.Host)
	defer os.Unsetenv("TF_CLOUD_HOSTNAME")

	// Run the cloud command
	ui := cli.NewMockUi()
	c := &CloudCommand{
		Meta: Meta{
			Ui:               ui,
			testingOverrides: metaOverridesForProvider(testProvider()),
			Services:         disco,
			WorkingDir:       wd,
		},
	}

	args := []string{"version"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("expected exit 0, got %d: \n%s", code, ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	expected := "HCP Terraform Plugin v0.1.0\n\n"
	if output != expected {
		t.Fatalf("the output did not equal the expected string:\n%s", cmp.Diff(expected, output))
	}
}

func TestCloudPluginConfig_ToMetadata(t *testing.T) {
	expected := metadata.Pairs(
		"tfc-address", "https://app.staging.terraform.io",
		"tfc-base-path", "/api/v2/",
		"tfc-display-hostname", "app.staging.terraform.io",
		"tfc-token", "not-a-legit-token",
		"tfc-organization", "example-corp",
		"tfc-current-workspace", "example-space",
		"tfc-workspace-name", "example-space",
		// Actually combining -name and -tags is an invalid scenario from
		// Terraform's point of view, but here we're just testing that every
		// field makes the trip safely if sent.
		"tfc-workspace-tags", "networking",
		// Duplicate is on purpose.
		"tfc-workspace-tags", "platform-team",
		"tfc-default-project-name", "production-services",
	)
	inputStruct := CloudPluginConfig{
		Address:            "https://app.staging.terraform.io",
		BasePath:           "/api/v2/",
		DisplayHostname:    "app.staging.terraform.io",
		Token:              "not-a-legit-token",
		Organization:       "example-corp",
		CurrentWorkspace:   "example-space",
		WorkspaceName:      "example-space",
		WorkspaceTags:      []string{"networking", "platform-team"},
		DefaultProjectName: "production-services",
	}
	result := inputStruct.ToMetadata()
	if !reflect.DeepEqual(expected, result) {
		t.Fatalf("Expected: %#v\nGot: %#v\n", expected, result)
	}
}
