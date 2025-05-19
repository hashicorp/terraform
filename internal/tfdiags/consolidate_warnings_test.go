// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestConsolidateWarnings(t *testing.T) {
	var diags Diagnostics

	for i := 0; i < 4; i++ {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Warning 1",
			Detail:   fmt.Sprintf("This one has a subject %d", i),
			Subject: &hcl.Range{
				Filename: "foo.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
			},
		})
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Error 1",
			Detail:   fmt.Sprintf("This one has a subject %d", i),
			Subject: &hcl.Range{
				Filename: "foo.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
			},
		})
		diags = diags.Append(Sourceless(
			Warning,
			"Warning 2",
			fmt.Sprintf("This one is sourceless %d", i),
		))
		diags = diags.Append(SimpleWarning("Warning 3"))
	}

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Warning 4",
		Detail:   "Only one of this one",
		Subject: &hcl.Range{
			Filename: "foo.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
		},
	})

	// Finally, we'll just add a set of diags that should not be consolidated.

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "do not consolidate",
		Detail:   "warning 1, I should not have been consolidated",
		Subject: &hcl.Range{
			Filename: "bar.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
		},
		Extra: doNotConsolidate(true),
	})

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "do not consolidate",
		Detail:   "warning 2, I should not have been consolidated",
		Subject: &hcl.Range{
			Filename: "bar.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
		},
		Extra: doNotConsolidate(true),
	})

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "do not consolidate",
		Detail:   "warning 3, I should not have been consolidated",
		Subject: &hcl.Range{
			Filename: "bar.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 0},
		},
		Extra: doNotConsolidate(true),
	})

	got := diags.ConsolidateWarnings(2)
	want := Diagnostics{
		// First set
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 1",
			Detail_:   "This one has a subject 0",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "Error 1",
			Detail_:   "This one has a subject 0",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 2",
			Detail_:   "This one is sourceless 0",
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 3",
		},

		// Second set (consolidation begins; note additional paragraph in Warning 1 detail)
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 1",
			Detail_:   "This one has a subject 1\n\n(and 2 more similar warnings elsewhere)",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "Error 1",
			Detail_:   "This one has a subject 1",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 2",
			Detail_:   "This one is sourceless 1",
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 3",
		},

		// Third set (no more Warning 1, because it's consolidated)
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "Error 1",
			Detail_:   "This one has a subject 2",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 2",
			Detail_:   "This one is sourceless 2",
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 3",
		},

		// Fourth set (still no warning 1)
		&rpcFriendlyDiag{
			Severity_: Error,
			Summary_:  "Error 1",
			Detail_:   "This one has a subject 3",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 2",
			Detail_:   "This one is sourceless 3",
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 3",
		},

		// Special straggler warning gets to show up unconsolidated, because
		// there is only one of it.
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "Warning 4",
			Detail_:   "Only one of this one",
			Subject_: &SourceRange{
				Filename: "foo.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},

		// The final set of warnings should not have been consolidated because
		// of our filter function.
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "do not consolidate",
			Detail_:   "warning 1, I should not have been consolidated",
			Subject_: &SourceRange{
				Filename: "bar.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "do not consolidate",
			Detail_:   "warning 2, I should not have been consolidated",
			Subject_: &SourceRange{
				Filename: "bar.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
		&rpcFriendlyDiag{
			Severity_: Warning,
			Summary_:  "do not consolidate",
			Detail_:   "warning 3, I should not have been consolidated",
			Subject_: &SourceRange{
				Filename: "bar.tf",
				Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
				End:      SourcePos{Line: 1, Column: 1, Byte: 0},
			},
		},
	}

	AssertDiagnosticsMatch(t, want, got)
}

type doNotConsolidate bool

var _ DiagnosticExtraDoNotConsolidate = doNotConsolidate(true)

func (d doNotConsolidate) DoNotConsolidateDiagnostic() bool {
	return bool(d)
}
