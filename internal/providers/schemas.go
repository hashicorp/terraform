// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// ProviderSchema is an overall container for all of the schemas for all
// configurable objects defined within a particular provider. All storage of
// provider schemas should use this type.
type ProviderSchema = GetProviderSchemaResponse

// SchemaForResourceType attempts to find a schema for the given mode and type.
// Returns nil if no such schema is available.
func (ss ProviderSchema) SchemaForResourceType(mode addrs.ResourceMode, typeName string) (schema *configschema.Block, version uint64) {
	switch mode {
	case addrs.ManagedResourceMode:
		res := ss.ResourceTypes[typeName]
		return res.Block, uint64(res.Version)
	case addrs.DataResourceMode:
		// Data resources don't have schema versions right now, since state is discarded for each refresh
		return ss.DataSources[typeName].Block, 0
	case addrs.EphemeralResourceMode:
		// Ephemeral resources don't have schema versions because their objects never outlive a single phase
		return ss.EphemeralResourceTypes[typeName].Block, 0
	default:
		// Shouldn't happen, because the above cases are comprehensive.
		return nil, 0
	}
}

// SchemaForResourceAddr attempts to find a schema for the mode and type from
// the given resource address. Returns nil if no such schema is available.
func (ss ProviderSchema) SchemaForResourceAddr(addr addrs.Resource) (schema *configschema.Block, version uint64) {
	return ss.SchemaForResourceType(addr.Mode, addr.Type)
}
