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
		ret.Subject = &SourceRange{
			Filename: d.diag.Subject.Filename,
			Start: SourcePos{
				Line:   d.diag.Subject.Start.Line,
				Column: d.diag.Subject.Start.Column,
				Byte:   d.diag.Subject.Start.Byte,
			},
			End: SourcePos{
				Line:   d.diag.Subject.End.Line,
				Column: d.diag.Subject.End.Column,
				Byte:   d.diag.Subject.End.Byte,
			},
		}
	}
	if d.diag.Context != nil {
		ret.Context = &SourceRange{
			Filename: d.diag.Context.Filename,
			Start: SourcePos{
				Line:   d.diag.Context.Start.Line,
				Column: d.diag.Context.Start.Column,
				Byte:   d.diag.Context.Start.Byte,
			},
			End: SourcePos{
				Line:   d.diag.Context.End.Line,
				Column: d.diag.Context.End.Column,
				Byte:   d.diag.Context.End.Byte,
			},
		}
	}
	return ret
}
