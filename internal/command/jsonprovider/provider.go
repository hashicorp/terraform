// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonprovider

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/command/jsonfunction"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "1.0"

// Providers is the top-level object returned when exporting provider schemas
type Providers struct {
	FormatVersion string               `json:"format_version"`
	Schemas       map[string]*Provider `json:"provider_schemas,omitempty"`
}

type Provider struct {
	Provider          *Schema            `json:"provider,omitempty"`
	ResourceSchemas   map[string]*Schema `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]*Schema `json:"data_source_schemas,omitempty"`

	// Functions are serialized by the jsonfunction package
	Functions               json.RawMessage `json:"functions,omitempty"`
	providerSchemaFunctions map[string]providers.FunctionDecl
}

func (p Provider) MarshalJSON() ([]byte, error) {
	type provider Provider

	tmp := provider(p)

	var err error
	tmp.Functions, err = jsonfunction.MarshalProviderFunctions(p.providerSchemaFunctions)
	if err != nil {
		return nil, err
	}

	return json.Marshal(tmp)
}

func newProviders() *Providers {
	schemas := make(map[string]*Provider)
	return &Providers{
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

func marshalProvider(tps providers.ProviderSchema) *Provider {
	return &Provider{
		Provider:                marshalSchema(tps.Provider),
		ResourceSchemas:         marshalSchemas(tps.ResourceTypes),
		DataSourceSchemas:       marshalSchemas(tps.DataSources),
		providerSchemaFunctions: tps.Functions,
	}
}
