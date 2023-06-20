package rpcapi

import (
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func diagnosticsToProto(diags tfdiags.Diagnostics) []*terraform1.Diagnostic {
	if len(diags) == 0 {
		return nil
	}

	ret := make([]*terraform1.Diagnostic, len(diags))
	for i, diag := range diags {
		protoDiag := &terraform1.Diagnostic{}
		ret[i] = protoDiag

		switch diag.Severity() {
		case tfdiags.Error:
			protoDiag.Severity = terraform1.Diagnostic_ERROR
		case tfdiags.Warning:
			protoDiag.Severity = terraform1.Diagnostic_WARNING
		default:
			protoDiag.Severity = terraform1.Diagnostic_INVALID
		}

		desc := diag.Description()
		protoDiag.Summary = desc.Summary
		protoDiag.Detail = desc.Detail

		srcRngs := diag.Source()
		if srcRngs.Subject != nil {
			protoDiag.Subject = sourceRangeToProto(*srcRngs.Subject)
		}
		if srcRngs.Context != nil {
			protoDiag.Context = sourceRangeToProto(*srcRngs.Context)
		}
	}
	return ret
}

func sourceRangeToProto(rng tfdiags.SourceRange) *terraform1.SourceRange {
	return &terraform1.SourceRange{
		// RPC API operations use source address syntax for "filename" by
		// convention, because the physical filesystem layout is an
		// implementation detail.
		SourceAddr: rng.Filename,

		Start: sourcePosToProto(rng.Start),
		End:   sourcePosToProto(rng.End),
	}
}

func sourcePosToProto(pos tfdiags.SourcePos) *terraform1.SourcePos {
	return &terraform1.SourcePos{
		Byte:   int64(pos.Byte),
		Line:   int64(pos.Line),
		Column: int64(pos.Column),
	}
}
