// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ConfigComponentExpressionScope is an extension to ExpressionScope that
// also provides access to the underlying configuration module that is being
// evaluated by this scope.
//
// This is typically used to share code between removed and component blocks
// which both load and execute Terraform configurations.
type ConfigComponentExpressionScope[Addr any] interface {
	ExpressionScope

	Addr() Addr
	ModuleTree(ctx context.Context) *configs.Config
	DeclRange(ctx context.Context) *hcl.Range
}

// EvalProviderTypes evaluates the provider configurations for a component,
// ensuring that all required providers are present and have the correct type.
//
// This function should be called during static evaluations of components and
// removed blocks.
func EvalProviderTypes(ctx context.Context, stack *StackConfig, providers map[addrs.LocalProviderConfig]hcl.Expression, phase EvalPhase, scope ConfigComponentExpressionScope[stackaddrs.ConfigComponent]) (addrs.Set[addrs.RootProviderConfig], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	neededProviders := requiredProviderInstances(ctx, scope)

	ret := addrs.MakeSet[addrs.RootProviderConfig]()
	for _, elem := range neededProviders.Elems {

		// sourceAddr is the addrs.RootProviderConfig that should be used to
		// set this provider in the component later.
		sourceAddr := elem.Key

		// componentAddr is the addrs.LocalProviderConfig that specifies the
		// local name and (optional) alias of the provider in the component.
		componentAddr := elem.Value.Local

		// typeAddr is the absolute address of the provider type itself.
		typeAddr := sourceAddr.Provider

		expr, exists := providers[componentAddr]
		if !exists {
			// Then this provider isn't listed in the `providers` block of this
			// component. Which is bad!
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Missing required provider configuration",
				Detail: fmt.Sprintf(
					"The root module for %s requires a provider configuration named %q for provider %q, which is not assigned in the block's \"providers\" argument.",
					scope.Addr(), componentAddr.StringCompact(), typeAddr.ForDisplay(),
				),
				Subject: scope.DeclRange(ctx),
			})
			continue
		}

		// This means we now have an expression that should be providing the
		// configuration for this required provider. We'll evaluate it now.

		result, hclDiags := EvalExprAndEvalContext(ctx, expr, phase, scope)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			continue
		}

		// Now, we received something from the expression. We need to make sure
		// it's a valid provider configuration and it's the right type of
		// provider.

		const errSummary = "Invalid provider configuration"
		if actualTy := result.Value.Type(); stackconfigtypes.IsProviderConfigType(actualTy) {
			// Then we at least got a provider reference of some kind.
			actualTypeAddr := stackconfigtypes.ProviderForProviderConfigType(actualTy)
			if actualTypeAddr != typeAddr {
				var errorDetail string

				stackName, matchingTypeExists := stack.ProviderLocalName(ctx, typeAddr)
				_, matchingNameExists := stack.ProviderForLocalName(ctx, componentAddr.LocalName)
				moduleProviderTypeExplicit := elem.Value.Explicit
				if !matchingTypeExists && !matchingNameExists {
					// Then the user just hasn't declared the target provider
					// type or name at all. We'll return a generic error message
					// asking the user to update the required_providers list.
					errorDetail = "\n\nDeclare the required provider in the stack's required_providers block, and then assign a configuration for that provider in this block's \"providers\" argument."
				} else if !matchingNameExists {
					// Then we have a type that matches, but the name doesn't.
					errorDetail = fmt.Sprintf("\n\nThis stack has a configured provider of the correct type under the name %q. Update this block's \"providers\" argument to reference this provider.", stackName)
				} else if !matchingTypeExists {
					// Then we have a name that matches, but the type doesn't.

					// If the types don't match and the names do, then maybe
					// the user hasn't properly filled in the required types
					// within the module.
					if !moduleProviderTypeExplicit {
						// Yes! The provider type within the module has been
						// implied by Terraform and not explicitly set within
						// the required_providers block. We'll suggest the user
						// to update the required_providers block of the module.
						errorDetail = fmt.Sprintf("\n\nThe module does not declare a source address for %q in its required_providers block, so Terraform assumed %q for backward-compatibility with older versions of Terraform", componentAddr.LocalName, elem.Key.Provider.ForDisplay())
					}

					// Otherwise the user has explicitly set the provider type
					// within the module, but it doesn't match the provider type
					// within the stack configuration. The generic error message
					// should be sufficient.
				}

				// But, unfortunately, the underlying types of the providers
				// do not match up.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail: fmt.Sprintf(
						"The provider configuration slot %q requires a configuration for provider %q, not for provider %q.%s",
						componentAddr.StringCompact(), typeAddr, actualTypeAddr, errorDetail,
					),
					Subject: result.Expression.Range().Ptr(),
				})
				continue
			}
		} else if result.Value == cty.DynamicVal {
			// Then we don't know the concrete type of this reference at this
			// time, so we'll just have to accept it. This is somewhat expected
			// during the validation phase, and even during the planning phase
			// if we have deferred attributes. We'll get an error later (ie.
			// during the plan phase) if the type doesn't match up then.
		} else {
			// We got something that isn't a provider reference at all.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"The provider configuration slot %s requires a configuration for provider %q.",
					componentAddr.StringCompact(), typeAddr,
				),
				Subject: result.Expression.Range().Ptr(),
			})
			continue
		}

		// If we made it here, the types all matched up so we've done everything
		// we can. component_instance.go will do additional checks to make sure
		// the result is known and not null when it comes time to actually
		// check the plan.

		ret.Add(sourceAddr)
	}

	return ret, diags
}

