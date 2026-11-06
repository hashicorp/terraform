// Copyright IBM Corp. 2014, 2026
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
	ListResourceSchemas      map[string]*Schema                         `json:"list_resource_schemas,omitempty"`
	Functions                map[string]*jsonfunction.FunctionSignature `json:"functions,omitempty"`
	ResourceIdentitySchemas  map[string]*IdentitySchema                 `json:"resource_identity_schemas,omitempty"`
	ActionSchemas            map[string]*ActionSchema                   `json:"action_schemas,omitempty"`
	StateStoreSchemas        map[string]*Schema                         `json:"state_store_schemas,omitempty"`
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
	p := &Provider{}
	if tps.Provider.Body != nil { // provider config not included in schema
		p.Provider = marshalSchema(tps.Provider)
	}
	if tps.ResourceTypes != nil {
		p.ResourceSchemas = marshalSchemas(tps.ResourceTypes)
		p.ResourceIdentitySchemas = marshalIdentitySchemas(tps.ResourceTypes)
	}
	if tps.DataSources != nil {
		p.DataSourceSchemas = marshalSchemas(tps.DataSources)
	}
	if tps.EphemeralResourceTypes != nil {
		p.EphemeralResourceSchemas = marshalSchemas(tps.EphemeralResourceTypes)
	}
	if tps.Functions != nil {
		p.Functions = jsonfunction.MarshalProviderFunctions(tps.Functions)
	}
	if tps.Actions != nil {
		p.ActionSchemas = marshalActionSchemas(tps.Actions)
	}
	if tps.StateStores != nil {
		p.StateStoreSchemas = marshalSchemas(tps.StateStores)
	}

	// List resource schemas are nested under a "config" block, so we need to
	// extract that block to get the actual provider schema for the list resource.
	// When getting the provider schemas, Terraform adds this extra level to
	// better match the actual configuration structure.
	listSchemas := make(map[string]providers.Schema, len(tps.ListResourceTypes))
	for k, v := range tps.ListResourceTypes {
		listSchemas[k] = providers.Schema{
			Body:    &v.Body.BlockTypes["config"].Block,
			Version: v.Version,
		}
	}
	p.ListResourceSchemas = marshalSchemas(listSchemas)

	return p
}
