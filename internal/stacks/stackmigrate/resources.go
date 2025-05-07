// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// stackResource represents a resource that was found in the terraform state.
// It contains the stack and component configuration for the resource.
type stackResource struct {
	// The unexpanded resource address
	AbsResource stackaddrs.AbsResource

	// The address of the terraform module that the resource belongs to.
	ContainingModule addrs.Module

	// The stack and component configuration for the resource.
	StackConfig     *stackconfig.Stack
	ComponentConfig *stackconfig.Component

	// The root module configuration for the resource's component.
	ModuleConfig *configs.Config
}

// implement the UniqueKeyer interface for stackResource
// The key of a stackResource pointer is simply itself.
func (r *stackResource) UniqueKey() collections.UniqueKey[*stackResource] {
	return r
}

// implement the UniqueKey interface for stackResource
func (r *stackResource) IsUniqueKey(*stackResource) {}

func (m *migration) migrateResources(resources map[string]string, modules map[string]string) collections.Map[Instance, collections.Set[*stackResource]] {
	components := collections.NewMap[Instance, collections.Set[*stackResource]]()

	// for each resource in the config, we track the instances that belong to the
	// same component.
	trackComponent := func(resource *stackResource) {
		instance := resource.AbsResource.Component
		if !components.HasKey(instance) {
			components.Put(instance, collections.NewSet[*stackResource]())
		}
		components.Get(instance).Add(resource)
	}

	for _, resource := range m.stateResources() {
		// check if the state resource has been requested for migration,
		// either by being in the resources map, or its module being in the modules map.
		// The returned target builds a new address for the resource within the
		// stack component where it will be migrated to.
		target, diags := m.search(resource.Addr, resources, modules)
		if diags.HasErrors() {
			// if there are errors, we can't migrate this resource.
			m.emitDiags(diags)
			continue
		}

		// We have the component address, now load the stack and component configuration
		// for the resource.
		// If this is successful, we can now start adding source information
		// to diagnostics.
		diags = m.loadConfig(target)
		if diags.HasErrors() {
			m.emitDiags(diags)
			continue
		}
		component := target.AbsResource.Component
		componentAddr := target.AbsResource.Item

		trackComponent(target)

		// retrieve the provider that was uses to create the resource instance.
		providerAddr, provider, diags := m.getOwningProvider(target)
		if diags.HasErrors() {
			m.emitDiags(diags)
			continue
		}

		schema := provider.GetProviderSchema().SchemaForResourceType(resource.Addr.Resource.Mode, resource.Addr.Resource.Type)
		if schema.Body == nil {
			m.emitDiags(diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Resource type not found",
				Detail:   fmt.Sprintf("Resource type %s not found in provider schema.", resource.Addr.Resource.Type),
				Subject:  target.ModuleConfig.SourceAddrRange.Ptr(),
			}))
			continue
		}

		for instanceKey, instance := range resource.Instances {
			instanceAddr := stackaddrs.AbsResourceInstance{
				Component: component,
				Item:      componentAddr.Instance(instanceKey),
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
	return components
}

// search searches for the state resource in the resource mappings and when found, converts and returns the relevant
// stackResource.
//
// If the resource or module is nested within the root module, they will be migrated to the component with the address structure retained.
// For example, a resource with the address module.my_module.module.child.aws_instance.foo will be migrated to
// component.my_component.module.child.aws_instance.foo if the corresponding map key is found.
// E.g module.child.aws_instance.foo will be replaced with component.child.aws_instance.foo
func (m *migration) search(resource addrs.AbsResource, resources map[string]string, modules map[string]string) (*stackResource, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &stackResource{
		ContainingModule: resource.Module.Module(),
	}

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
		target, ok := resources[resource.Resource.String()]
		if !ok {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Resource not found", fmt.Sprintf("Resource %q not found in mapping.", resource.Resource.String())))
			return ret, diags
		}

		inst, diags := parseComponentInstance(target)
		if diags.HasErrors() {
			return ret, diags
		}
		ret.AbsResource = stackaddrs.AbsResource{
			Component: inst,
			Item:      resource,
		}
		return ret, diags
	}

	// The resource is in a child module, so we need to find the component.
	// When found, we replace the module with the component instance, i.e
	// a resource of module.child.aws_instance.foo will be replaced with
	// component.child.aws_instance.foo
	if targetComponent, ok := modules[resource.Module[0].Name]; ok {
		inst, diags := parseComponentInstance(targetComponent)
		if diags.HasErrors() {
			return ret, diags
		}
		// retain the instance key
		inst.Item.Key = resource.Module[0].InstanceKey
		ret.AbsResource = stackaddrs.AbsResource{
			Component: inst,
			Item: addrs.AbsResource{
				Module:   resource.Module[1:], // the first module instance is replaced by the component instance
				Resource: resource.Resource,
			},
		}
		return ret, diags
	} else {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Module not found", fmt.Sprintf("Module %q not found in mapping.", resource.Module[0].Name)))
		return ret, diags
	}
}

