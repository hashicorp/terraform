// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/stackconfigtypes"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ComponentInstance struct {
	call *Component
	key  addrs.InstanceKey

	main *Main

	repetition instances.RepetitionData

	moduleTreePlan promising.Once[withDiagnostics[*plans.Plan]]
}

var _ Plannable = (*ComponentInstance)(nil)
var _ ExpressionScope = (*ComponentInstance)(nil)

func newComponentInstance(call *Component, key addrs.InstanceKey, repetition instances.RepetitionData) *ComponentInstance {
	return &ComponentInstance{
		call:       call,
		key:        key,
		main:       call.main,
		repetition: repetition,
	}
}

func (c *ComponentInstance) Addr() stackaddrs.AbsComponentInstance {
	callAddr := c.call.Addr()
	stackAddr := callAddr.Stack
	return stackaddrs.AbsComponentInstance{
		Stack: stackAddr,
		Item: stackaddrs.ComponentInstance{
			Component: callAddr.Item,
			Key:       c.key,
		},
	}
}

func (c *ComponentInstance) RepetitionData() instances.RepetitionData {
	return c.repetition
}

func (c *ComponentInstance) InputVariableValues(ctx context.Context, phase EvalPhase) cty.Value {
	ret, _ := c.CheckInputVariableValues(ctx, phase)
	return ret
}

func (c *ComponentInstance) CheckInputVariableValues(ctx context.Context, phase EvalPhase) (cty.Value, tfdiags.Diagnostics) {
	config := c.call.Config(ctx)
	wantTy, defs := config.InputsType(ctx)
	decl := c.call.Declaration(ctx)
	varDecls := config.RootModuleVariableDecls(ctx)

	if wantTy == cty.NilType {
		// Suggests that the target module is invalid in some way, so we'll
		// just report that we don't know the input variable values and trust
		// that the module's problems will be reported by some other return
		// path.
		return cty.DynamicVal, nil
	}

	// We actually checked the errors statically already, so we only care about
	// the value here.
	return EvalComponentInputVariables(ctx, varDecls, wantTy, defs, decl, phase, c)
}

// inputValuesForModulesRuntime adapts the result of
// [ComponentInstance.InputVariableValues] to the representation that the
// main Terraform modules runtime expects.
//
// The second argument (expectedValues) is the value that the apply operation
// expects to see for the input variables, which is typically the input
// values from the plan.
//
// During the planning phase, the expectedValues should be nil, as they will
// only be checked during the apply phase.
func (c *ComponentInstance) inputValuesForModulesRuntime(ctx context.Context, previousValues map[string]plans.DynamicValue, phase EvalPhase) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	valsObj := c.InputVariableValues(ctx, phase)
	if valsObj == cty.NilVal {
		return nil, diags
	}

	// valsObj might be an unknown value during the planning phase, in which
	// case we'll return an InputValues with all of the expected variables
	// defined as unknown values of their expected type constraints. To
	// achieve that, we'll do our work with the configuration's object type
	// constraint instead of with the value we've been given directly.
	wantTy, _ := c.call.Config(ctx).InputsType(ctx)
	if wantTy == cty.NilType {
		// The configuration is too invalid for us to know what type we're
		// expecting, so we'll just bail.
		return nil, diags
	}
	wantAttrs := wantTy.AttributeTypes()
	ret := make(terraform.InputValues, len(wantAttrs))
	for name, aty := range wantAttrs {
		v := valsObj.GetAttr(name)
		if !v.IsKnown() {
			// We'll ensure that it has the expected type even if
			// InputVariableValues didn't know what types to use.
			v = cty.UnknownVal(aty)
		}
		ret[name] = &terraform.InputValue{
			Value:      v,
			SourceType: terraform.ValueFromCaller,
		}

		// While we're here, we'll just add a diagnostic if the value has
		// somehow changed between the planning and apply phases. All of these
		// diagnostics acknowledge that the root cause here is a bug in
		// Terraform.
		if phase == ApplyPhase {
			raw, ok := previousValues[name]
			if !ok {
				// This shouldn't happen because we should have a value for
				// every input variable that we have a value for in the plan.
				// TODO: Support for ephemeral values is incoming, once that
				//   is implemented it will be possible for there to be a
				//   different set of input variables between the plan and the
				//   apply phase.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing input variable value",
					Detail: fmt.Sprintf(
						"The input variable %q is required but was not set in the plan for %s. This is a bug in Terraform - please report it.",
						name, c.Addr(),
					),
					Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
				})
				continue
			}

			plannedValue, err := raw.Decode(cty.DynamicPseudoType)
			if err != nil {
				// Then something has gone wrong when decoding the value.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid planned input variable value",
					Detail:   fmt.Sprintf("Failed to decode the planned value for input variable %q: %s. This is a bug in Terraform - please report it.", name, err),
					Subject:  c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
				})
				continue
			}

			if equals, _ := plannedValue.Equals(v).Unmark(); !equals.IsKnown() {
				// We unmark the value as we don't care about the actual value,
				// only whether it was equal or not.
				//
				// An unknown equals value means that the value was unknown
				// during the planning stage so we'll just accept the apply
				// value and not raise any diagnostics.
			} else if !equals.True() {
				// Then the value has changed between the planning and apply
				// phases. This is a bug in Terraform.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Planned input variable value changed",
					Detail: fmt.Sprintf(
						"The planned value for input variable %q has changed between the planning and apply phases for %s. This is a bug in Terraform - please report it.",
						name, c.Addr(),
					),
					Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
				})
			}
		}
	}
	return ret, diags
}

// Providers evaluates the "providers" argument from the component
// configuration and returns a mapping from the provider configuration
// addresses that the component's root module expect to have populated
// to the address of the [ProviderInstance] from the stack configuration
// to pass into that slot.
//
// If the second return value "valid" is true then the providers argument
// is valid and so the returned map should be complete. If "valid" is false
// then there are some problems with the providers argument and so the
// map might be incomplete, and so callers should use it only with a great
// deal of care.
func (c *ComponentInstance) Providers(ctx context.Context, phase EvalPhase) (map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, map[addrs.RootProviderConfig]addrs.Provider, bool) {
	known, unknown, diags := c.CheckProviders(ctx, phase)
	return known, unknown, !diags.HasErrors()
}

