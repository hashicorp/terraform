// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderSchema is an overall container for all of the schemas for all
// configurable objects defined within a particular provider. All storage of
// provider schemas should use this type.
type ProviderSchema = GetProviderSchemaResponse

// SchemaForResourceType attempts to find a schema for the given mode and type.
// Returns an empty schema if none is available.
func (ss ProviderSchema) SchemaForResourceType(mode addrs.ResourceMode, typeName string) (schema Schema) {
	switch mode {
	case addrs.ManagedResourceMode:
		return ss.ResourceTypes[typeName]
	case addrs.DataResourceMode:
		return ss.DataSources[typeName]
	case addrs.EphemeralResourceMode:
		return ss.EphemeralResourceTypes[typeName]
	case addrs.ListResourceMode:
		return ss.ListResourceTypes[typeName]
	default:
		// Shouldn't happen, because the above cases are comprehensive.
		return Schema{}
	}
}

// SchemaForResourceAddr attempts to find a schema for the mode and type from
// the given resource address. Returns an empty schema if none is available.
func (ss ProviderSchema) SchemaForResourceAddr(addr addrs.Resource) (schema Schema) {
	return ss.SchemaForResourceType(addr.Mode, addr.Type)
}

type ResourceIdentitySchemas = GetResourceIdentitySchemasResponse
