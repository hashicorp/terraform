// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// Tests the output of a plan that includes a policy evaluation
func TestPlan_WithPolicy(t *testing.T) {

	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)
	policyClient := policy.NewTestMockClient(t)
	overrides.PolicyClient = policyClient
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}
	resp := policy.EvaluationFromProtoResponse(
		proto.EvaluateResult_DENY_EVALUATE_RESULT, []*proto.PolicyEvaluationDetail{
			{
				Address: "resource_policy.foo",
				Result:  proto.EvaluateResult_DENY_EVALUATE_RESULT,
				File:    "policy_file.tfpolicy.hcl",
				DefRange: &proto.Range{
					Filename: "policy_file.tfpolicy.hcl",
					Start: &proto.Position{
						Line:   1,
						Column: 1,
					},
					End: &proto.Position{
						Line:   2,
						Column: 4,
					},
				},
				EnforceResults: []*proto.EnforceBlockResult{
					{
						Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostics: []*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "policy denied",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   2,
										Column: 4,
									},
								},
								Context: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   4,
										Column: 10,
									},
								},
								Snippet: &proto.Snippet{
									Code:                 policyCode,
									StartLine:            1,
									HighlightStartOffset: 1,
									HighlightEndOffset:   100,
								},
							},
						},
					},
				},
			},
		})
	policyClient.EvaluateResponse = &resp

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if !strings.Contains(output.Stderr(), "policy denied") {
		t.Fatalf("expected policy diagnostic in stderr, got:\n%s", output.Stderr())
	}
}

func TestPlan_WithPolicyDiagnosticsJSON(t *testing.T) {

	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)
	policyClient := policy.NewTestMockClient(t)
	overrides.PolicyClient = policyClient
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}
	resp := policy.EvaluationFromProtoResponse(
		proto.EvaluateResult_DENY_EVALUATE_RESULT, []*proto.PolicyEvaluationDetail{
			{
				Address:              "resource_policy.foo",
				Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
				File:                 "policy_file.tfpolicy.hcl",
				PolicySetEnforcement: "mandatory",
				DefRange: &proto.Range{
					Filename: "policy_file.tfpolicy.hcl",
					Start: &proto.Position{
						Line:   1,
						Column: 1,
					},
					End: &proto.Position{
						Line:   2,
						Column: 4,
					},
				},
				EnforceResults: []*proto.EnforceBlockResult{
					{
						Result:     proto.EvaluateResult_DENY_EVALUATE_RESULT,
						BlockIndex: 1,
						Diagnostics: []*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "policy denied",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   2,
										Column: 4,
									},
								},
								Context: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   4,
										Column: 10,
									},
								},
								Snippet: &proto.Snippet{
									Code:                 policyCode,
									StartLine:            1,
									HighlightStartOffset: 1,
									HighlightEndOffset:   100,
								},
							},
						},
					},
				},
			},
			{
				Address:              "resource_policy.bar",
				Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
				File:                 "policy_file.tfpolicy.hcl",
				PolicySetEnforcement: "mandatory",
				DefRange: &proto.Range{
					Filename: "policy_file.tfpolicy.hcl",
					Start: &proto.Position{
						Line:   1,
						Column: 1,
					},
					End: &proto.Position{
						Line:   2,
						Column: 4,
					},
				},
				EnforceResults: []*proto.EnforceBlockResult{
					{
						Result:     proto.EvaluateResult_DENY_EVALUATE_RESULT,
						BlockIndex: 2,
						Diagnostics: []*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "policy failed for some other reason",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   2,
										Column: 4,
									},
								},
								Context: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start: &proto.Position{
										Line:   1,
										Column: 1,
									},
									End: &proto.Position{
										Line:   4,
										Column: 10,
									},
								},
								Snippet: &proto.Snippet{
									Code:                 policyCode,
									StartLine:            1,
									HighlightStartOffset: 1,
									HighlightEndOffset:   100,
								},
							},
						},
					},
				},
			},
		})
	policyClient.EvaluateResponse = &resp

	// implicit allow, in a case where the evaluated provider matched no policy in the engine
	policyClient.EvaluateProviderResponse = &policy.EvaluationResponse{
		Overall: policy.AllowResult,
		Policies: []*policy.Policy{{
			Result:           policy.AllowResult,
			PolicySetName:    "some_policy_set",
			Address:          "policy_name",
			Directory:        "some/path/to",
			Filename:         "provider_policy_file.tfpolicy.hcl",
			EnforcementLevel: "mandatory",
			Range: &hcl.Range{
				Filename: "provider_policy_file.tfpolicy.hcl",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
				},
				End: hcl.Pos{
					Line:   5,
					Column: 12,
				},
			},
		},
		},
	}

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color", "-json"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	all := output.All()
	for _, want := range []string{"policy denied", "policy failed for some other reason", "resource_policy.foo", "resource_policy.bar"} {
		if !strings.Contains(all, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, all)
		}
	}
}

