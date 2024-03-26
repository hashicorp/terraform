// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

// ProviderInstance represents one instance of a provider.
//
// A provider configuration block with the for_each argument appears as a
// single [ProviderConfig], then one [Provider] for each stack config instance
// the provider belongs to, and then one [ProviderInstance] for each
// element of for_each for each [Provider].
type ProviderInstance struct {
	provider   *Provider
	key        addrs.InstanceKey
	repetition instances.RepetitionData

	main *Main

	providerArgs perEvalPhase[promising.Once[withDiagnostics[cty.Value]]]
	client       perEvalPhase[promising.Once[withDiagnostics[providers.Interface]]]
}

func newProviderInstance(provider *Provider, key addrs.InstanceKey, repetition instances.RepetitionData) *ProviderInstance {
	return &ProviderInstance{
		provider:   provider,
		key:        key,
		main:       provider.main,
		repetition: repetition,
	}
}

func (p *ProviderInstance) Addr() stackaddrs.AbsProviderConfigInstance {
	providerAddr := p.provider.Addr()
	return stackaddrs.AbsProviderConfigInstance{
		Stack: providerAddr.Stack,
		Item: stackaddrs.ProviderConfigInstance{
			ProviderConfig: providerAddr.Item,
			Key:            p.key,
		},
	}
}

func (p *ProviderInstance) RepetitionData() instances.RepetitionData {
	return p.repetition
}

func (p *ProviderInstance) ProviderType(ctx context.Context) *ProviderType {
	return p.main.ProviderType(ctx, p.Addr().Item.ProviderConfig.Provider)
}

func (p *ProviderInstance) ProviderArgsDecoderSpec(ctx context.Context) (hcldec.Spec, error) {
	return p.provider.Config(ctx).ProviderArgsDecoderSpec(ctx)
}

// ProviderArgs returns an object value representing the provider configuration
// for this instance, or an unknown value of the correct type if the
// configuration is invalid. If a provider error occurs, it returns
// [cty.DynamicVal].
func (p *ProviderInstance) ProviderArgs(ctx context.Context, phase EvalPhase) cty.Value {
	v, _ := p.CheckProviderArgs(ctx, phase)
	return v
}

func (p *ProviderInstance) CheckProviderArgs(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, p.providerArgs.For(phase), p.main,
		func(ctx context.Context) (cty.Value, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			providerType := p.ProviderType(ctx)
			decl := p.provider.Declaration(ctx)
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

			var configVal cty.Value
			var moreDiags tfdiags.Diagnostics
			configBody := decl.Config
			if configBody == nil {
				configBody = hcl.EmptyBody()
			}
			configVal, moreDiags = EvalBody(ctx, configBody, spec, phase, p)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return cty.UnknownVal(hcldec.ImpliedType(spec)), diags
			}

			unconfClient, err := providerType.UnconfiguredClient(ctx)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to start provider plugin",
					Detail: fmt.Sprintf(
						"Error while instantiating %q: %s.",
						providerType.Addr(), err,
					),
					Subject: decl.DeclRange.ToHCL().Ptr(),
				})
				return cty.DynamicVal, diags
			}
			defer unconfClient.Close()
			// We unmark the config before making the RPC call, but will still
			// return the original possibly-marked config if successful.
			unmarkedConfigVal, _ := configVal.UnmarkDeep()
			validateResp := unconfClient.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
				Config: unmarkedConfigVal,
			})
			diags = diags.Append(validateResp.Diagnostics)
			if validateResp.Diagnostics.HasErrors() {
				return cty.DynamicVal, diags
			}

			return configVal, diags
		},
	)
}

// Client returns a client object for the provider instance, already configured
// per the provider configuration arguments and ready to use.
//
// If the configured arguments are invalid then this might return a stub
// provider client that implements all methods either as silent no-ops or as
// returning error diagnostics, so callers can just treat the returned client
// as always valid.
//
// Callers must call Close on the returned client once they have finished using
// the client.
func (p *ProviderInstance) Client(ctx context.Context, phase EvalPhase) providers.Interface {
	ret, _ := p.CheckClient(ctx, phase)
	return ret
}

