package cloud

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"

	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
)

func TestCloud(t *testing.T) {
	var _ backend.Enhanced = New(nil)
	var _ backend.CLI = New(nil)
}

func TestCloud_backendWithName(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	workspaces, err := b.Workspaces()
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(workspaces) != 1 || workspaces[0] != testBackendSingleWorkspaceName {
		t.Fatalf("should only have a single configured workspace matching the configured 'name' strategy, but got: %#v", workspaces)
	}

	if _, err := b.StateMgr("foo"); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected fetching a state which is NOT the single configured workspace to have an ErrWorkspacesNotSupported error, but got: %v", err)
	}

	if err := b.DeleteWorkspace(testBackendSingleWorkspaceName); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected deleting the single configured workspace name to result in an error, but got: %v", err)
	}

	if err := b.DeleteWorkspace("foo"); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected deleting a workspace which is NOT the configured workspace name to result in an error, but got: %v", err)
	}
}

func TestCloud_backendWithTags(t *testing.T) {
	b, bCleanup := testBackendWithTags(t)
	defer bCleanup()

	backend.TestBackendStates(t, b)

	// Test pagination works
	for i := 0; i < 25; i++ {
		_, err := b.StateMgr(fmt.Sprintf("foo-%d", i+1))
		if err != nil {
			t.Fatalf("error: %s", err)
		}
	}

	workspaces, err := b.Workspaces()
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	actual := len(workspaces)
	if actual != 26 {
		t.Errorf("expected 26 workspaces (over one standard paginated response), got %d", actual)
	}
}

func TestCloud_PrepareConfig(t *testing.T) {
	cases := map[string]struct {
		config      cty.Value
		expectedErr string
	}{
		"null organization": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid organization value: The "organization" attribute value must not be empty.`,
		},
		"null workspace": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces":   cty.NullVal(cty.String),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags" or "name" is required.`,
		},
		"workspace: empty tags, name": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags" or "name" is required.`,
		},
		"workspace: name present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags" or "name" is allowed.`,
		},
		"workspace: name and tags present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags" or "name" is allowed.`,
		},
	}

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Validate
		_, valDiags := b.PrepareConfig(tc.config)
		if valDiags.Err() != nil && tc.expectedErr != "" {
			actualErr := valDiags.Err().Error()
			if !strings.Contains(actualErr, tc.expectedErr) {
				t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
			}
		}
	}
}

func TestCloud_config(t *testing.T) {
	cases := map[string]struct {
		config  cty.Value
		confErr string
		valErr  string
	}{
		"with_a_nonexisting_organization": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("nonexisting"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			confErr: "organization \"nonexisting\" at host app.terraform.io not found",
		},
		"with_an_unknown_host": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("nonexisting.local"),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			confErr: "Failed to request discovery document",
		},
		// localhost advertises TFE services, but has no token in the credentials
		"without_a_token": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("localhost"),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			confErr: "terraform login localhost",
		},
		"with_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
		},
		"with_a_name": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
		},
		"without_a_name_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			valErr: `Missing workspace mapping strategy.`,
		},
		"with_both_a_name_and_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
			valErr: `Only one of workspace "tags" or "name" is allowed.`,
		},
		"null config": {
			config: cty.NullVal(cty.EmptyObject),
		},
	}

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Validate
		_, valDiags := b.PrepareConfig(tc.config)
		if (valDiags.Err() != nil || tc.valErr != "") &&
			(valDiags.Err() == nil || !strings.Contains(valDiags.Err().Error(), tc.valErr)) {
			t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
		}

		// Configure
		confDiags := b.Configure(tc.config)
		if (confDiags.Err() != nil || tc.confErr != "") &&
			(confDiags.Err() == nil || !strings.Contains(confDiags.Err().Error(), tc.confErr)) {
			t.Fatalf("%s: unexpected configure result: %v", name, confDiags.Err())
		}
	}
}

