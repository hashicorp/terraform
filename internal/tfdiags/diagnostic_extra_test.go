// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"testing"
)

// testDiagnosticWithExtra is a test helper that creates a diagnostic with extra info
type testDiagnosticWithExtra struct {
	severity Severity
	summary  string
	detail   string
	extra    interface{}
}

var _ Diagnostic = testDiagnosticWithExtra{}

func (d testDiagnosticWithExtra) Severity() Severity {
	return d.severity
}

func (d testDiagnosticWithExtra) Description() Description {
	return Description{
		Summary: d.summary,
		Detail:  d.detail,
	}
}

func (d testDiagnosticWithExtra) Source() Source {
	return Source{}
}

func (d testDiagnosticWithExtra) FromExpr() *FromExpr {
	return nil
}

func (d testDiagnosticWithExtra) ExtraInfo() interface{} {
	return d.extra
}

// Mock implementations for testing

type mockBecauseUnknown struct {
	caused bool
}

func (m mockBecauseUnknown) DiagnosticCausedByUnknown() bool {
	return m.caused
}

type mockBecauseEphemeral struct {
	caused bool
}

func (m mockBecauseEphemeral) DiagnosticCausedByEphemeral() bool {
	return m.caused
}

type mockBecauseSensitive struct {
	caused bool
}

func (m mockBecauseSensitive) DiagnosticCausedBySensitive() bool {
	return m.caused
}

type mockDoNotConsolidate struct {
	doNotConsolidate bool
}

func (m mockDoNotConsolidate) DoNotConsolidateDiagnostic() bool {
	return m.doNotConsolidate
}

type mockCausedByTestFailure struct {
	causedByTestFailure bool
	verboseMode         bool
}

func (m mockCausedByTestFailure) DiagnosticCausedByTestFailure() bool {
	return m.causedByTestFailure
}

func (m mockCausedByTestFailure) IsTestVerboseMode() bool {
	return m.verboseMode
}

func TestDiagnosticExtrasEqual(t *testing.T) {
	tests := map[string]struct {
		diag1     Diagnostic
		diag2     Diagnostic
		wantEqual bool
	}{
		"both nil extras": {
			diag1:     testDiagnosticWithExtra{extra: nil},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: true,
		},

		// DiagnosticExtraBecauseUnknown tests
		"unknown: both have same value true": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			wantEqual: true,
		},
		"unknown: both have same value false": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: false}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: false}},
			wantEqual: true,
		},
		"unknown: different values": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: false}},
			wantEqual: false,
		},
		"unknown: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},

		// DiagnosticExtraBecauseEphemeral tests
		"ephemeral: both have same value true": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseEphemeral{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseEphemeral{caused: true}},
			wantEqual: true,
		},
		"ephemeral: different values": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseEphemeral{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseEphemeral{caused: false}},
			wantEqual: false,
		},
		"ephemeral: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseEphemeral{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},

		// DiagnosticExtraBecauseSensitive tests
		"sensitive: both have same value true": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: true}},
			wantEqual: true,
		},
		"sensitive: different values": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: false}},
			wantEqual: false,
		},
		"sensitive: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},

		// DiagnosticExtraDoNotConsolidate tests
		"doNotConsolidate: both have same value true": {
			diag1:     testDiagnosticWithExtra{extra: mockDoNotConsolidate{doNotConsolidate: true}},
			diag2:     testDiagnosticWithExtra{extra: mockDoNotConsolidate{doNotConsolidate: true}},
			wantEqual: true,
		},
		"doNotConsolidate: different values": {
			diag1:     testDiagnosticWithExtra{extra: mockDoNotConsolidate{doNotConsolidate: true}},
			diag2:     testDiagnosticWithExtra{extra: mockDoNotConsolidate{doNotConsolidate: false}},
			wantEqual: false,
		},
		"doNotConsolidate: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: mockDoNotConsolidate{doNotConsolidate: true}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},

		// DiagnosticExtraCausedByTestFailure tests
		"testFailure: both have same values": {
			diag1:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: true}},
			diag2:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: true}},
			wantEqual: true,
		},
		"testFailure: different causedByTestFailure": {
			diag1:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: true}},
			diag2:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: false, verboseMode: true}},
			wantEqual: false,
		},
		"testFailure: different verboseMode": {
			diag1:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: true}},
			diag2:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: false}},
			wantEqual: false,
		},
		"testFailure: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: mockCausedByTestFailure{causedByTestFailure: true, verboseMode: true}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},

		// DiagnosticExtraDeprecationOrigin tests
		"deprecation: both have same description": {
			diag1:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"}},
			diag2:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"}},
			wantEqual: true,
		},
		"deprecation: different descriptions": {
			diag1:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 1"}},
			diag2:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "origin 2"}},
			wantEqual: false,
		},
		"deprecation: one nil one not": {
			diag1:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "test origin"}},
			diag2:     testDiagnosticWithExtra{extra: nil},
			wantEqual: false,
		},
		"deprecation: empty vs non-empty": {
			diag1:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: ""}},
			diag2:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "test"}},
			wantEqual: false,
		},
		"deprecation: both empty": {
			diag1:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: ""}},
			diag2:     testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: ""}},
			wantEqual: true,
		},

		// Different extra types (only matching interfaces matter)
		"different extra types with same interface values": {
			diag1:     testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			diag2:     testDiagnosticWithExtra{extra: mockBecauseSensitive{caused: true}},
			wantEqual: false, // different because one has unknown, other has sensitive
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := DiagnosticExtrasEqual(tc.diag1, tc.diag2)
			if got != tc.wantEqual {
				t.Errorf("DiagnosticExtrasEqual() = %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

// Test that the comparison is symmetric
func TestDiagnosticExtrasEqual_Symmetric(t *testing.T) {
	tests := []struct {
		name  string
		diag1 Diagnostic
		diag2 Diagnostic
	}{
		{
			name:  "unknown extras",
			diag1: testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
			diag2: testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: false}},
		},
		{
			name:  "deprecation extras",
			diag1: testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "a"}},
			diag2: testDiagnosticWithExtra{extra: &DeprecationOriginDiagnosticExtra{OriginDescription: "b"}},
		},
		{
			name:  "nil vs non-nil",
			diag1: testDiagnosticWithExtra{extra: nil},
			diag2: testDiagnosticWithExtra{extra: mockBecauseUnknown{caused: true}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result1 := DiagnosticExtrasEqual(tc.diag1, tc.diag2)
			result2 := DiagnosticExtrasEqual(tc.diag2, tc.diag1)
			if result1 != result2 {
				t.Errorf("DiagnosticExtrasEqual is not symmetric: (a,b)=%v, (b,a)=%v", result1, result2)
			}
		})
	}
}
