package tfdiags

import (
	"github.com/hashicorp/hcl2/hcl"
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