// CheckProviders evaluates the "providers" argument from the component
// configuration and returns a mapping from the provider configuration
// addresses that the component's root module expect to have populated
// to the address of the [ProviderInstance] from the stack configuration
// to pass into that slot.
//
// If the "providers" argument is invalid then this will return error
// diagnostics along with a partial result.
func (c *ComponentInstance) CheckProviders(ctx context.Context, phase EvalPhase) (map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance, map[addrs.RootProviderConfig]addrs.Provider, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	knownProviders := make(map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance)
	unknownProviders := make(map[addrs.RootProviderConfig]addrs.Provider)

	declConfigs := c.call.Declaration(ctx).ProviderConfigs
	configProviders := c.call.Config(ctx).RequiredProviderInstances(ctx)

	// First, we'll iterate through the configProviders and check that we have
	// a definition for each of them. We'll also resolve the reference that we
	// have and make sure it points to an actual provider instance.
	for _, elem := range configProviders.Elems {

		// sourceAddr is the addrs.RootProviderConfig that should be used to
		// set this provider in the component later.
		sourceAddr := elem.Key

		// componentAddr is the addrs.LocalProviderConfig that specifies the
		// local name and (optional) alias of the provider in the component.
		componentAddr := elem.Value.Local

		// We validated the config providers during the static analysis, so we
		// know this expression exists and resolves to the correct type.
		expr := declConfigs[componentAddr]

		inst, unknown, instDiags := c.checkProvider(ctx, sourceAddr, componentAddr, expr, phase)
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

	stack := c.call.Stack(ctx)
	stackConfig := stack.StackConfig(ctx)
	moduleTree := c.call.Config(ctx).ModuleTree(ctx)

	// We'll search through the declConfigs to find any keys that match the
	// type and alias of a any provider needed by the state. This is backwards
	// when compared to how we resolved the configProviders. But we don't have
	// the information we need to do it the other way around.

	previousProviders := c.main.PreviousProviderInstances(c.Addr(), phase)
	for localProviderAddr, expr := range declConfigs {
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

		inst, unknown, instDiags := c.checkProvider(ctx, sourceAddr, localProviderAddr, expr, phase)
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
				Summary:  "Component requires undeclared provider",
				Detail: fmt.Sprintf(
					"The root module for %s has resources in state that require a configuration for provider %q, which isn't declared as a dependency of this stack configuration.\n\nDeclare this provider in the stack's required_providers block, and then assign a configuration for that provider in this component's \"providers\" argument.",
					c.Addr(), provider.ForDisplay(),
				),
				Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
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
				"The root module for %s has resources in state that require a provider configuration named %q for provider %q, which is not assigned in the component's \"providers\" argument.",
				c.Addr(), localAddr.StringCompact(), previousProvider.Provider.ForDisplay(),
			),
			Subject: c.call.Declaration(ctx).DeclRange.ToHCL().Ptr(),
		})
	}

	return knownProviders, unknownProviders, diags
}

func (c *ComponentInstance) checkProvider(ctx context.Context, sourceAddr addrs.RootProviderConfig, componentAddr addrs.LocalProviderConfig, expr hcl.Expression, phase EvalPhase) (stackaddrs.AbsProviderConfigInstance, bool, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret stackaddrs.AbsProviderConfigInstance

	result, hclDiags := EvalExprAndEvalContext(ctx, expr, phase, c)
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

func (c *ComponentInstance) neededProviderSchemas(ctx context.Context, phase EvalPhase) (map[addrs.Provider]providers.ProviderSchema, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	decl := c.call.Declaration(ctx)
	moduleTree := c.call.Config(ctx).ModuleTree(ctx)
	if moduleTree == nil {
		// The configuration is presumably invalid, but it's not our
		// responsibility to report errors in the configuration.
		// We'll just return nothing and let a different codepath detect
		// and report this error.
		return nil, diags
	}

	providerSchemas := make(map[addrs.Provider]providers.ProviderSchema)
	for _, sourceAddr := range moduleTree.ProviderTypes() {
		pTy := c.main.ProviderType(ctx, sourceAddr)
		if pTy == nil {
			continue // not our job to report a missing provider type
		}
		schema, err := pTy.Schema(ctx)

		// If this phase has a dependency lockfile, check if the provider is in it.
		depLocks := c.main.DependencyLocks(phase)
		if depLocks != nil {
			providerLockfileDiags := CheckProviderInLockfile(*depLocks, pTy, decl.DeclRange)
			// We report these diagnostics in a different place
			if providerLockfileDiags.HasErrors() {
				continue
			}
		}

		if err != nil {
			// FIXME: it's not technically our job to report a schema
			// fetch failure, but currently there is no single other
			// place that definitely does it, so we'll do it here at
			// the risk of some redundant errors if we end up using
			// the same provider multiple times.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider initialization error",
				Detail:   fmt.Sprintf("Failed to fetch the provider schema for %s: %s.", sourceAddr, err),
				Subject:  decl.DeclRange.ToHCL().Ptr(),
			})
			continue
		}
		providerSchemas[sourceAddr] = schema
	}
	return providerSchemas, diags
}

