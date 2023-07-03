// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package providers

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Schemas is an overall container for all of the schemas for all configurable
// objects defined within a particular provider.
//
// The schema for each individual configurable object is represented by nested
// instances of type Schema (singular) within this data structure.
type Schemas struct {
	// Provider is the schema for the provider itself.
	Provider *configschema.Block

	// ProviderMeta is the schema for the provider's meta info in a module
	ProviderMeta *configschema.Block

	// ResourceTypes map the resource type name to that type's schema.
	ResourceTypes              map[string]*configschema.Block
	ResourceTypeSchemaVersions map[string]uint64

	// DataSources maps the data source name to that data source's schema.
	DataSources map[string]*configschema.Block

	// ServerCapabilities lists optional features supported by the provider.
	ServerCapabilities ServerCapabilities

	// Diagnostics contains any warnings or errors from the method call.
	// While diagnostics are only relevant to the initial call, we add these to
	// the cached structure so that concurrent calls can handle failures
	// gracefully when the original call did not succeed.
	// TODO: can we be sure the original failure get handled correctly, and
	// ignore this entirely?
	Diagnostics tfdiags.Diagnostics
}

// SchemaForResourceType attempts to find a schema for the given mode and type.
// Returns nil if no such schema is available.
func (ss *Schemas) SchemaForResourceType(mode addrs.ResourceMode, typeName string) (schema *configschema.Block, version uint64) {
	switch mode {
	case addrs.ManagedResourceMode:
		return ss.ResourceTypes[typeName], ss.ResourceTypeSchemaVersions[typeName]
	case addrs.DataResourceMode:
		// Data resources don't have schema versions right now, since state is discarded for each refresh
		return ss.DataSources[typeName], 0
	default:
		// Shouldn't happen, because the above cases are comprehensive.
		return nil, 0
	}
}

// SchemaForResourceAddr attempts to find a schema for the mode and type from
// the given resource address. Returns nil if no such schema is available.
func (ss *Schemas) SchemaForResourceAddr(addr addrs.Resource) (schema *configschema.Block, version uint64) {
	return ss.SchemaForResourceType(addr.Mode, addr.Type)
}

func SchemaResponseToSchemas(resp GetProviderSchemaResponse) *Schemas {
	var schemas = &Schemas{
		ResourceTypes:              make(map[string]*configschema.Block),
		ResourceTypeSchemaVersions: make(map[string]uint64),
		DataSources:                make(map[string]*configschema.Block),
		ServerCapabilities:         resp.ServerCapabilities,
		Diagnostics:                resp.Diagnostics,
	}

	schemas.Provider = resp.Provider.Block
	schemas.ProviderMeta = resp.ProviderMeta.Block

	for name, res := range resp.ResourceTypes {
		schemas.ResourceTypes[name] = res.Block
		schemas.ResourceTypeSchemaVersions[name] = uint64(res.Version)
	}

	for name, dat := range resp.DataSources {
		schemas.DataSources[name] = dat.Block
	}

	return schemas
}

// Schema pairs a provider or resource schema with that schema's version.
// This is used to be able to upgrade the schema in UpgradeResourceState.
//
// This describes the schema for a single object within a provider. Type
// "Schemas" (plural) instead represents the overall collection of schemas
// for everything within a particular provider.
type Schema struct {
	Version int64
	Block   *configschema.Block
}
