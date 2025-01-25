// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
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

var _ ExpressionScope = (*ProviderInstance)(nil)

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

			unconfClient, err := providerType.UnconfiguredClient()
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
				// We should have triggered and returned a stub.UnknownProvider
				// in this case, so there's a bug somewhere in Terraform if
				// this happens.
				panic("provider instance with unknown for_each key")
			}
			if p.repetition.CountIndex != cty.NilVal && !p.repetition.CountIndex.IsKnown() {
				// Providers don't even support the count index argument, so
				// something crazy is happening if we get here.
				panic("provider instance with unknown count index")
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
				return stubs.ErroredProvider(), diags
			}

			// If the context we recieved gets cancelled then we want providers
			// to try to cancel any operations they have in progress, so we'll
			// watch for that in a separate goroutine. This extra context
			// is here just so we can avoid leaking this goroutine if the
			// parent doesn't get cancelled.
			providerCtx, localCancel := context.WithCancel(ctx)
			go func() {
				<-providerCtx.Done()
				if ctx.Err() == context.Canceled {
					// Not all providers respond to this, but some will quickly
					// abort operations currently in progress and return a
					// cancellation error, thus allowing us to halt more quickly
					// when interrupted.
					client.Stop()
				}
			}()

			// If this provider is implemented as a separate plugin then we
			// must terminate its child process once evaluation is complete.
			p.main.RegisterCleanup(func(ctx context.Context) tfdiags.Diagnostics {
				var diags tfdiags.Diagnostics
				localCancel() // make sure our cancel-monitoring goroutine terminates
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
			unmarkedArgs, _ := p.ProviderArgs(ctx, phase).UnmarkDeep()
			if unmarkedArgs == cty.NilVal {
				// Then we had an error previously, so we'll rely on that error
				// being exposed elsewhere.
				return stubs.ErroredProvider(), diags
			}

			resp := client.ConfigureProvider(providers.ConfigureProviderRequest{
				TerraformVersion:   version.SemVer.String(),
				Config:             unmarkedArgs,
				ClientCapabilities: ClientCapabilities(),
			})
			diags = diags.Append(resp.Diagnostics)
			if resp.Diagnostics.HasErrors() {
				// If the provider didn't configure successfully then it won't
				// meet the expectations of our callers and so we'll return a
				// stub instead. (The real provider stays running until it
				// gets cleaned up by the cleanup function above, despite being
				// inaccessible to the caller.)
				return stubs.ErroredProvider(), diags
			}

			return unconfigurableProvider{
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

// ExternalFunctions implements ExpressionScope.
func (p *ProviderInstance) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return p.main.ProviderFunctions(ctx, p.main.StackConfig(ctx, p.Addr().Stack.ConfigAddr()))
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (p *ProviderInstance) PlanTimestamp() time.Time {
	return p.main.PlanTimestamp()
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

// CheckApply implements Applyable.
func (p *ProviderInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	return nil, p.checkValid(ctx, ApplyPhase)
}

// tracingName implements Plannable.
func (p *ProviderInstance) tracingName() string {
	return p.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (p *ProviderInstance) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	name := p.Addr().String()
	clientName := name + " plugin client"
	p.providerArgs.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[cty.Value]]) {
		cb(o.PromiseID(), name)
	})
	p.client.Each(func(ep EvalPhase, o *promising.Once[withDiagnostics[providers.Interface]]) {
		cb(o.PromiseID(), clientName)
	})
}