// getOwningProvider returns the address of the provider configuration,
// as well as the provider instance, that was used to create the given resource instance.
//
// The provided config address is the location within the previous configuration
// and we need to find the corresponding provider configuration in the new
// configuration.

func (m *migration) getOwningProvider(resource *stackResource) (addrs.AbsProviderConfig, providers.Interface, tfdiags.Diagnostics) {
	var ret addrs.AbsProviderConfig

	providerConfig, diags := m.findProviderConfig(resource.ContainingModule, resource.AbsResource.Item.Resource, resource.ModuleConfig)
	if diags.HasErrors() {
		return ret, nil, diags
	}
	component := resource.ComponentConfig
	stackCfg := resource.StackConfig

	// translate the local provider
	expr, ok := component.ProviderConfigs[providerConfig]
	if !ok {
		// Then the module uses a provider not referenced in the component.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider not found for component",
			Detail:   fmt.Sprintf("Provider %q not found in component %q.", providerConfig.LocalName, resource.AbsResource.Component.Item.Component.Name),
			Subject:  component.SourceAddrRange.ToHCL().Ptr(),
		})
		return ret, nil, diags
	}

	vars := expr.Variables()
	if len(vars) != 1 {
		// This should be an exact reference to a single provider, if it's not
		// we can't really do anything.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid provider reference",
			Detail:   "Provider references should be a simple reference to a single provider.",
			Subject:  expr.Range().Ptr(),
		})
		return ret, nil, diags
	}

	ref, _, moreDiags := stackaddrs.ParseReference(vars[0])
	diags = diags.Append(moreDiags)

	switch ref := ref.Target.(type) {
	case stackaddrs.ProviderConfigRef:
		providerAddr, ok := stackCfg.RequiredProviders.ProviderForLocalName(ref.ProviderLocalName)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider not found for component",
				Detail:   fmt.Sprintf("Provider %s was needed by the resource %s but was not found in the stack configuration.", ref.ProviderLocalName, resource.AbsResource.Item.Resource.String()),
				Subject:  component.SourceAddrRange.ToHCL().Ptr(),
			})
			return ret, nil, diags
		}

		addr := addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: providerAddr,
			Alias:    providerConfig.Alias, // we still use the alias from the module provider as this is referenced as if from within the module.
		}

		provider, pDiags := m.provider(providerAddr)
		// pull in source information for diagnostics if available.
		for _, diag := range pDiags {
			if diag.Source().Subject == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: diag.Severity().ToHCL(),
					Summary:  diag.Description().Summary,
					Detail:   diag.Description().Detail,
					Subject:  resource.ComponentConfig.SourceAddrRange.ToHCL().Ptr(),
				})
			}
		}

		return addr, provider, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   "Non-provider reference found in provider configuration.",
			Subject:  expr.Range().Ptr(),
		})
		return ret, nil, diags
	}
}

// findProviderConfig recursively searches through the module configuration to find the provider
// that was used to create the resource instance.
func (m *migration) findProviderConfig(module addrs.Module, resource addrs.Resource, config *configs.Config) (addrs.LocalProviderConfig, tfdiags.Diagnostics) {
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

	// the address points to a nested module, so we continue the search
	// within the next module's configuration.
	provider, moreDiags := m.findProviderConfig(module[1:], resource, next)
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

func (m *migration) loadConfig(resource *stackResource) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	instance := resource.AbsResource.Component
	stack := m.Config.Stack(instance.Stack.ConfigAddr())
	if stack == nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Stack not found", fmt.Sprintf("Stack %q not found in configuration.", instance.Stack.ConfigAddr())))
	}
	resource.StackConfig = stack

	component := m.Config.Component(stackaddrs.ConfigComponentForAbsInstance(instance))
	if component == nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Component not found", fmt.Sprintf("Component %q not found in stack %q.", instance.Item.Component.Name, instance.Stack.ConfigAddr())))
	}

	resource.ComponentConfig = component

	moduleConfig, diags := m.moduleConfig(component)
	if diags.HasErrors() {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module configuration not found",
			Detail:   fmt.Sprintf("Module configuration for component %q not found", instance.Item.Component.Name),
			Subject:  component.SourceAddrRange.ToHCL().Ptr(),
		})
	}
	resource.ModuleConfig = moduleConfig
	return diags
}