// EvalProviderValues evaluates the provider configuration for a component, and
// returns the provider configuration instances that should be used in the
// component.
//
// This function should be called during dynamic evaluations of components and
// removed blocks.
func EvalProviderValues(ctx context.Context, main *Main, providers map[addrs.LocalProviderConfig]hcl.Expression, phase EvalPhase, scope ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance]) (map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, map[addrs.RootProviderConfig]addrs.Provider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	knownProviders := make(map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance)
	unknownProviders := make(map[addrs.RootProviderConfig]addrs.Provider)

	neededProviders := requiredProviderInstances(ctx, scope)

	for _, elem := range neededProviders.Elems {
		// sourceAddr is the addrs.RootProviderConfig that should be used to
		// set this provider in the component later.
		sourceAddr := elem.Key

		// componentAddr is the addrs.LocalProviderConfig that specifies the
		// local name and (optional) alias of the provider in the component.
		componentAddr := elem.Value.Local

		// We validated the config providers during the static analysis, so we
		// know this expression exists and resolves to the correct type.
		expr := providers[componentAddr]

		inst, unknown, instDiags := evalProviderValue(ctx, sourceAddr, componentAddr, expr, phase, scope)
		diags = diags.Append(instDiags)
		if instDiags.HasErrors() {
			continue
		}

		if unknown {
			unknownProviders[sourceAddr] = sourceAddr.Provider
			continue
		}

		knownProviders[sourceAddr] = inst
	}

	// Second, we want to iterate through the providers that are required by
	// the state and not required by the configuration. Unfortunately, we don't
	// currently store enough information to be able to retrieve the original
	// provider directly from the state. We only store the provider type and
	// alias of the original provider. Stacks can have multiple instances of the
	// same provider type, local name, and alias. This means we need the user to
	// still provide an entry for this provider in the declConfigs.
	// TODO: There's another TODO in the state package that suggests we should
	//   store the additional information we need. Once this is fixed we can
	//   come and tidy this up as well.

	stackConfig := main.StackConfig(ctx, scope.Addr().Stack.ConfigAddr())
	moduleTree := scope.ModuleTree(ctx)

	// We'll search through the declConfigs to find any keys that match the
	// type and alias of a any provider needed by the state. This is backwards
	// when compared to how we resolved the configProviders. But we don't have
	// the information we need to do it the other way around.

	previousProviders := main.PreviousProviderInstances(scope.Addr(), phase)
	for localProviderAddr, expr := range providers {
		provider := moduleTree.ProviderForConfigAddr(localProviderAddr)

		sourceAddr := addrs.RootProviderConfig{
			Provider: provider,
			Alias:    localProviderAddr.Alias,
		}

		if _, exists := knownProviders[sourceAddr]; exists || !previousProviders.Has(sourceAddr) {
			// Then this declConfig either matches a configProvider and we've
			// already processed it, or it matches a provider that isn't
			// required by the config or the state. In the first case, this is
			// fine we have matched the right provider already. In the second
			// case, we could raise a warning or something but it's not a big
			// deal so we can ignore it.
			continue
		}

		// Otherwise, this is a declConfig for a provider that is not in the
		// configProviders and is in the previousProviders. So, we should
		// process it.

		inst, unknown, instDiags := evalProviderValue(ctx, sourceAddr, localProviderAddr, expr, phase, scope)
		diags = diags.Append(instDiags)
		if instDiags.HasErrors() {
			continue
		}

		if unknown {
			unknownProviders[sourceAddr] = provider
		} else {
			knownProviders[sourceAddr] = inst
		}

		if _, ok := stackConfig.ProviderLocalName(ctx, provider); !ok {
			// Even though we have an entry for this provider in the declConfigs
			// doesn't mean we have an entry for this in our required providers.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Block requires undeclared provider",
				Detail: fmt.Sprintf(
					"The root module for %s has resources in state that require a configuration for provider %q, which isn't declared as a dependency of this stack configuration.\n\nDeclare this provider in the stack's required_providers block, and then assign a configuration for that provider in this block's \"providers\" argument.",
					scope.Addr(), provider.ForDisplay(),
				),
				Subject: scope.DeclRange(ctx),
			})
		}
	}

	// Finally, let's check that we have a provider configuration for every
	// provider needed by the state.

	for _, previousProvider := range previousProviders {
		if _, ok := knownProviders[previousProvider]; ok {
			// Then we have a provider for this, so great!
			continue
		}

		// If we get here, then we didn't find an entry for this provider in
		// the declConfigs. This is an error because we need to have an entry
		// for every provider that we have in the state.

		// localAddr helps with the error message.
		localAddr := addrs.LocalProviderConfig{
			LocalName: moduleTree.Module.LocalNameForProvider(previousProvider.Provider),
			Alias:     previousProvider.Alias,
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing required provider configuration",
			Detail: fmt.Sprintf(
				"The root module for %s has resources in state that require a provider configuration named %q for provider %q, which is not assigned in the block's \"providers\" argument.",
				scope.Addr(), localAddr.StringCompact(), previousProvider.Provider.ForDisplay(),
			),
			Subject: scope.DeclRange(ctx),
		})
	}

	return knownProviders, unknownProviders, diags
}

