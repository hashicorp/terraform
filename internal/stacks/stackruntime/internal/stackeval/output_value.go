// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// OutputValue represents an input variable belonging to a [Stack].
type OutputValue struct {
	addr stackaddrs.AbsOutputValue

	main *Main

	resultValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Plannable = (*OutputValue)(nil)

func newOutputValue(main *Main, addr stackaddrs.AbsOutputValue) *OutputValue {
	return &OutputValue{
		addr: addr,
		main: main,
	}
}

func (v *OutputValue) Addr() stackaddrs.AbsOutputValue {
	return v.addr
}

func (v *OutputValue) Config(ctx context.Context) *OutputValueConfig {
	configAddr := stackaddrs.ConfigForAbs(v.Addr())
	stackConfig := v.main.StackConfig(ctx, configAddr.Stack)
	if stackConfig == nil {
		return nil
	}
	return stackConfig.OutputValue(ctx, configAddr.Item)
}

func (v *OutputValue) Stack(ctx context.Context, phase EvalPhase) *Stack {
	return v.main.Stack(ctx, v.Addr().Stack, phase)
}

func (v *OutputValue) Declaration(ctx context.Context) *stackconfig.OutputValue {
	cfg := v.Config(ctx)
	if cfg == nil {
		return nil
	}
	return cfg.Declaration(ctx)
}

func (v *OutputValue) ResultType(ctx context.Context) (cty.Type, *typeexpr.Defaults) {
	decl := v.Declaration(ctx)
	if decl == nil {
		// If we get here then something odd must be going on, but
		// we don't have enough context to guess why so we'll just
		// return, in effect, "I don't know".
		return cty.DynamicPseudoType, &typeexpr.Defaults{
			Type: cty.DynamicPseudoType,
		}
	}
	return decl.Type.Constraint, decl.Type.Defaults
}

func (v *OutputValue) CheckResultType(ctx context.Context) (cty.Type, *typeexpr.Defaults, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ty, defs := v.ResultType(ctx)
	decl := v.Declaration(ctx)
	if v.Addr().Stack.IsRoot() {
		// A root output value cannot return provider configuration references,
		// because root outputs outlive the operation that generated them but
		// provider instances are live only during a single evaluation.
		for _, path := range stackconfigtypes.ProviderConfigPathsInType(ty) {
			// We'll construct a synthetic error so that we can conveniently
			// use tfdiags.FormatError to help construct a more specific error
			// message.
			err := path.NewErrorf("cannot return provider configuration reference from the root stack")
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid output value type",
				Detail: fmt.Sprintf(
					"Unsupported output value type: %s.",
					tfdiags.FormatError(err),
				),
				Subject: decl.Type.Expression.Range().Ptr(),
			})
		}
	}
	return ty, defs, diags
}

func (v *OutputValue) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	val, _ := v.CheckResultValue(ctx, phase)
	return val
}

func (v *OutputValue) CheckResultValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return withCtyDynamicValPlaceholder(doOnceWithDiags(
		ctx, v.resultValue.For(phase), v.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			ty, defs := v.ResultType(ctx)

			stack := v.Stack(ctx, phase)
			if stack == nil {
				// If we're in a stack whose expansion isn't known yet then
				// we'll return an unknown value placeholder so that
				// downstreams can at least do type-checking.
				return cty.UnknownVal(ty), diags
			}
			result, moreDiags := EvalExprAndEvalContext(ctx, v.Declaration(ctx).Value, phase, stack)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return cty.UnknownVal(ty), diags
			}

			var err error
			if defs != nil {
				result.Value = defs.Apply(result.Value)
			}
			result.Value, err = convert.Convert(result.Value, ty)
			if err != nil {
				diags = diags.Append(result.Diagnostic(
					tfdiags.Error,
					"Invalid output value result",
					fmt.Sprintf("Unsuitable value for output %q: %s.", v.Addr().Item.Name, tfdiags.FormatError(err)),
				))
				return cty.UnknownVal(ty), diags
			}

			return result.Value, diags
		},
	))
}

func (v *OutputValue) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME: We should really check the type during the validation phase
	// in OutputValueConfig, rather than the planning phase in OutputValue.
	_, _, moreDiags := v.CheckResultType(ctx)
	diags = diags.Append(moreDiags)
	_, moreDiags = v.CheckResultValue(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

// PlanChanges implements Plannable as a plan-time validation of the variable's
// declaration and of the caller's definition of the variable.
func (v *OutputValue) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, PlanPhase)
}

// References implements Referrer
func (v *OutputValue) References(ctx context.Context) []stackaddrs.AbsReference {
	cfg := v.Declaration(ctx)
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(ctx, cfg.Value)...)
	return makeReferencesAbsolute(ret, v.Addr().Stack)
}

// RequiredComponents implements Applyable
func (v *OutputValue) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return v.main.requiredComponentsForReferrer(ctx, v, PlanPhase)
}

// CheckApply implements Applyable.
func (v *OutputValue) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, v.checkValid(ctx, ApplyPhase)
}

func (v *OutputValue) tracingName() string {
	return v.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (v *OutputValue) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := v.Addr().String()
	v.resultValue.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), name)
	})
}
