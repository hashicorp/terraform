package terraform

import (
	"github.com/hashicorp/terraform/config/configschema"
)

type Schemas struct {
	Providers ProviderSchemas
}

// ProviderSchemas is a map from provider names to provider schemas.
//
// The names in this map are the direct plugin name (e.g. "aws") rather than
// any alias name (e.g. "aws.foo"), since.
type ProviderSchemas map[string]*ProviderSchema

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
}

// ProviderSchemaRequest is used to describe to a ResourceProvider which
// aspects of schema are required, when calling the GetSchema method.
type ProviderSchemaRequest struct {
	ResourceTypes []string
	DataSources   []string
}
