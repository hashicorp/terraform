// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"github.com/hashicorp/hcl/v2"
)

func invalidNameDiagnostic(summary string, rng hcl.Range) *hcl.Diagnostic {
	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   "Names must be valid identifiers: beginning with a letter or underscore, followed by zero or more letters, digits, or underscores.",
		Subject:  &rng,
	}
}
