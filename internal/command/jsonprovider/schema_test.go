// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package jsonprovider

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestMarshalSchemas(t *testing.T) {
	tests := []struct {
		Input map[string]providers.Schema
		Want  map[string]*Schema
	}{
		{
			nil,
			map[string]*Schema{},
		},
	}

	for _, test := range tests {
		got := marshalSchemas(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}

func TestMarshalSchema(t *testing.T) {
	tests := map[string]struct {
		Input providers.Schema
		Want  *Schema
	}{
		"nil_block": {
			providers.Schema{},
			&Schema{},
		},
	}

	for _, test := range tests {
		got := marshalSchema(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}

func TestMarshalProviderMetaSchema(t *testing.T) {
	tests := map[string]struct {
		Input providers.Schema
		Want  *Schema
	}{
		"no_provider_meta_schema": {
			providers.Schema{},
			nil,
		},
		"provider_meta_schema_defined": {
			providers.Schema{
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"user_agent": {Type: cty.List(cty.String), Optional: true},
					},
				},
			},
			&Schema{
				Block: &Block{
					Attributes: map[string]*Attribute{
						"user_agent": {
							AttributeType:   json.RawMessage(`["list","string"]`),
							Optional:        true,
							DescriptionKind: "plain",
						},
					},
					DescriptionKind: "plain",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := marshalProviderMetaSchema(test.Input)
			if !cmp.Equal(got, test.Want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
			}
		})
	}
}
