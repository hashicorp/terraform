// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LocalValue represents a local value defined within a [Stack].
type LocalValue struct {
	addr stackaddrs.AbsLocalValue

	main *Main

	value perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Referenceable = (*LocalValue)(nil)
var _ Plannable = (*LocalValue)(nil)

func newLocalValue(main *Main, addr stackaddrs.AbsLocalValue) *LocalValue {
	return &LocalValue{
		addr: addr,
		main: main,
	}
}

func (v *LocalValue) Addr() stackaddrs.AbsLocalValue {
	return v.addr
}

func (v *LocalValue) Config(ctx context.Context) *LocalValueConfig {
	configAddr := stackaddrs.ConfigForAbs(v.Addr())
	stackCfg := v.main.StackConfig(ctx, configAddr.Stack)
	return stackCfg.LocalValue(ctx, configAddr.Item)
}

func (v *LocalValue) Declaration(ctx context.Context) *stackconfig.LocalValue {
	return v.Config(ctx).Declaration()
}

func (v *LocalValue) Stack(ctx context.Context, phase EvalPhase) *Stack {
	return v.main.Stack(ctx, v.Addr().Stack, phase)
}

func (v *LocalValue) Value(ctx context.Context, phase EvalPhase) cty.Value {
	val, _ := v.CheckValue(ctx, phase)
	return val
}

// ExprReferenceValue implements Referenceable.
func (v *LocalValue) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	return v.Value(ctx, phase)
}

func (v *LocalValue) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := v.CheckValue(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

func (v *LocalValue) CheckValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, v.value.For(phase), v.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			decl := v.Declaration(ctx)
			stack := v.Stack(ctx, phase)

			if stack == nil {
				return cty.DynamicVal, diags
			}

			result, moreDiags := EvalExprAndEvalContext(ctx, decl.Value, phase, stack)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return cty.DynamicVal, diags
			}

			return result.Value, diags
		},
	))
}

// PlanChanges implements Plannable as a plan-time validation of the local value
func (v *LocalValue) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, PlanPhase)
}

// References implements Referrer
func (v *LocalValue) References(ctx context.Context) []stackaddrs.AbsReference {
	cfg := v.Declaration(ctx)
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(ctx, cfg.Value)...)
	return makeReferencesAbsolute(ret, v.Addr().Stack)
}

// CheckApply implements Applyable.
func (v *LocalValue) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, ApplyPhase)
}

func (v *LocalValue) tracingName() string {
	return v.Addr().String()
}
