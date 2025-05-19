// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/schemarepo/loadschemas"
)

// simpleMockPluginLibrary returns a plugin library pre-configured with
// one provider and one provisioner, both called "test".
//
// The provider is built with simpleMockProvider and the provisioner with
// simpleMockProvisioner, and all schemas used in both are as built by
// function simpleTestSchema.
//
// Each call to this function produces an entirely-separate set of objects,
// so the caller can feel free to modify the returned value to further
// customize the mocks contained within.
func simpleMockPluginLibrary() *loadschemas.Plugins {
	// We create these out here, rather than in the factory functions below,
	// because we want each call to the factory to return the _same_ instance,
	// so that test code can customize it before passing this component
	// factory into real code under test.
	provider := simpleMockProvider()
	provisioner := simpleMockProvisioner()
	return loadschemas.NewPlugins(
		map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
				return provider, nil
			},
		},
		map[string]provisioners.Factory{
			"test": func() (provisioners.Interface, error) {
				return provisioner, nil
			},
		},
		nil,
	)
}

// simpleTestSchema returns a block schema that contains a few optional
// attributes for use in tests.
//
// The returned schema contains the following optional attributes:
//
//   - test_string, of type string
//   - test_number, of type number
//   - test_bool, of type bool
//   - test_list, of type list(string)
//   - test_map, of type map(string)
//
// Each call to this function produces an entirely new schema instance, so
// callers can feel free to modify it once returned.
func simpleTestSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"test_string": {
				Type:     cty.String,
				Optional: true,
			},
			"test_number": {
				Type:     cty.Number,
				Optional: true,
			},
			"test_bool": {
				Type:     cty.Bool,
				Optional: true,
			},
			"test_list": {
				Type:     cty.List(cty.String),
				Optional: true,
			},
			"test_map": {
				Type:     cty.Map(cty.String),
				Optional: true,
			},
		},
	}
}