func (p *ProviderInstance) CheckClient(ctx context.Context, phase EvalPhase) (providers.Interface, tfdiags.Diagnostics) {
	return doOnceWithDiags(
		ctx, p.client.For(phase), p.main,
		func(ctx context.Context) (providers.Interface, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			if p.repetition.EachKey != cty.NilVal && !p.repetition.EachKey.IsKnown() {
				// If we're a placeholder standing in for all instances of
				// a provider block whose for_each is unknown then we
				// can't configure.
				return stubConfiguredProvider{unknown: true}, diags
			}
			if p.repetition.CountIndex != cty.NilVal && !p.repetition.CountIndex.IsKnown() {
				// If we're a placeholder standing in for all instances of
				// a provider block whose count is unknown then we
				// can't configure.
				return stubConfiguredProvider{unknown: true}, diags
			}

			args := p.ProviderArgs(ctx, phase)
			if !args.IsKnown() {
				// If we don't know the provider configuration at all then
				// we'll just immediately return a stub client, since
				// no provider can accept a wholly-unknown configuration.
				// (Known objects with unknown attribute values inside are
				// okay to try and so don't return immediately here.)
				return stubConfiguredProvider{unknown: true}, diags
			}

			providerType := p.ProviderType(ctx)
			decl := p.provider.Declaration(ctx)

			client, err := p.main.ProviderFactories().NewUnconfiguredClient(providerType.Addr())
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to start provider plugin",
					Detail: fmt.Sprintf(
						"Could not create an instance of %s for %s: %s.",
						providerType.Addr(), p.Addr(), err,
					),
					Subject: decl.DeclRange.ToHCL().Ptr(),
				})
				return stubConfiguredProvider{unknown: false}, diags
			}

			// If this provider is implemented as a separate plugin then we
			// must terminate its child process once evaluation is complete.
			p.main.RegisterCleanup(func(ctx context.Context) tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				err := client.Close()
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to terminate provider plugin",
						Detail: fmt.Sprintf(
							"Error closing the instance of %s for %s: %s.",
							providerType.Addr(), p.Addr(), err,
						),
						Subject: decl.DeclRange.ToHCL().Ptr(),
					})
				}
				return diags
			})

			// TODO: Some providers will malfunction if the caller doesn't
			// fetch their schema at least once before use. That's not something
			// the provider protocol promises but it's an implementation
			// detail that a certain generation of providers relied on
			// nonetheless. We'll probably need to check whether the provider
			// supports the "I don't need you to fetch my schema" capability
			// and, if not, do a redundant re-fetch of the schema in here
			// somewhere. Refer to the corresponding behavior in the
			// "terraform" package for non-Stacks usage and try to mimick
			// what it does in as lightweight a way as possible.

			// We unmark the config before making the RPC call, as marks cannot
			// be serialized.
			unmarkedArgs, _ := args.UnmarkDeep()
			resp := client.ConfigureProvider(providers.ConfigureProviderRequest{
				TerraformVersion: version.SemVer.String(),
				Config:           unmarkedArgs,
			})
			diags = diags.Append(resp.Diagnostics)
			if resp.Diagnostics.HasErrors() {
				// If the provider didn't configure successfully then it won't
				// meet the expectations of our callers and so we'll return a
				// stub instead. (The real provider stays running until it
				// gets cleaned up by the cleanup function above, despite being
				// inaccessible to the caller.)
				return stubConfiguredProvider{unknown: false}, diags
			}

			return providerClose{
				close: func() error {
					// We just totally ignore close for configured providers,
					// because we'll deal with them in the cleanup phase instead.
					return nil
				},
				Interface: client,
			}, diags
		},
	)
}

// ResolveExpressionReference implements ExpressionScope for expressions other
// than the for_each argument inside a provider block, which get evaluated
// once per provider instance.
func (p *ProviderInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := p.provider.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, p.repetition)
}

