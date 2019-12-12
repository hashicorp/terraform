package schemas

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
)

// Schemas is a container for various kinds of schema that Terraform needs
// during processing.
type Schemas struct {
	Providers    map[string]*ProviderSchema
	Provisioners map[string]*configschema.Block
}

// ProviderSchema returns the entire ProviderSchema object that was produced
// by the plugin for the given provider, or nil if no such schema is available.
//
// It's usually better to go use the more precise methods offered by type
// Schemas to handle this detail automatically.
func (ss *Schemas) ProviderSchema(typeName string) *ProviderSchema {
	if ss.Providers == nil {
		return nil
	}
	return ss.Providers[typeName]
}

// ProviderConfig returns the schema for the provider configuration of the
// given provider type, or nil if no such schema is available.
func (ss *Schemas) ProviderConfig(typeName string) *configschema.Block {
	ps := ss.ProviderSchema(typeName)
	if ps == nil {
		return nil
	}
	return ps.Provider
}

// ResourceTypeConfig returns the schema for the configuration of a given
// resource type belonging to a given provider type, or nil of no such
// schema is available.
//
// In many cases the provider type is inferrable from the resource type name,
// but this is not always true because users can override the provider for
// a resource using the "provider" meta-argument. Therefore it's important to
// always pass the correct provider name, even though it many cases it feels
// redundant.
func (ss *Schemas) ResourceTypeConfig(providerType string, resourceMode addrs.ResourceMode, resourceType string) (block *configschema.Block, schemaVersion uint64) {
	ps := ss.ProviderSchema(providerType)
	if ps == nil || ps.ResourceTypes == nil {
		return nil, 0
	}
	return ps.SchemaForResourceType(resourceMode, resourceType)
}

// ProvisionerConfig returns the schema for the configuration of a given
// provisioner, or nil of no such schema is available.
func (ss *Schemas) ProvisionerConfig(name string) *configschema.Block {
	return ss.Provisioners[name]
}

// ProviderSchema represents the schema for a provider's own configuration
// and the configuration for some or all of its resources and data sources.
//
// The completeness of this structure depends on how it was constructed.
// When constructed for a configuration, it will generally include only
// resource types and data sources used by that configuration.
type ProviderSchema struct {
	Provider      *configschema.Block
	ResourceTypes map[string]*configschema.Block
	DataSources   map[string]*configschema.Block

	ResourceTypeSchemaVersions map[string]uint64
}

// SchemaForResourceType attempts to find a schema for the given mode and type.
// Returns nil if no such schema is available.
func (ps *ProviderSchema) SchemaForResourceType(mode addrs.ResourceMode, typeName string) (schema *configschema.Block, version uint64) {
	switch mode {
	case addrs.ManagedResourceMode:
		return ps.ResourceTypes[typeName], ps.ResourceTypeSchemaVersions[typeName]
	case addrs.DataResourceMode:
		// Data resources don't have schema versions right now, since state is discarded for each refresh
		return ps.DataSources[typeName], 0
	default:
		// Shouldn't happen, because the above cases are comprehensive.
		return nil, 0
	}
}

// SchemaForResourceAddr attempts to find a schema for the mode and type from
// the given resource address. Returns nil if no such schema is available.
func (ps *ProviderSchema) SchemaForResourceAddr(addr addrs.Resource) (schema *configschema.Block, version uint64) {
	return ps.SchemaForResourceType(addr.Mode, addr.Type)
}

// ProviderSchemaRequest is used to describe to a ResourceProvider which
// aspects of schema are required, when calling the GetSchema method.
type ProviderSchemaRequest struct {
	ResourceTypes []string
	DataSources   []string
}