func (c *ComponentInstance) neededProviderClients(ctx context.Context, phase EvalPhase) (map[addrs.RootProviderConfig]providers.Interface, func(), bool) {
	providerInstAddrs, unknownProviders, valid := c.Providers(ctx, phase)
	if !valid {
		return nil, nil, false
	}
	providerInsts := make(map[addrs.RootProviderConfig]providers.Interface)
	var closeableInsts []providers.Interface
	for calleeAddr, callerAddr := range providerInstAddrs {
		providerInstStack := c.main.Stack(ctx, callerAddr.Stack, phase)
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
	for calleeAddr, provider := range unknownProviders {
		pTy := c.main.ProviderType(ctx, provider)
		client, err := pTy.UnconfiguredClient(ctx)
		if err != nil {
			continue
		}
		closeableInsts = append(closeableInsts, client)
		providerInsts[calleeAddr] = stubs.UnknownProvider(client)
	}
	return providerInsts, func() {
		// We need to close the unconfigured clients we took for the unknown
		// providers.
		for _, inst := range closeableInsts {
			// Nothing we can really do if the close fails, so just ignore
			// the errors.
			inst.Close()
		}
	}, true
}

func (c *ComponentInstance) ModuleTreePlan(ctx context.Context) *plans.Plan {
	ret, _ := c.CheckModuleTreePlan(ctx)
	return ret
}

func (c *ComponentInstance) CheckModuleTreePlan(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
	if !c.main.Planning() {
		panic("called CheckModuleTreePlan with an evaluator not instantiated for planning")
	}

	return doOnceWithDiags(
		ctx, &c.moduleTreePlan, c.main,
		func(ctx context.Context) (*plans.Plan, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			addr := c.Addr()
			h := hooksFromContext(ctx)
			hookSingle(ctx, hooksFromContext(ctx).PendingComponentInstancePlan, c.Addr())
			seq, ctx := hookBegin(ctx, h.BeginComponentInstancePlan, h.ContextAttach, addr)

			decl := c.call.Declaration(ctx)

			// This is our main bridge from the stacks language into the main Terraform
			// module language during the planning phase. We need to ask the main
			// language runtime to plan the module tree associated with this
			// component and return the result.

			moduleTree := c.call.Config(ctx).ModuleTree(ctx)
			if moduleTree == nil {
				// Presumably the configuration is invalid in some way, so
				// we can't create a plan and the relevant diagnostics will
				// get reported when the plan driver visits the ComponentConfig
				// object.
				return nil, diags
			}
			prevState := c.PlanPrevState(ctx)

			providerSchemas, moreDiags := c.neededProviderSchemas(ctx, PlanPhase)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil, diags
			}

			tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
				Hooks: []terraform.Hook{
					&componentInstanceTerraformHook{
						ctx:   ctx,
						seq:   seq,
						hooks: hooksFromContext(ctx),
						addr:  c.Addr(),
					},
				},
				PreloadedProviderSchemas: providerSchemas,
				Provisioners:             c.main.availableProvisioners(),
			})
			if err != nil {
				// Should not get here because we should always pass a valid
				// ContextOpts above.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to instantiate Terraform modules runtime",
					fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
				))
				return nil, diags
			}

			stackPlanOpts := c.main.PlanningOpts()
			inputValues, inputValueDiags := c.inputValuesForModulesRuntime(ctx, nil, PlanPhase)
			diags = diags.Append(inputValueDiags)
			if inputValues == nil || diags.HasErrors() {
				return nil, diags
			}

			providerClients, closer, valid := c.neededProviderClients(ctx, PlanPhase)
			if !valid {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Cannot plan component",
					Detail:   fmt.Sprintf("Cannot generate a plan for %s because its provider configuration assignments are invalid.", c.Addr()),
					Subject:  decl.DeclRange.ToHCL().Ptr(),
				})
				return nil, diags
			}
			defer closer()

			// If any of our upstream components have incomplete plans then
			// we need to force treating everything in this component as
			// deferred so we can preserve the correct dependency ordering.
			deferred := false
			for _, depAddr := range c.call.RequiredComponents(ctx).Elems() {
				depStack := c.main.Stack(ctx, depAddr.Stack, PlanPhase)
				if depStack == nil {
					deferred = true // to be conservative
					break
				}
				depComponent := depStack.Component(ctx, depAddr.Item)
				if depComponent == nil {
					deferred = true // to be conservative
					break
				}
				if !depComponent.PlanIsComplete(ctx) {
					deferred = true
					break
				}
			}

			// The instance is also upstream deferred if the for_each value for
			// this instance or any parent stacks is unknown.
			if c.key == addrs.WildcardKey {
				deferred = true
			} else {
				for _, step := range c.call.addr.Stack {
					if step.Key == addrs.WildcardKey {
						deferred = true
						break
					}
				}
			}

			// When our given context is cancelled, we want to instruct the
			// modules runtime to stop the running operation. We use this
			// nested context to ensure that we don't leak a goroutine when the
			// parent context isn't cancelled.
			operationCtx, operationCancel := context.WithCancel(ctx)
			defer operationCancel()
			go func() {
				<-operationCtx.Done()
				if ctx.Err() == context.Canceled {
					tfCtx.Stop()
				}
			}()

			plantimestamp := c.main.PlanTimestamp()
			// NOTE: This ComponentInstance type only deals with component
			// instances currently declared in the configuration. See
			// [ComponentInstanceRemoved] for the model of a component instance
			// that existed in the prior state but is not currently declared
			// in the configuration.
			plan, moreDiags := tfCtx.Plan(moduleTree, prevState, &terraform.PlanOpts{
				Mode:                       stackPlanOpts.PlanningMode,
				SetVariables:               inputValues,
				ExternalProviders:          providerClients,
				DeferralAllowed:            true,
				ExternalDependencyDeferred: deferred,

				// We want the same plantimestamp between all components and the stacks language
				ForcePlanTimestamp: &plantimestamp,
			})
			diags = diags.Append(moreDiags)

			if plan != nil {
				cic := &hooks.ComponentInstanceChange{
					Addr: addr,
				}

				for _, rsrcChange := range plan.DriftedResources {
					hookMore(ctx, seq, h.ReportResourceInstanceDrift, &hooks.ResourceInstanceChange{
						Addr: stackaddrs.AbsResourceInstanceObject{
							Component: addr,
							Item:      rsrcChange.ObjectAddr(),
						},
						Change: rsrcChange,
					})
				}
				for _, rsrcChange := range plan.Changes.Resources {
					if rsrcChange.Importing != nil {
						cic.Import++
					}
					if rsrcChange.Moved() {
						cic.Move++
					}
					cic.CountNewAction(rsrcChange.Action)

					hookMore(ctx, seq, h.ReportResourceInstancePlanned, &hooks.ResourceInstanceChange{
						Addr: stackaddrs.AbsResourceInstanceObject{
							Component: addr,
							Item:      rsrcChange.ObjectAddr(),
						},
						Change: rsrcChange,
					})
				}
				for _, rsrcChange := range plan.DeferredResources {
					cic.Defer++
					hookMore(ctx, seq, h.ReportResourceInstanceDeferred, &hooks.DeferredResourceInstanceChange{
						Reason: rsrcChange.DeferredReason,
						Change: &hooks.ResourceInstanceChange{
							Addr: stackaddrs.AbsResourceInstanceObject{
								Component: addr,
								Item:      rsrcChange.ChangeSrc.ObjectAddr(),
							},
							Change: rsrcChange.ChangeSrc,
						},
					})
				}
				hookMore(ctx, seq, h.ReportComponentInstancePlanned, cic)
			}

			if diags.HasErrors() {
				hookMore(ctx, seq, h.ErrorComponentInstancePlan, addr)
			} else {
				if plan.Complete {
					hookMore(ctx, seq, h.EndComponentInstancePlan, addr)

				} else {
					hookMore(ctx, seq, h.DeferComponentInstancePlan, addr)
				}

			}

			return plan, diags
		},
	)
}

