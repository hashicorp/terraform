// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package jsonprovider

import (
	"encoding/json"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

// marshalTestSchemas builds a single-provider *terraform.Schemas containing a
// managed resource (with an identity), a data source, and a function so the
// marshal directive and filters echo can be exercised.
func marshalTestSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				Provider: providers.Schema{
					Body: &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"region": {Type: cty.String, Optional: true},
						},
					},
				},
				ResourceTypes: map[string]providers.Schema{
					"test_instance": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {Type: cty.String, Computed: true},
							},
						},
						Identity: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"id": {Type: cty.String, Required: true},
							},
						},
					},
				},
				DataSources: map[string]providers.Schema{
					"test_data": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {Type: cty.String, Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

// topLevelKeys parses marshaled output into the set of top-level JSON keys.
func topLevelKeys(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("failed to parse output: %s", err)
	}
	return top
}

// providerCategories parses the categories present for a single provider entry.
func providerCategories(t *testing.T, raw []byte, provider string) map[string]json.RawMessage {
	t.Helper()
	var top struct {
		Schemas map[string]map[string]json.RawMessage `json:"provider_schemas"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("failed to parse output: %s", err)
	}
	return top.Schemas[provider]
}

// TestMarshal_omitsFilters asserts that the unfiltered wildcard entry point
// never emits a top-level "filters" field and emits both ResourceTypes-derived
// categories (the EmitAll behavior).
func TestMarshal_omitsFilters(t *testing.T) {
	got, err := Marshal(marshalTestSchemas())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	top := topLevelKeys(t, got)
	if _, ok := top["filters"]; ok {
		t.Errorf("unfiltered Marshal output unexpectedly includes \"filters\":\n%s", got)
	}

	cats := providerCategories(t, got, "registry.terraform.io/hashicorp/test")
	if _, ok := cats["resource_schemas"]; !ok {
		t.Errorf("expected resource_schemas in EmitAll output:\n%s", got)
	}
	if _, ok := cats["resource_identity_schemas"]; !ok {
		t.Errorf("expected resource_identity_schemas in EmitAll output:\n%s", got)
	}
}

// TestMarshalWithFilters_resourceEmit asserts the directive controls which of
// the two ResourceTypes-derived categories is emitted.
func TestMarshalWithFilters_resourceEmit(t *testing.T) {
	const provider = "registry.terraform.io/hashicorp/test"

	tests := map[string]struct {
		emit         ResourceEmit
		wantResource bool
		wantIdentity bool
		wantDataSrc  bool
	}{
		"EmitAll": {
			emit:         EmitAll,
			wantResource: true,
			wantIdentity: true,
			wantDataSrc:  true,
		},
		"ResourceBlockOnly": {
			emit:         ResourceBlockOnly,
			wantResource: true,
			wantIdentity: false,
			// non-resource categories still render naturally; the directive
			// only governs the ResourceTypes-derived pair.
			wantDataSrc: true,
		},
		"ResourceIdentityOnly": {
			emit:         ResourceIdentityOnly,
			wantResource: false,
			wantIdentity: true,
			wantDataSrc:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := MarshalWithFilters(marshalTestSchemas(), tc.emit, nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			cats := providerCategories(t, got, provider)

			if _, ok := cats["resource_schemas"]; ok != tc.wantResource {
				t.Errorf("resource_schemas present=%t, want %t:\n%s", ok, tc.wantResource, got)
			}
			if _, ok := cats["resource_identity_schemas"]; ok != tc.wantIdentity {
				t.Errorf("resource_identity_schemas present=%t, want %t:\n%s", ok, tc.wantIdentity, got)
			}
			if _, ok := cats["data_source_schemas"]; ok != tc.wantDataSrc {
				t.Errorf("data_source_schemas present=%t, want %t:\n%s", ok, tc.wantDataSrc, got)
			}
		})
	}
}

// TestMarshalWithFilters_filtersEcho asserts the filters echo is emitted with
// only the dimensions that were set, and that omitted dimensions are absent.
func TestMarshalWithFilters_filtersEcho(t *testing.T) {
	filters := &Filters{
		Provider: "registry.terraform.io/hashicorp/test",
		Kind:     "resource",
		// Type intentionally omitted.
	}

	got, err := MarshalWithFilters(marshalTestSchemas(), ResourceBlockOnly, filters)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	var top struct {
		Filters *struct {
			Provider string `json:"provider"`
			Kind     string `json:"kind"`
			Type     string `json:"type"`
		} `json:"filters"`
	}
	if err := json.Unmarshal(got, &top); err != nil {
		t.Fatalf("failed to parse output: %s", err)
	}
	if top.Filters == nil {
		t.Fatalf("expected a top-level filters object:\n%s", got)
	}
	if top.Filters.Provider != filters.Provider {
		t.Errorf("filters.provider = %q, want %q", top.Filters.Provider, filters.Provider)
	}
	if top.Filters.Kind != filters.Kind {
		t.Errorf("filters.kind = %q, want %q", top.Filters.Kind, filters.Kind)
	}

	// Confirm omitted dimensions are physically absent from the JSON.
	raw := topLevelKeys(t, got)
	var filtersRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["filters"], &filtersRaw); err != nil {
		t.Fatalf("failed to parse filters: %s", err)
	}
	if _, ok := filtersRaw["type"]; ok {
		t.Errorf("omitted filters.type unexpectedly present:\n%s", got)
	}
	if _, ok := filtersRaw["provider"]; !ok {
		t.Errorf("filters.provider unexpectedly absent:\n%s", got)
	}
	if _, ok := filtersRaw["kind"]; !ok {
		t.Errorf("filters.kind unexpectedly absent:\n%s", got)
	}
}
