// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"github.com/hashicorp/hcl/v2"
)

// hclDiagnostic is a Diagnostic implementation that wraps a HCL Diagnostic
type hclDiagnostic struct {
	diag *hcl.Diagnostic
}

var _ Diagnostic = hclDiagnostic{}

func (d hclDiagnostic) Severity() Severity {
	switch d.diag.Severity {
	case hcl.DiagWarning:
		return Warning
	default:
		return Error
	}
}

func (d hclDiagnostic) Description() Description {
	return Description{
		Summary: d.diag.Summary,
		Detail:  d.diag.Detail,
	}
}

func (d hclDiagnostic) Source() Source {
	var ret Source
	if d.diag.Subject != nil {
		rng := SourceRangeFromHCL(*d.diag.Subject)
		ret.Subject = &rng
	}
	if d.diag.Context != nil {
		rng := SourceRangeFromHCL(*d.diag.Context)
		ret.Context = &rng
	}
	return ret
}

func (d hclDiagnostic) FromExpr() *FromExpr {
	if d.diag.Expression == nil || d.diag.EvalContext == nil {
		return nil
	}
	return &FromExpr{
		Expression:  d.diag.Expression,
		EvalContext: d.diag.EvalContext,
	}
}

func (d hclDiagnostic) ExtraInfo() interface{} {
	return d.diag.Extra
}

func (d hclDiagnostic) Equals(otherDiag ComparableDiagnostic) bool {
	od, ok := otherDiag.(hclDiagnostic)
	if !ok {
		return false
	}
	if d.diag.Severity != od.diag.Severity {
		return false
	}
	if d.diag.Summary != od.diag.Summary {
		return false
	}
	if d.diag.Detail != od.diag.Detail {
		return false
	}
	if !hclRangeEquals(d.diag.Subject, od.diag.Subject) {
		return false
	}

	return true
}

func hclRangeEquals(l, r *hcl.Range) bool {
	if l == nil || r == nil {
		return l == r
	}
	if l.Filename != r.Filename {
		return false
	}
	if l.Start.Byte != r.Start.Byte {
		return false
	}
	if l.End.Byte != r.End.Byte {
		return false
	}
	return true
}

// SourceRangeFromHCL constructs a SourceRange from the corresponding range
// type within the HCL package.
func SourceRangeFromHCL(hclRange hcl.Range) SourceRange {
	return SourceRange{
		Filename: hclRange.Filename,
		Start: SourcePos{
			Line:   hclRange.Start.Line,
			Column: hclRange.Start.Column,
			Byte:   hclRange.Start.Byte,
		},
		End: SourcePos{
			Line:   hclRange.End.Line,
			Column: hclRange.End.Column,
			Byte:   hclRange.End.Byte,
		},
	}
}

// ToHCL constructs a HCL Range from the receiving SourceRange. This is the
// opposite of SourceRangeFromHCL.
func (r SourceRange) ToHCL() hcl.Range {
	return hcl.Range{
		Filename: r.Filename,
		Start: hcl.Pos{
			Line:   r.Start.Line,
			Column: r.Start.Column,
			Byte:   r.Start.Byte,
		},
		End: hcl.Pos{
			Line:   r.End.Line,
			Column: r.End.Column,
			Byte:   r.End.Byte,
		},
	}
}

// ToHCL constructs a hcl.Diagnostics containing the same diagnostic messages
// as the receiving tfdiags.Diagnostics.
//
// This conversion preserves the data that HCL diagnostics are able to
// preserve but would be lossy in a round trip from tfdiags to HCL and then
// back to tfdiags, because it will lose the specific type information of
// the source diagnostics. In most cases this will not be a significant
// problem, but could produce an awkward result in some special cases such
// as converting the result of ConsolidateWarnings, which will force the
// resulting warning groups to be flattened early.
func (diags Diagnostics) ToHCL() hcl.Diagnostics {
	if len(diags) == 0 {
		return nil
	}
	ret := make(hcl.Diagnostics, len(diags))
	for i, diag := range diags {
		severity := diag.Severity()
		desc := diag.Description()
		source := diag.Source()
		fromExpr := diag.FromExpr()

		hclDiag := &hcl.Diagnostic{
			Summary:  desc.Summary,
			Detail:   desc.Detail,
			Severity: severity.ToHCL(),
		}
		if source.Subject != nil {
			hclDiag.Subject = source.Subject.ToHCL().Ptr()
		}
		if source.Context != nil {
			hclDiag.Context = source.Context.ToHCL().Ptr()
		}
		if fromExpr != nil {
			hclDiag.Expression = fromExpr.Expression
			hclDiag.EvalContext = fromExpr.EvalContext
		}

		ret[i] = hclDiag
	}
	return ret
}
