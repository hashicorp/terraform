// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	stackparser "github.com/hashicorp/terraform/internal/stacks/stackconfig/parser"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TODO: This file currently never includes source information in the diags it
//   emits when it totally could. This should be added when we productionise
//   everything.

type Migration struct {
	Providers     map[addrs.Provider]providers.Factory
	PreviousState *states.State
	Config        *stackconfig.Config
}

func (m *Migration) Migrate(resources map[string]string, modules map[string]string, emit func(change stackstate.AppliedChange), emitDiag func(diagnostic tfdiags.Diagnostic)) {

	migration := &migration{
		Migration: m,
		emit:      emit,
		emitDiag:  emitDiag,
		providers: make(map[addrs.Provider]providers.Interface),
		parser:    configs.NewSourceBundleParser(m.Config.Sources),
		configs:   make(map[sourceaddrs.FinalSource]*configs.Config),
	}

	defer migration.close() // cleanup any opened providers.

	components := migration.migrateResources(resources, modules)
	migration.migrateComponents(components)

	// Everything is migrated!
}

type migration struct {
	Migration *Migration

	emit     func(change stackstate.AppliedChange)
	emitDiag func(diagnostic tfdiags.Diagnostic)

	providers map[addrs.Provider]providers.Interface
	parser    *configs.SourceBundleParser
	configs   map[sourceaddrs.FinalSource]*configs.Config
}

func (m *migration) config(addr sourceaddrs.FinalSource) *configs.Config {

	if cfg, ok := m.configs[addr]; ok {
		return cfg
	}

	if !m.parser.IsConfigDir(addr) {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Component configuration not found", fmt.Sprintf("Component configuration not found at %s", addr)))
		return nil
	}

	var diags tfdiags.Diagnostics
	module, moreDiags := m.parser.LoadConfigDir(addr)
	diags = diags.Append(moreDiags)
	for _, diag := range diags {
		m.emitDiag(diag)
	}
	if module != nil {
		walker := stackparser.NewSourceBundleModuleWalker(addr, m.Migration.Config.Sources, m.parser)
		config, moreDiags := configs.BuildConfig(module, walker, nil)
		diags = diags.Append(moreDiags)
		for _, diag := range diags {
			m.emitDiag(diag)
		}

		m.configs[addr] = config
	}

	return m.configs[addr]
}

func (m *migration) provider(provider addrs.Provider) providers.Interface {
	if p, ok := m.providers[provider]; ok {
		return p
	}

	factory, ok := m.Migration.Providers[provider]
	if !ok {
		m.providers[provider] = nil
		return nil
	}

	p, err := factory()
	if err != nil {
		m.providers[provider] = nil
		return nil
	}

	m.providers[provider] = p
	return p
}

func (m *migration) close() {
	for addr, provider := range m.providers {
		if err := provider.Close(); err != nil {
			m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider cleanup failed", fmt.Sprintf("Failed to close provider %s: %s", addr.ForDisplay(), err.Error())))
		}
	}
}

func (m *migration) migrateResources(resources map[string]string, modules map[string]string) collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]] {
	components := collections.NewMap[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]()

	trackComponent := func(instance stackaddrs.AbsComponentInstance) {
		configComponent := stackaddrs.ConfigComponent{
			Stack: instance.Stack.ConfigAddr(),
			Item:  instance.Item.Component,
		}
		if !components.HasKey(configComponent) {
			components.Put(configComponent, collections.NewSet[stackaddrs.AbsComponentInstance]())
		}
		components.Get(configComponent).Add(instance)
	}

	for _, module := range m.Migration.PreviousState.Modules {
		for _, resource := range module.Resources {

			target, ok := m.search(resource.Addr, resources, modules)
			if !ok {
				// search should have emitted a diagnostic already if it returned
				// false
				continue
			}

			trackComponent(target.Component) // record the component instance

			providerAddress, ok := m.getProviderAddress(target)
			if !ok {
				// getProviderAddress should have emitted a diagnostic already
				continue
			}

			provider := m.provider(providerAddress.Provider)
			if provider == nil {
				// provider should have emitted a diagnostic already
				continue
			}

			schema, ok := provider.GetProviderSchema().ResourceTypes[resource.Addr.Resource.Type]
			if !ok {
				m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Resource type not found", fmt.Sprintf("Resource type %s not found in provider schema.", resource.Addr.Resource.Type)))
				continue
			}

			for instanceKey, instance := range resource.Instances {
				instanceAddr := stackaddrs.AbsResourceInstance{
					Component: target.Component,
					Item:      target.Item.Instance(instanceKey),
				}

				m.emit(&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
						Component: instanceAddr.Component,
						Item:      instanceAddr.Item.DeposedObject(addrs.NotDeposed),
					},
					NewStateSrc:        instance.Current,
					ProviderConfigAddr: providerAddress,
					Schema:             schema.Block,
				})

				for deposedKey, deposed := range instance.Deposed {
					m.emit(&stackstate.AppliedChangeResourceInstanceObject{
						ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
							Component: instanceAddr.Component,
							Item:      instanceAddr.Item.DeposedObject(deposedKey),
						},
						NewStateSrc:        deposed,
						ProviderConfigAddr: providerAddress,
						Schema:             schema.Block,
					})
				}
			}
		}
	}
	return components
}

