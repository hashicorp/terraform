package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/terraform"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "1.0"

// providers is the top-level object returned when exporting provider schemas
type providers struct {
	FormatVersion string               `json:"format_version"`
	Schemas       map[string]*Provider `json:"provider_schemas,omitempty"`
}

type Provider struct {
	Provider          *Schema            `json:"provider,omitempty"`
	ResourceSchemas   map[string]*Schema `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]*Schema `json:"data_source_schemas,omitempty"`
}

func newProviders() *providers {
	schemas := make(map[string]*Provider)
	return &providers{
		FormatVersion: FormatVersion,
		Schemas:       schemas,
	}
}

// MarshalForRenderer converts the provided internation representation of the
// schema into the public structured JSON versions.
//
// This is a format that can be read by the structured plan renderer.
func MarshalForRenderer(s *terraform.Schemas) map[string]*Provider {
	schemas := make(map[string]*Provider, len(s.Providers))
	for k, v := range s.Providers {
		schemas[k.String()] = marshalProvider(v)
	}
	return schemas
}

func Marshal(s *terraform.Schemas) ([]byte, error) {
	providers := newProviders()
	providers.Schemas = MarshalForRenderer(s)
	ret, err := json.Marshal(providers)
	return ret, err
}

func marshalProvider(tps *terraform.ProviderSchema) *Provider {
	if tps == nil {
		return &Provider{}
	}

	var ps *Schema
	var rs, ds map[string]*Schema

	if tps.Provider != nil {
		ps = marshalSchema(tps.Provider)
	}

	if tps.ResourceTypes != nil {
		rs = marshalSchemas(tps.ResourceTypes, tps.ResourceTypeSchemaVersions)
	}

	if tps.DataSources != nil {
		ds = marshalSchemas(tps.DataSources, tps.ResourceTypeSchemaVersions)
	}

	return &Provider{
		Provider:          ps,
		ResourceSchemas:   rs,
		DataSourceSchemas: ds,
	}
}
