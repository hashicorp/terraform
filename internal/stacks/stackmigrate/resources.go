// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
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

	for _, module := range m.Migration.PreviousState.Modules {
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

			providerAddr, ok := m.getOwningProvider(target, originalModule)
			if !ok {
				// getOwningProvider should have emitted a diagnostic already
				continue
			}

			provider := m.provider(providerAddr.Provider)
			if provider == nil {
				// provider should have emitted a diagnostic already
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

func fixRootAddrs(resources, modules map[string]string) (map[string]string, map[string]string) {
	fixedResources := make(map[string]string, len(resources))
	for resource, component := range resources {
		if !strings.Contains(component, "component.") {
			fixedResources[resource] = "component." + component
		} else {
			fixedResources[resource] = component
		}
	}

	fixedModules := make(map[string]string, len(modules))
	for module, component := range modules {
		if !strings.Contains(component, "component.") {
			fixedModules[module] = "component." + component
		} else {
			fixedModules[module] = component
		}
	}

	return fixedResources, fixedModules
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
	resources, modules = fixRootAddrs(resources, modules)

	parseComponentInstance := func(target string) (Instance, tfdiags.Diagnostics) {
		inst, _, diags := stackaddrs.ParseAbsComponentInstanceStrOnly(target)
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
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Resource not found", fmt.Sprintf("Resource %s not found in mapping.", resource.Resource.String())))
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
		inst.Item.Key = resource.Module[0].InstanceKey
		return stackaddrs.AbsResource{
			Component: inst,
			Item: addrs.AbsResource{
				Module:   resource.Module[1:], // the first module instance is replaced by the component instance
				Resource: resource.Resource,
			},
		}, diags
	} else {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Module not found", fmt.Sprintf("Module %s not found in mapping.", resource.Module[0].Name)))
		return empty, diags
	}
}
