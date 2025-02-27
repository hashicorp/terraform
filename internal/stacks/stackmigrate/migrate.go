// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"fmt"
	"iter"

	"github.com/hashicorp/go-slug/sourceaddrs"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Migration is a struct that aids in migrating a terraform state to a stack configuration.
type Migration struct {
	// Providers is a map of provider addresses available to the stack.
	Providers map[addrs.Provider]providers.Factory

	// PreviousState is the terraform core state that we are migrating from.
	PreviousState *states.State
	Config        *stackconfig.Config
}

// Alias common types to make the code more readable.
type (
	// ConfigComponent is the definition of a component in a stack configuration,
	// and therefore is unique for all instances of a component in a stack.
	Config = stackaddrs.ConfigComponent

	// Every instance of a component in a stack instance has a unique address.
	Instance = stackaddrs.AbsComponentInstance

	// Every instance of a component in a stack has the same AbsComponent address.
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
	*Migration

	emit     func(change stackstate.AppliedChange)
	emitDiag func(diagnostic tfdiags.Diagnostic)

	providers map[addrs.Provider]providers.Interface
	parser    *configs.SourceBundleParser
	configs   map[sourceaddrs.FinalSource]*configs.Config
}

func (m *migration) stateResources() iter.Seq2[addrs.AbsResource, *states.Resource] {
	return func(yield func(addrs.AbsResource, *states.Resource) bool) {
		for _, module := range m.PreviousState.Modules {
			for _, resource := range module.Resources {
				if !yield(resource.Addr, resource) {
					return
				}
			}
		}
	}
}

// moduleConfig returns the module configuration for the component. If the configuration
// has already been loaded, it will be returned from the cache.
func (m *migration) moduleConfig(component *stackconfig.Component) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if component.FinalSourceAddr == nil {
		// if there is no final source address, then the configuration was likely
		// loaded via a shallow load, but we need the full configuration.
		panic("component has no final source address")
	}
	if cfg, ok := m.configs[component.FinalSourceAddr]; ok {
		return cfg, diags
	}
	moduleConfig, diags := component.ModuleConfig(m.parser.Bundle())
	if diags.HasErrors() {
		return nil, diags
	}
	m.configs[component.FinalSourceAddr] = moduleConfig
	return moduleConfig, diags
}

func (m *migration) emitDiags(diags tfdiags.Diagnostics) {
	for _, diag := range diags {
		m.emitDiag(diag)
	}
}

func (m *migration) provider(provider addrs.Provider) (providers.Interface, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if p, ok := m.providers[provider]; ok {
		return p, diags
	}

	factory, ok := m.Migration.Providers[provider]
	if !ok {
		return nil, tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Provider not found", fmt.Sprintf("Provider %s not found in required_providers.", provider.ForDisplay()))}
	}

	p, err := factory()
	if err != nil {
		return nil, tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "Provider initialization failed", fmt.Sprintf("Failed to initialize provider %s: %s", provider.ForDisplay(), err.Error()))}
	}

	m.providers[provider] = p
	return p, diags
}

func (m *migration) close() {
	for addr, provider := range m.providers {
		if err := provider.Close(); err != nil {
			m.emitDiag(tfdiags.Sourceless(tfdiags.Error, "Provider cleanup failed", fmt.Sprintf("Failed to close provider %s: %s", addr.ForDisplay(), err.Error())))
		}
	}
}
