package cloud

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-svchost/disco"
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

func TestCloud_backendWithPrefix(t *testing.T) {
	b, bCleanup := testBackendWithPrefix(t)
	defer bCleanup()

	backend.TestBackendStates(t, b)
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid organization value: The "organization" attribute value must not be empty.`,
		},
		"null workspace": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces":   cty.NullVal(cty.String),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags", "name", or "prefix" is required.`,
		},
		"workspace: empty tags, name, and prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags", "name", or "prefix" is required.`,
		},
		"workspace: name and prefix present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.StringVal("app-"),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags", "name", or "prefix" is allowed.`,
		},
		"workspace: name and tags present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags", "name", or "prefix" is allowed.`,
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
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
					"name":   cty.NullVal(cty.String),
					"prefix": cty.NullVal(cty.String),
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
		},
		"with_a_prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.StringVal("my-app-"),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
		},
		"without_a_name_prefix_or_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			valErr: `Missing workspace mapping strategy.`,
		},
		"with_both_a_name_and_a_prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.StringVal("my-app-"),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			valErr: `Only one of workspace "tags", "name", or "prefix" is allowed.`,
		},
		"with_both_a_name_and_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
				}),
			}),
			valErr: `Only one of workspace "tags", "name", or "prefix" is allowed.`,
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

func TestCloud_setConfigurationFields(t *testing.T) {
	originalForceBackendEnv := os.Getenv("TF_FORCE_LOCAL_BACKEND")

	cases := map[string]struct {
		obj                     cty.Value
		expectedHostname        string
		expectedOrganziation    string
		expectedWorkspacePrefix string
		expectedWorkspaceName   string
		expectedWorkspaceTags   []string
		expectedForceLocal      bool
		setEnv                  func()
		resetEnv                func()
		expectedErr             string
	}{
		"with hostname set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
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
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:      "hashicorp.com",
			expectedOrganziation:  "hashicorp",
			expectedWorkspaceName: "prod",
		},
		"with workspace prefix set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.StringVal("prod"),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:        "hashicorp.com",
			expectedOrganziation:    "hashicorp",
			expectedWorkspacePrefix: "prod",
		},
		"with workspace tags set": {
			obj: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.NullVal(cty.String),
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
					"name":   cty.NullVal(cty.String),
					"prefix": cty.StringVal("prod"),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			expectedHostname:        "hashicorp.com",
			expectedOrganziation:    "hashicorp",
			expectedWorkspacePrefix: "prod",
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
		if tc.expectedWorkspacePrefix != "" && b.WorkspaceMapping.Prefix != tc.expectedWorkspacePrefix {
			t.Fatalf("%s: expected workspace prefix mapping (%s) to match configured workspace prefix (%s)", name, b.WorkspaceMapping.Prefix, tc.expectedWorkspacePrefix)
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

func TestCloud_versionConstraints(t *testing.T) {
	cases := map[string]struct {
		config     cty.Value
		prerelease string
		version    string
		result     string
	}{
		"compatible version": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			version: "0.11.1",
		},
		"version too old": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			version: "0.0.1",
			result:  "upgrade Terraform to >= 0.1.0",
		},
		"version too new": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
					"tags":   cty.NullVal(cty.Set(cty.String)),
				}),
			}),
			version: "10.0.1",
			result:  "downgrade Terraform to <= 10.0.0",
		},
	}

	// Save and restore the actual version.
	p := tfversion.Prerelease
	v := tfversion.Version
	defer func() {
		tfversion.Prerelease = p
		tfversion.Version = v
	}()

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Set the version for this test.
		tfversion.Prerelease = tc.prerelease
		tfversion.Version = tc.version

		// Validate
		_, valDiags := b.PrepareConfig(tc.config)
		if valDiags.HasErrors() {
			t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
		}

		// Configure
		confDiags := b.Configure(tc.config)
		if (confDiags.Err() != nil || tc.result != "") &&
			(confDiags.Err() == nil || !strings.Contains(confDiags.Err().Error(), tc.result)) {
			t.Fatalf("%s: unexpected configure result: %v", name, confDiags.Err())
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

func TestCloud_addAndRemoveWorkspacesWithPrefix(t *testing.T) {
	b, bCleanup := testBackendWithPrefix(t)
	defer bCleanup()

	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces := []string(nil)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected states %#+v, got %#+v", expectedWorkspaces, states)
	}

	if _, err := b.StateMgr(backend.DefaultStateName); err != backend.ErrDefaultWorkspaceNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultWorkspaceNotSupported, err)
	}

	expectedA := "test_A"
	if _, err := b.StateMgr(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = append(expectedWorkspaces, expectedA)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}

	expectedB := "test_B"
	if _, err := b.StateMgr(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = append(expectedWorkspaces, expectedB)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}

	if err := b.DeleteWorkspace(backend.DefaultStateName); err != backend.ErrDefaultWorkspaceNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultWorkspaceNotSupported, err)
	}

	if err := b.DeleteWorkspace(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = []string{expectedB}
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v got %#+v", expectedWorkspaces, states)
	}

	if err := b.DeleteWorkspace(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = []string(nil)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}
}

func TestCloud_checkConstraints(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	cases := map[string]struct {
		constraints *disco.Constraints
		prerelease  string
		version     string
		result      string
	}{
		"compatible version": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.11.1",
			result:  "",
		},
		"version too old": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.10.1",
			result:  "upgrade Terraform to >= 0.11.0",
		},
		"version too new": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.12.0",
			result:  "downgrade Terraform to <= 0.11.11",
		},
		"version excluded - ordered": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.7", "0.11.8"},
				Maximum:   "0.11.11",
			},
			version: "0.11.7",
			result:  "upgrade Terraform to > 0.11.8",
		},
		"version excluded - unordered": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.8", "0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "upgrade Terraform to > 0.11.8",
		},
		"list versions": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.10.1",
			result:  "versions >= 0.11.0, <= 0.11.11.",
		},
		"list exclusion": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "excluding version 0.11.6.",
		},
		"list exclusions": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.8", "0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "excluding versions 0.11.6, 0.11.8.",
		},
	}

	// Save and restore the actual version.
	p := tfversion.Prerelease
	v := tfversion.Version
	defer func() {
		tfversion.Prerelease = p
		tfversion.Version = v
	}()

	for name, tc := range cases {
		// Set the version for this test.
		tfversion.Prerelease = tc.prerelease
		tfversion.Version = tc.version

		// Check the constraints.
		diags := b.checkConstraints(tc.constraints)
		if (diags.Err() != nil || tc.result != "") &&
			(diags.Err() == nil || !strings.Contains(diags.Err().Error(), tc.result)) {
			t.Fatalf("%s: unexpected constraints result: %v", name, diags.Err())
		}
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
		local      string
		remote     string
		operations bool
		wantErr    bool
	}{
		{"0.13.5", "0.13.5", true, false},
		{"0.14.0", "0.13.5", true, true},
		{"0.14.0", "0.13.5", false, false},
		{"0.14.0", "0.14.1", true, false},
		{"0.14.0", "1.0.99", true, false},
		{"0.14.0", "1.1.0", true, false},
		{"0.14.0", "1.2.0", true, true},
		{"1.2.0", "1.2.99", true, false},
		{"1.2.0", "1.3.0", true, true},
		{"0.15.0", "latest", true, false},
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
					Operations:       tfe.Bool(tc.operations),
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
				if got := diags.Err().Error(); !strings.Contains(got, "Terraform version mismatch") {
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
	if got := diags.Err().Error(); !strings.Contains(got, "Error looking up workspace: Invalid Terraform version") {
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
	if got, want := diags[0].Description().Summary, "Terraform version mismatch"; got != want {
		t.Errorf("wrong summary: got %s, want %s", got, want)
	}
	wantDetail := "The local Terraform version (0.14.0) does not match the configured version for remote workspace hashicorp/app-prod (0.13.5)."
	if got := diags[0].Description().Detail; got != wantDetail {
		t.Errorf("wrong summary: got %s, want %s", got, wantDetail)
	}
}
