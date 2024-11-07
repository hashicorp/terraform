// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProviderConfig represents a single "provider" block in a stack configuration.
type ProviderConfig struct {
	addr   stackaddrs.ConfigProviderConfig
	config *stackconfig.ProviderConfig

	main *Main

	providerArgs promising.Once[withDiagnostics[cty.Value]]
}

func newProviderConfig(main *Main, addr stackaddrs.ConfigProviderConfig, config *stackconfig.ProviderConfig) *ProviderConfig {
	return &ProviderConfig{
		addr:   addr,
		config: config,
		main:   main,
	}
}

func (p *ProviderConfig) Addr() stackaddrs.ConfigProviderConfig {
	return p.addr
}

func (p *ProviderConfig) Declaration(ctx context.Context) *stackconfig.ProviderConfig {
	return p.config
}

func (p *ProviderConfig) ProviderType(ctx context.Context) *ProviderType {
	return p.main.ProviderType(ctx, p.Addr().Item.Provider)
}

func (p *ProviderConfig) InstRefValueType(ctx context.Context) cty.Type {
	decl := p.Declaration(ctx)
	return providerInstanceRefType(decl.ProviderAddr)
}

func (p *ProviderConfig) ProviderArgsDecoderSpec(ctx context.Context) (hcldec.Spec, error) {
	providerType := p.ProviderType(ctx)
	schema, err := providerType.Schema(ctx)
	if err != nil {
		return nil, err
	}
	if schema.Provider.Block == nil {
		return hcldec.ObjectSpec{}, nil
	}
	return schema.Provider.Block.DecoderSpec(), nil
}

// ProviderArgs returns an object value representing an approximation of all
// provider instances declared by this provider configuration, or
// an unknown value (possibly [cty.DynamicVal]) if the configuration is too
// invalid to produce any answer at all.
func (p *ProviderConfig) ProviderArgs(ctx context.Context, phase EvalPhase) cty.Value {
	v, _ := p.CheckProviderArgs(ctx, phase)
	return v
}

func CheckProviderInLockfile(locks depsfile.Locks, providerType *ProviderType, declRange *hcl.Range) (diags tfdiags.Diagnostics) {
	if !depsfile.ProviderIsLockable(providerType.Addr()) {
		return diags
	}

	if p := locks.Provider(providerType.Addr()); p == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider missing from lockfile",
			Detail: fmt.Sprintf(
				"Provider %q is not in the lockfile. This provider must be in the lockfile to be used in the configuration. Please run `tfstacks providers lock` to update the lockfile and run this operation again with an updated configuration.",
				providerType.Addr(),
			),
			Subject: declRange,
		})
	}
	return diags
}

func (p *ProviderConfig) CheckProviderArgs(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, &p.providerArgs, p.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			providerType := p.ProviderType(ctx)
			decl := p.Declaration(ctx)

			depLocks := p.main.DependencyLocks(phase)
			if depLocks != nil {
				// Check if the provider is in the lockfile,
				// if it is not we can not read the provider schema
				lockfileDiags := CheckProviderInLockfile(*depLocks, providerType, decl.DeclRange.ToHCL().Ptr())
				if lockfileDiags.HasErrors() {
					return cty.DynamicVal, lockfileDiags
				}
				diags = diags.Append(lockfileDiags)
			}

			spec, err := p.ProviderArgsDecoderSpec(ctx)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to read provider schema",
					Detail: fmt.Sprintf(
						"Error while reading the schema for %q: %s.",
						providerType.Addr(), err,
					),
					Subject: decl.DeclRange.ToHCL().Ptr(),
				})
				return cty.DynamicVal, diags
			}

			client, err := providerType.UnconfiguredClient()
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to initialize provider",
					Detail: fmt.Sprintf(
						"Error initializing %q to validate %s: %s.",
						providerType.Addr(), p.Addr(), err,
					),
					Subject: decl.DeclRange.ToHCL().Ptr(),
				})
				return cty.UnknownVal(hcldec.ImpliedType(spec)), diags
			}

			body := decl.Config
			if body == nil {
				// A provider with no configuration is valid (just means no
				// attributes or blocks), but we need to pass an empty body to
				// the evaluator to avoid a panic.
				body = hcl.EmptyBody()
			}

			configVal, moreDiags := EvalBody(ctx, body, spec, phase, p)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return cty.UnknownVal(hcldec.ImpliedType(spec)), diags
			}
			// We unmark the config before making the RPC call, but will still
			// return the original possibly-marked config if successful.
			unmarkedConfigVal, _ := configVal.UnmarkDeep()
			validateResp := client.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
				Config: unmarkedConfigVal,
			})
			diags = diags.Append(validateResp.Diagnostics)
			if validateResp.Diagnostics.HasErrors() {
				return cty.UnknownVal(hcldec.ImpliedType(spec)), diags
			}

			return configVal, diags
		},
	)
}