// search looks through the resource mappings and returns the relevant
// stackaddrs.AbsResourceInstance.
func (m *migration) search(resource addrs.AbsResource, resources map[string]string, modules map[string]string) (stackaddrs.AbsResource, bool) {
	if resource.Module.IsRoot() {
		if target, ok := resources[resource.Resource.String()]; ok {
			return stackaddrs.AbsResource{
				Component: stackaddrs.AbsComponentInstance{
					Stack: stackaddrs.RootStackInstance,
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: target,
						},
						Key: addrs.NoKey,
					},
				},
				Item: resource,
			}, true
		} else {
			m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Resource not found", fmt.Sprintf("Resource %s not found in mapping.", resource.Resource.String())))
			return stackaddrs.AbsResource{}, false
		}
	}

	if target, ok := modules[resource.Module[0].Name]; ok {
		return stackaddrs.AbsResource{
			Component: stackaddrs.AbsComponentInstance{
				Stack: stackaddrs.RootStackInstance,
				Item: stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{
						Name: target,
					},
					Key: resource.Module[0].InstanceKey,
				},
			},
			Item: addrs.AbsResource{
				Module:   resource.Module[1:], // the first module instance is replaced by the component instance
				Resource: resource.Resource,
			},
		}, true
	} else {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Module not found", fmt.Sprintf("Module %s not found in mapping.", resource.Module[0].Name)))
		return stackaddrs.AbsResource{}, false
	}
}

// getProviderAddress returns the address of the provider configuration
// that was used to create the given resource instance.
//
// The provided config address is the location within the previous configuration
// and we need to find the corresponding provider configuration in the new
// configuration.
func (m *migration) getProviderAddress(target stackaddrs.AbsResource) (addrs.AbsProviderConfig, bool) {

	stack := m.Migration.Config.Stack(target.Component.Stack.ConfigAddr())
	if stack == nil {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Stack not found", fmt.Sprintf("Stack %s not found in configuration.", target.Component.Stack.ConfigAddr())))
		return addrs.AbsProviderConfig{}, false
	}

	component := stack.Components[target.Component.Item.Component.Name]
	if component == nil {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Component not found", fmt.Sprintf("Component %s not found in stack %s.", target.Component.Item.Component.Name, target.Component.Stack.ConfigAddr())))
		return addrs.AbsProviderConfig{}, false
	}

	config := m.config(component.FinalSourceAddr)
	if config == nil {
		// We should have emitted diagnostics for this already.
		return addrs.AbsProviderConfig{}, false
	}

	moduleProvider, ok := m.findProviderInModule(target.Item.Module.Module(), target.Item.Resource, config)
	if !ok {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider not found for resource %s in component %s.", target.Item.Resource.String(), target.Component.Item.Component.Name)))
		return addrs.AbsProviderConfig{}, false
	}

	// translate the local provider

	expr, ok := component.ProviderConfigs[moduleProvider]
	if !ok {
		// Then the module uses a provider not referenced in the component.
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s not found in component %s.", moduleProvider.LocalName, target.Component.Item.Component.Name)))
		return addrs.AbsProviderConfig{}, false
	}

	vars := expr.Variables()
	if len(vars) != 1 {
		// This should be an exact reference to a single provider, if it's not
		// we can't really do anything.
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Invalid reference", "Provider references should be a simple reference to a single provider."))
		return addrs.AbsProviderConfig{}, false
	}

	var diags tfdiags.Diagnostics
	ref, _, moreDiags := stackaddrs.ParseReference(vars[0])
	diags = diags.Append(moreDiags)
	for _, diag := range diags {
		m.emitDiag(diag)
	}

	switch ref := ref.Target.(type) {
	case stackaddrs.ProviderConfigRef:
		provider, ok := stack.RequiredProviders.ProviderForLocalName(ref.ProviderLocalName)
		if !ok {
			m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s not found in required_providers.", ref.ProviderLocalName)))
			return addrs.AbsProviderConfig{}, false
		}

		return addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: provider,
			Alias:    moduleProvider.Alias, // we still use the alias from the module provider as this is referenced as if from within the module.
		}, true
	default:
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Invalid reference", "Non-provider reference found in provider configuration."))
		return addrs.AbsProviderConfig{}, false
	}
}

