// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package loadschemas

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LoadSchemas loads all of the schemas that might be needed to work with the
// given configuration and state, using the given plugins.
func LoadSchemas(config *configs.Config, state *states.State, plugins *Plugins) (*schemarepo.Schemas, error) {
	schemas := &schemarepo.Schemas{
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

func loadProviderSchemas(schemas map[addrs.Provider]providers.ProviderSchema, config *configs.Config, state *states.State, plugins *Plugins) tfdiags.Diagnostics {
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

		identitySchemas, err := plugins.ResourceIdentitySchemas(fqn)
		if err != nil {
			// We just log and ignore the error
			log.Printf("[WARN] Failed to obtain resource identity schemas for provider %s: %s", fqn, err)
		} else {
			// Iterate over the identity schemas and merge them into the provider schema
			for name, identitySchema := range identitySchemas.IdentityTypes {
				if resource, ok := schema.ResourceTypes[name]; !ok {
					// This shouldn't happen, but in case we get an identity for a non-existent resource type
					log.Printf("[WARN] Failed to find resource type %s for provider %s", name, fqn)
					continue
				} else {
					schema.ResourceTypes[name] = providers.Schema{
						Body:    resource.Body,
						Version: resource.Version,

						Identity:        identitySchema.Body,
						IdentityVersion: uint64(identitySchema.Version),
					}
				}
			}
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

func loadProvisionerSchemas(schemas map[string]*configschema.Block, config *configs.Config, plugins *Plugins) tfdiags.Diagnostics {
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
