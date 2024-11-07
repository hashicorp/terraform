// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// InputVariableConfig represents a "variable" block in a stack configuration.
type InputVariableConfig struct {
	addr   stackaddrs.ConfigInputVariable
	config *stackconfig.InputVariable

	main *Main
}

var _ Validatable = (*InputVariableConfig)(nil)
var _ Referenceable = (*InputVariableConfig)(nil)
var _ namedPromiseReporter = (*InputVariableConfig)(nil)

func newInputVariableConfig(main *Main, addr stackaddrs.ConfigInputVariable, config *stackconfig.InputVariable) *InputVariableConfig {
	if config == nil {
		panic("newInputVariableConfig with nil configuration")
	}
	return &InputVariableConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (v *InputVariableConfig) Addr() stackaddrs.ConfigInputVariable {
	return v.addr
}

func (v *InputVariableConfig) tracingName() string {
	return v.Addr().String()
}

func (v *InputVariableConfig) Declaration() *stackconfig.InputVariable {
	return v.config
}

func (v *InputVariableConfig) TypeConstraint() cty.Type {
	return v.config.Type.Constraint
}

func (v *InputVariableConfig) NestedDefaults() *typeexpr.Defaults {
	return v.config.Type.Defaults
}

// DefaultValue returns the effective default value for this input variable,
// or cty.NilVal if this variable is required.
//
// If the configured default value is invalid, this returns a placeholder
// unknown value of the correct type. Use
// [InputVariableConfig.ValidateDefaultValue] instead if you are intending
// to report configuration diagnostics to the user.
func (v *InputVariableConfig) DefaultValue(ctx context.Context) cty.Value {
	ret, _ := v.ValidateDefaultValue(ctx)
	return ret
}

// ValidateDefaultValue verifies that the specified default value is valid
// and then returns the validated value. If the result is cty.NilVal then
// this input variable is required and so has no default value.
//
// If the returned diagnostics has errors then the returned value is a
// placeholder unknown value of the correct type.
func (v *InputVariableConfig) ValidateDefaultValue(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val := v.config.DefaultValue
	if val == cty.NilVal {
		return cty.NilVal, diags
	}
	want := v.TypeConstraint()
	if defs := v.NestedDefaults(); defs != nil {
		val = defs.Apply(val)
	}
	val, err := convert.Convert(val, want)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid default value for input variable",
			Detail:   fmt.Sprintf("The default value does not conform to the variable's type constraint: %s.", err),
			// TODO: Better to indicate the default value itself, but
			// stackconfig.InputVariable doesn't currently retain that.
			Subject: v.config.DeclRange.ToHCL().Ptr(),
		})
		return cty.UnknownVal(want), diags
	}
	return val, diags
}

// StackConfig returns the stack configuration that this input variable belongs
// to.
func (v *InputVariableConfig) StackConfig(ctx context.Context) *StackConfig {
	return v.main.mustStackConfig(ctx, v.Addr().Stack)
}

// StackCallConfig returns the stack call that would be providing the value
// for this input variable, or nil if this input variable belongs to the
// main (root) stack and therefore its value would come from outside of
// the configuration.
func (v *InputVariableConfig) StackCallConfig(ctx context.Context) *StackCallConfig {
	calleeAddr := v.Addr().Stack
	if calleeAddr.IsRoot() {
		return nil
	}
	callerAddr := calleeAddr.Parent()
	caller := v.main.mustStackConfig(ctx, callerAddr)
	return caller.StackCall(ctx, stackaddrs.StackCall{Name: calleeAddr[len(calleeAddr)-1].Name})
}

// ExprReferenceValue implements Referenceable
func (v *InputVariableConfig) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	if v.Addr().Stack.IsRoot() {
		// During validation the root input variable values are always unknown,
		// because validation tests whether the configuration is valid for
		// _any_ inputs, rather than for _specific_ inputs.
		return v.markValue(cty.UnknownVal(v.TypeConstraint()))
	} else {
		// Our apparent value is the value assigned in the definition object
		// in the parent call.
		call := v.StackCallConfig(ctx)
		val := call.InputVariableValues(ctx, phase)[v.Addr().Item]
		if val == cty.NilVal {
			val = cty.UnknownVal(v.TypeConstraint())
		}
		return v.markValue(val)
	}
}

// markValue returns the given value with any additional cty marks that
// ought to be applied to the value of the variable based on its configuration.
func (v *InputVariableConfig) markValue(val cty.Value) cty.Value {
	if val == cty.NilVal {
		return val
	}
	if v.config.Sensitive {
		val = val.Mark(marks.Sensitive)
	}
	if v.config.Ephemeral {
		val = val.Mark(marks.Ephemeral)
	}
	return val
}

func (v *InputVariableConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	_, moreDiags := v.ValidateDefaultValue(ctx)
	diags = diags.Append(moreDiags)
	return diags
}

// Validate implements Validatable
func (v *InputVariableConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return v.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (v *InputVariableConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, PlanPhase)
}

// reportNamedPromises implements namedPromiseReporter.
func (s *InputVariableConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	// Nothing to report yet
}