// ApplyModuleTreePlan applies a plan returned by a previous call to
// [ComponentInstance.CheckModuleTreePlan].
//
// Applying a plan often has significant externally-visible side-effects, and
// so this method should be called only once for a given plan. In practice
// we currently ensure that is true by calling it only from the package-level
// [ApplyPlan] function, which arranges for this function to be called
// concurrently with the same method on other component instances and with
// a whole-tree walk to gather up results and diagnostics.
func (c *ComponentInstance) ApplyModuleTreePlan(ctx context.Context, plan *plans.Plan) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if !c.main.Applying() {
		panic("called ApplyModuleTreePlan with an evaluator not instantiated for applying")
	}

	// NOTE WELL: This function MUST either successfully apply the component
	// instance's plan or return at least one error diagnostic explaining why
	// it cannot.
	//
	// All return paths must include a non-nil ComponentInstanceApplyResult.
	// If an error occurs before we even begin applying the plan then the
	// result should report that the changes are incomplete and that the
	// new state is exactly the previous run state.
	//
	// If the underlying modules runtime raises errors when asked to apply the
	// plan, then this function should pass all of those errors through to its
	// own diagnostics while still returning the presumably-partially-updated
	// result state.

	addr := c.Addr()
	decl := c.call.Declaration(ctx)

	// This is the result to return along with any errors that prevent us from
	// even starting the modules runtime apply phase. It reports that nothing
	// changed at all.
	noOpResult := c.PlaceholderApplyResultForSkippedApply(ctx, plan)

	// We'll gather up our set of potentially-affected objects before we do
	// anything else, because the modules runtime tends to mutate the objects
	// accessible through the given plan pointer while it does its work and
	// so we're likely to get a different/incomplete answer if we ask after
	// work has already been done.
	affectedResourceInstanceObjects := resourceInstanceObjectsAffectedByPlan(plan)

	h := hooksFromContext(ctx)
	hookSingle(ctx, hooksFromContext(ctx).PendingComponentInstanceApply, c.Addr())
	seq, ctx := hookBegin(ctx, h.BeginComponentInstanceApply, h.ContextAttach, addr)

	moduleTree := c.call.Config(ctx).ModuleTree(ctx)
	if moduleTree == nil {
		// We should not get here because if the configuration was statically
		// invalid then we should've detected that during the plan phase.
		// We'll emit a diagnostic about it just to make sure we're explicit
		// that the plan didn't get applied, but if anyone sees this error
		// it suggests a bug in whatever calling system sent us the plan
		// and configuration -- it's sent us the wrong configuration, perhaps --
		// and so we cannot know exactly what to blame with only the information
		// we have here.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component configuration is invalid during apply",
			fmt.Sprintf(
				"Despite apparently successfully creating a plan earlier, %s seems to have an invalid configuration during the apply phase. This should not be possible, and suggests a bug in whatever subsystem is managing the plan and apply workflow.",
				addr.String(),
			),
		))
		return noOpResult, diags
	}

	providerSchemas, moreDiags := c.neededProviderSchemas(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return noOpResult, diags
	}

	tfHook := &componentInstanceTerraformHook{
		ctx:   ctx,
		seq:   seq,
		hooks: hooksFromContext(ctx),
		addr:  c.Addr(),
	}
	tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
		Hooks: []terraform.Hook{
			tfHook,
		},
		PreloadedProviderSchemas: providerSchemas,
		Provisioners:             c.main.availableProvisioners(),
	})
	if err != nil {
		// Should not get here because we should always pass a valid
		// ContextOpts above.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to instantiate Terraform modules runtime",
			fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
		))
		return noOpResult, diags
	}

	// We'll need to make some light modifications to the plan to include
	// information we've learned in other parts of the apply walk that
	// should've filled in some unknown value placeholders. It would be rude
	// to modify the plan that our caller is holding though, so we'll
	// shallow-copy it. This is NOT a deep copy, so don't modify anything
	// that's reachable through any pointers without copying those first too.
	modifiedPlan := *plan
	inputValues, inputValueDiags := c.inputValuesForModulesRuntime(ctx, plan.VariableValues, ApplyPhase)
	diags = diags.Append(inputValueDiags)
	if inputValues == nil || inputValueDiags.HasErrors() {
		// inputValuesForModulesRuntime uses nil (as opposed to a
		// non-nil zerolen map) to represent that the definition of
		// the input variables was so invalid that we cannot do
		// anything with it, in which case we'll just return early
		// and assume the plan walk driver will find the diagnostics
		// via another return path.
		return noOpResult, diags
	}
	// TODO: Check that the final input values are consistent with what
	// we had during planning. If not, that suggests a bug elsewhere.
	//
	// UGH: the "modules runtime"'s model of planning was designed around
	// the goal of producing a traditional Terraform CLI-style saved plan
	// file and so it has the input variable values already encoded as
	// plans.DynamicValue opaque byte arrays, and so we need to convert
	// our resolved input values into that format. It would be better
	// if plans.Plan used the typical in-memory format for input values
	// and let the plan file serializer worry about encoding, but we'll
	// defer that API change for now to avoid disrupting other codepaths.
	modifiedPlan.VariableValues = make(map[string]plans.DynamicValue, len(inputValues))
	modifiedPlan.VariableMarks = make(map[string][]cty.PathValueMarks, len(inputValues))
	for name, iv := range inputValues {
		val, pvm := iv.Value.UnmarkDeepWithPaths()
		dv, err := plans.NewDynamicValue(val, cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to encode input variable value",
				fmt.Sprintf(
					"Could not encode the value of input variable %q of %s: %s.\n\nThis is a bug in Terraform; please report it!",
					name, c.Addr(), err,
				),
			))
			continue
		}
		modifiedPlan.VariableValues[name] = dv
		modifiedPlan.VariableMarks[name] = pvm
	}
	if diags.HasErrors() {
		return noOpResult, diags
	}

	providerClients, closer, valid := c.neededProviderClients(ctx, ApplyPhase)
	if !valid {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot apply component plan",
			Detail:   fmt.Sprintf("Cannot apply the plan for %s because the configured provider configuration assignments are invalid.", c.Addr()),
			Subject:  decl.DeclRange.ToHCL().Ptr(),
		})
		return noOpResult, diags
	}
	defer closer()

	var newState *states.State
	if modifiedPlan.Applyable {
		// When our given context is cancelled, we want to instruct the
		// modules runtime to stop the running operation. We use this
		// nested context to ensure that we don't leak a goroutine when the
		// parent context isn't cancelled.
		operationCtx, operationCancel := context.WithCancel(ctx)
		defer operationCancel()
		go func() {
			<-operationCtx.Done()
			if ctx.Err() == context.Canceled {
				tfCtx.Stop()
			}
		}()

		// NOTE: tfCtx.Apply tends to make changes to the given plan while it
		// works, and so code after this point should not make any further use
		// of either "modifiedPlan" or "plan" (since they share lots of the same
		// pointers to mutable objects and so both can get modified together.)
		newState, moreDiags = tfCtx.Apply(&modifiedPlan, moduleTree, &terraform.ApplyOpts{
			ExternalProviders: providerClients,
		})
		diags = diags.Append(moreDiags)
	} else {
		// For a non-applyable plan, we just skip trying to apply it altogether
		// and just propagate the prior state (including any refreshing we
		// did during the plan phase) forward.
		newState = modifiedPlan.PriorState
	}

	if newState != nil {
		cic := &hooks.ComponentInstanceChange{
			Addr: addr,

			// We'll increment these gradually as we visit each change below.
			Add:    0,
			Change: 0,
			Import: 0,
			Remove: 0,
			Move:   0,
			Forget: 0,

			// Defer changes will always be 0 during the apply as we don't
			// actually apply them.
			Defer: 0,
		}

		// We need to report what changes were applied, which is mostly just
		// re-announcing what was planned but we'll check to see if our
		// terraform.Hook implementation saw a "successfully applied" event
		// for each resource instance object before counting it.
		applied := tfHook.ResourceInstanceObjectsSuccessfullyApplied()
		for _, rioAddr := range applied {
			action := tfHook.ResourceInstanceObjectAppliedAction(rioAddr)
			cic.CountNewAction(action)
		}

		// The state management actions (create, import, forget) don't emit
		// actions during an apply so they're not being counted by looking
		// at the ResourceInstanceObjectAppliedAction above.
		//
		// Instead, we'll recheck the planned actions here to count them.
		plan := c.main.PlanBeingApplied().Components.Get(c.Addr())
		for _, rioAddr := range affectedResourceInstanceObjects {
			if applied.Has(rioAddr) {
				// Then we processed this above.
				continue
			}

			change, exists := plan.ResourceInstancePlanned.GetOk(rioAddr)
			if !exists {
				// This is a bit weird, but not something we should prevent
				// the apply from continuing for. We'll just ignore it and
				// assume that the plan was incomplete in some way.
				continue
			}

			// Otherwise, we have a change that wasn't successfully applied
			// for some reason. If the change was a no-op and a move or import
			// then it was still successful so we'll count it as such. Also,
			// forget actions don't count as applied changes but still happened
			// so we'll count them here.

			switch change.Action {
			case plans.NoOp:
				if change.Importing != nil {
					cic.Import++
				}
				if change.Moved() {
					cic.Move++
				}
			case plans.Forget:
				cic.Forget++
			}
		}
		cic.Defer = plan.DeferredResourceInstanceChanges.Len()

		hookMore(ctx, seq, h.ReportComponentInstanceApplied, cic)
	}

	if diags.HasErrors() {
		hookMore(ctx, seq, h.ErrorComponentInstanceApply, addr)
	} else {
		hookMore(ctx, seq, h.EndComponentInstanceApply, addr)
	}

	if newState == nil {
		// The modules runtime returns a nil state only if an error occurs
		// so early that it couldn't take any actions at all, and so we
		// must assume that the state is totally unchanged in that case.
		newState = plan.PrevRunState
		affectedResourceInstanceObjects = nil
	}

	return &ComponentInstanceApplyResult{
		FinalState:                      newState,
		AffectedResourceInstanceObjects: affectedResourceInstanceObjects,

		// Currently our definition of "complete" is that the apply phase
		// didn't return any errors, since we expect the modules runtime
		// to either perform all of the actions that were planned or
		// return errors explaining why it cannot.
		Complete: !diags.HasErrors(),
	}, diags
}

