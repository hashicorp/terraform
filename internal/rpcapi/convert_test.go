// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	stackhooks "github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestDiagnosticsToProto(t *testing.T) {
	tests := map[string]struct {
		Input tfdiags.Diagnostics
		Want  []*terraform1.Diagnostic
	}{
		"nil": {
			Input: nil,
			Want:  nil,
		},
		"empty": {
			Input: make(tfdiags.Diagnostics, 0, 5),
			Want:  nil,
		},
		"sourceless": {
			Input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Something annoying",
					"But I'll get over it.",
				),
			},
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
				},
			},
		},
		"warning": {
			Input: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"I have a very bad feeling about this",
					"That's no moon; it's a space station.",
				),
			},
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_WARNING,
					Summary:  "I have a very bad feeling about this",
					Detail:   "That's no moon; it's a space station.",
				},
			},
		},
		"with subject": {
			Input: tfdiags.Diagnostics{}.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
				},
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
				},
			},
		},
		"with subject and context": {
			Input: tfdiags.Diagnostics{}.Append(
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
					Context: &hcl.Range{
						Filename: "git::https://example.com/foo.git",
						Start:    hcl.InitialPos,
						End: hcl.Pos{
							Byte:   5,
							Line:   6,
							Column: 7,
						},
					},
				},
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "Something annoying",
					Detail:   "But I'll get over it.",
					Subject: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   2,
							Line:   3,
							Column: 4,
						},
					},
					Context: &terraform1.SourceRange{
						SourceAddr: "git::https://example.com/foo.git",
						Start: &terraform1.SourcePos{
							Byte:   0,
							Line:   1,
							Column: 1,
						},
						End: &terraform1.SourcePos{
							Byte:   5,
							Line:   6,
							Column: 7,
						},
					},
				},
			},
		},
		"with only severity and summary": {
			// This is the kind of degenerate diagnostic we produce when
			// we're just naively wrapping a Go error, as tends to arise
			// in providers that are just passing through their SDK errors.
			Input: tfdiags.Diagnostics{}.Append(
				fmt.Errorf("oh no bad"),
			),
			Want: []*terraform1.Diagnostic{
				{
					Severity: terraform1.Diagnostic_ERROR,
					Summary:  "oh no bad",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := diagnosticsToProto(test.Input)
			want := test.Want

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestComponentInstancePolicyEvaluationProto(t *testing.T) {
	policyObj := &policy.Policy{
		Result:           policy.AllowResult,
		Address:          "policy_name",
		PolicySetName:    "some_policy_set",
		Filename:         "policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
	}

	snippetContext := `module_policy "example"`

	exprValueBytes, err := msgpack.Marshal(cty.StringVal("bar"), cty.String)
	if err != nil {
		t.Fatalf("failed to marshal expression value: %s", err)
	}

	resourceAddr, diags := addrs.ParseAbsResourceInstanceStr("test_instance.example")
	if diags.HasErrors() {
		t.Fatalf("failed to parse resource address: %s", diags.Err())
	}

	testCases := map[string]struct {
		componentAddr string
		policyResults func() map[string]policy.EvaluationResponse
		want          *stacks.ComponentInstancePolicyEvaluation
	}{
		"no results": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{},
		},
		"policy result with diagnostic": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{
					addrs.RootModule.Child("example").String(): {
						Overall: policy.DenyResult,
						Diagnostics: policy.Diagnostics{
							policy.NewErrorDiagnostic(
								"module policy denied",
								"module policy blocked usage",
								policy.DenyResult,
							),
						},
						Policies: []*policy.Policy{{
							Address:          "module_policy.example",
							Filename:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
							Result:           policy.DenyResult,
						}},
					},
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Results: []*stacks.PolicyResult{
					{
						TargetAddress: "module.example",
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "module_policy.example",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						Result: stacks.EvaluateResult_DENY_EVALUATE_RESULT,
					},
				},
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: "module.example",
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "module policy denied",
							Detail:   "module policy blocked usage",
						},
						PolicyMetadata: &stacks.PolicyMetaData{},
					},
				},
			},
		},
		"policy info with snippet and range": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{
					addrs.RootModule.Child("example").String(): {
						Overall:  policy.AllowResult,
						Policies: []*policy.Policy{policyObj},
						Enforcements: []policy.EnforcementResult{
							{
								Result:     policy.AllowResult,
								Message:    "module policy allowed usage",
								BlockIndex: 1,
								Policy:     policyObj,
								Range: &hcl.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start:    hcl.Pos{Line: 3, Column: 5, Byte: 10},
									End:      hcl.Pos{Line: 4, Column: 10, Byte: 30},
								},
								Snippet: &proto.Snippet{
									Code:                 `key = attr.value == "foo"`,
									Context:              &snippetContext,
									StartLine:            3,
									HighlightStartOffset: 0,
									HighlightEndOffset:   5,
								},
							},
							{
								// Enforcements without a message are skipped.
								Result: policy.AllowResult,
								Policy: policyObj,
							},
						},
					},
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Results: []*stacks.PolicyResult{
					{
						TargetAddress: "module.example",
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						Result: stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
					},
				},
				Infos: []*stacks.PolicyInfo{
					{
						TargetAddress: "module.example",
						Result:        stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
						Message:       "module policy allowed usage",
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
							EnforceIndex:     1,
						},
						PolicySnippet: &stacks.PolicySnippet{
							Code:                 `key = attr.value == "foo"`,
							Context:              snippetContext,
							StartLine:            3,
							HighlightStartOffset: 0,
							HighlightEndOffset:   5,
						},
						PolicyRange: &terraform1.SourceRange{
							SourceAddr: "policy_file.tfpolicy.hcl",
							Start:      &terraform1.SourcePos{Byte: 10, Line: 3, Column: 5},
							End:        &terraform1.SourcePos{Byte: 30, Line: 4, Column: 10},
						},
					},
				},
			},
		},
		"policy diagnostic with extra data": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{
					addrs.RootModule.Child("example").String(): {
						Overall: policy.DenyResult,
						Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "policy error",
								Detail:   "the module is not allowed",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start:    &proto.Position{Byte: 10, Line: 2, Column: 3},
									End:      &proto.Position{Byte: 20, Line: 2, Column: 13},
								},
								Snippet: &proto.Snippet{
									Context:              &snippetContext,
									Code:                 `key = attr.value == "foo"`,
									StartLine:            2,
									HighlightStartOffset: 6,
									HighlightEndOffset:   11,
								},
								ExpressionValues: []*proto.ExpressionValue{
									{
										Traversal: &proto.AttributePath{
											Steps: []*proto.AttributePath_Step{{
												Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "value"},
											}},
										},
										Value: exprValueBytes,
									},
								},
							},
						}, policyObj),
					},
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: "module.example",
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "policy error",
							Detail:   "the module is not allowed",
						},
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						PolicySnippet: &stacks.PolicySnippet{
							Context:              snippetContext,
							Code:                 `key = attr.value == "foo"`,
							StartLine:            2,
							HighlightStartOffset: 6,
							HighlightEndOffset:   11,
						},
						PolicyRange: &terraform1.SourceRange{
							SourceAddr: "policy_file.tfpolicy.hcl",
							Start:      &terraform1.SourcePos{Byte: 10, Line: 2, Column: 3},
							End:        &terraform1.SourcePos{Byte: 20, Line: 2, Column: 13},
						},
						ExpressionValues: []*stacks.ExpressionValue{
							{
								Traversal: stacks.NewAttributePath(cty.GetAttrPath("value")),
								Value:     exprValueBytes,
							},
						},
					},
				},
			},
		},
		"policy diagnostic with enforce block index": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{
					addrs.RootModule.Child("example").String(): policy.EvaluationFromProtoResponse(
						proto.EvaluateResult_DENY_EVALUATE_RESULT,
						[]*proto.PolicyEvaluationDetail{{
							Address:              "module_policy.example",
							Result:               proto.EvaluateResult_DENY_EVALUATE_RESULT,
							File:                 "policy_file.tfpolicy.hcl",
							PolicySetName:        "some_policy_set",
							PolicySetEnforcement: "mandatory",
							DefRange: &proto.Range{
								Filename: "policy_file.tfpolicy.hcl",
								Start:    &proto.Position{Byte: 0, Line: 1, Column: 1},
								End:      &proto.Position{Byte: 40, Line: 5, Column: 2},
							},
							EnforceResults: []*proto.EnforceBlockResult{{
								Result:     proto.EvaluateResult_DENY_EVALUATE_RESULT,
								BlockIndex: 2,
								Diagnostics: []*proto.Diagnostic{{
									Severity: proto.Severity_ERROR,
									Summary:  "enforce block denied module",
									Detail:   "the enforce block condition failed",
									Result: &proto.DiagnosticResult{
										Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
									},
								}},
							}},
						}},
					),
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Results: []*stacks.PolicyResult{
					{
						TargetAddress: "module.example",
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "module_policy.example",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						Result: stacks.EvaluateResult_DENY_EVALUATE_RESULT,
					},
				},
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: "module.example",
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "enforce block denied module",
							Detail:   "the enforce block condition failed",
						},
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "module_policy.example",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
							EnforceIndex:     2,
						},
					},
				},
			},
		},
		"policy diagnostic with warning severity and local range": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				diags := policy.DiagsFromProto([]*proto.Diagnostic{{
					Severity: proto.Severity_WARNING,
					Summary:  "resource policy warning",
					Detail:   "the resource is discouraged",
					Result: &proto.DiagnosticResult{
						Result: proto.EvaluateResult_ALLOW_EVALUATE_RESULT,
					},
				}}, policyObj)

				// WithLocalRange attaches the Terraform source location, which
				// becomes the diagnostic's Subject and Context.
				diags[0] = diags[0].WithLocalRange(&hcl.Range{
					Filename: "main.tf",
					Start:    hcl.Pos{Line: 5, Column: 1, Byte: 50},
					End:      hcl.Pos{Line: 5, Column: 20, Byte: 70},
				})

				return map[string]policy.EvaluationResponse{
					resourceAddr.String(): {
						Overall:     policy.AllowResult,
						Diagnostics: diags,
					},
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: "test_instance.example",
						Result:        stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_WARNING,
							Summary:  "resource policy warning",
							Detail:   "the resource is discouraged",
							Subject: &terraform1.SourceRange{
								SourceAddr: "main.tf",
								Start:      &terraform1.SourcePos{Line: 5, Column: 1, Byte: 50},
								End:        &terraform1.SourcePos{Line: 5, Column: 20, Byte: 70},
							},
							Context: &terraform1.SourceRange{
								SourceAddr: "main.tf",
								Start:      &terraform1.SourcePos{Line: 5, Column: 1, Byte: 50},
								End:        &terraform1.SourcePos{Line: 5, Column: 20, Byte: 70},
							},
						},
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
					},
				},
			},
		},
		"policy diagnostic with duplicate and invalid expression values": {
			componentAddr: "component.test",
			policyResults: func() map[string]policy.EvaluationResponse {
				return map[string]policy.EvaluationResponse{
					resourceAddr.String(): {
						Overall: policy.DenyResult,
						Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{{
							Severity: proto.Severity_ERROR,
							Summary:  "policy error",
							Detail:   "the resource is not allowed",
							Result: &proto.DiagnosticResult{
								Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
							},
							ExpressionValues: []*proto.ExpressionValue{
								{
									// first occurrence of "value" is kept
									Traversal: &proto.AttributePath{Steps: []*proto.AttributePath_Step{{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "value"}}}},
									Value:     exprValueBytes,
								},
								{
									// duplicate "value" path is skipped
									Traversal: &proto.AttributePath{Steps: []*proto.AttributePath_Step{{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "value"}}}},
									Value:     exprValueBytes,
								},
							},
						}}, policyObj),
					},
				}
			},
			want: &stacks.ComponentInstancePolicyEvaluation{
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: "test_instance.example",
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "policy error",
							Detail:   "the resource is not allowed",
						},
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						ExpressionValues: []*stacks.ExpressionValue{
							{
								Traversal: stacks.NewAttributePath(cty.GetAttrPath("value")),
								Value:     exprValueBytes,
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			componentAddr := mustAbsComponentInstance(t, tc.componentAddr)

			want := tc.want
			want.Addr = &stacks.ComponentInstanceInStackAddr{
				ComponentAddr:         stackaddrs.ConfigComponentForAbsInstance(componentAddr).String(),
				ComponentInstanceAddr: componentAddr.String(),
			}

			got := componentInstancePolicyEvaluationProto(componentAddr, tc.policyResults())

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestProviderInstancePolicyEvaluationProto(t *testing.T) {
	policyObj := &policy.Policy{
		Result:           policy.AllowResult,
		Address:          "policy_name",
		PolicySetName:    "some_policy_set",
		Filename:         "policy_file.tfpolicy.hcl",
		EnforcementLevel: "mandatory",
	}

	snippetContext := `provider_policy "example"`

	exprValueBytes, err := msgpack.Marshal(cty.StringVal("bar"), cty.String)
	if err != nil {
		t.Fatalf("failed to marshal expression value: %s", err)
	}

	providerInstanceAddr := stackaddrs.AbsProviderConfigInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ProviderConfigInstance{
			ProviderConfig: stackaddrs.ProviderConfig{
				Provider: addrs.NewDefaultProvider("testing"),
				Name:     "default",
			},
			Key: addrs.NoKey,
		},
	}

	testCases := map[string]struct {
		policyResults func() *stackhooks.ProviderInstancePolicyResults
		want          *stacks.ProviderInstancePolicyEvaluation
	}{
		"no results": {
			policyResults: func() *stackhooks.ProviderInstancePolicyResults {
				return &stackhooks.ProviderInstancePolicyResults{
					Addr:         providerInstanceAddr,
					ProviderAddr: `provider["registry.terraform.io/hashicorp/testing"]`,
				}
			},
			want: &stacks.ProviderInstancePolicyEvaluation{},
		},
		"policy result with diagnostic": {
			policyResults: func() *stackhooks.ProviderInstancePolicyResults {
				return &stackhooks.ProviderInstancePolicyResults{
					Addr:         providerInstanceAddr,
					ProviderAddr: `provider["registry.terraform.io/hashicorp/testing"]`,
					Result: policy.EvaluationResponse{
						Overall: policy.DenyResult,
						Diagnostics: policy.Diagnostics{
							policy.NewErrorDiagnostic(
								"provider policy denied",
								"provider policy blocked usage",
								policy.DenyResult,
							),
						},
						Policies: []*policy.Policy{
							{
								Address:          "provider_policy.example",
								Filename:         "policy_file.tfpolicy.hcl",
								EnforcementLevel: "mandatory",
								Result:           policy.DenyResult,
							},
						},
					},
				}
			},
			want: &stacks.ProviderInstancePolicyEvaluation{
				Results: []*stacks.PolicyResult{
					{
						TargetAddress: `provider["registry.terraform.io/hashicorp/testing"]`,
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "provider_policy.example",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						Result: stacks.EvaluateResult_DENY_EVALUATE_RESULT,
					},
				},
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: `provider["registry.terraform.io/hashicorp/testing"]`,
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "provider policy denied",
							Detail:   "provider policy blocked usage",
						},
						PolicyMetadata: &stacks.PolicyMetaData{},
					},
				},
			},
		},
		"policy info with snippet and range": {
			policyResults: func() *stackhooks.ProviderInstancePolicyResults {
				return &stackhooks.ProviderInstancePolicyResults{
					Addr:         providerInstanceAddr,
					ProviderAddr: `provider["registry.terraform.io/hashicorp/testing"]`,
					Result: policy.EvaluationResponse{
						Overall:  policy.AllowResult,
						Policies: []*policy.Policy{policyObj},
						Enforcements: []policy.EnforcementResult{
							{
								Result:     policy.AllowResult,
								Message:    "provider policy allowed usage",
								BlockIndex: 1,
								Policy:     policyObj,
								Range: &hcl.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start:    hcl.Pos{Line: 3, Column: 5, Byte: 10},
									End:      hcl.Pos{Line: 4, Column: 10, Byte: 30},
								},
								Snippet: &proto.Snippet{
									Code:                 `key = attr.value == "foo"`,
									Context:              &snippetContext,
									StartLine:            3,
									HighlightStartOffset: 0,
									HighlightEndOffset:   5,
								},
							},
							{
								// Enforcements without a message are skipped.
								Result: policy.AllowResult,
								Policy: policyObj,
							},
						},
					},
				}
			},
			want: &stacks.ProviderInstancePolicyEvaluation{
				Results: []*stacks.PolicyResult{
					{
						TargetAddress: `provider["registry.terraform.io/hashicorp/testing"]`,
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						Result: stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
					},
				},
				Infos: []*stacks.PolicyInfo{
					{
						TargetAddress: `provider["registry.terraform.io/hashicorp/testing"]`,
						Result:        stacks.EvaluateResult_ALLOW_EVALUATE_RESULT,
						Message:       "provider policy allowed usage",
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
							EnforceIndex:     1,
						},
						PolicySnippet: &stacks.PolicySnippet{
							Code:                 `key = attr.value == "foo"`,
							Context:              snippetContext,
							StartLine:            3,
							HighlightStartOffset: 0,
							HighlightEndOffset:   5,
						},
						PolicyRange: &terraform1.SourceRange{
							SourceAddr: "policy_file.tfpolicy.hcl",
							Start:      &terraform1.SourcePos{Byte: 10, Line: 3, Column: 5},
							End:        &terraform1.SourcePos{Byte: 30, Line: 4, Column: 10},
						},
					},
				},
			},
		},
		"policy diagnostic with extra data": {
			policyResults: func() *stackhooks.ProviderInstancePolicyResults {
				return &stackhooks.ProviderInstancePolicyResults{
					Addr:         providerInstanceAddr,
					ProviderAddr: `provider["registry.terraform.io/hashicorp/testing"]`,
					Result: policy.EvaluationResponse{
						Overall: policy.DenyResult,
						Diagnostics: policy.DiagsFromProto([]*proto.Diagnostic{
							{
								Severity: proto.Severity_ERROR,
								Summary:  "policy error",
								Detail:   "the provider is not allowed",
								Result: &proto.DiagnosticResult{
									Result: proto.EvaluateResult_DENY_EVALUATE_RESULT,
								},
								Subject: &proto.Range{
									Filename: "policy_file.tfpolicy.hcl",
									Start:    &proto.Position{Byte: 10, Line: 2, Column: 3},
									End:      &proto.Position{Byte: 20, Line: 2, Column: 13},
								},
								Snippet: &proto.Snippet{
									Context:              &snippetContext,
									Code:                 `key = attr.value == "foo"`,
									StartLine:            2,
									HighlightStartOffset: 6,
									HighlightEndOffset:   11,
								},
								ExpressionValues: []*proto.ExpressionValue{
									{
										Traversal: &proto.AttributePath{
											Steps: []*proto.AttributePath_Step{{
												Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "value"},
											}},
										},
										Value: exprValueBytes,
									},
								},
							},
						}, policyObj),
					},
				}
			},
			want: &stacks.ProviderInstancePolicyEvaluation{
				Diagnostics: []*stacks.PolicyDiagnostic{
					{
						TargetAddress: `provider["registry.terraform.io/hashicorp/testing"]`,
						Result:        stacks.EvaluateResult_DENY_EVALUATE_RESULT,
						Diagnostic: &terraform1.Diagnostic{
							Severity: terraform1.Diagnostic_ERROR,
							Summary:  "policy error",
							Detail:   "the provider is not allowed",
						},
						PolicyMetadata: &stacks.PolicyMetaData{
							PolicyName:       "policy_name",
							PolicySetName:    "some_policy_set",
							FileName:         "policy_file.tfpolicy.hcl",
							EnforcementLevel: "mandatory",
						},
						PolicySnippet: &stacks.PolicySnippet{
							Context:              snippetContext,
							Code:                 `key = attr.value == "foo"`,
							StartLine:            2,
							HighlightStartOffset: 6,
							HighlightEndOffset:   11,
						},
						PolicyRange: &terraform1.SourceRange{
							SourceAddr: "policy_file.tfpolicy.hcl",
							Start:      &terraform1.SourcePos{Byte: 10, Line: 2, Column: 3},
							End:        &terraform1.SourcePos{Byte: 20, Line: 2, Column: 13},
						},
						ExpressionValues: []*stacks.ExpressionValue{
							{
								Traversal: stacks.NewAttributePath(cty.GetAttrPath("value")),
								Value:     exprValueBytes,
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			want := tc.want
			want.Addr = &stacks.ProviderInstanceInStackAddr{
				ProviderAddr:         stackaddrs.ConfigProviderConfigForAbsInstance(providerInstanceAddr).String(),
				ProviderInstanceAddr: providerInstanceAddr.String(),
			}

			got := providerInstancePolicyEvaluationProto(tc.policyResults())

			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
