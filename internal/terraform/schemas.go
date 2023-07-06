// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Schemas is a container for various kinds of schema that Terraform needs
// during processing.
type Schemas struct {
	Providers    map[addrs.Provider]providers.ProviderSchema
	Provisioners map[string]*configschema.Block
}

// ProviderSchema returns the entire ProviderSchema object that was produced
// by the plugin for the given provider, or nil if no such schema is available.
//
// It's usually better to go use the more precise methods offered by type
// Schemas to handle this detail automatically.
func (ss *Schemas) ProviderSchema(provider addrs.Provider) providers.ProviderSchema {
	return ss.Providers[provider]
}

// ProviderConfig returns the schema for the provider configuration of the
// given provider type, or nil if no such schema is available.
func (ss *Schemas) ProviderConfig(provider addrs.Provider) *configschema.Block {
	return ss.ProviderSchema(provider).Provider.Block
}

// ResourceTypeConfig returns the schema for the configuration of a given
// resource type belonging to a given provider type, or nil of no such
// schema is available.
//
// In many cases the provider type is inferrable from the resource type name,
// but this is not always true because users can override the provider for
// a resource using the "provider" meta-argument. Therefore it's important to
// always pass the correct provider name, even though it many cases it feels
// redundant.
func (ss *Schemas) ResourceTypeConfig(provider addrs.Provider, resourceMode addrs.ResourceMode, resourceType string) (block *configschema.Block, schemaVersion uint64) {
	ps := ss.ProviderSchema(provider)
	if ps.ResourceTypes == nil {
		return nil, 0
	}
	return ps.SchemaForResourceType(resourceMode, resourceType)
}

// ProvisionerConfig returns the schema for the configuration of a given
// provisioner, or nil of no such schema is available.
func (ss *Schemas) ProvisionerConfig(name string) *configschema.Block {
	return ss.Provisioners[name]
}

// loadSchemas searches the given configuration, state  and plan (any of which
// may be nil) for constructs that have an associated schema, requests the
// necessary schemas from the given component factory (which must _not_ be nil),
// and returns a single object representing all of the necessary schemas.
//
// If an error is returned, it may be a wrapped tfdiags.Diagnostics describing
// errors across multiple separate objects. Errors here will usually indicate
// either misbehavior on the part of one of the providers or of the provider
// protocol itself. When returned with errors, the returned schemas object is
// still valid but may be incomplete.
func loadSchemas(config *configs.Config, state *states.State, plugins *contextPlugins) (*Schemas, error) {
	schemas := &Schemas{
		Providers:    map[addrs.Provider]providers.ProviderSchema{},
		Provisioners: map[string]*configschema.Block{},
	}
	var diags tfdiags.Diagnostics

	newDiags := loadProviderSchemas(schemas.Providers, config, state, plugins)
	diags = diags.Append(newDiags)
	newDiags = loadProvisionerSchemas(schemas.Provisioners, config, plugins)
	diags = diags.Append(newDiags)

	return schemas, diags.Err()
}

func loadProviderSchemas(schemas map[addrs.Provider]providers.ProviderSchema, config *configs.Config, state *states.State, plugins *contextPlugins) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ensure := func(fqn addrs.Provider) {
		name := fqn.String()

		if _, exists := schemas[fqn]; exists {
			return
		}

		log.Printf("[TRACE] LoadSchemas: retrieving schema for provider type %q", name)
		schema, err := plugins.ProviderSchema(fqn)
		if err != nil {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls, which would then repeat the same error message
			// multiple times.
			schemas[fqn] = providers.ProviderSchema{}
			diags = diags.Append(
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to obtain provider schema",
					fmt.Sprintf("Could not load the schema for provider %s: %s.", fqn, err),
				),
			)
			return
		}

		schemas[fqn] = schema
	}

	if config != nil {
		for _, fqn := range config.ProviderTypes() {
			ensure(fqn)
		}
	}

	if state != nil {
		needed := providers.AddressedTypesAbs(state.ProviderAddrs())
		for _, typeAddr := range needed {
			ensure(typeAddr)
		}
	}

	return diags
}

func loadProvisionerSchemas(schemas map[string]*configschema.Block, config *configs.Config, plugins *contextPlugins) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ensure := func(name string) {
		if _, exists := schemas[name]; exists {
			return
		}

		log.Printf("[TRACE] LoadSchemas: retrieving schema for provisioner %q", name)
		schema, err := plugins.ProvisionerSchema(name)
		if err != nil {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls, which would then repeat the same error message
			// multiple times.
			schemas[name] = &configschema.Block{}
			diags = diags.Append(
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to obtain provisioner schema",
					fmt.Sprintf("Could not load the schema for provisioner %q: %s.", name, err),
				),
			)
			return
		}

		schemas[name] = schema
	}

	if config != nil {
		for _, rc := range config.Module.ManagedResources {
			for _, pc := range rc.Managed.Provisioners {
				ensure(pc.Type)
			}
		}

		// Must also visit our child modules, recursively.
		for _, cc := range config.Children {
			childDiags := loadProvisionerSchemas(schemas, cc, plugins)
			diags = diags.Append(childDiags)
		}
	}

	return diags
}
