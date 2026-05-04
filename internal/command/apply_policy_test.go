// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"os"
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

	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.2"}
{"@level":"info","@message":"data.test_data_source.a: Refreshing...","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read"},"type":"apply_start"}
{"@level":"info","@message":"data.test_data_source.a: Refresh complete after 0s [id=zzzzz]","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read","id_key":"id","id_value":"zzzzz","elapsed_seconds":0},"type":"apply_complete"}
{"@level":"info","@message":"test_instance.foo: Plan to create","@module":"terraform.ui","change":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"planned_change"}
{"@level":"info","@message":"Plan: 1 to add, 0 to change, 0 to destroy.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"plan"},"type":"change_summary"}
{"@level":"info","@message":"test_instance.foo: Creating...","@module":"terraform.ui","hook":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"apply_start"}
{"@level":"info","@message":"test_instance.foo: Creation complete after 0s","@module":"terraform.ui","hook":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create","elapsed_seconds":0},"type":"apply_complete"}
{"@level":"error","@message":"Error: policy denied","@module":"terraform.ui","@policy":"true","target_address":"test_instance.foo","policy_diagnostic":{"severity":"error","summary":"policy denied","detail":"","range":{"filename":"main.tf","start":{"line":1,"column":1,"byte":0},"end":{"line":1,"column":31,"byte":30}},"snippet":{"context":null,"code":"resource \"test_instance\" \"foo\" {","start_line":1,"highlight_start_offset":0,"highlight_end_offset":30,"values":[]},"policy_range":{"filename":"policy_file.tfpolicy.hcl","start":{"line":1,"column":1,"byte":0},"end":{"line":2,"column":4,"byte":0}},"policy_snippet":{"context":null,"code":"\t\tresource_policy \"resource_type\" \"policy_name\" {\n\t\t  enforce_attrs {\n\t\t    key = attr.value == \"foo\"\n\t\t  }\n\t\t}\n\t","start_line":1,"highlight_start_offset":1,"highlight_end_offset":100,"values":null}},"policy_metadata":{"enforce_index":1,"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.foo","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_diagnostic"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","policy_address":"resource_policy.foo","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.foo","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","target_address":"test_instance.foo","type":"policy_result"}
{"@level":"info","@message":"Apply complete! Resources: 1 added, 0 changed, 0 destroyed.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"apply"},"type":"change_summary"}
{"@level":"info","@message":"Outputs: 0","@module":"terraform.ui","outputs":{},"type":"outputs"}`
	checkGoldenReferenceStr(t, output, expected)
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

	// The resulting json only contains the policy result, because the object that
	// had a failed policy evaluation during the plan succeeded during apply.
	// This can occur when more references become known.
	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.2"}
{"@level":"info","@message":"data.test_data_source.a: Refreshing...","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read"},"type":"apply_start"}
{"@level":"info","@message":"data.test_data_source.a: Refresh complete after 0s [id=zzzzz]","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read","id_key":"id","id_value":"zzzzz","elapsed_seconds":0},"type":"apply_complete"}
{"@level":"info","@message":"test_instance.foo: Plan to create","@module":"terraform.ui","change":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"planned_change"}
{"@level":"info","@message":"Plan: 1 to add, 0 to change, 0 to destroy.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"plan"},"type":"change_summary"}
{"@level":"info","@message":"test_instance.foo: Creating...","@module":"terraform.ui","hook":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"apply_start"}
{"@level":"info","@message":"test_instance.foo: Creation complete after 0s","@module":"terraform.ui","hook":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create","elapsed_seconds":0,"id_key":"id"},"type":"apply_complete"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","policy_address":"resource_policy.foo","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.foo","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"AllowResult","target_address":"test_instance.foo","type":"policy_result"}
{"@level":"info","@message":"Apply complete! Resources: 1 added, 0 changed, 0 destroyed.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"apply"},"type":"change_summary"}
{"@level":"info","@message":"Outputs: 0","@module":"terraform.ui","outputs":{},"type":"outputs"}`
	checkGoldenReferenceStr(t, output, expected)
}
