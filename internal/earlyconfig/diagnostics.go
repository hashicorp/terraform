package earlyconfig

import (
	"fmt"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/tfdiags"
)

func wrapDiagnostics(diags tfconfig.Diagnostics) tfdiags.Diagnostics {
	ret := make(tfdiags.Diagnostics, len(diags))
	for i, diag := range diags {
		ret[i] = wrapDiagnostic(diag)
	}
	return ret
}

func wrapDiagnostic(diag tfconfig.Diagnostic) tfdiags.Diagnostic {
	return wrappedDiagnostic{
		d: diag,
	}
}

type wrappedDiagnostic struct {
	d tfconfig.Diagnostic
}

func (d wrappedDiagnostic) Severity() tfdiags.Severity {
	switch d.d.Severity {
	case tfconfig.DiagError:
		return tfdiags.Error
	case tfconfig.DiagWarning:
		return tfdiags.Warning
	default:
		// Should never happen since there are no other severities
		return 0
	}
}

func (d wrappedDiagnostic) Description() tfdiags.Description {
	// Since the inspect library doesn't produce precise source locations,
	// we include the position information as part of the error message text.
	// See the comment inside method "Source" for more information.
	switch {
	case d.d.Pos == nil:
		return tfdiags.Description{
			Summary: d.d.Summary,
			Detail:  d.d.Detail,
		}
	case d.d.Detail != "":
		return tfdiags.Description{
			Summary: d.d.Summary,
			Detail:  fmt.Sprintf("On %s line %d: %s", d.d.Pos.Filename, d.d.Pos.Line, d.d.Detail),
		}
	default:
		return tfdiags.Description{
			Summary: fmt.Sprintf("%s (on %s line %d)", d.d.Summary, d.d.Pos.Filename, d.d.Pos.Line),
		}
	}
}

func (d wrappedDiagnostic) Source() tfdiags.Source {
	// Since the inspect library is constrained by the lowest common denominator
	// between legacy HCL and modern HCL, it only returns ranges at whole-line
	// granularity, and that isn't sufficient to populate a tfdiags.Source
	// and so we'll just omit ranges altogether and include the line number in
	// the Description text.
	//
	// Callers that want to return nicer errors should consider reacting to
	// earlyconfig errors by attempting a follow-up parse with the normal
	// config loader, which can produce more precise source location
	// information.
	return tfdiags.Source{}
}

func (d wrappedDiagnostic) FromExpr() *tfdiags.FromExpr {
	return nil
}
