package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/terraform"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// providers is the top-level object returned when exporting provider schemas
type providers struct {
	FormatVersion string              `json:"format_version"`
	Schemas       map[string]Provider `json:"provider_schemas"`
}

type Provider struct {
	Provider          *schema            `json:"provider,omitempty"`
	ResourceSchemas   map[string]*schema `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]*schema `json:"data_source_schemas,omitempty"`
}

func newProviders() *providers {
	schemas := make(map[string]Provider)
	return &providers{
		FormatVersion: FormatVersion,
		Schemas:       schemas,
	}
}

func Marshal(s *terraform.Schemas) ([]byte, error) {
	if len(s.Providers) == 0 {
		return nil, nil
	}

	providers := newProviders()

	for k, v := range s.Providers {
		providers.Schemas[k] = marshalProvider(v)
	}

	// add some polish for the human consumers
	ret, err := json.MarshalIndent(providers, "", "  ")
	return ret, err
}

func marshalProvider(tps *terraform.ProviderSchema) Provider {
	if tps == nil {
		return Provider{}
	}

	var ps *schema
	var rs, ds map[string]*schema

	if tps.Provider != nil {
		ps = marshalSchema(tps.Provider)
	}

	if tps.ResourceTypes != nil {
		rs = marshalSchemas(tps.ResourceTypes, tps.ResourceTypeSchemaVersions)
	}

	if tps.DataSources != nil {
		ds = marshalSchemas(tps.DataSources, tps.ResourceTypeSchemaVersions)
	}

	return Provider{
		Provider:          ps,
		ResourceSchemas:   rs,
		DataSourceSchemas: ds,
	}
}