// PlanPrevState returns the previous state for this component instance during
// the planning phase, or panics if called in any other phase.
func (c *ComponentInstance) PlanPrevState(ctx context.Context) *states.State {
	// The following call will panic if we aren't in the plan phase.
	stackState := c.main.PlanPrevState()
	ret := stackState.ComponentInstanceStateForModulesRuntime(c.Addr())
	if ret == nil {
		ret = states.NewState() // so caller doesn't need to worry about nil
	}
	return ret
}

// ApplyResult returns the result from applying a plan for this object using
// [ApplyModuleTreePlan].
//
// Use the Complete field of the returned object to determine whether the
// apply ran to completion successfully enough for dependent work to proceed.
// If Complete is false then dependent work should not start, and instead
// dependents should unwind their stacks in a way that describes a no-op result.
func (c *ComponentInstance) ApplyResult(ctx context.Context) *ComponentInstanceApplyResult {
	ret, _ := c.CheckApplyResult(ctx)
	return ret
}

// CheckApplyResult returns the results from applying a plan for this object
// using [ApplyModuleTreePlan], and diagnostics describing any problems
// encountered when applying it.
func (c *ComponentInstance) CheckApplyResult(ctx context.Context) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	changes := c.main.ApplyChangeResults()
	applyResult, moreDiags, err := changes.ComponentInstanceResult(ctx, c.Addr())
	diags = diags.Append(moreDiags)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component instance apply not scheduled",
			fmt.Sprintf("Terraform needs the result from applying changes to %s, but that apply was apparently not scheduled to run: %s. This is a bug in Terraform.", c.Addr(), err),
		))
	}
	return applyResult, diags
}

// PlaceholderApplyResultForSkippedApply returns a [ComponentInstanceApplyResult]
// which describes the hypothetical result of skipping the apply phase for
// this component instance altogether.
//
// It doesn't have any logic to check whether the apply _was_ actually skipped;
// the caller that's orchestrating the changes during the apply phase must
// decided that for itself and then choose between either calling
// [ComponentInstance.ApplyModuleTreePlan] to apply as normal, or returning
// the result of this function instead to explain that the apply was skipped.
func (c *ComponentInstance) PlaceholderApplyResultForSkippedApply(ctx context.Context, plan *plans.Plan) *ComponentInstanceApplyResult {
	// (We have this in here as a method just because it helps keep all of
	// the logic for constructing [ComponentInstanceApplyResult] objects
	// together in the same file, rather than having the caller synthesize
	// a result itself only in this one special situation.)
	return &ComponentInstanceApplyResult{
		FinalState: plan.PrevRunState,
		Complete:   false,
	}
}

// ApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan], or nil if the apply failed and
// so there is no new state to return.
func (c *ComponentInstance) ApplyResultState(ctx context.Context) *states.State {
	ret, _ := c.CheckApplyResultState(ctx)
	return ret
}

// CheckApplyResultState returns the new state resulting from applying a plan for
// this object using [ApplyModuleTreePlan] and diagnostics describing any
// problems encountered when applying it.
func (c *ComponentInstance) CheckApplyResultState(ctx context.Context) (*states.State, tfdiags.Diagnostics) {
	result, diags := c.CheckApplyResult(ctx)
	var newState *states.State
	if result != nil {
		newState = result.FinalState
	}
	return newState, diags
}

// InspectingState returns the state as captured in the snapshot provided when
// instantiating [Main] for [InspectPhase] evaluation.
func (c *ComponentInstance) InspectingState(ctx context.Context) *states.State {
	wholeState := c.main.InspectingState()
	return wholeState.ComponentInstanceStateForModulesRuntime(c.Addr())
}

