// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"os"
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

	expected := `
Error: policy denied

  on policy_file.tfpolicy.hcl line 1:
   1: 		resource_policy "resource_type" "policy_name" {
   2: 		  enforce_attrs {
   3: 		    key = attr.value == "foo"
   4: 		  }
   5: 		}
   6: 	

  while evaluating policy for main.tf line 1:
   1: resource "test_instance" "foo" {

`

	if diff := cmp.Diff(expected, output.Stderr()); diff != "" {
		t.Fatalf("unexpected output:\n%s", diff)
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

	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.2"}
{"@level":"info","@message":"data.test_data_source.a: Refreshing...","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read"},"type":"apply_start"}
{"@level":"info","@message":"data.test_data_source.a: Refresh complete after 0s [id=zzzzz]","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read","id_key":"id","id_value":"zzzzz","elapsed_seconds":0},"type":"apply_complete"}
{"@level":"info","@message":"test_instance.foo: Plan to create","@module":"terraform.ui","change":{"resource":{"addr":"test_instance.foo","implied_provider":"test","module":"","resource":"test_instance.foo","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"planned_change"}
{"@level":"info","@message":"Plan: 1 to add, 0 to change, 0 to destroy.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"plan"},"type":"change_summary"}
{"@level":"error","@message":"Error: policy denied","@module":"terraform.ui","@policy":"true","target_address":"test_instance.foo","policy_diagnostic":{"severity":"error","summary":"policy denied","detail":"","range":{"filename":"main.tf","start":{"line":1,"column":1,"byte":0},"end":{"line":1,"column":31,"byte":30}},"snippet":{"context":null,"code":"resource \"test_instance\" \"foo\" {","start_line":1,"highlight_start_offset":0,"highlight_end_offset":30,"values":[]},"policy_range":{"filename":"policy_file.tfpolicy.hcl","start":{"line":1,"column":1,"byte":0},"end":{"line":2,"column":4,"byte":0}},"policy_snippet":{"context":null,"code":"\t\tresource_policy \"resource_type\" \"policy_name\" {\n\t\t  enforce_attrs {\n\t\t    key = attr.value == \"foo\"\n\t\t  }\n\t\t}\n\t","start_line":1,"highlight_start_offset":1,"highlight_end_offset":100,"values":null}},"policy_metadata":{"enforce_index":1,"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.foo","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_diagnostic"}
{"@level":"error","@message":"Error: policy failed for some other reason","@module":"terraform.ui","@policy":"true","target_address":"test_instance.foo","policy_diagnostic":{"severity":"error","summary":"policy failed for some other reason","detail":"","range":{"filename":"main.tf","start":{"line":1,"column":1,"byte":0},"end":{"line":1,"column":31,"byte":30}},"snippet":{"context":null,"code":"resource \"test_instance\" \"foo\" {","start_line":1,"highlight_start_offset":0,"highlight_end_offset":30,"values":[]},"policy_range":{"filename":"policy_file.tfpolicy.hcl","start":{"line":1,"column":1,"byte":0},"end":{"line":2,"column":4,"byte":0}},"policy_snippet":{"context":null,"code":"\t\tresource_policy \"resource_type\" \"policy_name\" {\n\t\t  enforce_attrs {\n\t\t    key = attr.value == \"foo\"\n\t\t  }\n\t\t}\n\t","start_line":1,"highlight_start_offset":1,"highlight_end_offset":100,"values":null}},"policy_metadata":{"enforce_index":2,"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.bar","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_diagnostic"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","target_address":"test_instance.foo","policy_address":"resource_policy.foo","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.foo","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_result"}
{"@level":"info","@message":"Policy Result","@module":"terraform.ui","@policy":"true","target_address":"test_instance.foo","policy_address":"resource_policy.bar","policy_metadata":{"policy_set_path":"policy_file.tfpolicy.hcl","policy_name":"resource_policy.bar","file_name":"policy_file.tfpolicy.hcl","enforcement_level":"mandatory"},"result":"DenyResult","type":"policy_result"}`

	checkGoldenReferenceStr(t, output, expected)
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

	expected := `data.test_data_source.a: Reading...
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

Warning: policy with unknowns

  on policy_file.tfpolicy.hcl line 1:
   1: 		resource_policy "resource_type" "policy_name" {
   2: 		  enforce_attrs {
   3: 		    key = attr.value == "foo"
   4: 		  }
   5: 		}
   6: 	

  while evaluating policy for main.tf line 1:
   1: resource "test_instance" "foo" {


─────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't
guarantee to take exactly these actions if you run "terraform apply" now.
`

	if actual, diff := output.Stdout(), cmp.Diff(expected, output.Stdout()); diff != "" {
		t.Fatalf("unexpected output:\n%s. \nDiff: %s", actual, diff)
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

	expected := `data.test_data_source.a: Reading...
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

Policy Info:
on policy_file.tfpolicy.hcl line 3, in resource_policy "test_policy" "name"
"Something about this enforcement"

on main.tf line 1, in resource "test_instance" "foo"

Policy Info:
on provider_policy_file.tfpolicy.hcl line 3, in provider_policy "test_policy" "name"
"Something about this enforcement"



─────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't
guarantee to take exactly these actions if you run "terraform apply" now.
`

	if actual, diff := output.Stdout(), cmp.Diff(expected, output.Stdout()); diff != "" {
		t.Fatalf("unexpected output:\n%s. \nDiff: %s", actual, diff)
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

	expected := `{"@level":"info","@message":"Terraform 1.15.0-dev","@module":"terraform.ui","terraform":"1.15.0-dev","type":"version","ui":"1.3"}
{"@level":"error","@message":"Error: Failed to connect to policy engine","@module":"terraform.ui","@policy":"true","policy_diagnostic":{"severity":"error","summary":"Failed to connect to policy engine","detail":"Failed to connect to policy engine: failed to connect to plugin: exec: \"tfpolicy-plugin\": executable file not found in $PATH."},"policy_metadata":{},"result":"SetupErrorResult","type":"policy_diagnostic"}
{"@level":"info","@message":"data.test_data_source.a: Refreshing...","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read"},"type":"apply_start"}
{"@level":"info","@message":"data.test_data_source.a: Refresh complete after 0s [id=zzzzz]","@module":"terraform.ui","hook":{"resource":{"addr":"data.test_data_source.a","module":"","resource":"data.test_data_source.a","implied_provider":"test","resource_type":"test_data_source","resource_name":"a","resource_key":null},"action":"read","id_key":"id","id_value":"zzzzz","elapsed_seconds":0},"type":"apply_complete"}
{"@level":"info","@message":"test_instance.foo: Plan to create","@module":"terraform.ui","change":{"resource":{"addr":"test_instance.foo","module":"","resource":"test_instance.foo","implied_provider":"test","resource_type":"test_instance","resource_name":"foo","resource_key":null},"action":"create"},"type":"planned_change"}
{"@level":"info","@message":"Plan: 1 to add, 0 to change, 0 to destroy.","@module":"terraform.ui","changes":{"add":1,"change":0,"import":0,"remove":0,"action_invocation":0,"operation":"plan"},"type":"change_summary"}`
	fmt.Println(output.Stdout())
	checkGoldenReferenceStr(t, output, expected)
}
