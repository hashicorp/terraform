// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

func TestInit_WithModulePolicy(t *testing.T) {

	cases := []struct {
		name     string
		policy   *policy.EvaluationResponse
		expected int
	}{
		{
			name: "unknown module policy",
			// This unknown evaluation should still lead to success of the init operation.
			policy:   &policy.EvaluationResponse{Overall: policy.UnknownResult},
			expected: 0,
		},
		{
			name: "advisory deny module policy",
			// Advisory policies may return deny without error diagnostics and should
			// not block init.
			policy:   &policy.EvaluationResponse{Overall: policy.DenyResult},
			expected: 0,
		},
		{
			name:     "success module policy",
			policy:   &policy.EvaluationResponse{Overall: policy.AllowResult},
			expected: 0,
		},
		{
			name: "errored module policy",
			policy: &policy.EvaluationResponse{
				Overall: policy.DenyResult,
				Diagnostics: policy.Diagnostics{
					policy.NewErrorDiagnostic("test error", "test error detail", policy.DenyResult),
				},
			},
			expected: 1,
		},
	}

	for _, tc := range cases {
		td := t.TempDir()
		testCopyDir(t, testFixturePath("dynamic-module-sources/local-source-with-variable"), td)
		t.Chdir(td)

		ui := new(cli.MockUi)
		view, done := testView(t)

		overrides := metaOverridesForProvider(testProvider())
		policyClient := policy.NewTestMockClient(t)

		t.Run(tc.name, func(t *testing.T) {

			policyClient.EvaluateModuleResponse = tc.policy
			overrides.PolicyClient = policyClient

			c := &InitCommand{
				Meta: Meta{
					testingOverrides:          overrides,
					Ui:                        ui,
					View:                      view,
					AllowExperimentalFeatures: true,
				},
			}

			code := c.Run([]string{"-policies", td, "-var", "module_name=example"})
			output := done(t)
			if code != tc.expected {
				t.Fatalf("got exit status %d; want %d\nstderr:\n%s\n\nstdout:\n%s", code, tc.expected, output.Stderr(), output.Stdout())
			}

			if !policyClient.EvaluateModuleCalled {
				t.Fatal("expected EvaluateModule to be called")
			}

			if got, want := policyClient.EvaluateModuleRequest.Target, "./modules/example"; got != want {
				t.Fatalf("wrong module policy target\ngot:  %q\nwant: %q", got, want)
			}

			if !policyClient.EvaluateModuleCalled {
				t.Fatal("expected EvaluateModule to be called")
			}

			if got, want := policyClient.EvaluateModuleRequest.Target, "./modules/example"; got != want {
				t.Fatalf("wrong module policy target\ngot:  %q\nwant: %q", got, want)
			}

			if policyClient.EvaluateModuleRequest.Meta == nil {
				t.Fatal("expected module metadata to be set")
			}

			if got, want := policyClient.EvaluateModuleRequest.Meta.Address, "module.example"; got != want {
				t.Fatalf("wrong module address\ngot:  %q\nwant: %q", got, want)
			}

			if got, want := policyClient.EvaluateModuleRequest.Meta.Version, ""; got != want {
				t.Fatalf("wrong module version\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

func TestInit_WithPolicyClientStopAfterInit(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("dynamic-module-sources/local-source-with-variable"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	var stopCalled atomic.Bool
	var policyEvaluated atomic.Bool
	policyClient.StopFn = func() {
		stopCalled.Store(true)
	}
	policyClient.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) policy.EvaluationResponse {
		policyEvaluated.Store(true)
		if stopCalled.Load() {
			t.Fatal("policy client Stop was called before init finished")
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-var", "module_name=example"})
	output := done(t)
	if code != 0 {
		t.Fatalf("got exit status %d; want 0\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}
	if !policyEvaluated.Load() {
		t.Fatal("expected module policy evaluation to be called during init")
	}
	if !stopCalled.Load() {
		t.Fatal("expected policy client Stop to be called after init")
	}
}

func TestInit_WithModulePolicy_AlreadyInstalled(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("dynamic-module-sources/add-version-constraint"), td)
	t.Chdir(td)

	if err := os.WriteFile(filepath.Join(td, ".terraform/modules/modules.json"), []byte(`{
    "Modules": [
        {
            "Key": "",
            "Source": "",
            "Dir": ""
        },
        {
            "Key": "child",
            "Source": "registry.terraform.io/hashicorp/module-installer-acctest/aws",
            "Version": "0.0.1",
            "Dir": ".terraform/modules/child"
        }
    ]
}`), 0644); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) policy.EvaluationResponse {
		if got, want := req.Meta.Version, "0.0.1"; got != want {
			t.Fatalf("wrong module version\ngot:  %q\nwant: %q", got, want)
		}
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Diagnostics: policy.Diagnostics{
				policy.NewErrorDiagnostic(
					"module policy denied",
					"module policy blocked installation",
					policy.DenyResult,
				),
			},
		}
	}
	overs := overrides
	overs.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overs,
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-no-color"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateModuleCalled {
		t.Fatal("expected EvaluateModule to be called for already-installed module")
	}

	expected := `
Error: module policy denied

  on add-version-constraint.tf line 7:
   7: module "child" {

module policy blocked installation

Error: Policy evaluation failed

Module download blocked due to policy violations. Please review other
diagnostics for details.
`
	if diff := cmp.Diff(expected, output.Stderr()); diff != "" {
		t.Fatalf("unexpected stderr:\n%s", diff)
	}
}

func TestInit_WithPolicySetupFailureJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-nested-provider-requirements"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test":                        {"1.2.3", "1.2.4"},
		"tf.example.com/awesomecorp/happycloud": {"1.0.0"},
		"hashicorp/null":                        {"2.0.1"},
		"hashicorp/grandchild":                  {"1.0.0"},
	})

	ui := new(cli.MockUi)
	view, done := testView(t)

	c := &InitCommand{
		Meta: Meta{
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-no-color", "-json"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.3"}
{"@level":"info","@message":"Initializing the backend...","@module":"terraform.ui","message_code":"initializing_backend_message","type":"init_output"}
{"@level":"error","@message":"Error: Failed to connect to policy engine","@module":"terraform.ui","@policy":"true","policy_diagnostic":{"severity":"error","summary":"Failed to connect to policy engine","detail":"Failed to connect to policy engine: failed to connect to plugin: exec: \"tfpolicy-plugin\": executable file not found in $PATH."},"policy_metadata":{},"result":"SetupErrorResult","type":"policy_diagnostic"}`
	checkGoldenReferenceStr(t, output, expected)
}

func TestInit_WithModulePolicyDiagnostics(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("dynamic-module-sources/local-source-with-variable"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateModuleResponse = &policy.EvaluationResponse{
		Overall: policy.DenyResult,
		Diagnostics: policy.Diagnostics{
			policy.DiagsFromProto([]*proto.Diagnostic{
				{
					Severity: proto.Severity_ERROR,
					Summary:  "module policy denied",
					Result: &proto.DiagnosticResult{
						Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
					},
				},
			}, &policy.Policy{
				Address:          "module_policy.example",
				Filename:         "policy_file.tfpolicy.hcl",
				EnforcementLevel: "mandatory",
				Result:           policy.DenyResult,
			})[0],
		},
		Policies: []*policy.Policy{
			{
				Address:          "module_policy.example",
				Filename:         "policy_file.tfpolicy.hcl",
				EnforcementLevel: "mandatory",
				Result:           policy.DenyResult,
			},
		},
	}
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-var", "module_name=example", "-no-color"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateModuleCalled {
		t.Fatal("expected EvaluateModule to be called")
	}

	stderr := output.Stderr()
	expected := `
Error: module policy denied

  on main.tf line 6:
   6: module "example" {


Error: Policy evaluation failed

Module download blocked due to policy violations. Please review other
diagnostics for details.
`
	if diff := cmp.Diff(expected, stderr); diff != "" {
		t.Fatalf("unexpected stderr:\n%s", diff)
	}
}

func TestInit_WithNestedModulePolicyDiagnostics(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-nested-provider-requirements"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)

	// assert modules are evaluated correctly
	expected := map[string]policy.EvaluateResult{
		"./grandchild":    policy.DenyResult,
		"./modules/child": policy.AllowResult,
	}
	actual := map[string]policy.EvaluateResult{}

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	policyClient.EvaluateModuleFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) policy.EvaluationResponse {
		if req.Target != "./grandchild" {
			actual[req.Target] = policy.AllowResult
			return policy.EvaluationResponse{Overall: policy.AllowResult}
		}
		actual[req.Target] = policy.DenyResult
		return policy.EvaluationResponse{
			Overall: policy.DenyResult,
			Diagnostics: policy.Diagnostics{
				policy.DiagsFromProto([]*proto.Diagnostic{
					{
						Severity: proto.Severity_ERROR,
						Summary:  "nested module policy denied",
						Result: &proto.DiagnosticResult{
							Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						},
					},
				}, &policy.Policy{
					Address:          "module_policy.nested",
					Filename:         "policy_file.tfpolicy.hcl",
					EnforcementLevel: "mandatory",
					Result:           policy.DenyResult,
				})[0],
			},
		}
	}
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-no-color"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateModuleCalled {
		t.Fatal("expected EvaluateModule to be called")
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Fatalf("unexpected module policy evaluation results:\n%s", diff)
	}

	stderr := output.Stderr()
	expectedStderr := `
Error: nested module policy denied

  on modules/child/main.tf line 12:
  12: module "nested" {


Error: Policy evaluation failed

Module download blocked due to policy violations. Please review other
diagnostics for details.
`
	if diff := cmp.Diff(expectedStderr, stderr); diff != "" {
		t.Fatalf("unexpected stderr:\n%s\nexpected:\n%s\n diff:\n%s", stderr, expectedStderr, diff)
	}
}

func TestInit_WithModulePolicyJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("dynamic-module-sources/local-source-with-variable"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	resp := policy.EvaluationFromProtoResponse(
		proto.EvaluateResult_DENY_EVALUATE_RESULT,
		[]*proto.PolicyEvaluationDetail{
			{
				Address:              "module_policy.example",
				Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
				File:                 "policy_file.tfpolicy.hcl",
				PolicySetEnforcement: "mandatory",
				Diagnostics: []*proto.Diagnostic{
					{
						Severity: proto.Severity_ERROR,
						Summary:  "module policy denied",
						Result: &proto.DiagnosticResult{
							Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						},
					},
				},
				DefRange: &proto.Range{
					Filename: "policy_set.policy.hcl",
				},
			},
		},
	)
	policyClient.EvaluateModuleResponse = &resp
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-var", "module_name=example", "-no-color", "-json"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 0\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateModuleCalled {
		t.Fatal("expected EvaluateModule to be called")
	}

	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.3"}
{"@level":"info","@message":"Initializing the backend...","@module":"terraform.ui","message_code":"initializing_backend_message","type":"init_output"}
{"@level":"info","@message":"Initializing modules...","@module":"terraform.ui","message_code":"initializing_modules_message","type":"init_output"}
{"@level":"error","@message":"Error: module policy denied","@module":"terraform.ui","@policy":"true","policy_diagnostic":{"severity":"error","summary":"module policy denied","detail":"","range":{"filename":"main.tf","start":{"line":6,"column":1,"byte":60},"end":{"line":6,"column":17,"byte":76}},"snippet":{"context":null,"code":"module \"example\" {","start_line":6,"highlight_start_offset":0,"highlight_end_offset":16,"values":[]}},"policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"module_policy.example","file_name":"policy_set.policy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","target_address":"module.example","type":"policy_diagnostic"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","target_address":"module.example","policy_address":"module_policy.example","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"module_policy.example","file_name":"policy_set.policy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_result"}
{"@level":"error","@message":"Error: Policy evaluation failed","@module":"terraform.ui","diagnostic":{"severity":"error","summary":"Policy evaluation failed","detail":"Module download blocked due to policy violations. Please review other diagnostics for details."},"type":"diagnostic"}`
	checkGoldenReferenceStr(t, output, expected)
}

func TestInit_WithProviderPolicy(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-nested-provider-requirements"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test":                        {"1.2.3", "1.2.4"},
		"tf.example.com/awesomecorp/happycloud": {"1.0.0"},
		"hashicorp/null":                        {"2.0.1"},
		"hashicorp/grandchild":                  {"1.0.0"},
	})

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	actualTargets := []string{}
	expectedTargets := []string{"test", "happycloud", "null", "grandchild"}

	policyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) policy.EvaluationResponse {
		actualTargets = append(actualTargets, req.Target)
		var expected *proto.PolicyEvaluateProviderRequest_ProviderMetadata
		switch req.Target {
		case "test":
			expected = &proto.PolicyEvaluateProviderRequest_ProviderMetadata{
				Name:      "test",
				Namespace: "hashicorp",
				Source:    "registry.terraform.io/hashicorp/test",
				Version:   "1.2.3",
			}
		case "happycloud":
			expected = &proto.PolicyEvaluateProviderRequest_ProviderMetadata{
				Name:      "happycloud",
				Namespace: "awesomecorp",
				Source:    "tf.example.com/awesomecorp/happycloud",
				Version:   "1.0.0",
			}
		case "null":
			expected = &proto.PolicyEvaluateProviderRequest_ProviderMetadata{
				Name:      "null",
				Namespace: "hashicorp",
				Source:    "registry.terraform.io/hashicorp/null",
				Version:   "2.0.1",
			}
		case "grandchild":
			expected = &proto.PolicyEvaluateProviderRequest_ProviderMetadata{
				Name:      "grandchild",
				Namespace: "hashicorp",
				Source:    "registry.terraform.io/hashicorp/grandchild",
				Version:   "1.0.0",
			}
		}
		if diff := cmp.Diff(expected, req.Meta, protocmp.Transform(), cmp.Comparer(cty.Value.RawEquals)); diff != "" {
			t.Fatalf("wrong provider metadata\ngot:  %s\nwant: %v", diff, expected)
		}

		t.Cleanup(func() {
			if !slices.Contains(expectedTargets, req.Target) {
				t.Errorf("expected target %q to be in %v", req.Target, expectedTargets)
			}
		})

		if req.Target == "test" {
			// This unknown evaluation should still lead to success of the init operation.
			return policy.EvaluationResponse{Overall: policy.UnknownResult}
		}
		if req.Target == "happycloud" {
			// Advisory policies may return deny without error diagnostics and should
			// not block init.
			return policy.EvaluationResponse{Overall: policy.DenyResult}
		}
		return policy.EvaluationResponse{Overall: policy.AllowResult}
	}
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td})
	output := done(t)
	if code != 0 {
		t.Fatalf("got exit status %d; want 0\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateProviderCalled {
		t.Fatal("expected EvaluateProvider to be called")
	}
}

func TestInit_WithProviderPolicyDiagnostics(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-nested-provider-requirements"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test":                        {"1.2.3", "1.2.4"},
		"tf.example.com/awesomecorp/happycloud": {"1.0.0"},
		"hashicorp/null":                        {"2.0.1"},
		"hashicorp/grandchild":                  {"1.0.0"},
	})

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	expectedTargets := []string{"test", "happycloud", "null", "grandchild"}
	actualTargets := []string{}
	policyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) policy.EvaluationResponse {
		actualTargets = append(actualTargets, req.Target)
		return policy.EvaluationFromProtoResponse(
			proto.EvaluateResult_DENY_EVALUATE_RESULT,
			[]*proto.PolicyEvaluationDetail{
				{
					Address:              "provider_policy." + req.Target,
					Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
					File:                 "policy_file.tfpolicy.hcl",
					PolicySetEnforcement: "mandatory",
					Diagnostics: []*proto.Diagnostic{
						{
							Severity: proto.Severity_ERROR,
							Summary:  req.Target + " provider policy denied",
							Result: &proto.DiagnosticResult{
								Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
							},
						},
					},
				},
			},
		)
	}
	overs := overrides
	overs.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overs,
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-no-color"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateProviderCalled {
		t.Fatal("expected EvaluateProvider to be called")
	}

	if got, want := len(actualTargets), len(expectedTargets); got != want {
		t.Fatalf("wrong number of evaluated providers\ngot:  %d (%v)\nwant: %d", got, actualTargets, want)
	}
	for _, target := range expectedTargets {
		if !slices.Contains(actualTargets, target) {
			t.Fatalf("missing provider evaluation for %q in %v", target, actualTargets)
		}
	}

	stderr := output.Stderr()
	for _, target := range expectedTargets {
		if !strings.Contains(stderr, "Error: "+target+" provider policy denied") {
			t.Fatalf("missing policy diagnostic for %q in stderr:\n%s", target, stderr)
		}
	}
	if !strings.Contains(stderr, "Provider download blocked due to policy violations. Please review other diagnostics for details.") {
		t.Fatalf("missing provider policy error in stderr:\n%s", stderr)
	}
}