func (c *ComponentInstance) ResultValue(ctx context.Context, phase EvalPhase) cty.Value {
	switch phase {
	case PlanPhase:
		plan := c.ModuleTreePlan(ctx)
		if plan == nil {
			// Planning seems to have failed so we cannot decide a result value yet.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			return cty.DynamicVal
		}

		// We need to vary our behavior here slightly depending on what action
		// we're planning to take with this overall component: normally we want
		// to use the "planned new state"'s output values, but if we're actually
		// planning to destroy all of the infrastructure managed by this
		// component then the planned new state has no output values at all,
		// so we'll use the prior state's output values instead just in case
		// we also need to plan destroying another component instance
		// downstream of this one which will make use of this instance's
		// output values _before_ we destroy it.
		//
		// FIXME: We're using UIMode for this decision, despite its doc comment
		// saying we shouldn't, because this behavior is an offshoot of the
		// already-documented annoying exception to that rule where various
		// parts of Terraform use UIMode == DestroyMode in particular to deal
		// with necessary variations during a "full destroy". Hopefully we'll
		// eventually find a more satisfying solution for that, in which case
		// we should update the following to use that solution too.
		attrs := make(map[string]cty.Value)
		if plan.UIMode != plans.DestroyMode {
			outputChanges := plan.Changes.Outputs
			for _, changeSrc := range outputChanges {
				if len(changeSrc.Addr.Module) > 0 {
					// Only include output values of the root module as part
					// of the component.
					continue
				}

				name := changeSrc.Addr.OutputValue.Name
				change, err := changeSrc.Decode()
				if err != nil {
					attrs[name] = cty.DynamicVal
					continue
				}

				if changeSrc.Sensitive {
					// For our purposes here, a static sensitive flag on the
					// output value is indistinguishable from the value having
					// been dynamically marked as sensitive.
					attrs[name] = change.After.Mark(marks.Sensitive)
					continue
				}

				// Otherwise, just use the value as-is.
				attrs[name] = change.After
			}
		} else {
			// The "prior state" of the plan includes any new information we
			// learned by "refreshing" before we planned to destroy anything,
			// and so should be as close as possible to the current
			// (pre-destroy) state of whatever infrastructure this component
			// instance is managing.
			for _, os := range plan.PriorState.RootOutputValues {
				v := os.Value
				if os.Sensitive {
					// For our purposes here, a static sensitive flag on the
					// output value is indistinguishable from the value having
					// been dynamically marked as sensitive.
					v = v.Mark(marks.Sensitive)
				}
				attrs[os.Addr.OutputValue.Name] = v
			}
		}
		if decl := c.call.Config(ctx).ModuleTree(ctx); decl != nil {
			// If the plan only ran partially then we might be missing
			// some planned changes for output values, which could
			// cause "attrs" to have an incomplete set of attributes.
			// To avoid confusing downstream errors we'll insert unknown
			// values for any declared output values that don't yet
			// have a final value.
			for name := range decl.Module.Outputs {
				if _, ok := attrs[name]; !ok {
					// We can't do any better than DynamicVal because
					// output values in the modules language don't
					// have static type constraints.
					attrs[name] = cty.DynamicVal
				}
			}
			// In the DestroyMode case above we might also find ourselves
			// with some remnant additional output values that have since
			// been removed from the configuration, but yet remain in the
			// state. Destroying with a different configuration than was
			// most recently applied is not guaranteed to work, but we
			// can make it more likely to work by dropping anything that
			// isn't currently declared, since referring directly to these
			// would be a static validation error anyway, and including
			// them might cause aggregate operations like keys(component.foo)
			// to produce broken results.
			for name := range attrs {
				_, declared := decl.Module.Outputs[name]
				if !declared {
					// (deleting map elements during iteration is valid in Go,
					// unlike some other languages.)
					delete(attrs, name)
				}
			}
		}
		return cty.ObjectVal(attrs)

	case ApplyPhase, InspectPhase:
		// As a special case, if we're applying and the planned action is
		// to destroy then we'll just return the planned output values
		// verbatim without waiting for anything, so that downstreams can
		// begin their own destroy phases before we start ours.
		if phase == ApplyPhase {
			fullPlan := c.main.PlanBeingApplied()
			ourPlan := fullPlan.Components.Get(c.Addr())
			if ourPlan == nil {
				// Weird, but we'll tolerate it.
				return cty.DynamicVal
			}
			if ourPlan.PlannedAction == plans.Delete {
				// In this case our result was already decided during the
				// planning phase, because we can't block on anything else
				// here to make sure we don't create a self-dependency
				// while our downstreams are trying to destroy themselves.
				attrs := make(map[string]cty.Value, len(ourPlan.PlannedOutputValues))
				for addr, val := range ourPlan.PlannedOutputValues {
					attrs[addr.Name] = val
				}
				return cty.ObjectVal(attrs)
			}
		}

		var state *states.State
		switch phase {
		case ApplyPhase:
			state = c.ApplyResultState(ctx)
		case InspectPhase:
			state = c.InspectingState(ctx)
		default:
			panic(fmt.Sprintf("unsupported evaluation phase %s", state)) // should not get here
		}
		if state == nil {
			// Applying seems to have failed so we cannot provide a result
			// value, and so we'll return a placeholder to help our caller
			// unwind gracefully with its own placeholder result.
			// We can't do any better than DynamicVal here because in the
			// modules language output values don't have statically-declared
			// result types.
			// (This should not typically happen in InspectPhase if the caller
			// provided a valid state snapshot, but we'll still tolerate it in
			// that case because InspectPhase is sometimes used in our unit
			// tests which might provide contrived input if testing component
			// instances is not their primary focus.)
			return cty.DynamicVal
		}

		// For apply and inspect phases we use the root module output values
		// from the state to construct our value.
		outputVals := state.RootOutputValues
		attrs := make(map[string]cty.Value, len(outputVals))
		for _, ov := range outputVals {
			name := ov.Addr.OutputValue.Name

			if ov.Sensitive {
				// For our purposes here, a static sensitive flag on the
				// output value is indistinguishable from the value having
				// been dynamically marked as sensitive.
				attrs[name] = ov.Value.Mark(marks.Sensitive)
				continue
			}

			// Otherwise, just set the value as is.
			attrs[name] = ov.Value
		}

		// If the apply operation was unsuccessful for any reason then we
		// might have some output values that are missing from the state,
		// because the state is only updated with the results of successful
		// operations. To avoid downstream errors we'll insert unknown values
		// for any declared output values that don't yet have a final value.
		//
		// The status of the apply operation will have been recorded elsewhere
		// so we don't need to worry about that here. This also ensures that
		// nothing will actually attempt to apply the unknown values here.
		config := c.call.Config(ctx).ModuleTree(ctx)
		for _, output := range config.Module.Outputs {
			if _, ok := attrs[output.Name]; !ok {
				attrs[output.Name] = cty.DynamicVal
			}
		}

		return cty.ObjectVal(attrs)

	default:
		// We can't produce a concrete value for any other phase.
		return cty.DynamicVal
	}
}

// ResolveExpressionReference implements ExpressionScope.
func (c *ComponentInstance) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	stack := c.call.Stack(ctx)
	return stack.resolveExpressionReference(ctx, ref, nil, c.repetition)
}

// PlanTimestamp implements ExpressionScope, providing the timestamp at which
// the current plan is being run.
func (c *ComponentInstance) PlanTimestamp() time.Time {
	return c.main.PlanTimestamp()
}

