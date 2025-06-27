// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// provider config isn't used when creating a hash of state_store config
// as long as the schema for the state store returned from the provider
// is accurate.
func TestStateStore_Hash(t *testing.T) {

	// This test assumes a configuration like this,
	// where the "fs" state store is implemented in
	// the "foobar" provider:
	//
	// terraform {
	//   required_providers = {
	//     # entries would be here
	//   }
	//   state_store "foobar_fs" {
	//     # Nested provider block
	//     provider "foobar" {
	//       foo = "bar"
	//     }

	//     # Attributes for configuring the state store
	//     path          = "mystate.tfstate"
	//     workspace_dir = "foobar"
	//   }
	// }

	// hcl.Body includes all of the contents of the state_store block shown above
	configWithProvider := SynthBody("TestStateStore_Hash", map[string]cty.Value{
		"provider": cty.ObjectVal(map[string]cty.Value{
			"foo": cty.StringVal("bar"),
		}),

		"path":          cty.StringVal("mystate.tfstate"),
		"workspace_dir": cty.StringVal("foobar"),
	})

	// hcl.Body that represents the state_store block shown above
	// but excludes the provider block
	configWithoutProvider := SynthBody("TestStateStore_Hash", map[string]cty.Value{
		"path":          cty.StringVal("mystate.tfstate"),
		"workspace_dir": cty.StringVal("foobar"),
	})

	// schema from GetProviderSchema for the given state store
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"path": {
				Type:     cty.String,
				Optional: true,
			},
			"workspace_dir": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	// The hash of the two different hcl.Body values should be the same,
	// assuming the schema only describes the schema for configuring
	// a state store.
	sWithProvider := StateStore{
		Config: configWithProvider,
	}
	sWithoutProvider := StateStore{
		Config: configWithoutProvider,
	}

	hashWithProvider := sWithProvider.Hash(schema)
	hashWithoutProvider := sWithoutProvider.Hash(schema)

	if hashWithProvider != hashWithoutProvider {
		t.Fatalf("presence of provider config should not impact the hash created for a state store, yet these hashes don't match:\n hash with provider: %d\n hash without provider: %d",
			hashWithProvider,
			hashWithoutProvider)
	}
}
