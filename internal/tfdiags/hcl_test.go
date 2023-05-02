// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestDiagnosticsToHCL(t *testing.T) {
	var diags Diagnostics
	diags = diags.Append(Sourceless(
		Error,
		"A sourceless diagnostic",
		"...that has a detail",
	))
	diags = diags.Append(fmt.Errorf("a diagnostic promoted from an error"))
	diags = diags.Append(SimpleWarning("A diagnostic from a simple warning"))
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "A diagnostic from HCL",
		Detail:   "...that has a detail and source information",
		Subject: &hcl.Range{
			Filename: "test.tf",
			Start:    hcl.Pos{Line: 1, Column: 2, Byte: 1},
			End:      hcl.Pos{Line: 1, Column: 3, Byte: 2},
		},
		Context: &hcl.Range{
			Filename: "test.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 4, Byte: 3},
		},
		EvalContext: &hcl.EvalContext{},
		Expression:  &fakeHCLExpression{},
	})

	got := diags.ToHCL()
	want := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "A sourceless diagnostic",
			Detail:   "...that has a detail",
		},
		{
			Severity: hcl.DiagError,
			Summary:  "a diagnostic promoted from an error",
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "A diagnostic from a simple warning",
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "A diagnostic from HCL",
			Detail:   "...that has a detail and source information",
			Subject: &hcl.Range{
				Filename: "test.tf",
				Start:    hcl.Pos{Line: 1, Column: 2, Byte: 1},
				End:      hcl.Pos{Line: 1, Column: 3, Byte: 2},
			},
			Context: &hcl.Range{
				Filename: "test.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 4, Byte: 3},
			},
			EvalContext: &hcl.EvalContext{},
			Expression:  &fakeHCLExpression{},
		},
	}

	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(hcl.EvalContext{})); diff != "" {
		t.Errorf("incorrect result\n%s", diff)
	}
}

// We have this here just to give us something easy to compare in the test
// above, because we only care that the expression passes through, not about
// how exactly it is shaped.
type fakeHCLExpression struct {
}

func (e *fakeHCLExpression) Range() hcl.Range {
	return hcl.Range{}
}

func (e *fakeHCLExpression) StartRange() hcl.Range {
	return hcl.Range{}
}

func (e *fakeHCLExpression) Variables() []hcl.Traversal {
	return nil
}

func (e *fakeHCLExpression) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return cty.DynamicVal, nil
}
