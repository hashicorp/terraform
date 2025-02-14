// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (m *migration) migrateResources(resources map[string]string, modules map[string]string) collections.Map[Config, collections.Set[Instance]] {
	components := collections.NewMap[Config, collections.Set[Instance]]()

	// for each component in the config, we track the instances that are associated with it.
	trackComponent := func(instance Instance) {
		configComponent := Config{
			Stack: instance.Stack.ConfigAddr(),
			Item:  instance.Item.Component,
		}
		if !components.HasKey(configComponent) {
			components.Put(configComponent, collections.NewSet[Instance]())
		}
		components.Get(configComponent).Add(instance)
	}

	for _, module := range m.PreviousState.Modules {
		for _, resource := range module.Resources {
			// the search will replace the target's module with the component instance,
			// therefore, any further function call that needs the original module
			// should retrieve it from the original resource.
			originalModule := resource.Addr.Module.Module()

			target, diags := m.search(resource.Addr, resources, modules)
			if diags.HasErrors() {
				// if there are errors, we can't migrate this resource.
				m.emitDiags(diags)
				continue
			}

			trackComponent(target.Component) // record the component instance

			providerAddr, diags := m.getOwningProvider(target, originalModule)
			if diags.HasErrors() {
				m.emitDiags(diags)
				continue
			}

			provider, diags := m.provider(providerAddr.Provider)
			if diags.HasErrors() {
				m.emitDiags(diags)
				continue
			}
			schema, _ := provider.GetProviderSchema().SchemaForResourceType(resource.Addr.Resource.Mode, resource.Addr.Resource.Type)
			if schema == nil {
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
					ProviderConfigAddr: providerAddr,
					Schema:             schema,
				})

				for deposedKey, deposed := range instance.Deposed {
					m.emit(&stackstate.AppliedChangeResourceInstanceObject{
						ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
							Component: instanceAddr.Component,
							Item:      instanceAddr.Item.DeposedObject(deposedKey),
						},
						NewStateSrc:        deposed,
						ProviderConfigAddr: providerAddr,
						Schema:             schema,
					})
				}
			}
		}
	}
	return components
}

// search searches for the state resource in the resource mappings and when found, converts and returns the relevant
// stackaddrs.AbsResourceInstance.
//
// If the resource or module is nested within the root module, they will be migrated to the component with the address structure retained.
// For example, a resource with the address module.my_module.module.child.aws_instance.foo will be migrated to
// component.my_component.module.child.aws_instance.foo if the corresponding map key is found.
// E.g module.child.aws_instance.foo will be replaced with component.child.aws_instance.foo
func (m *migration) search(resource addrs.AbsResource, resources map[string]string, modules map[string]string) (stackaddrs.AbsResource, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var empty stackaddrs.AbsResource

	parseComponentInstance := func(target string) (Instance, tfdiags.Diagnostics) {
		fullTarget := "component." + strings.TrimPrefix(target, "component.")
		if len(strings.Split(fullTarget, ".")) > 2 {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Invalid component instance", fmt.Sprintf("Only root component instances are allowed, got %q", target)))
			return Instance{}, diags
		}
		inst, _, diags := stackaddrs.ParseAbsComponentInstanceStrOnly(fullTarget)
		return inst, diags
	}

	if resource.Module.IsRoot() {
		// If there is no resource mapping, we check for a root module mapping
		target, ok := resources[resource.Resource.String()]
		if !ok {
			target, ok = modules[addrs.RootModule.String()]
		}

		if ok {
			inst, diags := parseComponentInstance(target)
			if diags.HasErrors() {
				return empty, diags
			}
			return stackaddrs.AbsResource{
				Component: inst,
				Item:      resource,
			}, diags
		} else {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Resource not found", fmt.Sprintf("Resource %q not found in mapping.", resource.Resource.String())))
			return empty, diags
		}
	}

	// The resource is in a child module, so we need to find the component.
	// When found, we replace the module with the component instance, i.e
	// a resource of module.child.aws_instance.foo will be replaced with
	// component.child.aws_instance.foo
	if targetComponent, ok := modules[resource.Module[0].Name]; ok {
		inst, diags := parseComponentInstance(targetComponent)
		if diags.HasErrors() {
			return empty, diags
		}
		// retain the instance key
		inst.Item.Key = resource.Module[0].InstanceKey
		return stackaddrs.AbsResource{
			Component: inst,
			Item: addrs.AbsResource{
				Module:   resource.Module[1:], // the first module instance is replaced by the component instance
				Resource: resource.Resource,
			},
		}, diags
	} else {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Module not found", fmt.Sprintf("Module %q not found in mapping.", resource.Module[0].Name)))
		return empty, diags
	}
}

