// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	version "github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestCloud(t *testing.T) {
	var _ backendrun.OperationsBackend = New(nil)
	var _ backendrun.CLI = New(nil)
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

	if err := b.DeleteWorkspace(testBackendSingleWorkspaceName, true); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected deleting the single configured workspace name to result in an error, but got: %v", err)
	}

	if err := b.DeleteWorkspace("foo", true); err != backend.ErrWorkspacesNotSupported {
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

func TestCloud_backendWithKVTags(t *testing.T) {
	b, bCleanup := testBackendWithKVTags(t)
	defer bCleanup()

	_, err := b.client.Workspaces.Create(context.Background(), "hashicorp", tfe.WorkspaceCreateOptions{
		Name: tfe.String("ws-billing-101"),
		TagBindings: []*tfe.TagBinding{
			{Key: "dept", Value: "billing"},
			{Key: "costcenter", Value: "101"},
		},
	})
	if err != nil {
		t.Fatalf("error creating workspace: %s", err)
	}

	workspaces, err := b.Workspaces()
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	actual := len(workspaces)
	if actual != 1 {
		t.Fatalf("expected 1 workspaces, got %d", actual)
	}

	if workspaces[0] != "ws-billing-101" {
		t.Fatalf("expected workspace name to be 'ws-billing-101', got %s", workspaces[0])
	}
}

func TestCloud_DescribeTags(t *testing.T) {
	cases := map[string]struct {
		expected   string
		expectedOr []string
		config     cty.Value
	}{
		"one set tag": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal([]cty.Value{
						cty.StringVal("billing"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expected: "billing",
		},
		"two set tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal([]cty.Value{
						cty.StringVal("billing"),
						cty.StringVal("cc101"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expected: "billing, cc101",
		},
		"one kv tag": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.MapVal(map[string]cty.Value{
						"dept": cty.StringVal("billing"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expected: "dept=billing",
		},
		"two kv tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.MapVal(map[string]cty.Value{
						"dept":       cty.StringVal("billing"),
						"costcenter": cty.StringVal("101"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedOr: []string{"costcenter=101, dept=billing", "dept=billing, costcenter=101"},
		},
		"no tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("default"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expected: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, cleanup := testUnconfiguredBackend(t)
			t.Cleanup(cleanup)

			_, valDiags := b.PrepareConfig(tc.config)
			if valDiags.Err() != nil {
				t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
			}

			diags := b.Configure(tc.config)
			if diags.Err() != nil {
				t.Fatalf("%s: unexpected configure result: %v", name, diags.Err())
			}

			actual := b.WorkspaceMapping.DescribeTags()
			if tc.expectedOr != nil {
				for _, expected := range tc.expectedOr {
					if actual == expected {
						return
					}
				}
				t.Fatalf("expected one of %v, got %q", tc.expectedOr, actual)
			}

			if actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

// Test b.PrepareConfig, which actually does very little; most real validation
// happens later in resolveCloudConfig.
func TestCloud_PrepareConfig(t *testing.T) {
	cases := map[string]struct {
		config      cty.Value
		expectedErr string
	}{
		"workspace: name and tags conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags" or "name" is allowed.`,
		},
		"totally empty config block is ok": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.NullVal(cty.String),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
		},
	}

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Validate
		_, valDiags := b.PrepareConfig(tc.config)

		if tc.expectedErr != "" {
			if valDiags.Err() == nil {
				t.Fatalf("%s: expected validation error %q but didn't get one", name, tc.expectedErr)
			} else {
				actualErr := valDiags.Err().Error()
				if !strings.Contains(actualErr, tc.expectedErr) {
					t.Fatalf("%s: expected validation error %q but instead got %q", name, tc.expectedErr, actualErr)
				}
			}
		} else {
			if valDiags.Err() != nil {
				t.Fatalf("%s: expected validation to pass but got error %q", name, valDiags.Err().Error())
			}
		}
	}
}

func TestCloud_configWithEnvVars(t *testing.T) {
	cases := map[string]struct {
		setup                 func(b *Cloud)
		config                cty.Value
		vars                  map[string]string
		expectedOrganization  string
		expectedHostname      string
		expectedWorkspaceName string
		expectedProjectName   string
		expectedErr           string
	}{
		"with no organization specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_ORGANIZATION": "hashicorp",
			},
			expectedOrganization: "hashicorp",
		},
		"with both organization and env var specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_ORGANIZATION": "we-should-not-see-this",
			},
			expectedOrganization: "hashicorp",
		},
		"with no hostname specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_HOSTNAME": "private.hashicorp.engineering",
			},
			expectedHostname: "private.hashicorp.engineering",
		},
		"with hostname and env var specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("private.hashicorp.engineering"),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_HOSTNAME": "mycool.tfe-host.io",
			},
			expectedHostname: "private.hashicorp.engineering",
		},
		"an invalid workspace env var": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
			vars: map[string]string{
				"TF_WORKSPACE": "i-dont-exist-in-org",
			},
			expectedErr: `Invalid workspace selection: Terraform failed to find workspace "i-dont-exist-in-org" in organization hashicorp`,
		},
		"workspaces and env var specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("mt-doom"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_WORKSPACE": "shire",
			},
			expectedErr: `conflicts with TF_WORKSPACE environment variable`,
		},
		"env var workspace does not have specified tag": {
			setup: func(b *Cloud) {
				b.client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
					Name: tfe.String("mordor"),
				})

				b.client.Workspaces.Create(context.Background(), "mordor", tfe.WorkspaceCreateOptions{
					Name: tfe.String("shire"),
				})
			},
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal([]cty.Value{
						cty.StringVal("cloud"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_WORKSPACE": "shire",
			},
			expectedErr: "Terraform failed to find workspace \"shire\" with the tags specified in your configuration:\n[cloud]",
		},
		"env var workspace has specified tag": {
			setup: func(b *Cloud) {
				b.client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
					Name: tfe.String("mordor"),
				})

				b.client.Workspaces.Create(context.Background(), "mordor", tfe.WorkspaceCreateOptions{
					Name: tfe.String("shire"),
					Tags: []*tfe.Tag{
						{
							Name: "hobbity",
						},
					},
				})
			},
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal([]cty.Value{
						cty.StringVal("hobbity"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_WORKSPACE": "shire",
			},
			expectedWorkspaceName: "", // No error is raised, but workspace is not set
		},
		"project specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("mt-doom"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.StringVal("my-project"),
				}),
			}),
			expectedWorkspaceName: "mt-doom",
			expectedProjectName:   "my-project",
		},
		"project env var specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("mt-doom"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_PROJECT": "other-project",
			},
			expectedWorkspaceName: "mt-doom",
			expectedProjectName:   "other-project",
		},
		"project and env var specified": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("mt-doom"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.StringVal("my-project"),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_PROJECT": "other-project",
			},
			expectedWorkspaceName: "mt-doom",
			expectedProjectName:   "my-project",
		},
		"workspace exists but in different project": {
			setup: func(b *Cloud) {
				b.client.Organizations.Create(context.Background(), tfe.OrganizationCreateOptions{
					Name: tfe.String("mordor"),
				})

				project, _ := b.client.Projects.Create(context.Background(), "mordor", tfe.ProjectCreateOptions{
					Name: "another-project",
				})

				b.client.Workspaces.Create(context.Background(), "mordor", tfe.WorkspaceCreateOptions{
					Name:    tfe.String("shire"),
					Project: project,
				})
			},
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.StringVal("mordor"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal([]cty.Value{
						cty.StringVal("hobbity"),
					}),
					"project": cty.StringVal("my-project"),
				}),
			}),
			expectedProjectName: "my-project",
			// No error is raised, and workspace is still in its original
			// project, but the configured project for any future workspaces
			// created with `terraform workspace new` is unaffected.
		},
		"with everything set as env vars": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.NullVal(cty.String),
				"workspaces":   cty.NullVal(cty.String),
			}),
			vars: map[string]string{
				"TF_CLOUD_ORGANIZATION": "mordor",
				"TF_WORKSPACE":          "mt-doom",
				"TF_CLOUD_HOSTNAME":     "mycool.tfe-host.io",
				"TF_CLOUD_PROJECT":      "my-project",
			},
			expectedOrganization:  "mordor",
			expectedWorkspaceName: "mt-doom",
			expectedHostname:      "mycool.tfe-host.io",
			expectedProjectName:   "my-project",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, cleanup := testUnconfiguredBackend(t)
			t.Cleanup(cleanup)

			for k, v := range tc.vars {
				os.Setenv(k, v)
			}

			t.Cleanup(func() {
				for k := range tc.vars {
					os.Unsetenv(k)
				}
			})

			_, valDiags := b.PrepareConfig(tc.config)
			if valDiags.Err() != nil {
				t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
			}

			if tc.setup != nil {
				tc.setup(b)
			}

			diags := b.Configure(tc.config)
			if (diags.Err() != nil || tc.expectedErr != "") &&
				(diags.Err() == nil || !strings.Contains(diags.Err().Error(), tc.expectedErr)) {
				t.Fatalf("%s: unexpected configure result: %v", name, diags.Err())
			}

			if tc.expectedOrganization != "" && tc.expectedOrganization != b.Organization {
				t.Fatalf("%s: organization not valid: %s, expected: %s", name, b.Organization, tc.expectedOrganization)
			}

			if tc.expectedHostname != "" && tc.expectedHostname != b.Hostname {
				t.Fatalf("%s: hostname not valid: %s, expected: %s", name, b.Hostname, tc.expectedHostname)
			}

			if tc.expectedWorkspaceName != "" && tc.expectedWorkspaceName != b.WorkspaceMapping.Name {
				t.Fatalf("%s: workspace name not valid: %s, expected: %s", name, b.WorkspaceMapping.Name, tc.expectedWorkspaceName)
			}

			if tc.expectedProjectName != "" && tc.expectedProjectName != b.WorkspaceMapping.Project {
				t.Fatalf("%s: project name not valid: %s, expected: %s", name, b.WorkspaceMapping.Project, tc.expectedProjectName)
			}
		})
	}
}

func TestCloud_config(t *testing.T) {
	cases := map[string]struct {
		config  cty.Value
		confErr string
		valErr  string
	}{
		"with_a_non_tfe_host": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("nontfe.local"),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			confErr: "Host nontfe.local does not provide a tfe service",
		},
		// localhost advertises TFE services, but has no token in the credentials
		"without_a_token": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("localhost"),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
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
					"project": cty.NullVal(cty.String),
				}),
			}),
		},
		"with_kv_tags": {
			config: cty.ObjectVal((map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.MapVal(map[string]cty.Value{
						"dept": cty.StringVal("billing"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			})),
		},
		"with_a_name": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
		},
		"without_a_name_tags": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.NullVal(cty.String),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			confErr: `Missing workspace mapping strategy.`,
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
					"project": cty.NullVal(cty.String),
				}),
			}),
			valErr: `Only one of workspace "tags" or "name" is allowed.`,
		},
		"invalid tags dynamic type": {
			config: cty.ObjectVal((map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.NullVal(cty.String),
					"tags":    cty.StringVal("dept:billing"),
					"project": cty.NullVal(cty.String),
				}),
			})),
			confErr: `tags must be a set or object, not string`,
		},
		"invalid tags dynamic type, object with non-string": {
			config: cty.ObjectVal((map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.MapVal(map[string]cty.Value{
						"dept": cty.NumberIntVal(1),
					}),
					"project": cty.NullVal(cty.String),
				}),
			})),
			confErr: `tag object values must be strings`,
		},
		"invalid tags dynamic type, tuple non-string": {
			config: cty.ObjectVal((map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.NullVal(cty.String),
					"tags":    cty.TupleVal([]cty.Value{cty.NumberIntVal(1)}),
					"project": cty.NullVal(cty.String),
				}),
			})),
			confErr: `tag elements must be strings`,
		},
		"null config": {
			config: cty.NullVal(cty.EmptyObject),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, cleanup := testUnconfiguredBackend(t)
			t.Cleanup(cleanup)

			// Validate
			_, valDiags := b.PrepareConfig(tc.config)
			if (valDiags.Err() != nil || tc.valErr != "") &&
				(valDiags.Err() == nil || !strings.Contains(valDiags.Err().Error(), tc.valErr)) {
				t.Fatalf("unexpected validation result: %v", valDiags.Err())
			}

			// Configure
			confDiags := b.Configure(tc.config)
			if (confDiags.Err() != nil || tc.confErr != "") &&
				(confDiags.Err() == nil || !strings.Contains(confDiags.Err().Error(), tc.confErr)) {
				t.Fatalf("unexpected configure result: %v", confDiags.Err())
			}
		})
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
			"project": cty.NullVal(cty.String),
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
			"project": cty.NullVal(cty.String),
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

	expected := `This version of HCP Terraform does not support the state mechanism
attempting to be used by the platform. This should never happen.`
	if !strings.Contains(confDiags.Err().Error(), expected) {
		t.Fatalf("expected configure to error with %q, got %q", expected, confDiags.Err().Error())
	}
}

func TestCloud_setUnavailableTerraformVersion(t *testing.T) {
	// go-tfe returns an error IRL if you try to set a Terraform version that's
	// not available in your HCP Terraform instance. To test this, tfe_client_mock errors if
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
			"project": cty.NullVal(cty.String),
		}),
	})

	b, _, bCleanup := testBackend(t, config, nil)
	defer bCleanup()

	// Make sure the workspace doesn't exist yet -- otherwise, we can't test what
	// happens when a workspace gets created. This is why we can't use "name" in
	// the backend config above, btw: if you do, testBackend() creates the default
	// workspace before we get a chance to do anything.
	_, err := b.client.Workspaces.Read(context.Background(), b.Organization, workspaceName)
	if err != tfe.ErrResourceNotFound {
		t.Fatalf("the workspace we were about to try and create (%s/%s) already exists in the mocks somehow, so this test isn't trustworthy anymore", b.Organization, workspaceName)
	}

	_, err = b.StateMgr(workspaceName)
	if err != nil {
		t.Fatalf("expected no error from StateMgr, despite not being able to set remote Terraform version: %#v", err)
	}
	// Make sure the workspace was created:
	workspace, err := b.client.Workspaces.Read(context.Background(), b.Organization, workspaceName)
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

func TestCloud_resolveCloudConfig(t *testing.T) {
	cases := map[string]struct {
		config         cty.Value
		vars           map[string]string
		expectedResult cloudConfig
		expectedErr    string
	}{
		"hostname/org/name/token/project all set": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("app.staging.terraform.io"),
				"organization": cty.StringVal("examplecorp"),
				"token":        cty.StringVal("password123"),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.StringVal("networking"),
				}),
			}),
			expectedResult: cloudConfig{
				hostname:     "app.staging.terraform.io",
				organization: "examplecorp",
				token:        "password123",
				workspaceMapping: WorkspaceMapping{
					Name:    "prod",
					Project: "networking",
				},
			},
		},
		"totally empty config block": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.NullVal(cty.String),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
			vars: map[string]string{
				"TF_CLOUD_HOSTNAME":     "app.staging.terraform.io",
				"TF_CLOUD_ORGANIZATION": "examplecorp",
				"TF_WORKSPACE":          "prod",
				"TF_CLOUD_PROJECT":      "networking",
			},
			expectedResult: cloudConfig{
				hostname:     "app.staging.terraform.io",
				organization: "examplecorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name:    "prod",
					Project: "networking",
				},
			},
		},
		"null organization": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.NullVal(cty.String),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedErr: `Invalid or missing required argument: "organization" must be set in the cloud configuration or as an environment variable: TF_CLOUD_ORGANIZATION.`,
		},
		"null workspace": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags" or "name" is required.`,
		},
		"workspace: empty tags, name": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.NullVal(cty.String),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Missing workspace mapping strategy. Either workspace "tags" or "name" is required.`,
		},
		"workspace: name and tags present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("org"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.StringVal("prod"),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
						},
					),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedErr: `Invalid workspaces configuration: Only one of workspace "tags" or "name" is allowed.`,
		},
		"organization from environment": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.NullVal(cty.String),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_ORGANIZATION": "example-org",
			},
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "example-org",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name: "prod",
				},
			},
		},
		"null workspace, TF_WORKSPACE present": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
			vars: map[string]string{
				"TF_WORKSPACE": "my-workspace",
			},
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "hashicorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name: "my-workspace",
				},
			},
		},
		"organization and workspace and project env var": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"organization": cty.NullVal(cty.String),
				"workspaces": cty.NullVal(cty.Object(map[string]cty.Type{
					"name":    cty.String,
					"tags":    cty.Set(cty.String),
					"project": cty.String,
				})),
			}),
			vars: map[string]string{
				"TF_CLOUD_ORGANIZATION": "hashicorp",
				"TF_WORKSPACE":          "my-workspace",
				"TF_CLOUD_PROJECT":      "example-project",
			},
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "hashicorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name:    "my-workspace",
					Project: "example-project",
				},
			},
		},
		"with no project": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("examplecorp"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "examplecorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name: "prod",
				},
			},
		},
		"with project from environment": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("examplecorp"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_PROJECT": "example-project",
			},
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "examplecorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name:    "prod",
					Project: "example-project",
				},
			},
		},
		"with both project env var and config value": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("examplecorp"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.StringVal("config-project"),
				}),
			}),
			vars: map[string]string{
				"TF_CLOUD_PROJECT": "env-project",
			},
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "examplecorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name:    "prod",
					Project: "config-project", // config wins
				},
			},
		},
		"with hostname not set, set to default hostname": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.NullVal(cty.String),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":    cty.StringVal("prod"),
					"tags":    cty.NullVal(cty.Set(cty.String)),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedResult: cloudConfig{
				hostname:     defaultHostname,
				organization: "hashicorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					Name: "prod",
				},
			},
		},
		"with workspace tags set": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.SetVal(
						[]cty.Value{
							cty.StringVal("billing"),
							cty.StringVal("applications"),
						},
					),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedResult: cloudConfig{
				hostname:     "hashicorp.com",
				organization: "hashicorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					TagsAsSet: []string{"billing", "applications"},
				},
			},
		},
		"with kv tags set": {
			config: cty.ObjectVal(map[string]cty.Value{
				"organization": cty.StringVal("hashicorp"),
				"hostname":     cty.StringVal("hashicorp.com"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name": cty.NullVal(cty.String),
					"tags": cty.MapVal(map[string]cty.Value{
						"dept": cty.StringVal("billing"),
					}),
					"project": cty.NullVal(cty.String),
				}),
			}),
			expectedResult: cloudConfig{
				hostname:     "hashicorp.com",
				organization: "hashicorp",
				token:        "",
				workspaceMapping: WorkspaceMapping{
					TagsAsMap: map[string]string{"dept": "billing"},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.vars {
				os.Setenv(k, v)
			}
			t.Cleanup(func() {
				for k := range tc.vars {
					os.Unsetenv(k)
				}
			})

			result, diags := resolveCloudConfig(tc.config)

			// Validate either expected error or result, not both
			if tc.expectedErr != "" {
				if diags.Err() == nil {
					t.Fatalf("%s: expected validation error %q but didn't get one", name, tc.expectedErr)
				} else {
					actualErr := diags.Err().Error()
					if !strings.Contains(actualErr, tc.expectedErr) {
						t.Fatalf("%s: expected validation error %q but instead got %q", name, tc.expectedErr, actualErr)
					}
				}
			} else { // check result
				// Sort tags first to avoid flaking, since the slice order from
				// a cty.Set isn't guaranteed:
				sort.Strings(tc.expectedResult.workspaceMapping.TagsAsSet)
				sort.Strings(result.workspaceMapping.TagsAsSet)
				if !reflect.DeepEqual(tc.expectedResult, result) {
					t.Fatalf("%s: expected final config of %#v but instead got %#v", name, tc.expectedResult, result)
				}

				if !reflect.DeepEqual(tc.expectedResult.workspaceMapping.asTFETagBindings(), result.workspaceMapping.asTFETagBindings()) {
					t.Fatalf("%s: expected final config of %#v but instead got %#v", name, tc.expectedResult, result)
				}
			}
		})
	}
}

func TestCloud_forceLocalBackend(t *testing.T) {
	b, cleanup := testUnconfiguredBackend(t)
	t.Cleanup(cleanup)

	os.Setenv("TF_FORCE_LOCAL_BACKEND", "true")
	t.Cleanup(func() {
		os.Unsetenv("TF_FORCE_LOCAL_BACKEND")
	})

	obj := cty.ObjectVal(map[string]cty.Value{
		"organization": cty.StringVal("hashicorp"),
		"hostname":     cty.NullVal(cty.String),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name":    cty.StringVal("prod"),
			"tags":    cty.NullVal(cty.Set(cty.String)),
			"project": cty.NullVal(cty.String),
		}),
	})

	obj, valDiags := b.PrepareConfig(obj)
	if valDiags.Err() != nil {
		t.Fatalf("unexpected validation result: %v", valDiags.Err())
	}

	diags := b.Configure(obj)
	if diags.Err() != nil {
		t.Fatalf("unexpected configure result: %v", diags.Err())
	}

	if b.forceLocal != true {
		t.Fatalf("expected force local backend to be set due to env var")
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

	if err := b.DeleteWorkspace(testBackendSingleWorkspaceName, true); err != backend.ErrWorkspacesNotSupported {
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
		b.Organization,
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
		b.Organization,
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
		b.Organization,
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
		{"0.14.0", "1.3.0", "remote", true},
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
				b.Organization,
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
		b.Organization,
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
		b.Organization,
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

func TestCloudBackend_DeleteWorkspace_SafeAndForce(t *testing.T) {
	b, bCleanup := testBackendWithTags(t)
	defer bCleanup()
	safeDeleteWorkspaceName := "safe-delete-workspace"
	forceDeleteWorkspaceName := "force-delete-workspace"

	_, err := b.StateMgr(safeDeleteWorkspaceName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	_, err = b.StateMgr(forceDeleteWorkspaceName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	// sanity check that the mock now contains two workspaces
	wl, err := b.Workspaces()
	if err != nil {
		t.Fatalf("error fetching workspace names: %v", err)
	}
	if len(wl) != 2 {
		t.Fatalf("expected 2 workspaced but got %d", len(wl))
	}

	c := context.Background()
	safeDeleteWorkspace, err := b.client.Workspaces.Read(c, b.Organization, safeDeleteWorkspaceName)
	if err != nil {
		t.Fatalf("error fetching workspace: %v", err)
	}

	// Lock a workspace so that it should fail to be safe deleted
	_, err = b.client.Workspaces.Lock(context.Background(), safeDeleteWorkspace.ID, tfe.WorkspaceLockOptions{Reason: tfe.String("test")})
	if err != nil {
		t.Fatalf("error locking workspace: %v", err)
	}
	err = b.DeleteWorkspace(safeDeleteWorkspaceName, false)
	if err == nil {
		t.Fatalf("workspace should have failed to safe delete")
	}

	// unlock the workspace and confirm that safe-delete now works
	_, err = b.client.Workspaces.Unlock(context.Background(), safeDeleteWorkspace.ID)
	if err != nil {
		t.Fatalf("error unlocking workspace: %v", err)
	}
	err = b.DeleteWorkspace(safeDeleteWorkspaceName, false)
	if err != nil {
		t.Fatalf("error safe deleting workspace: %v", err)
	}

	// lock a workspace and then confirm that force deleting it works
	forceDeleteWorkspace, err := b.client.Workspaces.Read(c, b.Organization, forceDeleteWorkspaceName)
	if err != nil {
		t.Fatalf("error fetching workspace: %v", err)
	}
	_, err = b.client.Workspaces.Lock(context.Background(), forceDeleteWorkspace.ID, tfe.WorkspaceLockOptions{Reason: tfe.String("test")})
	if err != nil {
		t.Fatalf("error locking workspace: %v", err)
	}
	err = b.DeleteWorkspace(forceDeleteWorkspaceName, true)
	if err != nil {
		t.Fatalf("error force deleting workspace: %v", err)
	}
}

func TestCloudBackend_DeleteWorkspace_DoesNotExist(t *testing.T) {
	b, bCleanup := testBackendWithTags(t)
	defer bCleanup()

	err := b.DeleteWorkspace("non-existent-workspace", false)
	if err != nil {
		t.Fatalf("expected deleting a workspace which does not exist to succeed")
	}
}

func TestCloud_ServiceDiscoveryAliases(t *testing.T) {
	s := testServer(t)
	b := New(testDisco(s))

	diag := b.Configure(cty.ObjectVal(map[string]cty.Value{
		"hostname":     cty.NullVal(cty.String), // Forces aliasing to test server
		"organization": cty.StringVal("hashicorp"),
		"token":        cty.NullVal(cty.String),
		"workspaces": cty.ObjectVal(map[string]cty.Value{
			"name":    cty.StringVal("prod"),
			"tags":    cty.NullVal(cty.Set(cty.String)),
			"project": cty.NullVal(cty.String),
		}),
	}))
	if diag.HasErrors() {
		t.Fatalf("expected no diagnostic errors, got %s", diag.Err())
	}

	aliases, err := b.ServiceDiscoveryAliases()
	if err != nil {
		t.Fatalf("expected no errors, got %s", err)
	}
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias but got %d", len(aliases))
	}
}
