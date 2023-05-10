// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfdiags

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

var _ Diagnostic = CheckBlockDiagnostic{}

// CheckBlockDiagnostic is a diagnostic produced by a Terraform config Check block.
//
// It only ever returns warnings, and will not be consolidated as part of the
// Diagnostics.ConsolidateWarnings function.
type CheckBlockDiagnostic struct {
	diag Diagnostic
}

// AsCheckBlockDiagnostics will wrap every diagnostic in diags in a
// CheckBlockDiagnostic.
func AsCheckBlockDiagnostics(diags Diagnostics) Diagnostics {
	if len(diags) == 0 {
		return nil
	}

	ret := make(Diagnostics, len(diags))
	for i, diag := range diags {
		ret[i] = CheckBlockDiagnostic{diag}
	}
	return ret
}

// AsCheckBlockDiagnostic will wrap a Diagnostic or a hcl.Diagnostic in a
// CheckBlockDiagnostic.
func AsCheckBlockDiagnostic(diag interface{}) Diagnostic {
	switch d := diag.(type) {
	case Diagnostic:
		return CheckBlockDiagnostic{d}
	case *hcl.Diagnostic:
		return CheckBlockDiagnostic{hclDiagnostic{d}}
	default:
		panic(fmt.Errorf("can't construct diagnostic from %T", diag))
	}
}

// IsFromCheckBlock returns true if the specified Diagnostic is a
// CheckBlockDiagnostic.
func IsFromCheckBlock(diag Diagnostic) bool {
	_, ok := diag.(CheckBlockDiagnostic)
	return ok
}

func (c CheckBlockDiagnostic) Severity() Severity {
	// Regardless of the severity of the underlying diagnostic, check blocks
	// only ever report Warning severity.
	return Warning
}

func (c CheckBlockDiagnostic) Description() Description {
	return c.diag.Description()
}

func (c CheckBlockDiagnostic) Source() Source {
	return c.diag.Source()
}

func (c CheckBlockDiagnostic) FromExpr() *FromExpr {
	return c.diag.FromExpr()
}

func (c CheckBlockDiagnostic) ExtraInfo() interface{} {
	return c.diag.ExtraInfo()
}