func (p *ProviderInstance) checkValid(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	_, moreDiags := p.CheckProviderArgs(ctx, phase)
	diags = diags.Append(moreDiags)

	// NOTE: CheckClient starts and configures the provider as a side-effect.
	// If this is a plugin-based provider then the plugin process will stay
	// running for the remainder of the specified evaluation phase.
	_, moreDiags = p.CheckClient(ctx, phase)
	diags = diags.Append(moreDiags)

	return diags
}

// PlanChanges implements Plannable.
func (p *ProviderInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, PlanPhase)
}

// RequiredComponents implements Applyable
func (p *ProviderInstance) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return p.provider.RequiredComponents(ctx)
}

// CheckApply implements Applyable.
func (p *ProviderInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, ApplyPhase)
}

// tracingName implements Plannable.
func (p *ProviderInstance) tracingName() string {
	return p.Addr().String()
}

// stubConfiguredProvider is a placeholder provider used when ConfigureProvider
// on a real provider fails, so that callers can still receieve a usable client
// that will just produce placeholder values from its operations.
//
// This is essentially the cty.DynamicVal equivalent for providers.Interface,
// allowing us to follow our usual pattern that only one return path carries
// diagnostics up to the caller and all other codepaths just do their best
// to unwind with placeholder values. It's intended only for use in situations
// that would expect an already-configured provider, so it's incorrect to call
// [ConfigureProvider] on a value of this type.
//
// Some methods of this type explicitly return errors saying that the provider
// configuration was invalid, while others just optimistically do nothing at
// all. The general rule is that anything that would for a normal provider
// be expected to perform externally-visible side effects must return an error
// to be explicit that those side effects did not occur, but we can silently
// skip anything that is a Terraform-only detail.
//
// As usual with provider calls, the returned diagnostics must be annotated
// using [tfdiags.Diagnostics.InConfigBody] with the relevant configuration body
// so that they can be attributed to the appropriate configuration element.
type stubConfiguredProvider struct {
	// If unknown is true then the implementation will assume it's acting
	// as a placeholder for a provider whose configuration isn't yet
	// sufficiently known to be properly instantiated, which means that
	// plan-time operations will return totally-unknown values.
	// Otherwise any operation that is supposed to perform a side-effect
	// will fail with an error saying that the provider configuration
	// is invalid.
	unknown bool
}

var _ providers.Interface = stubConfiguredProvider{}

// ApplyResourceChange implements providers.Interface.
func (stubConfiguredProvider) ApplyResourceChange(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot apply changes because this resource's associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ApplyResourceChangeResponse{
		Diagnostics: diags,
	}
}

func (stubConfiguredProvider) CallFunction(providers.CallFunctionRequest) providers.CallFunctionResponse {
	panic("can't call functions on the stub provider")
}

// Close implements providers.Interface.
func (stubConfiguredProvider) Close() error {
	return nil
}

// ConfigureProvider implements providers.Interface.
func (stubConfiguredProvider) ConfigureProvider(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
	// This provider is used only in situations where ConfigureProvider on
	// a real provider fails and the recipient was expecting a configured
	// provider, so it doesn't make sense to configure it.
	panic("can't configure the stub provider")
}

// GetProviderSchema implements providers.Interface.
func (stubConfiguredProvider) GetProviderSchema() providers.GetProviderSchemaResponse {
	return providers.GetProviderSchemaResponse{}
}

// ImportResourceState implements providers.Interface.
func (p stubConfiguredProvider) ImportResourceState(req providers.ImportResourceStateRequest) providers.ImportResourceStateResponse {
	var diags tfdiags.Diagnostics
	if p.unknown {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is deferred",
			"Cannot import an existing object into this resource because its associated provider configuration is deferred to a later operation due to unknown expansion.",
			nil, // nil attribute path means the overall configuration block
		))
	} else {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is invalid",
			"Cannot import an existing object into this resource because its associated provider configuration is invalid.",
			nil, // nil attribute path means the overall configuration block
		))
	}
	return providers.ImportResourceStateResponse{
		Diagnostics: diags,
	}
}