// findProviderInModule returns the provider configuration within config that
// was used to create the given resource in module.
//
// Within a stack, providers cannot be defined in modules we basically return
// the name and alias of the provider that was used to either manage the
// resource or was passed into the module that does manage the resource.
//
// A false return value indicates that the resource does not actually exist
// within the configuration.
func (m *migration) findProviderInModule(module addrs.Module, resource addrs.Resource, config *configs.Config) (addrs.LocalProviderConfig, bool) {
	if len(module) == 0 {
		r := config.Module.ResourceByAddr(resource)
		if r == nil {
			return addrs.LocalProviderConfig{}, false
		}

		return r.ProviderConfigAddr(), true
	}

	next, ok := config.Children[module[0]]
	if !ok {
		return addrs.LocalProviderConfig{}, false
	}

	provider, ok := m.findProviderInModule(module[1:], resource, next)
	if !ok {
		return addrs.LocalProviderConfig{}, false
	}

	call, ok := config.Module.ModuleCalls[module[0]]
	if !ok {
		return addrs.LocalProviderConfig{}, false
	}

	for _, p := range call.Providers {
		if p.InChild.Name == provider.LocalName && p.InChild.Alias == provider.Alias {
			return p.InParent.Addr(), true
		}
	}
	return addrs.LocalProviderConfig{}, false
}

func (m *migration) migrateComponents(components collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]) {
	dependencies, dependents := m.calculateDependencies(components)

	for _, cmpnts := range components.All() {
		for component := range cmpnts.All() {
			cfg := m.Migration.Config.Component(stackaddrs.ConfigComponent{
				Stack: component.Stack.ConfigAddr(),
				Item:  component.Item.Component,
			})
			if cfg.FinalSourceAddr == nil {
				panic("component has no final source address")
			}

			// We need to see the inputs and outputs from the component, so we can
			// create the component instance with the correct values.

			config := m.config(cfg.FinalSourceAddr)
			if config == nil {
				// We should have emitted diagnostics for this already.
				continue
			}

			// We can put unknown values into the state for now, as Stacks should
			// perform a refresh before actually using any of these anyway.

			inputs := make(map[addrs.InputVariable]cty.Value, len(config.Module.Variables))
			for name := range config.Module.Variables {
				inputs[addrs.InputVariable{Name: name}] = cty.DynamicVal
			}
			outputs := make(map[addrs.OutputValue]cty.Value, len(config.Module.Outputs))
			for name := range config.Module.Outputs {
				outputs[addrs.OutputValue{Name: name}] = cty.DynamicVal
			}

			// We need this address to be able to look up dependencies and
			// dependents later.

			addr := stackaddrs.AbsComponent{
				Stack: component.Stack,
				Item:  component.Item.Component,
			}

			m.emit(&stackstate.AppliedChangeComponentInstance{
				ComponentAddr: stackaddrs.AbsComponent{
					Stack: stackaddrs.RootStackInstance,
					Item:  component.Item.Component,
				},
				ComponentInstanceAddr: component,

				OutputValues:   outputs,
				InputVariables: inputs,

				// If a destroy plan, or a removed block, is executed before the
				// next plan is applied, the component will break without this
				// metadata.
				Dependencies: dependencies.Get(addr),
				Dependents:   dependents.Get(addr),
			})
		}
	}
}

