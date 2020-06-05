package tfdiags

import (
	"fmt"

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
func (d Diagnostics) ToHCL() hcl.Diagnostics {
	if len(d) == 0 {
		return nil
	}
	ret := make(hcl.Diagnostics, len(d))
	for i, diag := range d {
		severity := diag.Severity()
		desc := diag.Description()
		source := diag.Source()
		fromExpr := diag.FromExpr()

		hclDiag := &hcl.Diagnostic{
			Summary: desc.Summary,
			Detail:  desc.Detail,
		}

		switch severity {
		case Warning:
			hclDiag.Severity = hcl.DiagWarning
		case Error:
			hclDiag.Severity = hcl.DiagError
		default:
			// The above should always be exhaustive for all of the valid
			// Severity values in this package.
			panic(fmt.Sprintf("unknown diagnostic severity %s", severity))
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
