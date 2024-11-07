// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Provider represents a provider configuration in a particular stack config.
type Provider struct {
	addr   stackaddrs.AbsProviderConfig
	config *stackconfig.ProviderConfig

	main *Main

	forEachValue perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	instances    perEvalPhase[promising.Once[withDiagnostics[instancesResult[*ProviderInstance]]]]
}

func newProvider(main *Main, addr stackaddrs.AbsProviderConfig, config *stackconfig.ProviderConfig) *Provider {
	return &Provider{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (p *Provider) Addr() stackaddrs.AbsProviderConfig {
	return p.addr
}

func (p *Provider) Declaration(ctx context.Context) *stackconfig.ProviderConfig {
	return p.config
}

func (p *Provider) Config(ctx context.Context) *ProviderConfig {
	configAddr := stackaddrs.ConfigForAbs(p.Addr())
	stackConfig := p.main.StackConfig(ctx, configAddr.Stack)
	if stackConfig == nil {
		return nil
	}
	return stackConfig.Provider(ctx, configAddr.Item)
}

func (p *Provider) ProviderType(ctx context.Context) *ProviderType {
	return p.main.ProviderType(ctx, p.Addr().Item.Provider)
}

func (p *Provider) Stack(ctx context.Context) *Stack {
	// Unchecked because we should've been constructed from the same stack
	// object we're about to return, and so this should be valid unless
	// the original construction was from an invalid object itself.
	return p.main.StackUnchecked(ctx, p.Addr().Stack)
}

// InstRefValueType returns the type of any values that represent references to
// instances of this provider configuration.
//
// All configurations for the same provider share the same type.
func (p *Provider) InstRefValueType(ctx context.Context) cty.Type {
	decl := p.Declaration(ctx)
	return providerInstanceRefType(decl.ProviderAddr)
}

// ForEachValue returns the result of evaluating the "for_each" expression
// for this provider configuration, with the following exceptions:
//   - If the provider config doesn't use "for_each" at all, returns [cty.NilVal].
//   - If the for_each expression is present but too invalid to evaluate,
//     returns [cty.DynamicVal] to represent that the for_each value cannot
//     be determined.
//
// A present and valid "for_each" expression produces a result that's
// guaranteed to be:
// - Either a set of strings, a map of any element type, or an object type
// - Known and not null (only the top-level value)
// - Not sensitive (only the top-level value)
func (p *Provider) ForEachValue(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := p.CheckForEachValue(ctx, phase)
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
func (p *Provider) CheckForEachValue(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	val, diags := doOnceWithDiags(
		ctx, p.forEachValue.For(phase), p.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics
			cfg := p.Declaration(ctx)

			switch {

			case cfg.ForEach != nil:
				result, moreDiags := evaluateForEachExpr(ctx, cfg.ForEach, phase, p.Stack(ctx), "provider")
				diags = diags.Append(moreDiags)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
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

// Instances returns all of the instances of the provider config known to be
// declared by the configuration.
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
func (p *Provider) Instances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*ProviderInstance, bool) {
	ret, unknown, _ := p.CheckInstances(ctx, phase)
	return ret, unknown
}

func (p *Provider) CheckInstances(ctx context.Context, phase EvalPhase) (map[addrs.InstanceKey]*ProviderInstance, bool, tfdiags.Diagnostics) {
	result, diags := doOnceWithDiags(
		ctx, p.instances.For(phase), p.main,
		func(ctx context.Context) (instancesResult[*ProviderInstance], tfdiags.Diagnostics) {
			forEachVal, diags := p.CheckForEachValue(ctx, phase)
			if diags.HasErrors() {
				return instancesResult[*ProviderInstance]{}, diags
			}

			return instancesMap(forEachVal, func(ik addrs.InstanceKey, rd instances.RepetitionData) *ProviderInstance {
				return newProviderInstance(p, ik, rd)
			}), diags
		},
	)
	return result.insts, result.unknown, diags
}

// ExprReferenceValue implements Referenceable, returning a value containing
// one or more values that act as references to instances of the provider.
func (p *Provider) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	decl := p.Declaration(ctx)
	insts, unknown := p.Instances(ctx, phase)
	refType := p.InstRefValueType(ctx)

	switch {
	case decl.ForEach != nil:
		if unknown {
			return cty.UnknownVal(cty.Map(refType))
		}

		if insts == nil {
			// Then we errored during instance calculation, this should have
			// been caught before we got here.
			return cty.NilVal
		}
		elems := make(map[string]cty.Value, len(insts))
		for instKey := range insts {
			k, ok := instKey.(addrs.StringKey)
			if !ok {
				panic(fmt.Sprintf("provider config with for_each has invalid instance key of type %T", instKey))
			}
			elems[string(k)] = cty.CapsuleVal(refType, &stackaddrs.AbsProviderConfigInstance{
				Stack: p.Addr().Stack,
				Item: stackaddrs.ProviderConfigInstance{
					ProviderConfig: p.Addr().Item,
					Key:            instKey,
				},
			})
		}
		if len(elems) == 0 {
			return cty.MapValEmpty(refType)
		}
		return cty.MapVal(elems)
	default:
		if insts == nil {
			return cty.UnknownVal(refType)
		}
		return cty.CapsuleVal(refType, &stackaddrs.AbsProviderConfigInstance{
			Stack: p.Addr().Stack,
			Item: stackaddrs.ProviderConfigInstance{
				ProviderConfig: p.Addr().Item,
				Key:            addrs.NoKey,
			},
		})
	}
}

func (p *Provider) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := p.CheckForEachValue(ctx, phase)
	diags = diags.Append(moreDiags)
	_, _, moreDiags = p.CheckInstances(ctx, phase)
	diags = diags.Append(moreDiags)
	// Everything else is instance-specific and so the plan walk driver must
	// call p.Instances and ask each instance to plan itself.

	return diags
}

// PlanChanges implements Plannable.
func (p *Provider) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, PlanPhase)
}

// References implements Referrer
func (p *Provider) References(ctx context.Context) []stackaddrs.AbsReference {
	cfg := p.Declaration(ctx)
	var ret []stackaddrs.Reference
	ret = append(ret, ReferencesInExpr(ctx, cfg.ForEach)...)
	if schema, err := p.ProviderType(ctx).Schema(ctx); err == nil {
		ret = append(ret, ReferencesInBody(ctx, cfg.Config, schema.Provider.Block.DecoderSpec())...)
	}
	return makeReferencesAbsolute(ret, p.Addr().Stack)
}

// CheckApply implements ApplyChecker.
func (p *Provider) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, ApplyPhase)
}

// tracingName implements Plannable.
func (p *Provider) tracingName() string {
	return p.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (p *Provider) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := p.Addr().String()
	forEachName := name + " for_each"
	instsName := name + " instances"
	p.forEachValue.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), forEachName)
	})
	p.instances.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[instancesResult[*ProviderInstance]]]) {
		cb(o.PromiseID(), instsName)
	})
	// FIXME: We should call reportNamedPromises on the individual
	// ProviderInstance objects too, but promising.Once doesn't allow us
	// to peek to see if the Once was already resolved without blocking on
	// it, and we don't want to block on any promises in here.
	// Without this, any promises belonging to the individual instances will
	// not be named in a self-dependency error report, but since references
	// to provider instances are always indirect through the provider this
	// shouldn't be a big deal in most cases.
}
