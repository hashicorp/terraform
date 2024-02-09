// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
func (p *ProviderConfig) ProviderArgs(ctx context.Context) cty.Value {
	v, _ := p.CheckProviderArgs(ctx)
	return v
}

func (p *ProviderConfig) CheckProviderArgs(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, &p.providerArgs, p.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			providerType := p.ProviderType(ctx)
			decl := p.Declaration(ctx)
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

			client, err := providerType.UnconfiguredClient(ctx)
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
			defer client.Close()

			configVal, moreDiags := EvalBody(ctx, decl.Config, spec, ValidatePhase, p)
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
	return p.main.
		mustStackConfig(ctx, p.Addr().Stack).
		resolveExpressionReference(ctx, ref, repetition, nil)
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

// Validate implements Validatable.
func (p *ProviderConfig) Validate(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// TODO: Actually validate the configuration against the schema.
	// Currently we're doing that only during the plan phase, but
	// it would be better to catch statically-detectable problems
	// earlier and only once per provider block, rather than repeatedly
	// for each instance of a provider.

	return diags
}

// tracingName implements Validatable.
func (p *ProviderConfig) tracingName() string {
	return p.Addr().String()
}
