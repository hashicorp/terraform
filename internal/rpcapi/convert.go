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

	}
	return ret
}