func TestCloud_configVerifyMinimumTFEVersion(t *testing.T) {
	config := cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String),
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name": cty.NullVal(cty.String),
			"tags": cty.SetVal(
				[]cty.Value{
					cty.StringVal("billing"),
				},
			),
		}),
	})

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("TFP-API-Version", "2.4")
		},
	}
	s := testServerWithHandlers(handlers)

	b := New(testDisco(s))

	confDiags := b.Configure(config)
	if confDiags.Err() == nil {
		t.Fatalf("expected configure to error")
	}

	expected := `The 'cloud' option is not supported with this version of Terraform Enterprise.`
	if !strings.Contains(confDiags.Err().Error(), expected) {
		t.Fatalf("expected configure to error with %q, got %q", expected, confDiags.Err().Error())
	}
}

func TestCloud_configVerifyMinimumTFEVersionInAutomation(t *testing.T) {
	config := cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String),
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name": cty.NullVal(cty.String),
			"tags": cty.SetVal(
				[]cty.Value{
					cty.StringVal("billing"),
				},
			),
		}),
	})

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/ping": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("TFP-API-Version", "2.4")
		},
	}
	s := testServerWithHandlers(handlers)

	b := New(testDisco(s))
	b.runningInAutomation = true

	confDiags := b.Configure(config)
	if confDiags.Err() == nil {
		t.Fatalf("expected configure to error")
	}

	expected := `This version of Terraform Cloud/Enterprise does not support the state mechanism
attempting to be used by the platform. This should never happen.`
	if !strings.Contains(confDiags.Err().Error(), expected) {
		t.Fatalf("expected configure to error with %q, got %q", expected, confDiags.Err().Error())
	}
}

func TestCloud_setUnavailableTerraformVersion(t *testing.T) {
	// go-tfe returns an error IRL if you try to set a Terraform version that's
	// not available in your TFC instance. To test this, tfe_client_mock errors if
	// you try to set any Terraform version for this specific workspace name.
	workspaceName := "unavailable-terraform-version"

	config := cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String),
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name": cty.NullVal(cty.String),
			"tags": cty.SetVal(
				[]cty.Value{
					cty.StringVal("sometag"),
				},
			),
		}),
	})

	b, bCleanup := testBackend(t, config)
	defer bCleanup()

	// Make sure the workspace doesn't exist yet -- otherwise, we can't test what
	// happens when a workspace gets created. This is why we can't use "name" in
	// the backend config above, btw: if you do, testBackend() creates the default
	// workspace before we get a chance to do anything.
	_, err := b.client.Workspaces.Read(context.Background(), b.organization, workspaceName)
	if err != tfe.ErrResourceNotFound {
		t.Fatalf("the workspace we were about to try and create (%s/%s) already exists in the mocks somehow, so this test isn't trustworthy anymore", b.organization, workspaceName)
	}

	_, err = b.StateMgr(workspaceName)
	if err != nil {
		t.Fatalf("expected no error from StateMgr, despite not being able to set remote Terraform version: %#v", err)
	}
	// Make sure the workspace was created:
	workspace, err := b.client.Workspaces.Read(context.Background(), b.organization, workspaceName)
	if err != nil {
		t.Fatalf("b.StateMgr() didn't actually create the desired workspace")
	}
	// Make sure our mocks still error as expected, using the same update function b.StateMgr() would call:
	_, err = b.client.Workspaces.UpdateByID(
		context.Background(),
		workspace.ID,
		tfe.WorkspaceUpdateOptions{TerraformVersion: tfe.String("1.1.0")},
	)
	if err == nil {
		t.Fatalf("the mocks aren't emulating a nonexistent remote Terraform version correctly, so this test isn't trustworthy anymore")
	}
}

