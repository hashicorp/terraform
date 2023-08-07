// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestCheckRuleDiagnosticExtra_WrapsExtra(t *testing.T) {
	var originals tfdiags.Diagnostics
	originals = originals.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
		Extra:    "extra",
	})

	overridden := tfdiags.OverrideAll(originals, tfdiags.Warning, func() tfdiags.DiagnosticExtraWrapper {
		return &CheckRuleDiagnosticExtra{}
	})

	if overridden[0].ExtraInfo().(*CheckRuleDiagnosticExtra).wrapped.(string) != "extra" {
		t.Errorf("unexpected extra info: %v", overridden[0].ExtraInfo())
	}
}

func TestCheckRuleDiagnosticExtra_Unwraps(t *testing.T) {
	var originals tfdiags.Diagnostics
	originals = originals.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
		Extra:    "extra",
	})

	overridden := tfdiags.OverrideAll(originals, tfdiags.Warning, func() tfdiags.DiagnosticExtraWrapper {
		return &CheckRuleDiagnosticExtra{}
	})

	result := tfdiags.ExtraInfo[string](overridden[0])
	if result != "extra" {
		t.Errorf("unexpected extra info: %v", result)
	}
}

func TestCheckRuleDiagnosticExtra_DoNotConsolidate(t *testing.T) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
		Extra: &CheckRuleDiagnosticExtra{
			CheckRule: NewCheckRule(AbsOutputValue{
				Module: RootModuleInstance,
				OutputValue: OutputValue{
					Name: "output",
				},
			}, OutputPrecondition, 0),
		},
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
		Extra: &CheckRuleDiagnosticExtra{
			CheckRule: NewCheckRule(AbsCheck{
				Module: RootModuleInstance,
				Check: Check{
					Name: "check",
				},
			}, CheckAssertion, 0),
		},
	})

	if tfdiags.DoNotConsolidateDiagnostic(diags[0]) {
		t.Errorf("first diag should be consolidated but was not")
	}

	if !tfdiags.DoNotConsolidateDiagnostic(diags[1]) {
		t.Errorf("second diag should not be consolidated but was")
	}

}

func TestDiagnosticOriginatesFromCheckRule_Passes(t *testing.T) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "original error",
		Detail:   "this is an error",
		Extra:    &CheckRuleDiagnosticExtra{},
	})

	if _, ok := DiagnosticOriginatesFromCheckRule(diags[0]); ok {
		t.Errorf("first diag did not originate from check rule but thinks it did")
	}

	if _, ok := DiagnosticOriginatesFromCheckRule(diags[1]); !ok {
		t.Errorf("second diag did originate from check rule but this it did not")
	}
}
