// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/schemarepo/loadschemas"
	"github.com/hashicorp/terraform/internal/states"
)

// contextPlugins is a deprecated old name for loadschemas.Plugins
type contextPlugins = loadschemas.Plugins

func newContextPlugins(
	providerFactories map[addrs.Provider]providers.Factory,
	provisionerFactories map[string]provisioners.Factory,
	preloadedProviderSchemas map[addrs.Provider]providers.ProviderSchema,
) *loadschemas.Plugins {
	return loadschemas.NewPlugins(providerFactories, provisionerFactories, preloadedProviderSchemas)
}

// Schemas is a deprecated old name for schemarepo.Schemas
type Schemas = schemarepo.Schemas

func loadSchemas(config *configs.Config, state *states.State, plugins *loadschemas.Plugins) (*schemarepo.Schemas, error) {
	return loadschemas.LoadSchemas(config, state, plugins)
}
