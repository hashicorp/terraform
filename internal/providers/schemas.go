package providers

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// Schemas is an overall container for all of the schemas for all configurable
// objects defined within a particular provider.
//
// The schema for each individual configurable object is represented by nested
// instances of type Schema (singular) within this data structure.
//
// This type used to be known as terraform.ProviderSchema, but moved out here
// as part of our ongoing efforts to shrink down the "terraform" package.
// There's still a type alias at the old name, but we should prefer using
// providers.Schema in new code. However, a consequence of this transitional
// situation is that the "terraform" package still has the responsibility for
// constructing a providers.Schemas object based on responses from the provider
// API; hopefully we'll continue this refactor later so that functions in this
// package totally encapsulate the unmarshalling and include this as part of
// providers.GetProviderSchemaResponse.
type Schemas struct {
	Provider      *configschema.Block
	ProviderMeta  *configschema.Block
	ResourceTypes map[string]*configschema.Block
	DataSources   map[string]*configschema.Block

	ResourceTypeSchemaVersions map[string]uint64
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
