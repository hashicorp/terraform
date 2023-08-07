// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import "github.com/hashicorp/terraform/internal/tfdiags"

// DiagnosticExtraCheckRule provides an interface for diagnostic ExtraInfo to
// retrieve an embedded CheckRule from within a tfdiags.Diagnostic.
type DiagnosticExtraCheckRule interface {
	// DiagnosticOriginatesFromCheckRule returns the CheckRule that the
	// surrounding diagnostic originated from.
	DiagnosticOriginatesFromCheckRule() CheckRule
}

// DiagnosticOriginatesFromCheckRule checks if the provided diagnostic contains
// a CheckRule as ExtraInfo and returns that CheckRule and true if it does. This
// function returns an empty CheckRule and false if the diagnostic does not
// contain a CheckRule.
func DiagnosticOriginatesFromCheckRule(diag tfdiags.Diagnostic) (CheckRule, bool) {
	maybe := tfdiags.ExtraInfo[DiagnosticExtraCheckRule](diag)
	if maybe == nil {
		return CheckRule{}, false
	}
	return maybe.DiagnosticOriginatesFromCheckRule(), true
}

// CheckRuleDiagnosticExtra is an object that can be attached to diagnostics
// that originate from check rules.
//
// It implements the DiagnosticExtraCheckRule interface for retrieving the
// concrete CheckRule that spawned the diagnostic.
//
// It also implements the tfdiags.DiagnosticExtraDoNotConsolidate interface, to
// stop diagnostics created by check blocks being consolidated.
//
// It also implements the tfdiags.DiagnosticExtraUnwrapper interface, as nested
// data blocks will attach this struct but do want to lose any extra info
// embedded in the original diagnostic.
type CheckRuleDiagnosticExtra struct {
	CheckRule CheckRule

	wrapped interface{}
}

var (
	_ DiagnosticExtraCheckRule                = (*CheckRuleDiagnosticExtra)(nil)
	_ tfdiags.DiagnosticExtraDoNotConsolidate = (*CheckRuleDiagnosticExtra)(nil)
	_ tfdiags.DiagnosticExtraUnwrapper        = (*CheckRuleDiagnosticExtra)(nil)
	_ tfdiags.DiagnosticExtraWrapper          = (*CheckRuleDiagnosticExtra)(nil)
)

func (c *CheckRuleDiagnosticExtra) UnwrapDiagnosticExtra() interface{} {
	return c.wrapped
}

func (c *CheckRuleDiagnosticExtra) WrapDiagnosticExtra(inner interface{}) {
	if c.wrapped != nil {
		// This is a logical inconsistency, the caller should know whether they
		// have already wrapped an extra or not.
		panic("Attempted to wrap a diagnostic extra into a CheckRuleDiagnosticExtra that is already wrapping a different extra. This is a bug in Terraform, please report it.")
	}
	c.wrapped = inner
}

func (c *CheckRuleDiagnosticExtra) DoNotConsolidateDiagnostic() bool {
	// Do not consolidate warnings from check blocks.
	return c.CheckRule.Container.CheckableKind() == CheckableCheck
}

func (c *CheckRuleDiagnosticExtra) DiagnosticOriginatesFromCheckRule() CheckRule {
	return c.CheckRule
}
