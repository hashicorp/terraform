// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

// filterTestSchemas builds a small single-provider *terraform.Schemas suitable
// for exercising the pure filter function.
func filterTestSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test_instance": {
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

// TestFilterProviderSchemas_plumbing verifies the pure filter function threads
// the resource-emission directive and filters echo through, and that the
// no-selector case is an exact pass-through of the loaded schemas.
func TestFilterProviderSchemas_plumbing(t *testing.T) {
	schemas := filterTestSchemas()

	t.Run("no selectors is a pass-through", func(t *testing.T) {
		filtered, emit, filters, diags := filterProviderSchemas(schemas, selectors{})
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if filtered != schemas {
			t.Errorf("expected the loaded schemas to be returned unchanged")
		}
		if emit != jsonprovider.EmitAll {
			t.Errorf("expected EmitAll, got %v", emit)
		}
		if filters != nil {
			t.Errorf("expected nil filters echo, got %+v", filters)
		}
	})

	t.Run("directive and filters echo are threaded through", func(t *testing.T) {
		sel := selectors{
			kind:    arguments.KindResourceIdentity,
			kindSet: true,
		}
		_, emit, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if emit != jsonprovider.ResourceIdentityOnly {
			t.Errorf("expected ResourceIdentityOnly, got %v", emit)
		}
		if filters == nil {
			t.Fatalf("expected a non-nil filters echo")
		}
		if filters.Kind != string(arguments.KindResourceIdentity) {
			t.Errorf("expected filters.kind=%q, got %q", arguments.KindResourceIdentity, filters.Kind)
		}
	})
}
