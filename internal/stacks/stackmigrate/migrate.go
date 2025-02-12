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
	*Migration

	emit     func(change stackstate.AppliedChange)
	emitDiag func(diagnostic tfdiags.Diagnostic)

	providers map[addrs.Provider]providers.Interface
	parser    *configs.SourceBundleParser
	configs   map[sourceaddrs.FinalSource]*configs.Config
}

// moduleConfig returns the configuration for the given address. If the configuration
// has already been loaded, it will be returned from the cache.
func (m *migration) moduleConfig(addr sourceaddrs.FinalSource) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if cfg, ok := m.configs[addr]; ok {
		return cfg, diags
	}

	if !m.parser.IsConfigDir(addr) {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Component configuration not found", fmt.Sprintf("Component configuration not found at %s", addr)))
		return nil, diags
	}

	module, moreDiags := m.parser.LoadConfigDir(addr)
	diags = diags.Append(moreDiags)

	if module != nil {
		walker := stackparser.NewSourceBundleModuleWalker(addr, m.Config.Sources, m.parser)
		config, moreDiags := configs.BuildConfig(module, walker, nil)
		diags = diags.Append(moreDiags)

		m.configs[addr] = config
	}

	return m.configs[addr], diags
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
