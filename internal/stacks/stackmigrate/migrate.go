// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"

	"github.com/hashicorp/go-slug/sourceaddrs"

	"github.com/hashicorp/terraform/internal/addrs"
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

// Alias common types to make the code more readable.
type (
	Config       = stackaddrs.ConfigComponent
	Instance     = stackaddrs.AbsComponentInstance
	AbsComponent = stackaddrs.AbsComponent
)

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

// moduleConfig returns the configuration for the given address. If the configuration
// has already been loaded, it will be returned from the cache.
func (m *migration) moduleConfig(addr sourceaddrs.FinalSource) *configs.Config {
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

	if module != nil {
		walker := stackparser.NewSourceBundleModuleWalker(addr, m.Migration.Config.Sources, m.parser)
		config, moreDiags := configs.BuildConfig(module, walker, nil)
		diags = diags.Append(moreDiags)

		m.configs[addr] = config
	}

	m.emitDiags(diags)
	return m.configs[addr]
}

func (m *migration) emitDiags(diags tfdiags.Diagnostics) {
	for _, diag := range diags {
		m.emitDiag(diag)
	}
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

// getOwningProvider returns the address of the provider configuration
// that was used to create the given resource instance.
//
// The provided config address is the location within the previous configuration
// and we need to find the corresponding provider configuration in the new
// configuration.
func (m *migration) getOwningProvider(target stackaddrs.AbsResource, origModule addrs.Module) (addrs.AbsProviderConfig, bool) {
	stack, component := m.getStackComponent(target.Component)
	if stack == nil || component == nil {
		// We should have emitted diagnostics for this already.
		return addrs.AbsProviderConfig{}, false
	}

	moduleConfig := m.moduleConfig(component.FinalSourceAddr)
	if moduleConfig == nil {
		// We should have emitted diagnostics for this already.
		return addrs.AbsProviderConfig{}, false
	}

	moduleProvider, ok := m.findProviderInModule(origModule, target.Item.Resource, moduleConfig)
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
	m.emitDiags(diags)

	switch ref := ref.Target.(type) {
	case stackaddrs.ProviderConfigRef:
		provider, ok := stack.RequiredProviders.ProviderForLocalName(ref.ProviderLocalName)
		if !ok {
			m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s was needed by the resource %s but was not found in the stack configuration.", ref.ProviderLocalName, target.Item.Resource.String())))
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
	if module.IsRoot() {
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

	// if the nested module does not have a provide, TODO: Test for it

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

func (m *migration) getStackComponent(instance Instance) (stack *stackconfig.Stack, component *stackconfig.Component) {
	cfg := m.Migration.Config
	stack = cfg.Stack(instance.Stack.ConfigAddr())
	if stack == nil {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Stack not found", fmt.Sprintf("Stack %s not found in configuration.", instance.Stack.ConfigAddr())))
		return nil, nil
	}

	component = cfg.Component(stackaddrs.ConfigComponentForAbsInstance(instance))
	if component == nil {
		m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Component not found", fmt.Sprintf("Component %s not found in stack %s.", instance.Item.Component.Name, instance.Stack.ConfigAddr())))
		return stack, nil
	}

	return stack, component
}