// PlanChanges implements Plannable by validating that all of the per-instance
// arguments are suitable, and then asking the main Terraform language runtime
// to produce a plan in terms of the component's selected module.
func (c *ComponentInstance) PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics) {
	var changes []stackplan.PlannedChange
	var diags tfdiags.Diagnostics

	_, moreDiags := c.CheckInputVariableValues(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	_, _, moreDiags = c.CheckProviders(ctx, PlanPhase)
	diags = diags.Append(moreDiags)

	corePlan, moreDiags := c.CheckModuleTreePlan(ctx)
	diags = diags.Append(moreDiags)
	if corePlan != nil {
		existedBefore := false
		if prevState := c.main.PlanPrevState(); prevState != nil {
			existedBefore = prevState.HasComponentInstance(c.Addr())
		}
		destroying := corePlan.UIMode == plans.DestroyMode
		refreshOnly := corePlan.UIMode == plans.RefreshOnlyMode

		var action plans.Action
		switch {
		case destroying:
			action = plans.Delete
		case refreshOnly:
			action = plans.Read
		case existedBefore:
			action = plans.Update
		default:
			action = plans.Create
		}

		// FIXME: This is silly because we make ResultValue wrap the output
		// values map up into an object and then just unwrap it again
		// immediately.
		var outputVals map[string]cty.Value
		if resultVal := c.ResultValue(ctx, PlanPhase); resultVal.Type().IsObjectType() && resultVal.IsKnown() && !resultVal.IsNull() {
			outputVals = make(map[string]cty.Value, resultVal.LengthInt())
			for it := resultVal.ElementIterator(); it.Next(); {
				k, v := it.Element()
				outputVals[k.AsString()] = v
			}
		}

		// We must always at least announce that the component instance exists,
		// and that must come before any resource instance changes referring to it.
		changes = append(changes, &stackplan.PlannedChangeComponentInstance{
			Addr: c.Addr(),

			Action:                 action,
			PlanApplyable:          corePlan.Applyable,
			PlanComplete:           corePlan.Complete,
			RequiredComponents:     c.RequiredComponents(ctx),
			PlannedInputValues:     corePlan.VariableValues,
			PlannedInputValueMarks: corePlan.VariableMarks,
			PlannedOutputValues:    outputVals,
			PlannedCheckResults:    corePlan.Checks,

			// We must remember the plan timestamp so that the plantimestamp
			// function can return a consistent result during a later apply phase.
			PlanTimestamp: corePlan.Timestamp,
		})

		seenObjects := addrs.MakeSet[addrs.AbsResourceInstanceObject]()
		for _, rsrcChange := range corePlan.Changes.Resources {
			schema, err := c.resourceTypeSchema(
				ctx,
				rsrcChange.ProviderAddr.Provider,
				rsrcChange.Addr.Resource.Resource.Mode,
				rsrcChange.Addr.Resource.Resource.Type,
			)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Can't fetch provider schema to save plan",
					fmt.Sprintf(
						"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
						rsrcChange.Addr, rsrcChange.ProviderAddr.Provider, err,
					),
				))
				continue
			}

			objAddr := addrs.AbsResourceInstanceObject{
				ResourceInstance: rsrcChange.Addr,
				DeposedKey:       rsrcChange.DeposedKey,
			}
			var priorStateSrc *states.ResourceInstanceObjectSrc
			if corePlan.PriorState != nil {
				priorStateSrc = corePlan.PriorState.ResourceInstanceObjectSrc(objAddr)
			}

			changes = append(changes, &stackplan.PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: c.Addr(),
					Item:      objAddr,
				},
				ChangeSrc:          rsrcChange,
				Schema:             schema,
				PriorStateSrc:      priorStateSrc,
				ProviderConfigAddr: rsrcChange.ProviderAddr,

				// TODO: Also provide the previous run state, if it's
				// different from the prior state, and signal whether the
				// difference from previous run seems "notable" per
				// Terraform Core's heuristics. Only the external plan
				// description needs that info, to populate the
				// "changes outside of Terraform" part of the plan UI;
				// the raw plan only needs the prior state.
			})
			seenObjects.Add(objAddr)
		}

		// We also need to catch any objects that exist in the "prior state"
		// but don't have any actions planned, since we still need to capture
		// the prior state part in case it was updated by refreshing during
		// the plan walk.
		if priorState := corePlan.PriorState; priorState != nil {
			for _, addr := range priorState.AllResourceInstanceObjectAddrs() {
				if seenObjects.Has(addr) {
					// We're only interested in objects that didn't appear
					// in the plan, such as data resources whose read has
					// completed during the plan phase.
					continue
				}

				rs := priorState.Resource(addr.ResourceInstance.ContainingResource())
				os := priorState.ResourceInstanceObjectSrc(addr)
				schema, err := c.resourceTypeSchema(
					ctx,
					rs.ProviderConfig.Provider,
					addr.ResourceInstance.Resource.Resource.Mode,
					addr.ResourceInstance.Resource.Resource.Type,
				)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Can't fetch provider schema to save plan",
						fmt.Sprintf(
							"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
							addr, rs.ProviderConfig.Provider, err,
						),
					))
					continue
				}

				changes = append(changes, &stackplan.PlannedChangeResourceInstancePlanned{
					ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
						Component: c.Addr(),
						Item:      addr,
					},
					Schema:             schema,
					PriorStateSrc:      os,
					ProviderConfigAddr: rs.ProviderConfig,
					// We intentionally omit ChangeSrc, because we're not actually
					// planning to change this object during the apply phase, only
					// to update its state data.
				})
				seenObjects.Add(addr)
			}
		}

		// We also have one more unusual case to deal with: if an object
		// existed at the end of the previous run but was found to have
		// been deleted when we refreshed during planning then it will
		// not be present in either the prior state _or_ the plan, but
		// we still need to include a stubby object for it in the plan
		// so we can remember to discard it from the state during the
		// apply phase.
		if prevRunState := corePlan.PrevRunState; prevRunState != nil {
			for _, addr := range prevRunState.AllResourceInstanceObjectAddrs() {
				if seenObjects.Has(addr) {
					// We're only interested in objects that didn't appear
					// in the plan, such as data resources whose read has
					// completed during the plan phase.
					continue
				}

				rs := prevRunState.Resource(addr.ResourceInstance.ContainingResource())

				changes = append(changes, &stackplan.PlannedChangeResourceInstancePlanned{
					ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
						Component: c.Addr(),
						Item:      addr,
					},
					ProviderConfigAddr: rs.ProviderConfig,
					// Everything except the addresses are omitted in this case,
					// which represents that we should just delete the object
					// from the state when applied, and not take any other
					// action.
				})
				seenObjects.Add(addr)
			}
		}

		// We need to keep track of the deferred changes as well
		for _, dr := range corePlan.DeferredResources {
			rsrcChange := dr.ChangeSrc
			objAddr := addrs.AbsResourceInstanceObject{
				ResourceInstance: rsrcChange.Addr,
				DeposedKey:       rsrcChange.DeposedKey,
			}
			var priorStateSrc *states.ResourceInstanceObjectSrc
			if corePlan.PriorState != nil {
				priorStateSrc = corePlan.PriorState.ResourceInstanceObjectSrc(objAddr)
			}

			schema, err := c.resourceTypeSchema(
				ctx,
				rsrcChange.ProviderAddr.Provider,
				rsrcChange.Addr.Resource.Resource.Mode,
				rsrcChange.Addr.Resource.Resource.Type,
			)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Can't fetch provider schema to save plan",
					fmt.Sprintf(
						"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.",
						rsrcChange.Addr, rsrcChange.ProviderAddr.Provider, err,
					),
				))
				continue
			}

			plannedChangeResourceInstance := stackplan.PlannedChangeResourceInstancePlanned{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: c.Addr(),
					Item:      objAddr,
				},
				ChangeSrc:          rsrcChange,
				Schema:             schema,
				PriorStateSrc:      priorStateSrc,
				ProviderConfigAddr: rsrcChange.ProviderAddr,
			}
			changes = append(changes, &stackplan.PlannedChangeDeferredResourceInstancePlanned{
				DeferredReason:          dr.DeferredReason,
				ResourceInstancePlanned: plannedChangeResourceInstance,
			})
		}
	}

	return changes, diags
}