func TestCloud_setConfigurationFields(t *testing.T) {
	originalForceBackendEnv := os.Getenv("TF_FORCE_LOCAL_BACKEND")

	cases := map[string]struct {
		obj                   cty.Value
		expectedHostname      string
		expectedOrganziation  string
		expectedWorkspaceName string
		expectedWorkspaceTags []string
		expectedForceLocal    bool
		setEnv                func()
		resetEnv              func()
		expectedErr           string
	}{
		"with hostname set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:     "hashicorp.com",
			expectedOrganziation: "hashicorp",
		},
		"with hostname not set, set to default hostname": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:     defaultHostname,
			expectedOrganziation: "hashicorp",
		},
		"with workspace name set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:      "hashicorp.com",
			expectedOrganziation:  "hashicorp",
			expectedWorkspaceName: "prod",
		},
		"with workspace tags set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
			expectedHostname:      "hashicorp.com",
			expectedOrganziation:  "hashicorp",
			expectedWorkspaceTags: []string{"billing"},
		},
		"with force local set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:     "hashicorp.com",
			expectedOrganziation: "hashicorp",
			setEnv: func() {
				os.Setenv("TF_FORCE_LOCAL_BACKEND", "1")
			},
			resetEnv: func() {
				os.Setenv("TF_FORCE_LOCAL_BACKEND", originalForceBackendEnv)
			},
			expectedForceLocal: true,
		},
	}

	for name, tc := range cases {
		b := &Cloud{}

		// if `setEnv` is set, then we expect `resetEnv` to also be set
		if tc.setEnv != nil {
			tc.setEnv()
			defer tc.resetEnv()
		}

		errDiags := b.setConfigurationFields(tc.obj)
		if errDiags.HasErrors() || tc.expectedErr != "" {
			actualErr := errDiags.Err().Error()
			if !strings.Contains(actualErr, tc.expectedErr) {
				t.Fatalf("%s: unexpected validation result: %v", name, errDiags.Err())
			}
		}

		if tc.expectedHostname != "" && b.hostname != tc.expectedHostname {
			t.Fatalf("%s: expected hostname %s to match configured hostname %s", name, b.hostname, tc.expectedHostname)
		}
		if tc.expectedOrganziation != "" && b.organization != tc.expectedOrganziation {
			t.Fatalf("%s: expected organization (%s) to match configured organization (%s)", name, b.organization, tc.expectedOrganziation)
		}
		if tc.expectedWorkspaceName != "" && b.WorkspaceMapping.Name != tc.expectedWorkspaceName {
			t.Fatalf("%s: expected workspace name mapping (%s) to match configured workspace name (%s)", name, b.WorkspaceMapping.Name, tc.expectedWorkspaceName)
		}
		if len(tc.expectedWorkspaceTags) > 0 {
			presentSet := make(map[string]struct{})
			for _, tag := range b.WorkspaceMapping.Tags {
				presentSet[tag] = struct{}{}
			}

			expectedSet := make(map[string]struct{})
			for _, tag := range tc.expectedWorkspaceTags {
				expectedSet[tag] = struct{}{}
			}

			var missing []string
			var unexpected []string

			for _, expected := range tc.expectedWorkspaceTags {
				if _, ok := presentSet[expected]; !ok {
					missing = append(missing, expected)
				}
			}

			for _, actual := range b.WorkspaceMapping.Tags {
				if _, ok := expectedSet[actual]; !ok {
					unexpected = append(missing, actual)
				}
			}

			if len(missing) > 0 {
				t.Fatalf("%s: expected workspace tag mapping (%s) to contain the following tags: %s", name, b.WorkspaceMapping.Tags, missing)
			}

			if len(unexpected) > 0 {
				t.Fatalf("%s: expected workspace tag mapping (%s) to NOT contain the following tags: %s", name, b.WorkspaceMapping.Tags, unexpected)
			}

		}
		if tc.expectedForceLocal != false && b.forceLocal != tc.expectedForceLocal {
			t.Fatalf("%s: expected force local backend to be set ", name)
		}
	}
}

func TestCloud_localBackend(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	local, ok := b.local.(*backendLocal.Local)
	if !ok {
		t.Fatalf("expected b.local to be \"*local.Local\", got: %T", b.local)
	}

	cloud, ok := local.Backend.(*Cloud)
	if !ok {
		t.Fatalf("expected local.Backend to be *cloud.Cloud, got: %T", cloud)
	}
}

func TestCloud_addAndRemoveWorkspacesDefault(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	if _, err := b.StateMgr(testBackendSingleWorkspaceName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := b.DeleteWorkspace(testBackendSingleWorkspaceName); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrWorkspacesNotSupported, err)
	}
}

