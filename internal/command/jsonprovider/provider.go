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
	Provider                 *Schema                                    `json:"provider,omitempty"`
	ResourceSchemas          map[string]*Schema                         `json:"resource_schemas,omitempty"`
	DataSourceSchemas        map[string]*Schema                         `json:"data_source_schemas,omitempty"`
	EphemeralResourceSchemas map[string]*Schema                         `json:"ephemeral_resource_schemas,omitempty"`
	Functions                map[string]*jsonfunction.FunctionSignature `json:"functions,omitempty"`
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
		Provider:                 marshalSchema(tps.Provider),
		ResourceSchemas:          marshalSchemas(tps.ResourceTypes),
		DataSourceSchemas:        marshalSchemas(tps.DataSources),
		EphemeralResourceSchemas: marshalSchemas(tps.EphemeralResourceTypes),
		Functions:                jsonfunction.MarshalProviderFunctions(tps.Functions),
	}
}