func TestInit_WithProviderPolicyJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("init-nested-provider-requirements"), td)
	t.Chdir(td)

	providerSource := newMockProviderSource(t, map[string][]string{
		"hashicorp/test":         {"1.2.3", "1.2.4"},
		"awesomecorp/happycloud": {"1.0.0"},
		"hashicorp/null":         {"2.0.1"},
		"hashicorp/grandchild":   {"1.0.0"},
	})

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
	resp := policy.EvaluationFromProtoResponse(
		proto.EvaluateResult_DENY_EVALUATE_RESULT,
		[]*proto.PolicyEvaluationDetail{
			{
				Address:              "provider_policy.example",
				Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
				File:                 "policy_file.tfpolicy.hcl",
				PolicySetEnforcement: "mandatory",
				Diagnostics: []*proto.Diagnostic{
					{
						Severity: proto.Severity_ERROR,
						Summary:  "provider policy denied",
						Result: &proto.DiagnosticResult{
							Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						},
					},
				},
			},
		},
	)
	policyClient.EvaluateProviderResponse = &resp
	overrides.PolicyClient = policyClient

	c := &InitCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			Ui:                        ui,
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		},
	}

	code := c.Run([]string{"-policies", td, "-no-color", "-json"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateProviderCalled {
		t.Fatal("expected EvaluateProvider to be called")
	}

	allOutput := strings.SplitSeq(output.Stdout(), "\n")
	var foundPolicyDiagnostic bool
	for line := range allOutput {
		if strings.Contains(line, `"type":"policy_diagnostic"`) {
			foundPolicyDiagnostic = strings.Contains(line, "provider policy denied")
		}
	}
	if !foundPolicyDiagnostic {
		t.Fatal("expected diagnostic output")
	}
}
