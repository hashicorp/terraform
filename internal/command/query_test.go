// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"
)

func TestQuery(t *testing.T) {
	tests := []struct {
		name        string
		directory   string
		expectedOut string
		expectedErr []string
		initCode    int
		args        []string
	}{
		{
			name:      "basic query",
			directory: "basic",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:      "query referencing local variable",
			directory: "with-locals",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:        "config with no query block",
			directory:   "no-list-block",
			expectedOut: "",
			expectedErr: []string{`
Error: No resources to query

The configuration does not contain any resources that can be queried.
`},
		},
		{
			name:        "missing query file",
			directory:   "missing-query-file",
			expectedOut: "",
			expectedErr: []string{`
Error: No resources to query

The configuration does not contain any resources that can be queried.
`},
		},
		{
			name:        "missing configuration",
			directory:   "missing-configuration",
			expectedOut: "",
			expectedErr: []string{`
Error: No configuration files

Query requires a query configuration to be present. Create a Terraform query
configuration file (.tfquery.hcl file) and try again.
`},
		},
		{
			name:        "invalid query syntax",
			directory:   "invalid-syntax",
			expectedOut: "",
			initCode:    0,
			expectedErr: []string{`
Error: Unsupported block type

  on query.tfquery.hcl line 11:
  11: resource "test_instance" "example" {

Blocks of type "resource" are not expected here.
`},
		},
		{
			name:      "empty result",
			directory: "empty-result",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

Warning: list block(s) [list.test_instance.example2] returned 0 results.`,
		},
		{
			name:      "error - extra variables",
			directory: "basic",
			args:      []string{"-var", "instance_name=test-instance"},
			expectedErr: []string{`
Error: Value for undeclared variable

A variable named "instance_name" was assigned on the command line, but the
root module does not declare a variable of that name. To use this value, add
a "variable" block to the configuration.
				`},
		},
		{
			name:      "query with variables",
			directory: "with-variables",
			args:      []string{"-var", "instance_name=test-instance"},
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:      "query with variables defined in tf file",
			directory: "with-variables-in-tf",
			args:      []string{"-var", "instance_name=test-instance"},
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:      "query with variable files",
			directory: "with-variables-file",
			args:      []string{"-var-file=custom.tfvars"},
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:        "error - query with invalid variable value",
			directory:   "with-invalid-variables",
			args:        []string{"-var", "target_ami=ami-123"},
			expectedErr: []string{`AMI ID must be longer than 10 characters.`},
		},
		{
			name:        "error - query with missing required variable",
			directory:   "with-variables",
			expectedOut: "",
			expectedErr: []string{`
Error: No value for required variable

  on query.tfquery.hcl line 7:
   7: variable "instance_name" {

The root module input variable "instance_name" is not set, and has no default
value. Use a -var or -var-file command line argument to provide a value for
this variable.
`},
		},
		{
			name:        "error - query with missing required variable in tf file",
			directory:   "with-variables-in-tf",
			expectedOut: "",
			expectedErr: []string{`
Error: No value for required variable

  on main.tf line 15:
  15: variable "instance_name" {

The root module input variable "instance_name" is not set, and has no default
value. Use a -var or -var-file command line argument to provide a value for
this variable.
`},
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("query", ts.directory)), td)
			t.Chdir(td)
			providerSource := newMockProviderSource(t, map[string][]string{
				"hashicorp/test": {"1.0.0"},
			})

			p := queryFixtureProvider()
			view, done := testView(t)
			meta := Meta{
				testingOverrides:          metaOverridesForProvider(p),
				View:                      view,
				AllowExperimentalFeatures: true,
				ProviderSource:            providerSource,
			}

			// helper for asserting against the expected error(s)
			assertErr := func(actual string) (errored bool) {
				for _, expected := range ts.expectedErr {
					expected := strings.TrimSpace(expected)
					if !strings.Contains(actual, expected) {
						errored = true
						t.Errorf("expected error message to contain '%s', \ngot: %s: diff: %s", expected, actual, cmp.Diff(expected, actual))
					}
				}
				return
			}

			init := &InitCommand{Meta: meta}
			code := init.Run(nil)
			output := done(t)
			if code != ts.initCode {
				t.Fatalf("expected status code %d but got %d: %s", ts.initCode, code, output.All())
			}

			// If we expect an init error, perhaps we want to assert the error message
			if ts.initCode != 0 {
				actual := output.All()
				if errored := assertErr(actual); errored {
					t.FailNow()
					return
				}
			}

			view, done = testView(t)
			meta.View = view

			c := &QueryCommand{Meta: meta}
			code = c.Run(append([]string{"-no-color"}, ts.args...))
			output = done(t)
			if len(ts.expectedErr) == 0 {
				if code != 0 {
					t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
				}
				actual := strings.TrimSpace(output.Stdout())

				// Check that we have query output
				expected := strings.TrimSpace(ts.expectedOut)
				if diff := cmp.Diff(expected, actual); diff != "" {
					t.Errorf("expected query output to contain \n%q, \ngot: \n%q, \ndiff: %s", expected, actual, diff)
				}

			} else {
				actual := strings.TrimSpace(output.Stderr())
				assertErr(actual)
			}
		})
	}
}

// TestQuery_varFileDuplicateAttr is a regression test for a bug where a
// -var-file containing a duplicated attribute would print an error diagnostic
// but still exit 0, silently discarding the file and falling back to other
// variable sources. A malformed var file must cause the query to fail.
func TestQuery_varFileDuplicateAttr(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("query", "duplicate-var-file")), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})

	p := queryFixtureProvider()
	view, done := testView(t)
	meta := Meta{
		testingOverrides:          metaOverridesForProvider(p),
		View:                      view,
		AllowExperimentalFeatures: true,
		ProviderSource:            providerSource,
	}

	init := &InitCommand{Meta: meta}
	initCode := init.Run(nil)
	initOutput := done(t)
	if initCode != 0 {
		t.Fatalf("init failed: %d\n\n%s", initCode, initOutput.All())
	}

	view, done = testView(t)
	meta.View = view

	c := &QueryCommand{Meta: meta}
	code := c.Run([]string{"-no-color", "-var-file=duplicate.tfvars"})
	output := done(t)
	if code == 0 {
		t.Fatalf("succeeded; want failure with a non-zero exit code\n\n%s", output.Stdout())
	}

	if got, want := output.Stderr(), "Attribute redefined"; !strings.Contains(got, want) {
		t.Fatalf("missing expected error message\nwant message containing %q\ngot:\n%s", want, got)
	}
}

func TestQueryCommand_Validate(t *testing.T) {
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "main.policy.hcl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	td2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(td2, "allow.policy.hcl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	missingPath := filepath.Join(t.TempDir(), "does-not-exist")

	tests := []struct {
		name             string
		policyPaths      []string
		allowExperiments bool
		wantDiags        tfdiags.Diagnostics
	}{
		{
			name:             "no policies, flag omitted",
			policyPaths:      nil,
			allowExperiments: true,
		},
		{
			name:             "single valid path",
			policyPaths:      []string{td},
			allowExperiments: true,
		},
		{
			name:             "multiple valid paths",
			policyPaths:      []string{td, td2},
			allowExperiments: true,
		},
		{
			name:             "non-existent path",
			policyPaths:      []string{missingPath},
			allowExperiments: true,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid policy path",
					fmt.Sprintf("Terraform cannot find the policy path at %s. Please ensure the file or directory exists and the path is correct.", missingPath),
				),
			},
		},
		{
			name:             "experiments disallowed",
			policyPaths:      []string{td},
			allowExperiments: false,
			wantDiags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"The -policies flag is only valid in experimental builds of Terraform.",
				),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &QueryCommand{Meta: Meta{AllowExperimentalFeatures: tc.allowExperiments}}
			got := cmd.Validate(&arguments.Query{PolicyPaths: tc.policyPaths})
			if tc.wantDiags == nil {
				tfdiags.AssertNoDiagnostics(t, got)
				return
			}
			tfdiags.AssertDiagnosticsMatch(t, got, tc.wantDiags)
		})
	}
}

type queryPolicyRemoteCommandBackend struct {
	backendrun.OperationsBackend
	local bool
}

func (*queryPolicyRemoteCommandBackend) IgnoreVersionConflict() {}

func (*queryPolicyRemoteCommandBackend) VerifyWorkspaceTerraformVersion(string) tfdiags.Diagnostics {
	return nil
}

func (b *queryPolicyRemoteCommandBackend) IsLocalOperations() bool {
	return b.local
}

func TestQueryCommand_policyClientRouting(t *testing.T) {
	for _, tc := range []struct {
		name        string
		remote      bool
		forceLocal  bool
		policyPaths []string
		wantClient  bool
	}{
		{name: "remote", remote: true, policyPaths: []string{"policy.tfpolicy.hcl"}},
		{name: "forced local", remote: true, forceLocal: true, policyPaths: []string{"policy.tfpolicy.hcl"}, wantClient: true},
		{name: "local", policyPaths: []string{"policy.tfpolicy.hcl"}, wantClient: true},
		{name: "no policies"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var be backendrun.OperationsBackend = backendLocal.New()
			if tc.remote {
				be = &queryPolicyRemoteCommandBackend{OperationsBackend: be, local: tc.forceLocal}
			}

			client := policy.NewTestMockClient(t)
			cmd := &QueryCommand{Meta: Meta{
				AllowExperimentalFeatures: true,
				testingOverrides:          &testingOverrides{PolicyClient: client},
			}}
			op := &backendrun.Operation{PolicyPaths: tc.policyPaths}
			stop := cmd.configureQueryPolicyClient(be, op)
			stop()
			if tc.wantClient {
				if op.PolicyClient != client {
					t.Fatal("policy client was not propagated to the operation")
				}
			} else if op.PolicyClient != nil {
				t.Fatal("remote operation unexpectedly received a local policy client")
			}
			if client.StopCalled != tc.wantClient {
				t.Fatalf("policy client stopped = %t, want %t", client.StopCalled, tc.wantClient)
			}
		})
	}
}

func queryFixtureProvider() *testing_provider.MockProvider {
	p := testProvider()
	instanceListSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {
				Type:     cty.DynamicPseudoType,
				Computed: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {
							Type:     cty.String,
							Required: true,
						},
						"foo": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
				Nesting:  configschema.NestingSingle,
				MinItems: 1,
				MaxItems: 1,
			},
		},
	}
	databaseListSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {
				Type:     cty.DynamicPseudoType,
				Computed: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"engine": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
				Nesting:  configschema.NestingSingle,
				MinItems: 1,
				MaxItems: 1,
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"ami": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
				Identity: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
				IdentityVersion: 1,
			},
			"test_database": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"engine": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
				Identity: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
					},
					Nesting: configschema.NestingSingle,
				},
			},
		},
		ListResourceTypes: map[string]providers.Schema{
			"test_instance": {Body: instanceListSchema},
			"test_database": {Body: databaseListSchema},
		},
	}

	// Mock the ListResources method for query operations
	p.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		// Check the config to determine what kind of response to return
		wholeConfigMap := request.Config.AsValueMap()

		configMap := wholeConfigMap["config"]

		// For empty results test case
		ami, ok := configMap.AsValueMap()["ami"]
		if ok && ami.AsString() == "ami-nonexistent" {
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data":   cty.ListValEmpty(cty.DynamicPseudoType),
					"config": configMap,
				}),
			}
		}

		switch request.TypeName {
		case "test_instance":
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-instance-1"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":  cty.StringVal("test-instance-1"),
								"ami": cty.StringVal("ami-12345"),
							}),
							"display_name": cty.StringVal("Test Instance 1"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-instance-2"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":  cty.StringVal("test-instance-2"),
								"ami": cty.StringVal("ami-67890"),
							}),
							"display_name": cty.StringVal("Test Instance 2"),
						}),
					}),
					"config": configMap,
				}),
			}
		case "test_database":
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-db-1"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":     cty.StringVal("test-db-1"),
								"engine": cty.StringVal("mysql"),
							}),
							"display_name": cty.StringVal("Test Database 1"),
						}),
					}),
					"config": configMap,
				}),
			}
		default:
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data":   cty.ListVal([]cty.Value{}),
					"config": configMap,
				}),
			}
		}
	}

	return p
}

func TestQueryPolicyStatusReporting(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("query", "basic")), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	p := queryFixtureProvider()
	p.GetProviderSchemaResponse.ServerCapabilities.GenerateResourceConfig = true
	p.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		config := request.Config.GetAttr("config")
		rows := make([]cty.Value, 0, 3)
		for idx := 1; idx <= 3; idx++ {
			id := fmt.Sprintf("test-instance-%d", idx)
			rows = append(rows, cty.ObjectVal(map[string]cty.Value{
				"identity": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal(id),
				}),
				"state": cty.ObjectVal(map[string]cty.Value{
					"id":  cty.StringVal(id),
					"ami": cty.StringVal(id),
				}),
				"display_name": cty.StringVal(fmt.Sprintf("Test Instance %d", idx)),
			}))
		}
		return providers.ListResourceResponse{Result: cty.ObjectVal(map[string]cty.Value{
			"data":   cty.ListVal(rows),
			"config": config,
		})}
	}
	p.GenerateResourceConfigFn = func(request providers.GenerateResourceConfigRequest) providers.GenerateResourceConfigResponse {
		id := request.State.GetAttr("id").AsString()
		if id == "test-instance-1" {
			return providers.GenerateResourceConfigResponse{Diagnostics: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "config generation failed", "could not generate resource configuration"),
			}}
		}
		return providers.GenerateResourceConfigResponse{Config: cty.ObjectVal(map[string]cty.Value{
			"id":  cty.NullVal(cty.String),
			"ami": cty.StringVal(id),
		})}
	}

	var evaluated []string
	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateFn = func(_ context.Context, request policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		id := request.Attrs.Raw.GetAttr("ami").AsString()
		evaluated = append(evaluated, id)
		switch id {
		case "test-instance-2":
			return policy.ErrorEvalFromDiags([]*proto.Diagnostic{{
				Severity: proto.Severity_ERROR,
				Summary:  "Failed to evaluate Terraform Policy",
				Detail:   "policy transport unavailable",
			}})
		case "test-instance-3":
			return policy.EvaluationFromProtoResponse(proto.EvaluateResult_ALLOW_EVALUATE_RESULT, nil)
		default:
			t.Fatalf("unexpected policy evaluation for %q", id)
			return policy.EvaluationResponse{}
		}
	}

	overrides := metaOverridesForProvider(p)
	overrides.PolicyClient = policyClient
	view, done := testView(t)
	meta := Meta{
		testingOverrides:          overrides,
		View:                      view,
		AllowExperimentalFeatures: true,
		ProviderSource:            providerSource,
	}

	init := &InitCommand{Meta: meta}
	if code := init.Run(nil); code != 0 {
		t.Fatalf("init failed with status %d:\n%s", code, done(t).All())
	}

	view, done = testView(t)
	meta.View = view
	command := &QueryCommand{Meta: meta}
	code := command.Run([]string{"-no-color", "-json", "-policies=."})
	output := done(t)
	if code != 0 {
		t.Fatalf("query failed with status %d:\n%s", code, output.All())
	}

	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output.Stdout()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to decode JSON line %q: %s", line, err)
		}
		records = append(records, record)
	}

	var foundResources, summaries, policyDiagnostics []map[string]any
	for _, record := range records {
		switch record["type"] {
		case "list_resource_found":
			foundResources = append(foundResources, record)
		case "policy_query_summary":
			summaries = append(summaries, record)
		case "policy_diagnostic":
			policyDiagnostics = append(policyDiagnostics, record)
		}
	}
	if len(foundResources) != 3 {
		t.Fatalf("list_resource_found records = %d, want 3", len(foundResources))
	}
	if len(summaries) != 1 {
		t.Fatalf("policy_query_summary records = %d, want 1", len(summaries))
	}
	if len(policyDiagnostics) != 1 {
		t.Fatalf("policy_diagnostic records = %d, want 1", len(policyDiagnostics))
	}
	if len(evaluated) != 2 || slices.Contains(evaluated, "test-instance-1") || !slices.Contains(evaluated, "test-instance-2") || !slices.Contains(evaluated, "test-instance-3") {
		t.Fatalf("evaluated resources = %v, want only test-instance-2 and test-instance-3", evaluated)
	}

	summary := summaries[0]
	if summary["overall_result"] != "error" {
		t.Fatalf("summary overall_result = %v, want error", summary["overall_result"])
	}
	passedPolicies, ok := summary["passed_policies"].([]any)
	if !ok || len(passedPolicies) != 0 {
		t.Fatalf("passed_policies = %#v, want []", summary["passed_policies"])
	}
	rows := summary["results"].([]any)
	if len(rows) != 3 {
		t.Fatalf("summary rows = %d, want 3", len(rows))
	}
	rowsByID := make(map[string]map[string]any, len(rows))
	for _, rawRow := range rows {
		row := rawRow.(map[string]any)
		identity := row["identity"].(map[string]any)
		rowsByID[identity["id"].(string)] = row
		policies, ok := row["policies"].([]any)
		if !ok || len(policies) != 0 {
			t.Fatalf("row %v policies = %#v, want []", identity["id"], row["policies"])
		}
	}
	wantResults := map[string]string{
		"test-instance-1": "unknown",
		"test-instance-2": "error",
		"test-instance-3": "n/a",
	}
	for id, want := range wantResults {
		row := rowsByID[id]
		if row == nil || row["result"] != want {
			t.Fatalf("row %s = %#v, want result %s", id, row, want)
		}
	}

	diagnostic := policyDiagnostics[0]
	if diagnostic["target_address"] != "test_instance.example_1" {
		t.Fatalf("policy diagnostic target = %v, want test_instance.example_1", diagnostic["target_address"])
	}
}

func TestQueryPolicyStatusReporting_NoPoliciesArgument(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("query", "basic")), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	p := queryFixtureProvider()
	policyClient := policy.NewTestMockClient(t)
	var policyClientCalled atomic.Bool
	policyClient.EvaluateFn = func(context.Context, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) policy.EvaluationResponse {
		policyClientCalled.Store(true)
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	policyClient.StopFn = func() {
		policyClientCalled.Store(true)
	}

	overrides := metaOverridesForProvider(p)
	overrides.PolicyClient = policyClient
	view, done := testView(t)
	meta := Meta{
		testingOverrides:          overrides,
		View:                      view,
		AllowExperimentalFeatures: true,
		ProviderSource:            providerSource,
	}

	init := &InitCommand{Meta: meta}
	code := init.Run(nil)
	output := done(t)
	if code != 0 {
		t.Fatalf("init failed with status %d:\n%s", code, output.All())
	}

	view, done = testView(t)
	meta.View = view
	command := &QueryCommand{Meta: meta}
	code = command.Run([]string{"-no-color", "-json"})
	output = done(t)
	if code != 0 {
		t.Fatalf("query failed with status %d:\n%s", code, output.All())
	}

	var foundResources, summaries, policyDiagnostics int
	for _, line := range strings.Split(strings.TrimSpace(output.Stdout()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to decode JSON line %q: %s", line, err)
		}
		switch record["type"] {
		case "list_resource_found":
			foundResources++
		case "policy_query_summary":
			summaries++
		case "policy_diagnostic":
			policyDiagnostics++
		}
	}

	if foundResources != 2 {
		t.Fatalf("list_resource_found records = %d, want 2", foundResources)
	}
	if summaries != 0 {
		t.Fatalf("policy_query_summary records = %d, want 0", summaries)
	}
	if policyDiagnostics != 0 {
		t.Fatalf("policy_diagnostic records = %d, want 0", policyDiagnostics)
	}
	if policyClientCalled.Load() {
		t.Fatal("policy client was called without -policies")
	}
}

func TestQuery_JSON(t *testing.T) {
	tmp := t.TempDir()
	tests := []struct {
		name           string
		directory      string
		expectedRes    []map[string]any
		initCode       int
		opts           []string
		commandErrMsg  string   // non-empty when the query command fails after init
		selectResource []string // only these resources will be selected in the result if given
	}{
		{
			name:      "basic query",
			directory: "basic",
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"input_config":  map[string]any{"ami": "ami-12345", "foo": nil},
					},
					"type": "list_start",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
		{
			name:      "basic query - generate config",
			directory: "basic",
			opts:      []string{fmt.Sprintf("-generate-config-out=%s/new.tf", tmp)},
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"input_config":  map[string]any{"ami": "ami-12345", "foo": nil},
					},
					"type": "list_start",
				},
				{"@level": "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
						"config":        "resource \"test_instance\" \"example_0\" {\n  provider = test\n  ami      = \"ami-12345\"\n}",
						"import_config": "import {\n  to       = test_instance.example_0\n  provider = test\n  identity = {\n    id = \"test-instance-1\"\n  }\n}",
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
						"config":        "resource \"test_instance\" \"example_1\" {\n  provider = test\n  ami      = \"ami-67890\"\n}",
						"import_config": "import {\n  to       = test_instance.example_1\n  provider = test\n  identity = {\n    id = \"test-instance-2\"\n  }\n}",
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
		{
			name:      "list resource with an empty result",
			directory: "empty-result",
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example2: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example2",
						"resource_type": "test_instance",
						"total":         float64(0),
					},
					"type": "list_complete",
				},
			},
			selectResource: []string{"list.test_instance.example2"},
		},
		{
			name:      "error - list resource with an unknown config",
			directory: "unknown-config",
			expectedRes: []map[string]any{
				{
					"@level":   "error",
					"@message": "Error: config is not known",
					"diagnostic": map[string]any{
						"severity": "error",
						"summary":  "config is not known",
						"detail":   "",
					},
					"type": "diagnostic",
				},
			},
		},
		{
			name:      "error - generate-config-path already exists",
			directory: "basic",
			opts:      []string{fmt.Sprintf("-generate-config-out=%s", t.TempDir())},
			expectedRes: []map[string]any{
				{
					"@level":   "error",
					"@message": "Error: Target generated file already exists",
					"diagnostic": map[string]any{
						"detail":   "Terraform can only write generated config into a new file. Either choose a different target location or move all existing configuration out of the target file, delete it and try again.",
						"severity": "error",
						"summary":  "Target generated file already exists",
					},
					"type": "diagnostic",
				},
			},
		},
		{
			name:      "success with variables",
			directory: "with-variables",
			opts:      []string{"-var", "instance_name=test-instance"},
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"input_config":  map[string]any{"ami": "ami-12345", "foo": "test-instance"},
					},
					"type": "list_start",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
		{
			name:      "success with sensitive variables",
			directory: "with-sensitive-variables",
			opts:      []string{"-var", "sensitive_foo=test-instance"},
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":                   "list.test_instance.example",
						"resource_type":             "test_instance",
						"input_config":              map[string]any{"ami": "ami-12345", "foo": "test-instance"},
						"sensitive_attribute_paths": []any{string(".foo")},
					},
					"type": "list_start",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"identity_version": float64(1),
						"resource_type":    "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("query", ts.directory)), td)
			t.Chdir(td)
			providerSource := newMockProviderSource(t, map[string][]string{
				"hashicorp/test": {"1.0.0"},
			})

			p := queryFixtureProvider()
			view, done := testView(t)
			meta := Meta{
				testingOverrides:          metaOverridesForProvider(p),
				View:                      view,
				AllowExperimentalFeatures: true,
				ProviderSource:            providerSource,
			}

			init := &InitCommand{Meta: meta}
			code := init.Run(nil)
			output := done(t)
			if code != ts.initCode {
				t.Fatalf("expected status code %d but got %d: %s", ts.initCode, code, output.All())
			}

			view, done = testView(t)
			meta.View = view

			c := &QueryCommand{Meta: meta}
			args := []string{"-no-color", "-json"}
			code = c.Run(append(args, ts.opts...))
			output = done(t)
			if code != 0 {
				t.Logf("query command returned non-zero code '%d' and an error: \n\n%s", code, output.All())
			}

			// convert output to JSON array
			actual := strings.TrimSpace(output.Stdout())
			conc := fmt.Sprintf("[%s]", strings.Join(strings.Split(actual, "\n"), ","))
			rawRes := make([]map[string]any, 0)
			err := json.NewDecoder(strings.NewReader(conc)).Decode(&rawRes)
			if err != nil {
				t.Fatalf("failed to unmarshal: %s", err)
			}

			// remove unnecessary fields before comparison
			actualRes := make([]map[string]any, 0, len(rawRes))
			for _, item := range rawRes {
				delete(item, "@module")
				delete(item, "@timestamp")
				delete(item, "ui")

				// Clean up diagnostic fields that we don't want to compare
				if diagnostic, ok := item["diagnostic"].(map[string]any); ok {
					delete(diagnostic, "range")
					delete(diagnostic, "snippet")
				}

				// if we have a select list of resource addresses, we only check those addresses
				if len(ts.selectResource) > 0 {
					for _, addr := range ts.selectResource {
						if strings.Contains(item["@message"].(string), addr) {
							actualRes = append(actualRes, item)
						}
					}
				} else {
					actualRes = append(actualRes, item)
				}
			}

			// remove the version entry. Not relevant for testing
			actualRes = slices.Delete(actualRes, 0, 1)

			if diff := cmp.Diff(ts.expectedRes, actualRes); diff != "" {
				// Check that the output matches the expected results
				t.Errorf("expected query output to contain \n%q, \ngot: \n%q, \ndiff: %s", ts.expectedRes, actualRes, diff)
			}
		})
	}
}

func TestQuery_JSON_Raw(t *testing.T) {

	tfVer := tfversion.String()
	tests := []struct {
		name        string
		directory   string
		expectedOut string
		expectedErr []string
		initCode    int
		args        []string
	}{
		{
			name:      "basic query",
			directory: "basic",
			expectedOut: `{"@level":"info","@message":"Terraform ` + tfVer + `","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.596469+02:00","terraform":"1.14.0-dev","type":"version","ui":"` + views.JSON_UI_VERSION + `"}
{"@level":"info","@message":"list.test_instance.example: Starting query...","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600609+02:00","list_start":{"address":"list.test_instance.example","resource_type":"test_instance","input_config":{"ami":"ami-12345","foo":null}},"type":"list_start"}
{"@level":"info","@message":"list.test_instance.example: Result found","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600729+02:00","list_resource_found":{"address":"list.test_instance.example","display_name":"Test Instance 1","identity":{"id":"test-instance-1"},"identity_version":1,"resource_type":"test_instance","resource_object":{"ami":"ami-12345","id":"test-instance-1"}},"type":"list_resource_found"}
{"@level":"info","@message":"list.test_instance.example: Result found","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600759+02:00","list_resource_found":{"address":"list.test_instance.example","display_name":"Test Instance 2","identity":{"id":"test-instance-2"},"identity_version":1,"resource_type":"test_instance","resource_object":{"ami":"ami-67890","id":"test-instance-2"}},"type":"list_resource_found"}
{"@level":"info","@message":"list.test_instance.example: List complete","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600770+02:00","list_complete":{"address":"list.test_instance.example","resource_type":"test_instance","total":2},"type":"list_complete"}
`,
		},
		{
			name:      "empty result",
			directory: "empty-result",
			expectedOut: `{"@level":"info","@message":"Terraform ` + tfVer + `","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.596469+02:00","terraform":"1.14.0-dev","type":"version","ui":"` + views.JSON_UI_VERSION + `"}
{"@level":"info","@message":"list.test_instance.example: Starting query...","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600609+02:00","list_start":{"address":"list.test_instance.example","resource_type":"test_instance","input_config":{"ami":"ami-12345","foo":null}},"type":"list_start"}
{"@level":"info","@message":"list.test_instance.example: Result found","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600729+02:00","list_resource_found":{"address":"list.test_instance.example","display_name":"Test Instance 1","identity":{"id":"test-instance-1"},"identity_version":1,"resource_type":"test_instance","resource_object":{"ami":"ami-12345","id":"test-instance-1"}},"type":"list_resource_found"}
{"@level":"info","@message":"list.test_instance.example: Result found","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600759+02:00","list_resource_found":{"address":"list.test_instance.example","display_name":"Test Instance 2","identity":{"id":"test-instance-2"},"identity_version":1,"resource_type":"test_instance","resource_object":{"ami":"ami-67890","id":"test-instance-2"}},"type":"list_resource_found"}
{"@level":"info","@message":"list.test_instance.example: List complete","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600770+02:00","list_complete":{"address":"list.test_instance.example","resource_type":"test_instance","total":2},"type":"list_complete"}
{"@level":"info","@message":"list.test_instance.example2: Starting query...","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600609+02:00","list_start":{"address":"list.test_instance.example2","resource_type":"test_instance","input_config":{"ami":"ami-nonexistent","foo":"test-instance-1"}},"type":"list_start"}
{"@level":"info","@message":"list.test_instance.example2: List complete","@module":"terraform.ui","@timestamp":"2025-09-12T16:52:57.600770+02:00","list_complete":{"address":"list.test_instance.example2","resource_type":"test_instance","total":0},"type":"list_complete"}
`,
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("query", ts.directory)), td)
			t.Chdir(td)
			providerSource := newMockProviderSource(t, map[string][]string{
				"hashicorp/test": {"1.0.0"},
			})

			p := queryFixtureProvider()
			view, done := testView(t)
			meta := Meta{
				testingOverrides:          metaOverridesForProvider(p),
				View:                      view,
				AllowExperimentalFeatures: true,
				ProviderSource:            providerSource,
			}

			init := &InitCommand{Meta: meta}
			code := init.Run(nil)
			output := done(t)
			if code != 0 {
				t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
			}

			view, done = testView(t)
			meta.View = view

			c := &QueryCommand{Meta: meta}
			args := []string{"-no-color", "-json"}
			code = c.Run(args)
			output = done(t)
			if code != 0 {
				t.Logf("query command returned non-zero code '%d' and an error: \n\n%s", code, output.All())
			}

			// Use regex to normalize timestamps and version numbers for comparison
			timestampRegex := regexp.MustCompile(`"@timestamp":"[^"]*"`)
			versionRegex := regexp.MustCompile(`"terraform":"[^"]*"`)

			actualOutput := output.Stdout()
			expectedOutput := ts.expectedOut

			// Replace timestamps and version numbers with placeholders
			actualNormalized := timestampRegex.ReplaceAllString(actualOutput, `"@timestamp":"TIMESTAMP"`)
			actualNormalized = versionRegex.ReplaceAllString(actualNormalized, `"terraform":"VERSION"`)

			expectedNormalized := timestampRegex.ReplaceAllString(expectedOutput, `"@timestamp":"TIMESTAMP"`)
			expectedNormalized = versionRegex.ReplaceAllString(expectedNormalized, `"terraform":"VERSION"`)
			if diff := cmp.Diff(expectedNormalized, actualNormalized); diff != "" {
				t.Errorf("expected query output to match, diff: %s", diff)
			}
		})
	}
}
