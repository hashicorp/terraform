// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package proto

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestDiagnosticToHCL(t *testing.T) {
	protoDiag := &Diagnostic{
		Severity: Severity_WARNING,
		Summary:  "policy warning",
		Detail:   "diagnostic detail",
		Subject: &Range{
			Filename: "policy.tfpolicy.hcl",
			Start:    &Position{Byte: 10, Line: 2, Column: 3},
			End:      &Position{Byte: 20, Line: 2, Column: 13},
		},
		Context: &Range{
			Filename: "policy.tfpolicy.hcl",
			Start:    &Position{Byte: 1, Line: 1, Column: 1},
			End:      &Position{Byte: 30, Line: 3, Column: 5},
		},
		Result: &DiagnosticResult{
			Result: EvaluateResult_DENY_EVALUATE_RESULT,
		},
		Attribute: &AttributePath{
			Steps: []*AttributePath_Step{
				{Selector: &AttributePath_Step_AttributeName{AttributeName: "tags"}},
				{Selector: &AttributePath_Step_ElementKeyString{ElementKeyString: "name"}},
			},
		},
		Snippet: &Snippet{
			Context: func() *string {
				ret := "Some context around the code"
				return &ret
			}(),
			Code: "some policy code snippet"},
		ExpressionValues: []*ExpressionValue{{
			Traversal: &AttributePath{
				Steps: []*AttributePath_Step{{Selector: &AttributePath_Step_AttributeName{AttributeName: "example"}}},
			},
			Value: []byte("value-bytes"),
		}},
		FunctionCall: "getresources",
		PolicySet: &PolicySet{
			Name: "policy-set",
			Path: "/tmp/policies",
		},
	}

	diag := protoDiag.ToHCL()
	tfDiag := tfdiags.FromHCL(diag)
	// assert the basic diagnostic fields
	tfdiags.AssertDiagnosticMatch(t, tfDiag, tfdiags.FromHCL(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "policy warning",
		Detail:   "diagnostic detail",
	}))

	// assert the extra information nested in the diagnostic
	t.Run("attribute extra", func(t *testing.T) {
		attributeExtra := tfdiags.ExtraInfo[*AttributeExtra](tfDiag)
		if attributeExtra == nil {
			t.Fatalf("expected attribute extra, got nil")
		}
		expectedPath := cty.Path{
			cty.GetAttrStep{Name: "tags"},
			cty.IndexStep{Key: cty.StringVal("name")},
		}
		if !expectedPath.Equals(attributeExtra.Attribute) {
			t.Fatalf("unexpected attribute path: got %#v, want %#v", attributeExtra.Attribute, expectedPath)
		}
	})

	t.Run("range extra", func(t *testing.T) {
		rangeExtra := tfdiags.ExtraInfo[*RangeExtra](tfDiag)
		if rangeExtra == nil {
			t.Fatalf("expected range extra, got nil")
		}
		if rangeExtra.Subject == nil || rangeExtra.Subject.Filename != "policy.tfpolicy.hcl" {
			t.Fatalf("unexpected range subject: %#v", rangeExtra.Subject)
		}
		if rangeExtra.Context == nil || rangeExtra.Context.Start.Line != 1 {
			t.Fatalf("unexpected range context: %#v", rangeExtra.Context)
		}
	})

	t.Run("function call extra", func(t *testing.T) {
		functionCallExtra := tfdiags.ExtraInfo[*FunctionCallExtra](tfDiag)
		if functionCallExtra == nil {
			t.Fatalf("expected function call extra, got nil")
		}
		if functionCallExtra.FunctionCall != "getresources" {
			t.Fatalf("unexpected function call extra: got %q, want %q", functionCallExtra.FunctionCall, "getresources")
		}
	})

	t.Run("expression values extra", func(t *testing.T) {
		expressionValuesExtra := tfdiags.ExtraInfo[*ExpressionValuesExtra](tfDiag)
		if expressionValuesExtra == nil {
			t.Fatalf("expected expression values extra, got nil")
		}
		if len(expressionValuesExtra.ExpressionValues) != 1 {
			t.Fatalf("unexpected expression values count: got %d, want 1", len(expressionValuesExtra.ExpressionValues))
		}
		if expressionValuesExtra.ExpressionValues[0].Traversal == nil || len(expressionValuesExtra.ExpressionValues[0].Traversal.Steps) != 1 {
			t.Fatalf("unexpected expression value path: %#v", expressionValuesExtra.ExpressionValues[0].Traversal)
		}
		if expressionValuesExtra.ExpressionValues[0].Value == nil {
			t.Fatalf("expected expression value bytes to be present")
		}
	})

	t.Run("snippet extra", func(t *testing.T) {
		snippetExtra := tfdiags.ExtraInfo[*SnippetExtra](tfDiag)
		if snippetExtra == nil {
			t.Fatalf("expected snippet extra, got nil")
		}
		if snippetExtra.Snippet == nil || snippetExtra.Snippet.Code != "some policy code snippet" {
			t.Fatalf("unexpected snippet extra: %#v", snippetExtra.Snippet)
		}
	})

	t.Run("evaluate result extra", func(t *testing.T) {
		resultExtra := tfdiags.ExtraInfo[*EvaluateResultExtra](tfDiag)
		if resultExtra == nil {
			t.Fatalf("expected evaluate result extra, got nil")
		}
		if resultExtra.EvaluateResult != EvaluateResult_DENY_EVALUATE_RESULT {
			t.Fatalf("unexpected evaluate result: got %s, want %s", resultExtra.EvaluateResult, EvaluateResult_DENY_EVALUATE_RESULT)
		}
	})

	t.Run("policy extra", func(t *testing.T) {
		policyExtra := tfdiags.ExtraInfo[*PolicyExtra](tfDiag)
		if policyExtra == nil {
			t.Fatalf("expected policy extra, got nil")
		}
		if policyExtra.PolicySet.Name != "policy-set" {
			t.Fatalf("unexpected policy set name: got %q, want %q", policyExtra.PolicySet.Name, "policy-set")
		}
		if policyExtra.PolicySet.Path != "/tmp/policies" {
			t.Fatalf("unexpected policy set path: got %q, want %q", policyExtra.PolicySet.Path, "/tmp/policies")
		}
	})
}