func evalProviderValue(ctx context.Context, sourceAddr addrs.RootProviderConfig, componentAddr addrs.LocalProviderConfig, expr hcl.Expression, phase EvalPhase, scope ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance]) (stackaddrs.AbsProviderConfigInstance, bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret stackaddrs.AbsProviderConfigInstance

	result, hclDiags := EvalExprAndEvalContext(ctx, expr, phase, scope)
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return ret, false, diags
	}
	v := result.Value

	// The first set of checks can perform a redundant check in some cases. For
	// providers required by the configuration the type validation should have
	// been performed by the static analysis. However, we'll repeat the checks
	// here to also catch the case where providers are required by the existing
	// state but are not defined in the configuration. This isn't checked by
	// the static analysis.
	const errSummary = "Invalid provider configuration"
	if actualTy := result.Value.Type(); stackconfigtypes.IsProviderConfigType(actualTy) {
		// Then we at least got a provider reference of some kind.
		actualTypeAddr := stackconfigtypes.ProviderForProviderConfigType(actualTy)
		if actualTypeAddr != sourceAddr.Provider {
			// But, unfortunately, the underlying types of the providers
			// do not match up.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errSummary,
				Detail: fmt.Sprintf(
					"The provider configuration slot %s requires a configuration for provider %q, not for provider %q.",
					componentAddr.StringCompact(), sourceAddr.Provider, actualTypeAddr,
				),
				Subject: result.Expression.Range().Ptr(),
			})
			return ret, false, diags
		}
	} else {
		// We got something that isn't a provider reference at all.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail: fmt.Sprintf(
				"The provider configuration slot %s requires a configuration for provider %q.",
				componentAddr.StringCompact(), sourceAddr.Provider,
			),
			Subject: result.Expression.Range().Ptr(),
		})
		return ret, false, diags
	}

	// Now, we differ from the static analysis in that we should have
	// returned a concrete value while we may have got unknown during the
	// static analysis.
	if v.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  errSummary,
			Detail: fmt.Sprintf(
				"The provider configuration slot %s is required, but this definition returned null.",
				componentAddr.StringCompact(),
			),
			Subject: result.Expression.Range().Ptr(),
		})
		return ret, false, diags
	}
	if !v.IsKnown() {
		return ret, true, diags
	}

	// If it's of the correct type, known, and not null then we should
	// be able to retrieve a specific provider instance address that
	// this value refers to.
	return stackconfigtypes.ProviderInstanceForValue(v), false, diags
}