func TestCloud_StateMgr_versionCheck(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	// Some fixed versions for testing with. This logic is a simple string
	// comparison, so we don't need many test cases.
	v0135 := version.Must(version.NewSemver("0.13.5"))
	v0140 := version.Must(version.NewSemver("0.14.0"))

	// Save original local version state and restore afterwards
	p := tfversion.Prerelease
	v := tfversion.Version
	s := tfversion.SemVer
	defer func() {
		tfversion.Prerelease = p
		tfversion.Version = v
		tfversion.SemVer = s
	}()

	// For this test, the local Terraform version is set to 0.14.0
	tfversion.Prerelease = ""
	tfversion.Version = v0140.String()
	tfversion.SemVer = v0140

	// Update the mock remote workspace Terraform version to match the local
	// Terraform version
	if _, err := b.client.Workspaces.Update(
		context.Background(),
		b.organization,
		b.WorkspaceMapping.Name,
		tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String(v0140.String()),
		},
	); err != nil {
		t.Fatalf("error: %v", err)
	}

	// This should succeed
	if _, err := b.StateMgr(testBackendSingleWorkspaceName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Now change the remote workspace to a different Terraform version
	if _, err := b.client.Workspaces.Update(
		context.Background(),
		b.organization,
		b.WorkspaceMapping.Name,
		tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String(v0135.String()),
		},
	); err != nil {
		t.Fatalf("error: %v", err)
	}

	// This should fail
	want := `Remote workspace Terraform version "0.13.5" does not match local Terraform version "0.14.0"`
	if _, err := b.StateMgr(testBackendSingleWorkspaceName); err.Error() != want {
		t.Fatalf("wrong error\n got: %v\nwant: %v", err.Error(), want)
	}
}

