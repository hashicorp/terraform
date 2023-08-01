package s3

import (
	"strings"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func diagnosticString(diag tfdiags.Diagnostic) string {
	var buffer strings.Builder
	buffer.WriteString(diag.Severity().String() + ": ")
	buffer.WriteString(diag.Description().Summary)
	if diag.Description().Detail != "" {
		buffer.WriteString("\n\n")
		buffer.WriteString(diag.Description().Detail)
	}
	return buffer.String()
}

func diagnosticsString(diags tfdiags.Diagnostics) string {
	l := len(diags)
	if l == 0 {
		return ""
	}

	var buffer strings.Builder
	for i, d := range diags {
		buffer.WriteString(diagnosticString(d))
		if i < l-1 {
			buffer.WriteString(",\n")
		}
	}
	return buffer.String()
}
