// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (m *migration) migrateComponents(components collections.Map[Instance, collections.Set[*stackResource]]) {
	// We need to calculate the dependencies between components, so we can
	// populate the dependencies and dependents fields in the component instances.
	dependencies, dependents := m.calculateDependencies(components)

	for instance := range components.All() {
		// We need to see the inputs and outputs from the component, so we can
		// create the component instance with the correct values.
		// ignore the diag because we already found this when loading the config.
		config, _ := m.moduleConfig(m.Config.Component(stackaddrs.ConfigComponentForAbsInstance(instance)))

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
		addr := AbsComponent{
			Stack: instance.Stack,
			Item:  instance.Item.Component,
		}

		// We emit a change a change for each component instance
		m.emit(&stackstate.AppliedChangeComponentInstance{
			ComponentAddr: AbsComponent{
				Stack: stackaddrs.RootStackInstance,
				Item:  instance.Item.Component,
			},
			ComponentInstanceAddr: instance,

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

func (m *migration) calculateDependencies(components collections.Map[Instance, collections.Set[*stackResource]]) (collections.Map[AbsComponent, collections.Set[AbsComponent]], collections.Map[AbsComponent, collections.Set[AbsComponent]]) {
	// The dependency map cares only about config components rather than instances,
	// so we need to convert the map to use the config component address.
	cfgComponents := collections.NewMap[AbsComponent, collections.Set[*stackResource]]()
	for in, cmpnts := range components.All() {
		cfgComponents.Put(AbsComponent{
			Stack: in.Stack,
			Item:  in.Item.Component,
		}, cmpnts)
	}

	dependencies := collections.NewMap[AbsComponent, collections.Set[AbsComponent]]()
	dependents := collections.NewMap[AbsComponent, collections.Set[AbsComponent]]()

	// First, we're going to work out the dependencies between components.
	for addr, cmpnts := range cfgComponents.All() {
		for resource := range cmpnts.All() {
			instance := resource.AbsResourceInstance.Component

			compDepSet := collections.NewSet[AbsComponent]()

			// We collect the component's dependencies, and also
			// add the component to the dependent set of its dependencies.
			addDependencies := func(dss collections.Set[AbsComponent]) {
				compDepSet.AddAll(dss)
				for cmpt := range dss.All() {
					if !dependents.HasKey(cmpt) {
						dependents.Put(cmpt, collections.NewSet[AbsComponent]())
					}
					dependents.Get(cmpt).Add(addr)
				}
			}

			component := resource.ComponentConfig
			stack := resource.StackConfig
			// First, check the inputs.
			inputDependencies, inputDiags := m.componentDependenciesFromExpression(component.Inputs, instance.Stack, cfgComponents)
			m.emitDiags(inputDiags)
			addDependencies(inputDependencies)

			// Then, check the depends_on directly.
			for _, traversal := range component.DependsOn {
				dependsOnDependencies, dependsOnDiags := m.componentDependenciesFromTraversal(traversal, instance.Stack, cfgComponents)
				m.emitDiags(dependsOnDiags)
				addDependencies(dependsOnDependencies)
			}

			// Then, check the foreach.
			forEachDependencies, forEachDiags := m.componentDependenciesFromExpression(component.ForEach, instance.Stack, cfgComponents)
			m.emitDiags(forEachDiags)
			addDependencies(forEachDependencies)

			// Finally, we're going to look at the providers, and see if they
			// depend on any other components.
			for _, expr := range component.ProviderConfigs {
				pds, diags := m.providerDependencies(expr, instance.Stack, stack, cfgComponents)
				m.emitDiags(diags)
				addDependencies(pds)
			}

			// We're happy we got all the dependencies for this component, so we
			// can store them now.
			dependencies.Put(addr, compDepSet)
		}
	}
	return dependencies, dependents
}

// componentDependenciesFromExpression returns a set of components that are
// referenced in the given expression.
func (m *migration) componentDependenciesFromExpression(expr hcl.Expression, current stackaddrs.StackInstance, components collections.Map[AbsComponent, collections.Set[*stackResource]]) (ds collections.Set[AbsComponent], diags tfdiags.Diagnostics) {
	ds = collections.NewSet[AbsComponent]()
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
func (m *migration) componentDependenciesFromTraversal(traversal hcl.Traversal, current stackaddrs.StackInstance, components collections.Map[AbsComponent, collections.Set[*stackResource]]) (deps collections.Set[AbsComponent], diags tfdiags.Diagnostics) {
	deps = collections.NewSet[AbsComponent]()

	parsed, _, moreDiags := stackaddrs.ParseReference(traversal)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		// Then the configuration is invalid, so we'll skip this variable.
		// The user should have ran a separate validation step before
		// performing the migration to catch this.
		return deps, diags
	}

	switch ref := parsed.Target.(type) {
	case stackaddrs.Component:
		// We have a reference to a component in the current stack.
		deps.Add(AbsComponent{
			Stack: current,
			Item:  ref,
		})
		return deps, diags
	case stackaddrs.StackCall:
		targetStackAddress := append(current.ConfigAddr(), stackaddrs.StackStep(ref))
		stack := m.Config.Stack(targetStackAddress)

		if stack == nil {
			// reference to a stack that does not exist in the configuration.
			diags = diags.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Stack not found",
				Detail:   fmt.Sprintf("Stack %q not found in configuration.", targetStackAddress),
				Subject:  parsed.SourceRange.ToHCL().Ptr(),
			})
			return deps, diags
		}

		for name := range stack.Components {
			// If the component in the stack call is part of the mapping, then it will
			// be present in the map, and we will add it to the dependencies.
			// Otherwise, we will ignore it.
			componentAddr := AbsComponent{
				Stack: current.Child(ref.Name, addrs.NoKey),
				Item:  stackaddrs.Component{Name: name},
			}

			if _, ok := components.GetOk(componentAddr); ok {
				deps.Add(componentAddr)
			}
		}
		return deps, diags
	default:
		// This is not a component reference, and we only care about
		// component dependencies.
		return deps, diags
	}
}

func (m *migration) providerDependencies(expr hcl.Expression, current stackaddrs.StackInstance, stack *stackconfig.Stack, components collections.Map[AbsComponent, collections.Set[*stackResource]]) (ds collections.Set[AbsComponent], diags tfdiags.Diagnostics) {
	ds = collections.NewSet[AbsComponent]()
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

			provider, pDiags := m.provider(addr)
			if pDiags.HasErrors() {
				diags = diags.Append(pDiags)
				continue // skip this provider if we can't get the schema
			}

			spec := provider.GetProviderSchema().Provider.Body.DecoderSpec()
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