func TestCloud_StateMgr_versionCheckLatest(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	v0140 := version.Must(version.NewSemver("0.14.0"))

	// Save original local version state and restore afterwards
	p := tfversion.Prerelease
	v := tfversion.Version
	s := tfversion.SemVer
	defer func() {
		tfversion.Prerelease = p
		tfversion.Version = v
		tfversion.SemVer = s
	}()

	// For this test, the local Terraform version is set to 0.14.0
	tfversion.Prerelease = ""
	tfversion.Version = v0140.String()
	tfversion.SemVer = v0140

	// Update the remote workspace to the pseudo-version "latest"
	if _, err := b.client.Workspaces.Update(
		context.Background(),
		b.organization,
		b.WorkspaceMapping.Name,
		tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String("latest"),
		},
	); err != nil {
		t.Fatalf("error: %v", err)
	}

	// This should succeed despite not being a string match
	if _, err := b.StateMgr(testBackendSingleWorkspaceName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCloud_VerifyWorkspaceTerraformVersion(t *testing.T) {
	testCases := []struct {
		local         string
		remote        string
		executionMode string
		wantErr       bool
	}{
		{"0.13.5", "0.13.5", "agent", false},
		{"0.14.0", "0.13.5", "remote", true},
		{"0.14.0", "0.13.5", "local", false},
		{"0.14.0", "0.14.1", "remote", false},
		{"0.14.0", "1.0.99", "remote", false},
		{"0.14.0", "1.1.0", "remote", false},
		{"0.14.0", "1.2.0", "remote", true},
		{"1.2.0", "1.2.99", "remote", false},
		{"1.2.0", "1.3.0", "remote", true},
		{"0.15.0", "latest", "remote", false},
		{"1.1.5", "~> 1.1.1", "remote", false},
		{"1.1.5", "> 1.1.0, < 1.3.0", "remote", false},
		{"1.1.5", "~> 1.0.1", "remote", true},
		// pre-release versions are comparable within their pre-release stage (dev,
		// alpha, beta), but not comparable to different stages and not comparable
		// to final releases.
		{"1.1.0-beta1", "1.1.0-beta1", "remote", false},
		{"1.1.0-beta1", "~> 1.1.0-beta", "remote", false},
		{"1.1.0", "~> 1.1.0-beta", "remote", true},
		{"1.1.0-beta1", "~> 1.1.0-dev", "remote", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("local %s, remote %s", tc.local, tc.remote), func(t *testing.T) {
			b, bCleanup := testBackendWithName(t)
			defer bCleanup()

			local := version.Must(version.NewSemver(tc.local))

			// Save original local version state and restore afterwards
			p := tfversion.Prerelease
			v := tfversion.Version
			s := tfversion.SemVer
			defer func() {
				tfversion.Prerelease = p
				tfversion.Version = v
				tfversion.SemVer = s
			}()

			// Override local version as specified
			tfversion.Prerelease = ""
			tfversion.Version = local.String()
			tfversion.SemVer = local

			// Update the mock remote workspace Terraform version to the
			// specified remote version
			if _, err := b.client.Workspaces.Update(
				context.Background(),
				b.organization,
				b.WorkspaceMapping.Name,
				tfe.WorkspaceUpdateOptions{
					ExecutionMode:    &tc.executionMode,
					TerraformVersion: tfe.String(tc.remote),
				},
			); err != nil {
				t.Fatalf("error: %v", err)
			}

			diags := b.VerifyWorkspaceTerraformVersion(backend.DefaultStateName)
			if tc.wantErr {
				if len(diags) != 1 {
					t.Fatal("expected diag, but none returned")
				}
				if got := diags.Err().Error(); !strings.Contains(got, "Incompatible Terraform version") {
					t.Fatalf("unexpected error: %s", got)
				}
			} else {
				if len(diags) != 0 {
					t.Fatalf("unexpected diags: %s", diags.Err())
				}
			}
		})
	}
}

func TestCloud_VerifyWorkspaceTerraformVersion_workspaceErrors(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	// Attempting to check the version against a workspace which doesn't exist
	// should result in no errors
	diags := b.VerifyWorkspaceTerraformVersion("invalid-workspace")
	if len(diags) != 0 {
		t.Fatalf("unexpected error: %s", diags.Err())
	}

	// Use a special workspace ID to trigger a 500 error, which should result
	// in a failed check
	diags = b.VerifyWorkspaceTerraformVersion("network-error")
	if len(diags) != 1 {
		t.Fatal("expected diag, but none returned")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "Error looking up workspace: Workspace read failed") {
		t.Fatalf("unexpected error: %s", got)
	}

	// Update the mock remote workspace Terraform version to an invalid version
	if _, err := b.client.Workspaces.Update(
		context.Background(),
		b.organization,
		b.WorkspaceMapping.Name,
		tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String("1.0.cheetarah"),
		},
	); err != nil {
		t.Fatalf("error: %v", err)
	}
	diags = b.VerifyWorkspaceTerraformVersion(backend.DefaultStateName)

	if len(diags) != 1 {
		t.Fatal("expected diag, but none returned")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "Incompatible Terraform version: The remote workspace specified") {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestCloud_VerifyWorkspaceTerraformVersion_ignoreFlagSet(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	// If the ignore flag is set, the behaviour changes
	b.IgnoreVersionConflict()

	// Different local & remote versions to cause an error
	local := version.Must(version.NewSemver("0.14.0"))
	remote := version.Must(version.NewSemver("0.13.5"))

	// Save original local version state and restore afterwards
	p := tfversion.Prerelease
	v := tfversion.Version
	s := tfversion.SemVer
	defer func() {
		tfversion.Prerelease = p
		tfversion.Version = v
		tfversion.SemVer = s
	}()

	// Override local version as specified
	tfversion.Prerelease = ""
	tfversion.Version = local.String()
	tfversion.SemVer = local

	// Update the mock remote workspace Terraform version to the
	// specified remote version
	if _, err := b.client.Workspaces.Update(
		context.Background(),
		b.organization,
		b.WorkspaceMapping.Name,
		tfe.WorkspaceUpdateOptions{
			TerraformVersion: tfe.String(remote.String()),
		},
	); err != nil {
		t.Fatalf("error: %v", err)
	}

	diags := b.VerifyWorkspaceTerraformVersion(backend.DefaultStateName)
	if len(diags) != 1 {
		t.Fatal("expected diag, but none returned")
	}

	if got, want := diags[0].Severity(), tfdiags.Warning; got != want {
		t.Errorf("wrong severity: got %#v, want %#v", got, want)
	}
	if got, want := diags[0].Description().Summary, "Incompatible Terraform version"; got != want {
		t.Errorf("wrong summary: got %s, want %s", got, want)
	}
	wantDetail := "The local Terraform version (0.14.0) does not meet the version requirements for remote workspace hashicorp/app-prod (0.13.5)."
	if got := diags[0].Description().Detail; got != wantDetail {
		t.Errorf("wrong summary: got %s, want %s", got, wantDetail)
	}
}