// ResolveExpressionReference implements ExpressionScope for the purposes
// of validating the static provider configuration before it has been expanded
// into multiple instances.
func (p *ProviderConfig) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	repetition := instances.RepetitionData{}
	if p.Declaration(ctx).ForEach != nil {
		// We're producing an approximation across all eventual instances
		// of this call, so we'll set each.key and each.value to unknown
		// values.
		repetition.EachKey = cty.UnknownVal(cty.String).RefineNotNull()
		repetition.EachValue = cty.DynamicVal
	}
	ret, diags := p.main.
		mustStackConfig(ctx, p.Addr().Stack).
		resolveExpressionReference(ctx, ref, nil, repetition)

	if _, ok := ret.(*ProviderConfig); ok {
		// We can't reference other providers from anywhere inside a provider
		// configuration block.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("The object %s is not in scope at this location.", ref.Target.String()),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
	}

	return ret, diags
}

// ExternalFunctions implements ExpressionScope.
func (p *ProviderConfig) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return p.main.ProviderFunctions(ctx, p.main.StackConfig(ctx, p.Addr().Stack))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (p *ProviderConfig) PlanTimestamp() time.Time {
	return p.main.PlanTimestamp()
}

// ExprReferenceValue implements Referenceable.
func (p *ProviderConfig) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
	// We don't say anything about the contents of a provider during the
	// static evaluation phase. We still return the type of the provider so
	// we can use it to verify type constraints, but we don't return any
	// actual values.
	if p.config.ForEach != nil {
		return cty.UnknownVal(cty.Map(p.InstRefValueType(ctx)))
	}
	return cty.UnknownVal(p.InstRefValueType(ctx))
}

var providerInstanceRefTypes = map[addrs.Provider]cty.Type{}
var providerInstanceRefTypesMu sync.Mutex

// providerInstanceRefType returns the singleton cty capsule type for a given
// provider source address, creating a new type if a particular source address
// was not requested before.
func providerInstanceRefType(sourceAddr addrs.Provider) cty.Type {
	providerInstanceRefTypesMu.Lock()
	defer providerInstanceRefTypesMu.Unlock()

	ret, ok := providerInstanceRefTypes[sourceAddr]
	if ok {
		return ret
	}
	providerInstanceRefTypes[sourceAddr] = stackconfigtypes.ProviderConfigType(sourceAddr)
	return providerInstanceRefTypes[sourceAddr]
}

func (p *ProviderConfig) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	_, diags := p.CheckProviderArgs(ctx, phase)
	return diags
}

// Validate implements Validatable.
func (p *ProviderConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	return p.checkValid(ctx, ValidatePhase)
}

// PlanChanges implements Plannable.
func (p *ProviderConfig) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, PlanPhase)
}

// tracingName implements Validatable.
func (p *ProviderConfig) tracingName() string {
	return p.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (p *ProviderConfig) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(p.providerArgs.PromiseID(), p.Addr().String())
}