// MoveResourceState implements providers.Interface.
func (p stubConfiguredProvider) MoveResourceState(req providers.MoveResourceStateRequest) providers.MoveResourceStateResponse {
	var diags tfdiags.Diagnostics
	if p.unknown {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is deferred",
			"Cannot move an existing object to this resource because its associated provider configuration is deferred to a later operation due to unknown expansion.",
			nil, // nil attribute path means the overall configuration block
		))
	} else {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Provider configuration is invalid",
			"Cannot move an existing object to this resource because its associated provider configuration is invalid.",
			nil, // nil attribute path means the overall configuration block
		))
	}
	return providers.MoveResourceStateResponse{
		Diagnostics: diags,
	}
}

// PlanResourceChange implements providers.Interface.
func (p stubConfiguredProvider) PlanResourceChange(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
	if p.unknown {
		return providers.PlanResourceChangeResponse{
			PlannedState: cty.DynamicVal,
		}
	}
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot plan changes for this resource because its associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.PlanResourceChangeResponse{
		Diagnostics: diags,
	}
}

// ReadDataSource implements providers.Interface.
func (p stubConfiguredProvider) ReadDataSource(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
	if p.unknown {
		return providers.ReadDataSourceResponse{
			State: cty.DynamicVal,
		}
	}
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot read from this data source because its associated provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.ReadDataSourceResponse{
		Diagnostics: diags,
	}
}

// ReadResource implements providers.Interface.
func (stubConfiguredProvider) ReadResource(req providers.ReadResourceRequest) providers.ReadResourceResponse {
	// For this one we'll just optimistically assume that the remote object
	// hasn't changed. In many cases we'll fail calling PlanResourceChange
	// right afterwards anyway, and even if not we'll get another opportunity
	// to refresh on a future run once the provider configuration is fixed.
	return providers.ReadResourceResponse{
		NewState: req.PriorState,
		Private:  req.Private,
	}
}

// Stop implements providers.Interface.
func (stubConfiguredProvider) Stop() error {
	// This stub provider never actually does any real work, so there's nothing
	// for us to stop.
	return nil
}

// UpgradeResourceState implements providers.Interface.
func (p stubConfiguredProvider) UpgradeResourceState(req providers.UpgradeResourceStateRequest) providers.UpgradeResourceStateResponse {
	if p.unknown {
		return providers.UpgradeResourceStateResponse{
			UpgradedState: cty.DynamicVal,
		}
	}

	// Ideally we'd just skip this altogether and echo back what the caller
	// provided, but the request is in a different serialization format than
	// the response and so only the real provider can deal with this one.
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		"Provider configuration is invalid",
		"Cannot decode the prior state for this resource instance because its provider configuration is invalid.",
		nil, // nil attribute path means the overall configuration block
	))
	return providers.UpgradeResourceStateResponse{
		Diagnostics: diags,
	}
}

// ValidateDataResourceConfig implements providers.Interface.
func (stubConfiguredProvider) ValidateDataResourceConfig(req providers.ValidateDataResourceConfigRequest) providers.ValidateDataResourceConfigResponse {
	// We'll just optimistically assume the configuration is valid, so that
	// we can progress to planning and return an error there instead.
	return providers.ValidateDataResourceConfigResponse{
		Diagnostics: nil,
	}
}

// ValidateProviderConfig implements providers.Interface.
func (stubConfiguredProvider) ValidateProviderConfig(req providers.ValidateProviderConfigRequest) providers.ValidateProviderConfigResponse {
	// It doesn't make sense to call this one on stubProvider, because
	// we only use stubProvider for situations where ConfigureProvider failed
	// on a real provider and we should already have called
	// ValidateProviderConfig on that provider by then anyway.
	return providers.ValidateProviderConfigResponse{
		PreparedConfig: req.Config,
		Diagnostics:    nil,
	}
}

// ValidateResourceConfig implements providers.Interface.
func (stubConfiguredProvider) ValidateResourceConfig(req providers.ValidateResourceConfigRequest) providers.ValidateResourceConfigResponse {
	// We'll just optimistically assume the configuration is valid, so that
	// we can progress to reading and return an error there instead.
	return providers.ValidateResourceConfigResponse{
		Diagnostics: nil,
	}
}