// RequiredComponents implements Applyable
func (c *ComponentInstance) RequiredComponents(ctx context.Context) collections.Set[stackaddrs.AbsComponent] {
	return c.call.RequiredComponents(ctx)
}

// CheckApply implements Applyable.
func (c *ComponentInstance) CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var changes []stackstate.AppliedChange
	var diags tfdiags.Diagnostics

	// FIXME: We need to report an AppliedChange object for the component
	// instance itself, and we need to emit "interim" objects representing
	// the "prior state" (refreshed) in each resource instance change in
	// the plan, so that the effect of refreshing will still get committed
	// to the state even if other downstream changes don't succeed.

	_, moreDiags := c.CheckInputVariableValues(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	_, _, moreDiags = c.CheckProviders(ctx, ApplyPhase)
	diags = diags.Append(moreDiags)

	applyResult, moreDiags := c.CheckApplyResult(ctx)
	diags = diags.Append(moreDiags)

	if applyResult != nil {
		newState := applyResult.FinalState

		ourChange := &stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         c.call.Addr(),
			ComponentInstanceAddr: c.Addr(),
			OutputValues:          make(map[addrs.OutputValue]cty.Value, len(newState.RootOutputValues)),
		}
		for name, os := range newState.RootOutputValues {
			val := os.Value
			if os.Sensitive {
				val = val.Mark(marks.Sensitive)
			}
			ourChange.OutputValues[addrs.OutputValue{Name: name}] = val
		}
		changes = append(changes, ourChange)

		for _, rioAddr := range applyResult.AffectedResourceInstanceObjects {
			os := newState.ResourceInstanceObjectSrc(rioAddr)
			var providerConfigAddr addrs.AbsProviderConfig
			var schema *configschema.Block
			if os != nil {
				rAddr := rioAddr.ResourceInstance.ContainingResource()
				rs := newState.Resource(rAddr)
				if rs == nil {
					// We should not get here: it should be impossible to
					// have state for a resource instance object without
					// also having state for its containing resource, because
					// the object is nested inside the resource state.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Inconsistent updated state for resource",
						fmt.Sprintf(
							"There is a state for %s specifically, but somehow no state for its containing resource %s. This is a bug in Terraform.",
							rioAddr, rAddr,
						),
					))
					continue
				}
				providerConfigAddr = rs.ProviderConfig

				var err error
				schema, err = c.resourceTypeSchema(
					ctx,
					rs.ProviderConfig.Provider,
					rAddr.Resource.Mode,
					rAddr.Resource.Type,
				)
				if err != nil {
					// It shouldn't be possible to get here because we would've
					// used the same schema we were just trying to retrieve
					// to encode the dynamic data in this states.State object
					// in the first place. If we _do_ get here then we won't
					// actually be able to save the updated state, which will
					// force the user to manually clean things up.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Can't fetch provider schema to save new state",
						fmt.Sprintf(
							"Failed to retrieve the schema for %s from provider %s: %s. This is a bug in Terraform.\n\nThe new state for this object cannot be saved. If this object was only just created, you may need to delete it manually in the target system to reconcile with the Terraform state before trying again.",
							rAddr, rs.ProviderConfig.Provider, err,
						),
					))
					continue
				}
			} else {
				// Our model doesn't have any way to represent the absense
				// of a provider configuration, so if we're trying to describe
				// just that the object has been deleted then we'll just
				// use a synthetic provider config address, this won't get
				// used for anything significant anyway.
				providerAddr := addrs.ImpliedProviderForUnqualifiedType(rioAddr.ResourceInstance.Resource.Resource.ImpliedProvider())
				providerConfigAddr = addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: providerAddr,
				}
			}

			var previousAddress *stackaddrs.AbsResourceInstanceObject
			if plannedChange := c.main.PlanBeingApplied().Components.Get(c.Addr()).ResourceInstancePlanned.Get(rioAddr); plannedChange != nil && plannedChange.Moved() {
				// If we moved the resource instance object, we need to record
				// the previous address in the applied change. The planned
				// change might be nil if the resource instance object was
				// deleted.
				previousAddress = &stackaddrs.AbsResourceInstanceObject{
					Component: c.Addr(),
					Item: addrs.AbsResourceInstanceObject{
						ResourceInstance: plannedChange.PrevRunAddr,
						DeposedKey:       addrs.NotDeposed,
					},
				}
			}

			changes = append(changes, &stackstate.AppliedChangeResourceInstanceObject{
				ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
					Component: c.Addr(),
					Item:      rioAddr,
				},
				PreviousResourceInstanceObjectAddr: previousAddress,
				NewStateSrc:                        os,
				ProviderConfigAddr:                 providerConfigAddr,
				Schema:                             schema,
			})
		}

	}

	return changes, diags
}

func (c *ComponentInstance) resourceTypeSchema(ctx context.Context, providerTypeAddr addrs.Provider, mode addrs.ResourceMode, typ string) (*configschema.Block, error) {
	// This should not be able to fail with an error because we should
	// be retrieving the same schema that was already used to encode
	// the object we're working with. The error handling here is for
	// robustness but any error here suggests a bug in Terraform.

	providerType := c.main.ProviderType(ctx, providerTypeAddr)
	providerSchema, err := providerType.Schema(ctx)
	if err != nil {
		return nil, err
	}
	ret, _ := providerSchema.SchemaForResourceType(mode, typ)
	if ret == nil {
		return nil, fmt.Errorf("schema does not include %v %q", mode, typ)
	}
	return ret, nil
}

func (c *ComponentInstance) tracingName() string {
	return c.Addr().String()
}

// reportNamedPromises implements namedPromiseReporter.
func (c *ComponentInstance) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	cb(c.moduleTreePlan.PromiseID(), c.Addr().String()+" plan")
}