// requiredProviderInstances returns a description of all of the provider
// instance slots ("provider configurations" in main Terraform language
// terminology) that are either explicitly declared or implied by the
// root module of the scope's module tree.
//
// In the returned map the keys describe provider configurations from
// the perspective of an object inside the root module, and so the LocalName
// field values are an implementation detail that must not be exposed into
// the calling stack and are included here only so that we can potentially
// return error messages referring to declarations inside the module.
//
// If any modules in the component's root module tree are invalid then this
// result could under-promise or over-promise depending on the kind of
// invalidity.
func requiredProviderInstances[Addr any](ctx context.Context, scope ConfigComponentExpressionScope[Addr]) addrs.Map[addrs.RootProviderConfig, configs.RequiredProviderConfig] {
	moduleTree := scope.ModuleTree(ctx)
	if moduleTree == nil || moduleTree.Root == nil {
		return addrs.MakeMap[addrs.RootProviderConfig, configs.RequiredProviderConfig]()
	}
	return moduleTree.Root.EffectiveRequiredProviderConfigs()
}

// neededProviderSchemas returns the provider schemas for all of the providers
// required by the configuration of the given component, along with any
// diagnostics that were encountered while fetching those schemas.
func neededProviderSchemas[Addr any](ctx context.Context, main *Main, phase EvalPhase, scope ConfigComponentExpressionScope[Addr]) (map[addrs.Provider]providers.ProviderSchema, tfdiags.Diagnostics, bool) {
	var diags tfdiags.Diagnostics
	skipFutherValidation := false

	config := scope.ModuleTree(ctx)

	providerSchemas := make(map[addrs.Provider]providers.ProviderSchema)
	for _, sourceAddr := range config.ProviderTypes() {
		pTy := main.ProviderType(ctx, sourceAddr)
		if pTy == nil {
			continue // not our job to report a missing provider
		}

		// If this phase has a dependency lockfile, check if the provider is in it.
		depLocks := main.DependencyLocks(phase)
		if depLocks != nil {
			// Check if the provider is in the lockfile,
			// if it is not we can not read the provider schema
			providerLockfileDiags := CheckProviderInLockfile(*depLocks, pTy, scope.DeclRange(ctx))

			// We report these diagnostics in a different place
			if providerLockfileDiags.HasErrors() {
				skipFutherValidation = true
				continue
			}
		}

		schema, err := pTy.Schema(ctx)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider initialization error",
				Detail:   fmt.Sprintf("Failed to fetch the provider schema for %s: %s.", sourceAddr, err),
				Subject:  scope.DeclRange(ctx),
			})
			continue
		}
		providerSchemas[sourceAddr] = schema
	}
	return providerSchemas, diags, skipFutherValidation
}

// unconfiguredProviderClients returns the provider clients for the providers
// required by the configuration of the given component, along with any
// diagnostics that were encountered while fetching those clients.
func unconfiguredProviderClients(ctx context.Context, main *Main, ps addrs.Set[addrs.RootProviderConfig]) (map[addrs.RootProviderConfig]providers.Interface, bool) {
	insts := make(map[addrs.RootProviderConfig]providers.Interface)
	valid := true

	for _, provider := range ps {
		pTy := main.ProviderType(ctx, provider.Provider)
		if pTy == nil {
			valid = false
			continue // not our job to report a missing provider
		}

		// We don't need to configure the client for validate functionality.
		inst, err := pTy.UnconfiguredClient()
		if err != nil {
			valid = false
			continue
		}
		insts[provider] = inst
	}

	return insts, valid
}

// configuredProviderClients return s
func configuredProviderClients(ctx context.Context, main *Main, known map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, unknown map[addrs.RootProviderConfig]addrs.Provider, phase EvalPhase) map[addrs.RootProviderConfig]providers.Interface {
	providerInsts := make(map[addrs.RootProviderConfig]providers.Interface)
	for calleeAddr, callerAddr := range known {
		providerInstStack := main.Stack(ctx, callerAddr.Stack, phase)
		if providerInstStack == nil {
			continue
		}
		provider := providerInstStack.Provider(ctx, callerAddr.Item.ProviderConfig)
		if provider == nil {
			continue
		}
		insts, unknown := provider.Instances(ctx, phase)
		if unknown {
			// an unknown provider should have been added to the unknown
			// providers and not the known providers, so this is a bug if we get
			// here.
			panic(fmt.Errorf("provider %s returned unknown instances", callerAddr))
		}
		if insts == nil {
			continue
		}
		inst, exists := insts[callerAddr.Item.Key]
		if !exists {
			continue
		}
		providerInsts[calleeAddr] = inst.Client(ctx, phase)
	}
	for calleeAddr, provider := range unknown {
		pTy := main.ProviderType(ctx, provider)
		client, err := pTy.UnconfiguredClient()
		if err != nil {
			continue
		}
		providerInsts[calleeAddr] = stubs.UnknownProvider(client)
	}
	return providerInsts
}
