// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
)

// testDiagWithExtra is a test helper that creates a diagnostic with extra info
type testDiagWithExtra struct {
	severity Severity
	summary  string
	detail   string
	subject  *SourceRange
	extra    interface{}
}

var _ Diagnostic = testDiagWithExtra{}
var _ ComparableDiagnostic = testDiagWithExtra{}

func (d testDiagWithExtra) Severity() Severity {
	return d.severity
}

func (d testDiagWithExtra) Description() Description {
	return Description{
		Summary: d.summary,
		Detail:  d.detail,
	}
}

func (d testDiagWithExtra) Source() Source {
	return Source{
		Subject: d.subject,
	}
}

func (d testDiagWithExtra) FromExpr() *FromExpr {
	return nil
}

func (d testDiagWithExtra) ExtraInfo() interface{} {
	return d.extra
}

func (d testDiagWithExtra) Equals(other ComparableDiagnostic) bool {
	od, ok := other.(testDiagWithExtra)
	if !ok {
		return false
	}
	return d.severity == od.severity && d.summary == od.summary && d.detail == od.detail
}

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

func Test_assertDiagnosticsAndExtrasMatch(t *testing.T) {
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
		"both empty": {
			expectDiff: false,
			diags1:     Diagnostics{},
			diags2:     Diagnostics{},
		},
		"diagnostics match with no extras": {
			expectDiff: false,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2:     Diagnostics{makeRPCFriendlyDiag(hclDiagnostic{&baseError})},
		},
		"diagnostics match with same deprecation extra": {
			expectDiff: false,
			diags1: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"},
			}},
			diags2: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"},
			}},
		},
		"diagnostics match but different deprecation extras": {
			expectDiff: true,
			diags1: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"},
			}},
			diags2: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 2"},
			}},
		},
		"diagnostics match but one has extra and one doesn't": {
			expectDiff: true,
			diags1: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"},
			}},
			diags2: Diagnostics{testDiagWithExtra{
				severity: Error,
				summary:  "error",
				detail:   "this is an error",
				extra:    nil,
			}},
		},
		"diagnostics don't match - fails on base comparison": {
			expectDiff: true,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2: func() Diagnostics {
				d := baseError
				d.Severity = hcl.DiagWarning
				return Diagnostics{hclDiagnostic{&d}}
			}(),
		},
		"multiple diagnostics with matching extras": {
			expectDiff: false,
			diags1: Diagnostics{
				testDiagWithExtra{
					severity: Warning,
					summary:  "warning 1",
					detail:   "detail 1",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"},
				},
				testDiagWithExtra{
					severity: Error,
					summary:  "error 1",
					detail:   "detail 2",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 2"},
				},
			},
			diags2: Diagnostics{
				testDiagWithExtra{
					severity: Warning,
					summary:  "warning 1",
					detail:   "detail 1",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"},
				},
				testDiagWithExtra{
					severity: Error,
					summary:  "error 1",
					detail:   "detail 2",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 2"},
				},
			},
		},
		"multiple diagnostics with one mismatched extra": {
			expectDiff: true,
			diags1: Diagnostics{
				testDiagWithExtra{
					severity: Warning,
					summary:  "warning 1",
					detail:   "detail 1",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"},
				},
				testDiagWithExtra{
					severity: Error,
					summary:  "error 1",
					detail:   "detail 2",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 2"},
				},
			},
			diags2: Diagnostics{
				testDiagWithExtra{
					severity: Warning,
					summary:  "warning 1",
					detail:   "detail 1",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"},
				},
				testDiagWithExtra{
					severity: Error,
					summary:  "error 1",
					detail:   "detail 2",
					extra:    &DeprecationOriginDiagnosticExtra{OriginDescription: "different origin"},
				},
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			diff := assertDiagnosticsAndExtrasMatch(tc.diags1, tc.diags2)

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