func (m *migration) calculateDependencies(components collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]) (collections.Map[stackaddrs.AbsComponent, collections.Set[stackaddrs.AbsComponent]], collections.Map[stackaddrs.AbsComponent, collections.Set[stackaddrs.AbsComponent]]) {
	dependencies := collections.NewMap[stackaddrs.AbsComponent, collections.Set[stackaddrs.AbsComponent]]()
	dependents := collections.NewMap[stackaddrs.AbsComponent, collections.Set[stackaddrs.AbsComponent]]()

	// First, we're going to work out the dependencies between components.

	for _, cmpnts := range components.All() {
		for component := range cmpnts.All() {
			addr := stackaddrs.AbsComponent{
				Stack: component.Stack,
				Item:  component.Item.Component,
			}

			if dependencies.HasKey(addr) {
				// Then we've seen another instance of this component before, and
				// we don't need to process it again.
				continue
			}

			stack := m.Migration.Config.Stack(addr.Stack.ConfigAddr())
			if stack == nil {
				// Then we tried to create a component for a stack that doesn't
				// exist in the configuration.
				m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Stack not found", fmt.Sprintf("Stack %s not found in configuration.", addr.Stack.ConfigAddr())))
				continue
			}

			cmpt := stack.Components[component.Item.Component.Name]
			if cmpt == nil {
				// Then we tried to create a component that doesn't exist in
				// the configuration.
				m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Component not found", fmt.Sprintf("Component %s not found in stack %s.", component.Item.Component.Name, addr.Stack.ConfigAddr())))
				continue
			}

			ds := collections.NewSet[stackaddrs.AbsComponent]()
			addDependency := func(cmpt stackaddrs.AbsComponent) {
				ds.Add(cmpt)

				if !dependents.HasKey(cmpt) {
					dependents.Put(cmpt, collections.NewSet[stackaddrs.AbsComponent]())
				}
				dependents.Get(cmpt).Add(addr)
			}
			addDependencies := func(dss collections.Set[stackaddrs.AbsComponent]) {
				for d := range dss.All() {
					addDependency(d)
				}
			}

			// First, check the inputs.

			inputDependencies, inputDiags := m.componentDependenciesFromExpression(cmpt.Inputs, component.Stack, components)
			for _, diag := range inputDiags {
				m.emitDiag(diag)
			}
			addDependencies(inputDependencies)

			// Then, check the depends_on directly.

			for _, traversal := range cmpt.DependsOn {
				dependsOnDependencies, dependsOnDiags := m.componentDependenciesFromTraversal(traversal, component.Stack, components)
				for _, diag := range dependsOnDiags {
					m.emitDiag(diag)
				}
				addDependencies(dependsOnDependencies)
			}

			// Then, check the foreach.

			forEachDependencies, forEachDiags := m.componentDependenciesFromExpression(cmpt.ForEach, component.Stack, components)
			for _, diag := range forEachDiags {
				m.emitDiag(diag)
			}
			addDependencies(forEachDependencies)

			// Finally, we're going to look at the providers, and see if they
			// depend on any other components.

			for _, expr := range cmpt.ProviderConfigs {
				pds, diags := m.providerDependencies(expr, component.Stack, stack, components)
				for _, diag := range diags {
					m.emitDiag(diag)
				}
				for d := range pds.All() {
					addDependency(d)
				}
			}

			// We're happy we got all the dependencies for this component, so we
			// can store them now.

			dependencies.Put(addr, ds)
		}
	}
	return dependencies, dependents
}

