// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

// These tests are to ensure that the normalisation of the diagnostics' concrete
// types doesn't impact how the diagnostics are compared.
//
// Full tests of the comparison logic in DiagnosticComparer and
// DiagnosticComparerWithSource are in compare_test.go

func Test_assertDiagnosticMatch_differentConcreteTypes(t *testing.T) {
	baseError := hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "error",
		Detail:   "this is an error",
	}

	cases := map[string]struct {
		diag1      Diagnostic
		diag2      Diagnostic
		expectDiff bool
	}{
		"diagnostics match but are different concrete types": {
			expectDiff: false,
			diag1:      hclDiagnostic{&baseError},
			diag2:      makeRPCFriendlyDiag(hclDiagnostic{&baseError}),
		},
		"diagnostics don't match and are different concrete types": {
			expectDiff: true,
			diag1:      hclDiagnostic{&baseError},
			diag2: func() Diagnostic {
				d := baseError
				d.Severity = hcl.DiagWarning // Altered severity level
				return makeRPCFriendlyDiag(hclDiagnostic{&d})
			}(),
		},
	}
	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// This should show no diff as, internally, the two diags are transformed into the same
			// concrete type
			diff := assertDiagnosticMatch(tc.diag1, tc.diag2)

			if !tc.expectDiff && len(diff) > 0 {
				t.Fatalf("unexpected diff:\n%s", diff)
			}
			if tc.expectDiff && len(diff) == 0 {
				t.Fatalf("expected a diff but got none")
			}
		})
	}
}

func Test_assertDiagnosticsMatch_differentConcreteTypes(t *testing.T) {
	baseError := hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "error",
		Detail:   "this is an error",
	}

	cases := map[string]struct {
		diags1     Diagnostics
		diags2     Diagnostics
		expectDiff bool
	}{
		"diagnostics match but are different concrete types": {
			expectDiff: false,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2:     Diagnostics{makeRPCFriendlyDiag(hclDiagnostic{&baseError})},
		},
		"diagnostics don't match and are different concrete types": {
			expectDiff: true,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2: func() Diagnostics {
				d := baseError
				d.Severity = hcl.DiagWarning // Altered severity level
				return Diagnostics{makeRPCFriendlyDiag(hclDiagnostic{&d})}
			}(),
		},
	}
	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// This should show no diff as, internally, the two diags are transformed into the same
			// concrete type
			diff := assertDiagnosticsMatch(tc.diags1, tc.diags2)

			if !tc.expectDiff && len(diff) > 0 {
				t.Fatalf("unexpected diff:\n%s", diff)
			}
			if tc.expectDiff && len(diff) == 0 {
				t.Fatalf("expected a diff but got none")
			}
		})
	}
}
