package rpcapi

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func protoDiagnotics(from tfdiags.Diagnostics) []*tfcore1.Diagnostic {
	if len(from) == 0 {
		return nil
	}
	ret := make([]*tfcore1.Diagnostic, len(from))
	for i, diag := range from {
		protoDiag := &tfcore1.Diagnostic{}

		severity := diag.Severity()
		desc := diag.Description()
		source := diag.Source()

		switch severity {
		case tfdiags.Error:
			protoDiag.Severity = tfcore1.Diagnostic_ERROR
		case tfdiags.Warning:
			protoDiag.Severity = tfcore1.Diagnostic_WARNING
		default:
			panic(fmt.Sprintf("unsupported diagnostic severity %s", severity))
		}

		protoDiag.Summary = desc.Summary
		protoDiag.Detail = desc.Detail
		protoDiag.Address = desc.Address

		if source.Subject != nil {
			protoDiag.Subject = protoSourceRange(*source.Subject)
		}
		if source.Context != nil {
			protoDiag.Context = protoSourceRange(*source.Context)
		}

		ret[i] = protoDiag
	}
	return ret
}

func protoSourceRange(from tfdiags.SourceRange) *tfcore1.SourceRange {
	return &tfcore1.SourceRange{
		Filename: from.Filename,
		Start:    protoSourcePos(from.Start),
		End:      protoSourcePos(from.End),
	}
}

func protoSourcePos(from tfdiags.SourcePos) *tfcore1.SourceRange_Pos {
	return &tfcore1.SourceRange_Pos{
		Line:   int64(from.Line),
		Column: int64(from.Column),
		Byte:   int64(from.Byte),
	}
}
