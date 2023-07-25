package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type Component struct {
	addr stackaddrs.AbsComponent

	main *Main

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
}

var _ Plannable = (*Component)(nil)
var _ Referenceable = (*Component)(nil)

func newComponent(main *Main, addr stackaddrs.AbsComponent) *Component {
	return &Component{
		addr: addr,
		main: main,
	}
}

func (c *Component) Addr() stackaddrs.AbsComponent {
	return c.addr
}

func (c *Component) Config(ctx context.Context) *ComponentConfig {
	configAddr := stackaddrs.ConfigForAbs(c.Addr())
	stackConfig := c.main.StackConfig(ctx, configAddr.Stack)
	if stackConfig == nil {
		return nil
	}
	return stackConfig.Component(ctx, configAddr.Item)
}

func (c *Component) Declaration(ctx context.Context) *stackconfig.Component {
	cfg := c.Config(ctx)
	if cfg == nil {
		return nil
	}
	return cfg.Declaration(ctx)
}

func (c *Component) Stack(ctx context.Context) *Stack {
	// Unchecked because we should've been constructed from the same stack
	// object we're about to return, and so this should be valid unless
	// the original construction was from an invalid object itself.
	return c.main.StackUnchecked(ctx, c.Addr().Stack)
}

// ForEachValue returns the result of evaluating the "for_each" expression
// for this stack call, with the following exceptions:
//   - If the stack call doesn't use "for_each" at all, returns [cty.NilVal].
//   - If the for_each expression is present but too invalid to evaluate,
//     returns [cty.DynamicVal] to represent that the for_each value cannot
//     be determined.
//
// A present and valid "for_each" expression produces a result that's
// guaranteed to be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (only the top-level value)
// - Not sensitive (only the top-level value)
func (c *Component) ForEachValue(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckForEachValue(ctx, phase)
	return ret
}

// CheckForEachValue evaluates the "for_each" expression if present, validates
// that its value is valid, and then returns that value.
//
// If this call does not use "for_each" then this immediately returns cty.NilVal
// representing the absense of the value.
//
// If the diagnostics does not include errors and the result is not cty.NilVal
// then callers can assume that the result value will be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (except for nested map/object element values)
// - Not sensitive (only the top-level value)
//
// If the diagnostics _does_ include errors then the result might be
// [cty.DynamicVal], which represents that the for_each expression was so invalid
// that we cannot know the for_each value.
func (c *Component) CheckForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := doOnceWithDiags(
		ctx, c.forEachValue.For(phase), c.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := c.Declaration(ctx)

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, c.Stack(ctx))
				diags = diags.Append(moreDiags)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
				}

				if !result.Value.IsKnown() {
					// FIXME: We should somehow allow this and emit a
					// "deferred change" representing all of the as-yet-unknown
					// instances of this call and everything beneath it.
					diags = diags.Append(result.Diagnostic(
						tfdiags.Error,
						"Invalid for_each value",
						"The for_each value must not be derived from values that will be determined only during the apply phase.",
					))
				}

				return result.Value, diags

			default:
				// This stack config doesn't use for_each at all
				return cty.NilVal, diags
			}
		},
	)
	if val == cty.NilVal && diags.HasErrors() {
		// We use cty.DynamicVal as the placeholder for an invalid for_each,
		// to represent "unknown for_each value" as distinct from "no for_each
		// expression at all".
		val = cty.DynamicVal
	}
	return val, diags
}

// Instances returns all of the instances of the call known to be declared
// by the configuration.
//
// Calcluating this involves evaluating the call's for_each expression if any,
// and so this call may block on evaluation of other objects in the
// configuration.
//
// If the configuration has an invalid definition of the instances then the
// result will be nil. Callers that need to distinguish between invalid
// definitions and valid definitions of zero instances can rely on the
// result being a non-nil zero-length map in the latter case.
//
// This function doesn't return any diagnostics describing ways in which the
// for_each expression is invalid because we assume that the main plan walk
// will visit the stack call directly and ask it to check itself, and that
// call will be the one responsible for returning any diagnostics.
func (c *Component) Instances(ctx context.Context, phase EvalPhase) map[addrs.InstanceKey]*ComponentInstance {
	forEachVal := c.ForEachValue(ctx, phase)

	switch {
	case forEachVal == cty.NilVal:
		// No for_each expression at all, then. We have exactly one instance
		// without an instance key and with no repetition data.
		return map[addrs.InstanceKey]*ComponentInstance{
			addrs.NoKey: newComponentInstance(c, addrs.NoKey, instances.RepetitionData{
				// no repetition symbols available in this case
			}),
		}

	case !forEachVal.IsKnown():
		// The for_each expression is too invalid for us to be able to
		// know which instances exist. A totally nil map (as opposed to a
		// non-nil map of length zero) signals that situation.
		return nil

	default:
		// Otherwise we should be able to assume the value is valid per the
		// definition of [CheckForEachValue]. The following will panic if
		// that other function doesn't satisfy its documented contract;
		// if that happens, prefer to correct [CheckForEachValue] than to
		// add additional complexity here.

		// NOTE: We MUST return a non-nil map from every return path under
		// this case, even if there are zero elements in it, because a nil map
		// represents an _invalid_ for_each expression (handled above).

		ty := forEachVal.Type()
		switch {
		case ty.IsObjectType() || ty.IsMapType():
			elems := forEachVal.AsValueMap()
			ret := make(map[addrs.InstanceKey]*ComponentInstance, len(elems))
			for k, v := range elems {
				ik := addrs.StringKey(k)
				ret[ik] = newComponentInstance(c, ik, instances.RepetitionData{
					EachKey:   cty.StringVal(k),
					EachValue: v,
				})
			}
			return ret

		case ty.IsSetType():
			// ForEachValue should have already guaranteed us a set of strings,
			// but we'll check again here just so we can panic more intellgibly
			// if that function is buggy.
			if ty.ElementType() != cty.String {
				panic(fmt.Sprintf("ForEachValue returned invalid result %#v", forEachVal))
			}

			elems := forEachVal.AsValueSlice()
			ret := make(map[addrs.InstanceKey]*ComponentInstance, len(elems))
			for _, sv := range elems {
				k := addrs.StringKey(sv.AsString())
				ret[k] = newComponentInstance(c, k, instances.RepetitionData{
					EachKey:   sv,
					EachValue: sv,
				})
			}
			return ret

		default:
			panic(fmt.Sprintf("ForEachValue returned invalid result %#v", forEachVal))
		}
	}
}

// ExprReferenceValue implements Referenceable.
func (c *Component) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	panic("unimplemented")
}

// PlanChanges implements Plannable by performing plan-time validation of
// the component call itself.
//
// The plan walk driver must call [Component.Instances] and also call
// PlanChanges for each instance separately in order to produce a complete
// plan.
func (c *Component) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckForEachValue(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	return nil, diags
}

func (c *Component) tracingName() string {
	return c.Addr().String()
}
