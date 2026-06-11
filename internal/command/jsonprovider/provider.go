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

// ResourceEmit is a small directive that tells the marshaler which of the two
// JSON categories derived from the single ResourceTypes map should be emitted.
//
// This is the *only* schema-category awareness inside jsonprovider. It exists
// solely to resolve the collision where both resource_schemas and
// resource_identity_schemas are derived from ResourceTypes; the full -kind
// vocabulary lives in the command/arguments package (see
// proposals/provider-subcommand-filtering/design_decisions.md #1 and #7). All
// other categories are governed purely by which maps survive pruning and drop
// out via omitempty.
type ResourceEmit int

const (
	// EmitAll emits both resource_schemas and resource_identity_schemas. This
	// is the unfiltered/wildcard behavior.
	EmitAll ResourceEmit = iota
	// ResourceBlockOnly emits resource_schemas only (-kind=resource).
	ResourceBlockOnly
	// ResourceIdentityOnly emits resource_identity_schemas only
	// (-kind=resource-identity).
	ResourceIdentityOnly
)

// Filters records the normalized active selectors used to produce a filtered
// response. Omitted dimensions are absent rather than represented as wildcards.
type Filters struct {
	Provider string `json:"provider,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Providers is the top-level object returned when exporting provider schemas
type Providers struct {
	FormatVersion string `json:"format_version"`

	// Filters is present only when at least one selector flag was supplied.
	//
	// This field is additive and only appears for filtered invocations, so
	// unfiltered output remains byte-for-byte identical and FormatVersion is
	// intentionally not bumped (see
	// proposals/provider-subcommand-filtering/design_decisions.md #10).
	Filters *Filters `json:"filters,omitempty"`

	Schemas map[string]*Provider `json:"provider_schemas,omitempty"`
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
	return marshalProviders(s, EmitAll)
}

func marshalProviders(s *terraform.Schemas, emit ResourceEmit) map[string]*Provider {
	schemas := make(map[string]*Provider, len(s.Providers))
	for k, v := range s.Providers {
		schemas[k.String()] = marshalProvider(v, emit)
	}
	return schemas
}

// Marshal produces the unfiltered, wildcard provider-schema JSON document. It
// is defined as the EmitAll, nil-filters case of MarshalWithFilters so the two
// paths cannot drift.
func Marshal(s *terraform.Schemas) ([]byte, error) {
	return MarshalWithFilters(s, EmitAll, nil)
}

// MarshalWithFilters is the single entry point for producing the provider-schema
// JSON document. The emit directive resolves the resource/resource-identity
// collision; filters, when non-nil, is attached as the top-level metadata echo.
func MarshalWithFilters(s *terraform.Schemas, emit ResourceEmit, filters *Filters) ([]byte, error) {
	providers := newProviders()
	providers.Filters = filters
	providers.Schemas = marshalProviders(s, emit)
	ret, err := json.Marshal(providers)
	return ret, err
}

func marshalProvider(tps providers.ProviderSchema, emit ResourceEmit) *Provider {
	p := &Provider{
		DataSourceSchemas:        marshalSchemas(tps.DataSources),
		EphemeralResourceSchemas: marshalSchemas(tps.EphemeralResourceTypes),
		Functions:                jsonfunction.MarshalProviderFunctions(tps.Functions),
		ActionSchemas:            marshalActionSchemas(tps.Actions),
		StateStoreSchemas:        marshalSchemas(tps.StateStores),
	}

	// The provider configuration block is emitted only when the provider
	// actually exposes one. A pruned schema (for example when filtering to a
	// non-provider kind) zeroes this field; omitting it here keeps the
	// provider category out of the output instead of rendering an empty stub.
	// Real providers always return a (possibly empty) provider configuration
	// block, so the unfiltered output is unchanged.
	if tps.Provider.Body != nil {
		p.Provider = marshalSchema(tps.Provider)
	}

	// resource_schemas and resource_identity_schemas are both derived from the
	// single ResourceTypes map, so the emit directive decides which of them is
	// rendered. EmitAll preserves the unfiltered behavior of emitting both.
	switch emit {
	case ResourceBlockOnly:
		p.ResourceSchemas = marshalSchemas(tps.ResourceTypes)
	case ResourceIdentityOnly:
		p.ResourceIdentitySchemas = marshalIdentitySchemas(tps.ResourceTypes)
	default: // EmitAll
		p.ResourceSchemas = marshalSchemas(tps.ResourceTypes)
		p.ResourceIdentitySchemas = marshalIdentitySchemas(tps.ResourceTypes)
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