// componentDependenciesFromExpression returns a set of components that are
// referenced in the given expression.
func (m *migration) componentDependenciesFromExpression(expr hcl.Expression, current stackaddrs.StackInstance, components collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]) (ds collections.Set[stackaddrs.AbsComponent], diags tfdiags.Diagnostics) {
	ds = collections.NewSet[stackaddrs.AbsComponent]()
	if expr == nil {
		return ds, diags
	}

	for _, v := range expr.Variables() {
		dss, moreDiags := m.componentDependenciesFromTraversal(v, current, components)
		ds.AddAll(dss)
		diags = diags.Append(moreDiags)
	}
	return ds, diags
}

// componentDependenciesFromTraversal returns the component that is referenced
// in the given traversal, if it is a component reference.
func (m *migration) componentDependenciesFromTraversal(traversal hcl.Traversal, current stackaddrs.StackInstance, components collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]) (ds collections.Set[stackaddrs.AbsComponent], diags tfdiags.Diagnostics) {
	ds = collections.NewSet[stackaddrs.AbsComponent]()

	ref, _, moreDiags := stackaddrs.ParseReference(traversal)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		// Then the configuration is invalid, so we'll skip this variable.
		// The user should have ran a separate validation step before
		// performing the migration to catch this.
		return ds, diags
	}

	switch ref := ref.Target.(type) {
	case stackaddrs.Component:
		// We have a reference to a component in the current stack.
		ds.Add(stackaddrs.AbsComponent{
			Stack: current,
			Item:  ref,
		})
		return ds, diags
	case stackaddrs.StackCall:
		targetStackAddress := append(current.ConfigAddr(), stackaddrs.StackStep(ref))
		stack := m.Migration.Config.Stack(targetStackAddress)

		// we have the configurations for the components in this stack, we just
		// need to scope them down to the components that are in the current
		// stack instance.

		for name := range stack.Components {
			configComponentAddress := stackaddrs.ConfigComponent{
				Stack: targetStackAddress,
				Item:  stackaddrs.Component{Name: name},
			}

			if components, ok := components.GetOk(configComponentAddress); ok {
				for component := range components.All() {
					if current.Contains(component.Stack) {
						ds.Add(stackaddrs.AbsComponent{
							Stack: component.Stack,
							Item:  component.Item.Component,
						})
					}
				}
			}
		}
		return ds, diags
	default:
		// This is not a component reference, and we only care about
		// component dependencies.
		return ds, diags
	}
}

func (m *migration) providerDependencies(expr hcl.Expression, current stackaddrs.StackInstance, stack *stackconfig.Stack, components collections.Map[stackaddrs.ConfigComponent, collections.Set[stackaddrs.AbsComponentInstance]]) (ds collections.Set[stackaddrs.AbsComponent], diags tfdiags.Diagnostics) {
	ds = collections.NewSet[stackaddrs.AbsComponent]()
	for _, v := range expr.Variables() {
		ref, _, moreDiags := stackaddrs.ParseReference(v)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// Invalid configuration, so skip it.
			continue
		}

		switch ref := ref.Target.(type) {
		case stackaddrs.ProviderConfigRef:
			config := stack.ProviderConfigs[addrs.LocalProviderConfig{
				LocalName: ref.ProviderLocalName,
				Alias:     ref.Name,
			}]

			dss, moreDiags := m.componentDependenciesFromExpression(config.ForEach, current, components)
			diags = diags.Append(moreDiags)
			ds.AddAll(dss)

			if config.Config == nil {
				// if there is no configuration, then there won't be any
				// dependencies.
				break
			}

			addr, ok := stack.RequiredProviders.ProviderForLocalName(ref.ProviderLocalName)
			if !ok {
				diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s not found in required_providers.", ref.ProviderLocalName)))
				continue
			}

			provider := m.provider(addr)
			if provider == nil {
				// provider should have emitted a diagnostic already
				continue // skip this provider if we can't get the schema
			}

			spec := provider.GetProviderSchema().Provider.Block.DecoderSpec()
			traversals := hcldec.Variables(config.Config, spec)
			for _, traversal := range traversals {
				dss, moreDiags := m.componentDependenciesFromTraversal(traversal, current, components)
				diags = diags.Append(moreDiags)
				ds.AddAll(dss)
			}

		default:
			// This is not a provider reference, and we only care about
			// provider dependencies.
			continue
		}
	}
	return ds, diags
}
