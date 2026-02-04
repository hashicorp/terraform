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

func Test_assertDiagnosticsAndExtrasMatch(t *testing.T) {
	baseError := hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Deprecated value used",
		Detail:   "Deprecated resource attribute \"foo\" used",
	}

	cases := map[string]struct {
		diags1     Diagnostics
		diags2     Diagnostics
		expectDiff bool
	}{
		"diagnostics match without extras": {
			expectDiff: false,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2:     Diagnostics{hclDiagnostic{&baseError}},
		},
		"diagnostics don't match - different summary": {
			expectDiff: true,
			diags1:     Diagnostics{hclDiagnostic{&baseError}},
			diags2: func() Diagnostics {
				d := baseError
				d.Summary = "Different summary"
				return Diagnostics{hclDiagnostic{&d}}
			}(),
		},
		"diagnostics match with same extras": {
			expectDiff: false,
			diags1: func() Diagnostics {
				diag := hclDiagnostic{&baseError}
				wrapped := Override(diag, Warning, func() DiagnosticExtraWrapper {
					return &DeprecationOriginDiagnosticExtra{
						OriginDescription: "provider configuration",
					}
				})
				return Diagnostics{wrapped}
			}(),
			diags2: func() Diagnostics {
				diag := hclDiagnostic{&baseError}
				wrapped := Override(diag, Warning, func() DiagnosticExtraWrapper {
					return &DeprecationOriginDiagnosticExtra{
						OriginDescription: "provider configuration",
					}
				})
				return Diagnostics{wrapped}
			}(),
		},
		"diagnostics don't match - different extras": {
			expectDiff: true,
			diags1: func() Diagnostics {
				diag := hclDiagnostic{&baseError}
				wrapped := Override(diag, Warning, func() DiagnosticExtraWrapper {
					return &DeprecationOriginDiagnosticExtra{
						OriginDescription: "provider configuration",
					}
				})
				return Diagnostics{wrapped}
			}(),
			diags2: func() Diagnostics {
				diag := hclDiagnostic{&baseError}
				wrapped := Override(diag, Warning, func() DiagnosticExtraWrapper {
					return &DeprecationOriginDiagnosticExtra{
						OriginDescription: "module configuration",
					}
				})
				return Diagnostics{wrapped}
			}(),
		},
		"diagnostics don't match - one has extras, other doesn't": {
			expectDiff: true,
			diags1: func() Diagnostics {
				diag := hclDiagnostic{&baseError}
				wrapped := Override(diag, Warning, func() DiagnosticExtraWrapper {
					return &DeprecationOriginDiagnosticExtra{
						OriginDescription: "provider configuration",
					}
				})
				return Diagnostics{wrapped}
			}(),
			diags2: Diagnostics{hclDiagnostic{&baseError}},
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

// Test extras comparison with various extra interface types
func Test_extrasMatch(t *testing.T) {
	cases := map[string]struct {
		extra1    interface{}
		extra2    interface{}
		wantMatch bool
	}{
		"both nil": {
			extra1:    nil,
			extra2:    nil,
			wantMatch: true,
		},
		"one nil": {
			extra1:    &DeprecationOriginDiagnosticExtra{OriginDescription: "test"},
			extra2:    nil,
			wantMatch: false,
		},
		"same deprecation origin": {
			extra1:    &DeprecationOriginDiagnosticExtra{OriginDescription: "provider.aws"},
			extra2:    &DeprecationOriginDiagnosticExtra{OriginDescription: "provider.aws"},
			wantMatch: true,
		},
		"different deprecation origin": {
			extra1:    &DeprecationOriginDiagnosticExtra{OriginDescription: "provider.aws"},
			extra2:    &DeprecationOriginDiagnosticExtra{OriginDescription: "provider.gcp"},
			wantMatch: false,
		},
		"same wrapped extras": {
			extra1: func() interface{} {
				e := &DeprecationOriginDiagnosticExtra{OriginDescription: "outer"}
				e.WrapDiagnosticExtra(&DeprecationOriginDiagnosticExtra{OriginDescription: "inner"})
				return e
			}(),
			extra2: func() interface{} {
				e := &DeprecationOriginDiagnosticExtra{OriginDescription: "outer"}
				e.WrapDiagnosticExtra(&DeprecationOriginDiagnosticExtra{OriginDescription: "inner"})
				return e
			}(),
			wantMatch: true,
		},
		"different wrapped extras": {
			extra1: func() interface{} {
				e := &DeprecationOriginDiagnosticExtra{OriginDescription: "outer"}
				e.WrapDiagnosticExtra(&DeprecationOriginDiagnosticExtra{OriginDescription: "inner1"})
				return e
			}(),
			extra2: func() interface{} {
				e := &DeprecationOriginDiagnosticExtra{OriginDescription: "outer"}
				e.WrapDiagnosticExtra(&DeprecationOriginDiagnosticExtra{OriginDescription: "inner2"})
				return e
			}(),
			wantMatch: false,
		},
		"same boolean extra - caused by unknown true": {
			extra1:    testExtraBecauseUnknown(true),
			extra2:    testExtraBecauseUnknown(true),
			wantMatch: true,
		},
		"same boolean extra - caused by unknown false": {
			extra1:    testExtraBecauseUnknown(false),
			extra2:    testExtraBecauseUnknown(false),
			wantMatch: true,
		},
		"different boolean extra - caused by unknown": {
			extra1:    testExtraBecauseUnknown(true),
			extra2:    testExtraBecauseUnknown(false),
			wantMatch: false,
		},
		"same boolean extra - caused by sensitive": {
			extra1:    testExtraBecauseSensitive(true),
			extra2:    testExtraBecauseSensitive(true),
			wantMatch: true,
		},
		"different boolean extra - caused by sensitive": {
			extra1:    testExtraBecauseSensitive(true),
			extra2:    testExtraBecauseSensitive(false),
			wantMatch: false,
		},
		"one has unknown interface, other doesn't": {
			extra1:    testExtraBecauseUnknown(true),
			extra2:    &DeprecationOriginDiagnosticExtra{OriginDescription: "test"},
			wantMatch: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := extrasMatch(tc.extra1, tc.extra2)
			if got != tc.wantMatch {
				t.Errorf("extrasMatch() = %v, want %v", got, tc.wantMatch)
			}
		})
	}
}

// Test helper types for extra interface testing
type testExtraBecauseUnknown bool

func (e testExtraBecauseUnknown) DiagnosticCausedByUnknown() bool {
	return bool(e)
}

type testExtraBecauseSensitive bool

func (e testExtraBecauseSensitive) DiagnosticCausedBySensitive() bool {
	return bool(e)
}