// getOwningProvider returns the address of the provider configuration
// that was used to create the given resource instance.
//
// The provided config address is the location within the previous configuration
// and we need to find the corresponding provider configuration in the new
// configuration.
func (m *migration) getOwningProvider(target stackaddrs.AbsResource, origModule addrs.Module) (addrs.AbsProviderConfig, tfdiags.Diagnostics) {
	stack, component, diags := m.stackAndComponentConfig(target.Component)
	if diags.HasErrors() {
		return addrs.AbsProviderConfig{}, diags
	}

	moduleConfig, diags := m.moduleConfig(component.FinalSourceAddr)
	if diags.HasErrors() {
		return addrs.AbsProviderConfig{}, diags
	}

	moduleProvider, diags := m.findProviderInModule(origModule, target.Item.Resource, moduleConfig)
	if diags.HasErrors() {
		return addrs.AbsProviderConfig{}, diags
	}

	// translate the local provider
	expr, ok := component.ProviderConfigs[moduleProvider]
	if !ok {
		// Then the module uses a provider not referenced in the component.
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %q not found in component %q.", moduleProvider.LocalName, target.Component.Item.Component.Name)))
		return addrs.AbsProviderConfig{}, diags
	}

	vars := expr.Variables()
	if len(vars) != 1 {
		// This should be an exact reference to a single provider, if it's not
		// we can't really do anything.
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Invalid reference", "Provider references should be a simple reference to a single provider."))
		return addrs.AbsProviderConfig{}, diags
	}

	ref, _, moreDiags := stackaddrs.ParseReference(vars[0])
	diags = diags.Append(moreDiags)

	switch ref := ref.Target.(type) {
	case stackaddrs.ProviderConfigRef:
		provider, ok := stack.RequiredProviders.ProviderForLocalName(ref.ProviderLocalName)
		if !ok {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s was needed by the resource %s but was not found in the stack configuration.", ref.ProviderLocalName, target.Item.Resource.String())))
			return addrs.AbsProviderConfig{}, diags
		}

		return addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: provider,
			Alias:    moduleProvider.Alias, // we still use the alias from the module provider as this is referenced as if from within the module.
		}, diags
	default:
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Invalid reference", "Non-provider reference found in provider configuration."))
		return addrs.AbsProviderConfig{}, diags
	}
}

// findProviderInModule searches for the provider configuration that was used to create the given resource instance.
func (m *migration) findProviderInModule(module addrs.Module, resource addrs.Resource, config *configs.Config) (addrs.LocalProviderConfig, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if module.IsRoot() {
		r := config.Module.ResourceByAddr(resource)
		if r == nil {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Resource %q not found in root module.", resource.String())))
			return addrs.LocalProviderConfig{}, diags
		}

		return r.ProviderConfigAddr(), diags
	}

	next, ok := config.Children[module[0]]
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Module %q not found in configuration.", module[0])))
		return addrs.LocalProviderConfig{}, diags
	}

	// the address points to another module, so we continue the search
	// within the next module's configuration.
	provider, moreDiags := m.findProviderInModule(module[1:], resource, next)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return addrs.LocalProviderConfig{}, diags
	}

	call, ok := config.Module.ModuleCalls[module[0]]
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Module call %q not found in configuration.", module[0])))
		return addrs.LocalProviderConfig{}, diags
	}

	for _, p := range call.Providers {
		if p.InChild.Name == provider.LocalName && p.InChild.Alias == provider.Alias {
			return p.InParent.Addr(), diags
		}
	}

	diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %q not found in module %q.", provider.LocalName, module[0])))
	return addrs.LocalProviderConfig{}, diags
}

func (m *migration) stackAndComponentConfig(instance Instance) (*stackconfig.Stack, *stackconfig.Component, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	stack := m.Config.Stack(instance.Stack.ConfigAddr())
	if stack == nil {
		return nil, nil, diags.Append(tfdiags.Sourceless(tfdiags.Error, "Stack not found", fmt.Sprintf("Stack %q not found in configuration.", instance.Stack.ConfigAddr())))
	}

	component := m.Config.Component(stackaddrs.ConfigComponentForAbsInstance(instance))
	if component == nil {
		return stack, nil, diags.Append(tfdiags.Sourceless(tfdiags.Error, "Component not found", fmt.Sprintf("Component %q not found in stack %q.", instance.Item.Component.Name, instance.Stack.ConfigAddr())))
	}

	return stack, component, diags
}
