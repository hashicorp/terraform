// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type InspectOpts struct {
	// Optional values to use when asked for the values of input variables.
	//
	// Any that are not specified will appear in expressions as an unknown
	// value using the declared type constraint, thereby acting as
	// placeholders for whatever real values might be defined as planning
	// options.
	InputVariableValues map[stackaddrs.InputVariable]ExternalInputValue

	// Provider factories to use for operations that involve provider clients.
	//
	// Populating this is optional but if not populated then operations which
	// expect to call into providers will return errors.
	ProviderFactories ProviderFactories

	// TestOnlyGlobals is optional and if set makes it possible to use
	// references like _test_only_global.name to refer to values from this
	// map from anywhere in the entire stack configuration.
	//
	// This is intended as a kind of "test double" so that we can write more
	// minimal unit tests that can avoid relying on too many language features
	// all at once, so that hopefully future maintenance will not require
	// making broad changes across many different tests at once, which would
	// then risk inadvertently treating a regression as expected behavior.
	//
	// Configurations that refer to test-only globals are not valid for use
	// outside of the test suite of this package.
	TestOnlyGlobals map[string]cty.Value
}

// EvalExpr evaluates an arbitrary expression in the main scope of the
// specified stack instance using the approach that's appropriate for the
// specified evaluation phase.
//
// Typical use of this method would be with a Main configured for "inspecting",
// using [InspectPhase] as the phase. This method can be used for any phase
// that supports dynamic expression evaluation in principle, but in that case
// evaluation might cause relatively-expensive effects such as creating
// plans for components.
func (m *Main) EvalExpr(ctx context.Context, expr hcl.Expression, scopeStackInst stackaddrs.StackInstance, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	ret, err := promising.MainTask(ctx, func(ctx context.Context) (withDiagnostics[cty.Value], error) {
		var diags tfdiags.Diagnostics

		scope := m.Stack(ctx, scopeStackInst, phase)
		if scope == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Evaluating expression in undeclared stack",
				fmt.Sprintf("Cannot evaluate an expression in %s, because it's not declared by the current configuration.", scopeStackInst),
			))
			return withDiagnostics[cty.Value]{
				Result:      cty.DynamicVal,
				Diagnostics: diags,
			}, nil
		}

		val, moreDiags := EvalExpr(ctx, expr, phase, scope)
		diags = diags.Append(moreDiags)
		return withDiagnostics[cty.Value]{
			Result:      val,
			Diagnostics: diags,
		}, nil
	})
	if err != nil {
		ret.Diagnostics = ret.Diagnostics.Append(diagnosticsForPromisingTaskError(err, m))
	}
	return ret.Result, ret.Diagnostics
}
