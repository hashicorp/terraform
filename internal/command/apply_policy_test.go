// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

// This tests that the apply policy diagnostics are reported.
func TestApply_WithPolicyDiagnosticsJSON(t *testing.T) {

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
	c := &ApplyCommand{
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
	code := c.Run(append(args, "-no-color", "-json", "-auto-approve"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	all := output.All()
	if !strings.Contains(all, "policy denied") {
		t.Fatalf("expected %q in output, got:\n%s", "policy denied", all)
	}
}

// This tests that the plan policy diagnostic is superceded by the apply policy evaluation.
func TestApply_WithPlanPolicyDiagnosticsJSON(t *testing.T) {

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
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides:          overrides,
			View:                      view,
			AllowExperimentalFeatures: true,
		},
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
		s := req.ProposedNewState.AsValueMap()
		s["id"] = cty.UnknownVal(cty.String)
		resp.PlannedState = cty.ObjectVal(s)
		return
	}
	evalRespFn := func(result proto.EvaluateResult) policy.EvaluationResponse {
		detail := &proto.PolicyEvaluationDetail{
			Address:              "resource_policy.foo",
			Result:               result,
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
			EnforceResults: []*proto.EnforceBlockResult{{
				Result: result,
			}},
		}
		if result == proto.EvaluateResult_DENY_EVALUATE_RESULT {
			detail.EnforceResults[0].Diagnostics = []*proto.Diagnostic{
				{
					Severity: proto.Severity_ERROR,
					Summary:  "policy denied",
					Result: &proto.DiagnosticResult{
						Result: result,
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
			}
		}
		return policy.EvaluationFromProtoResponse(result, []*proto.PolicyEvaluationDetail{detail})
	}

	policyClient.EvaluateFn = func(ctx context.Context, er policy.EvaluationRequest[*proto.ResourceMetadata]) policy.EvaluationResponse {
		// This is what is returned during the post-plan policy evaluation
		if !er.Attrs.GetAttr("id").IsWhollyKnown() {
			return evalRespFn(proto.EvaluateResult_DENY_EVALUATE_RESULT)
		}

		// This is for the post-apply policy evaluation
		return evalRespFn(proto.EvaluateResult_ALLOW_EVALUATE_RESULT)
	}

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
	code := c.Run(append(args, "-no-color", "-json", "-auto-approve"))
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stdout())
	}

	all := output.All()
	if !strings.Contains(all, "AllowResult") {
		t.Fatalf("expected %q in output, got:\n%s", "AllowResult", all)

	}
	if strings.Contains(all, "policy denied") {
		t.Fatalf("expected apply-time policy result to supersede plan-time denial, got:\n%s", all)
	}
}