func TestPlan_WithPolicyUnknown(t *testing.T) {

	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)
	policyClient := policy.NewTestMockClient(t)
	overrides.PolicyClient = policyClient
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	resp := policy.EvaluationFromProtoResponse(proto.EvaluateResult_UNKNOWN_EVALUATE_RESULT, []*proto.PolicyEvaluationDetail{
		{
			Result: proto.EvaluateResult_UNKNOWN_EVALUATE_RESULT,
			Diagnostics: []*proto.Diagnostic{
				{
					Severity: proto.Severity_WARNING,
					Summary:  "policy with unknowns",
					Result: &proto.DiagnosticResult{
						Result: proto.EvaluateResult_UNKNOWN_EVALUATE_RESULT,
					},
					Subject: &proto.Range{
						Filename: "policy_file.tfpolicy.hcl",
						Start: &proto.Position{
							Line:   1,
							Column: 1,
						},
						End: &proto.Position{
							Line:   2,
							Column: 4,
						},
					},
					Context: &proto.Range{
						Filename: "policy_file.tfpolicy.hcl",
						Start: &proto.Position{
							Line:   1,
							Column: 1,
						},
						End: &proto.Position{
							Line:   4,
							Column: 10,
						},
					},
					Snippet: &proto.Snippet{
						Code:                 policyCode,
						StartLine:            1,
						HighlightStartOffset: 1,
						HighlightEndOffset:   100,
					},
				},
			},
		},
	})
	policyClient.EvaluateResponse = &resp

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	all := output.All()
	if !strings.Contains(all, "policy with unknowns") {
		t.Fatalf("expected policy warning in output, got:\n%s", all)
	}
}

func TestPlan_WithPolicySuccessInfo(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	providerSource := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)
	policyClient := policy.NewTestMockClient(t)
	overrides.PolicyClient = policyClient
	meta := Meta{
		testingOverrides:          overrides,
		View:                      view,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
	}

	view, done = testView(t)
	meta.View = view

	c := &PlanCommand{
		Meta: meta,
	}

	policyObj := &policy.Policy{
		Result:           policy.AllowResult,
		PolicySetName:    "some_policy_set",
		Address:          "policy_name",
		Directory:        "some/path/to",
		Filename:         "provider_policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
		Range: &hcl.Range{
			Filename: "provider_policy_file.tfpolicy.hcl",
			Start: hcl.Pos{
				Line:   1,
				Column: 1,
			},
			End: hcl.Pos{
				Line:   5,
				Column: 12,
			},
		},
	}

	policyClient.EvaluateProviderFn = func(ctx context.Context, req policy.EvaluationRequest[*proto.ProviderMetadata]) policy.EvaluationResponse {
		if req.Meta.Version != "1.0.0" {
			t.Fatalf("Expected provider version to be 1.0.0")
		}

		return policy.EvaluationResponse{
			Overall:  policy.AllowResult,
			Policies: []*policy.Policy{policyObj},
			Enforcements: []policy.EnforcementResult{
				{
					Result:     policy.AllowResult,
					Message:    "Something about this enforcement",
					BlockIndex: 1,
					Snippet: &proto.Snippet{
						Code:                 "provider_policy \"test_policy\" \"name\"",
						StartLine:            1,
						HighlightStartOffset: 1,
						HighlightEndOffset:   100,
					},
					Range: &hcl.Range{
						Filename: "provider_policy_file.tfpolicy.hcl",
						Start: hcl.Pos{
							Line:   3,
							Column: 5,
						},
						End: hcl.Pos{
							Line:   4,
							Column: 10,
						},
					},
					Policy: policyObj,
				},
			},
		}
	}

	policyObj = &policy.Policy{
		Result:           policy.AllowResult,
		PolicySetName:    "some_policy_set",
		Address:          "policy_name",
		Directory:        "some/path/to",
		Filename:         "policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
		Range: &hcl.Range{
			Filename: "policy_file.tfpolicy.hcl",
			Start: hcl.Pos{
				Line:   1,
				Column: 1,
			},
			End: hcl.Pos{
				Line:   5,
				Column: 12,
			},
		},
	}
	policyClient.EvaluateResponse = &policy.EvaluationResponse{
		Overall:  policy.AllowResult,
		Policies: []*policy.Policy{policyObj},
		Enforcements: []policy.EnforcementResult{
			{
				Result:     policy.AllowResult,
				Message:    "Something about this enforcement",
				BlockIndex: 1,
				Snippet: &proto.Snippet{
					Code:                 "resource_policy \"test_policy\" \"name\"",
					StartLine:            1,
					HighlightStartOffset: 1,
					HighlightEndOffset:   100,
				},
				Range: &hcl.Range{
					Filename: "policy_file.tfpolicy.hcl",
					Start: hcl.Pos{
						Line:   3,
						Column: 5,
					},
					End: hcl.Pos{
						Line:   4,
						Column: 10,
					},
				},
				Policy: policyObj,
			},
		},
	}

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	all := output.All()
	if !strings.Contains(all, "Something about this enforcement") {
		t.Fatalf("expected policy info message in output, got:\n%s", all)
	}
}

func TestPlan_WithPolicySuccessInfoJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)
	policyClient := policy.NewTestMockClient(t)
	overrides.PolicyClient = policyClient
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	policyObj := &policy.Policy{
		Result:           policy.AllowResult,
		PolicySetName:    "some_policy_set",
		Address:          "policy_name",
		Directory:        "some/path/to",
		Filename:         "provider_policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
		Range: &hcl.Range{
			Filename: "provider_policy_file.tfpolicy.hcl",
			Start: hcl.Pos{
				Line:   1,
				Column: 1,
			},
			End: hcl.Pos{
				Line:   5,
				Column: 12,
			},
		},
	}

	policyClient.EvaluateProviderResponse = &policy.EvaluationResponse{
		Overall:  policy.AllowResult,
		Policies: []*policy.Policy{policyObj},
		Enforcements: []policy.EnforcementResult{
			{
				Result:     policy.AllowResult,
				Message:    "Something about this enforcement",
				BlockIndex: 1,
				Snippet: &proto.Snippet{
					Code:                 "provider_policy \"test_policy\" \"name\"",
					StartLine:            1,
					HighlightStartOffset: 1,
					HighlightEndOffset:   100,
				},
				Range: &hcl.Range{
					Filename: "provider_policy_file.tfpolicy.hcl",
					Start: hcl.Pos{
						Line:   3,
						Column: 5,
					},
					End: hcl.Pos{
						Line:   4,
						Column: 10,
					},
				},
				Policy: policyObj,
			},
		},
	}

	policyObj = &policy.Policy{
		Result:           policy.AllowResult,
		PolicySetName:    "some_policy_set",
		Address:          "policy_name",
		Directory:        "some/path/to",
		Filename:         "policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
		Range: &hcl.Range{
			Filename: "policy_file.tfpolicy.hcl",
			Start: hcl.Pos{
				Line:   1,
				Column: 1,
			},
			End: hcl.Pos{
				Line:   5,
				Column: 12,
			},
		},
	}
	policyClient.EvaluateResponse = &policy.EvaluationResponse{
		Overall:  policy.AllowResult,
		Policies: []*policy.Policy{policyObj},
		Enforcements: []policy.EnforcementResult{
			{
				Result:     policy.AllowResult,
				Message:    "Something about this enforcement",
				BlockIndex: 1,
				Snippet: &proto.Snippet{
					Code:                 "resource_policy \"test_policy\" \"name\"",
					StartLine:            1,
					HighlightStartOffset: 1,
					HighlightEndOffset:   100,
				},
				Range: &hcl.Range{
					Filename: "policy_file.tfpolicy.hcl",
					Start: hcl.Pos{
						Line:   3,
						Column: 5,
					},
					End: hcl.Pos{
						Line:   4,
						Column: 10,
					},
				},
				Policy: policyObj,
			},
		},
	}

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color", "-json"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	checkGoldenReference(t, output, "plan-policy")
}

func TestPlan_WithPolicySetupFailure(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)

	// We intentionally do not pass a policy client override here so the command
	// exercises the real policy client initialization path and emits any setup
	// diagnostics from attempting to connect to the policy engine.
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color"))
	output := done(t)
	// expect the operation to be a success
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	// we still display the policy output
	// and the plan still succeeds
	expectedOut := `
Error: Failed to connect to policy engine

Failed to connect to policy engine: failed to connect to plugin: exec:
"tfpolicy-plugin": executable file not found in $PATH.
data.test_data_source.a: Reading...
data.test_data_source.a: Read complete after 0s [id=zzzzz]

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_instance.foo will be created
  + resource "test_instance" "foo" {
      + ami = "bar"

      + network_interface {
          + description  = "Main network interface"
          + device_index = "0"
        }
    }

Plan: 1 to add, 0 to change, 0 to destroy.

─────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't
guarantee to take exactly these actions if you run "terraform apply" now.
`

	if diff := cmp.Diff(expectedOut, output.All()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
	}
}

func TestPlan_WithPolicySetupFailureJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan"), td)
	t.Chdir(td)
	policyCode := `		resource_policy "resource_type" "policy_name" {
		  enforce_attrs {
		    key = attr.value == "foo"
		  }
		}
	`
	if err := os.WriteFile("policy.hcl", []byte(policyCode), 0644); err != nil {
		t.Fatal(err)
	}

	p := planFixtureProvider()
	view, done := testView(t)
	overrides := metaOverridesForProvider(p)

	// We intentionally do not pass a policy client override here so the command
	// exercises the real policy client initialization path and emits any setup
	// diagnostics from attempting to connect to the policy engine.
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}

	args := []string{"-policies", td}
	code := c.Run(append(args, "-no-color", "-json"))
	output := done(t)
	// expect the operation to be a success
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	all := output.All()
	for _, want := range []string{"Failed to connect to policy engine", "Plan: 1 to add, 0 to change, 0 to destroy."} {
		if !strings.Contains(all, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, all)
		}
	}
}
