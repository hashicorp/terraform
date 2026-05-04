// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

func TestInit_WithModulePolicy(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("dynamic-module-sources/local-source-with-variable"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)

	overrides := metaOverridesForProvider(testProvider())
	policyClient := policy.NewTestMockClient(t)
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

	if got, want := policyClient.EvaluateModuleRequest.Meta.Source, "./modules/example"; got != want {
		t.Fatalf("wrong module source\ngot:  %q\nwant: %q", got, want)
	}

	if got, want := policyClient.EvaluateModuleRequest.Meta.Version, ""; got != want {
		t.Fatalf("wrong module version\ngot:  %q\nwant: %q", got, want)
	}
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
{"@level":"info","@message":"Initializing modules...","@module":"terraform.ui","message_code":"initializing_modules_message","type":"init_output"}
{"@level":"error","@message":"Error: module policy denied","@module":"terraform.ui","@policy":"true","policy_diagnostic":{"severity":"error","summary":"module policy denied","detail":"","range":{"filename":"main.tf","start":{"line":6,"column":1,"byte":60},"end":{"line":6,"column":17,"byte":76}},"snippet":{"context":null,"code":"module \"example\" {","start_line":6,"highlight_start_offset":0,"highlight_end_offset":16,"values":[]}},"policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"module_policy.example","file_name":".","enforcement_level":"mandatory"},"result":"DenyResult","target_address":"module.example","type":"policy_diagnostic"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","target_address":"module.example","policy_address":"module_policy.example","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"module_policy.example","file_name":".","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_result"}
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

	policyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ProviderMetadata]) policy.EvaluationResponse {
		actualTargets = append(actualTargets, req.Target)
		var expected *proto.ProviderMetadata
		switch req.Target {
		case "test":
			expected = &proto.ProviderMetadata{
				Name:       "test",
				Namespace:  "hashicorp",
				Type:       "test",
				Source:     "registry.terraform.io/hashicorp/test",
				Version:    "1.2.3",
				ModulePath: "",
			}
		case "happycloud":
			expected = &proto.ProviderMetadata{
				Name:       "happycloud",
				Namespace:  "awesomecorp",
				Type:       "happycloud",
				Source:     "tf.example.com/awesomecorp/happycloud",
				Version:    "1.0.0",
				ModulePath: "./modules/child",
			}
		case "null":
			expected = &proto.ProviderMetadata{
				Name:       "null",
				Namespace:  "hashicorp",
				Type:       "null",
				Source:     "registry.terraform.io/hashicorp/null",
				Version:    "2.0.1",
				ModulePath: "./modules/child",
			}
		case "grandchild":
			expected = &proto.ProviderMetadata{
				Name:       "grandchild",
				Namespace:  "hashicorp",
				Type:       "grandchild",
				Source:     "registry.terraform.io/hashicorp/grandchild",
				Version:    "1.0.0",
				ModulePath: "./grandchild",
			}
		}
		if diff := cmp.Diff(expected, req.Meta, protocmp.Transform()); diff != "" {
			t.Fatalf("wrong provider metadata\ngot:  %s\nwant: %v", diff, expected)
		}

		t.Cleanup(func() {
			if !slices.Contains(expectedTargets, req.Target) {
				t.Errorf("expected target %q to be in %v", req.Target, expectedTargets)
			}
		})

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

	code := c.Run([]string{"-policies", td, "-no-color"})
	output := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, output.Stderr(), output.Stdout())
	}

	if !policyClient.EvaluateProviderCalled {
		t.Fatal("expected EvaluateProvider to be called")
	}

	stderr := output.Stderr()
	expected := `
Error: provider policy denied


Error: Provider download failed due to policy violations. Please review other diagnostics for details.

`
	if diff := cmp.Diff(expected, stderr); diff != "" {
		t.Fatalf("unexpected stderr:\n%s", diff)
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
