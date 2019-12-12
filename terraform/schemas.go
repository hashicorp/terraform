package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/internal/schemas"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// Schemas is a container for various kinds of schema that Terraform needs
// during processing.
type Schemas = schemas.Schemas

// ProviderSchema represents the schema for a provider's own configuration
// and the configuration for some or all of its resources and data sources.
//
// The completeness of this structure depends on how it was constructed.
// When constructed for a configuration, it will generally include only
// resource types and data sources used by that configuration.
type ProviderSchema = schemas.ProviderSchema

// ProviderSchemaRequest is used to describe to a ResourceProvider which
// aspects of schema are required, when calling the GetSchema method.
type ProviderSchemaRequest = schemas.ProviderSchemaRequest

// LoadSchemas searches the given configuration, state  and plan (any of which
// may be nil) for constructs that have an associated schema, requests the
// necessary schemas from the given component factory (which must _not_ be nil),
// and returns a single object representing all of the necessary schemas.
//
// If an error is returned, it may be a wrapped tfdiags.Diagnostics describing
// errors across multiple separate objects. Errors here will usually indicate
// either misbehavior on the part of one of the providers or of the provider
// protocol itself. When returned with errors, the returned schemas object is
// still valid but may be incomplete.
func LoadSchemas(config *configs.Config, state *states.State, components contextComponentFactory) (*Schemas, error) {
	schemas := &Schemas{
		Providers:    map[string]*ProviderSchema{},
		Provisioners: map[string]*configschema.Block{},
	}
	var diags tfdiags.Diagnostics

	newDiags := loadProviderSchemas(schemas.Providers, config, state, components)
	diags = diags.Append(newDiags)
	newDiags = loadProvisionerSchemas(schemas.Provisioners, config, components)
	diags = diags.Append(newDiags)

	return schemas, diags.Err()
}

func loadProviderSchemas(schemas map[string]*ProviderSchema, config *configs.Config, state *states.State, components contextComponentFactory) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ensure := func(typeName string) {
		if _, exists := schemas[typeName]; exists {
			return
		}

		log.Printf("[TRACE] LoadSchemas: retrieving schema for provider type %q", typeName)
		provider, err := components.ResourceProvider(typeName, "early/"+typeName)
		if err != nil {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls.
			schemas[typeName] = &ProviderSchema{}
			diags = diags.Append(
				fmt.Errorf("Failed to instantiate provider %q to obtain schema: %s", typeName, err),
			)
			return
		}
		defer func() {
			provider.Close()
		}()

		resp := provider.GetSchema()
		if resp.Diagnostics.HasErrors() {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls.
			schemas[typeName] = &ProviderSchema{}
			diags = diags.Append(
				fmt.Errorf("Failed to retrieve schema from provider %q: %s", typeName, resp.Diagnostics.Err()),
			)
			return
		}

		s := &ProviderSchema{
			Provider:      resp.Provider.Block,
			ResourceTypes: make(map[string]*configschema.Block),
			DataSources:   make(map[string]*configschema.Block),

			ResourceTypeSchemaVersions: make(map[string]uint64),
		}

		if resp.Provider.Version < 0 {
			// We're not using the version numbers here yet, but we'll check
			// for validity anyway in case we start using them in future.
			diags = diags.Append(
				fmt.Errorf("invalid negative schema version provider configuration for provider %q", typeName),
			)
		}

		for t, r := range resp.ResourceTypes {
			s.ResourceTypes[t] = r.Block
			s.ResourceTypeSchemaVersions[t] = uint64(r.Version)
			if r.Version < 0 {
				diags = diags.Append(
					fmt.Errorf("invalid negative schema version for resource type %s in provider %q", t, typeName),
				)
			}
		}

		for t, d := range resp.DataSources {
			s.DataSources[t] = d.Block
			if d.Version < 0 {
				// We're not using the version numbers here yet, but we'll check
				// for validity anyway in case we start using them in future.
				diags = diags.Append(
					fmt.Errorf("invalid negative schema version for data source %s in provider %q", t, typeName),
				)
			}
		}

		schemas[typeName] = s
	}

	if config != nil {
		for _, typeName := range config.ProviderTypes() {
			ensure(typeName)
		}
	}

	if state != nil {
		needed := providers.AddressedTypesAbs(state.ProviderAddrs())
		for _, typeName := range needed {
			ensure(typeName)
		}
	}

	return diags
}

func loadProvisionerSchemas(schemas map[string]*configschema.Block, config *configs.Config, components contextComponentFactory) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	ensure := func(name string) {
		if _, exists := schemas[name]; exists {
			return
		}

		log.Printf("[TRACE] LoadSchemas: retrieving schema for provisioner %q", name)
		provisioner, err := components.ResourceProvisioner(name, "early/"+name)
		if err != nil {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls.
			schemas[name] = &configschema.Block{}
			diags = diags.Append(
				fmt.Errorf("Failed to instantiate provisioner %q to obtain schema: %s", name, err),
			)
			return
		}
		defer func() {
			if closer, ok := provisioner.(ResourceProvisionerCloser); ok {
				closer.Close()
			}
		}()

		resp := provisioner.GetSchema()
		if resp.Diagnostics.HasErrors() {
			// We'll put a stub in the map so we won't re-attempt this on
			// future calls.
			schemas[name] = &configschema.Block{}
			diags = diags.Append(
				fmt.Errorf("Failed to retrieve schema from provisioner %q: %s", name, resp.Diagnostics.Err()),
			)
			return
		}

		schemas[name] = resp.Provisioner
	}

	if config != nil {
		for _, rc := range config.Module.ManagedResources {
			for _, pc := range rc.Managed.Provisioners {
				ensure(pc.Type)
			}
		}

		// Must also visit our child modules, recursively.
		for _, cc := range config.Children {
			childDiags := loadProvisionerSchemas(schemas, cc, components)
			diags = diags.Append(childDiags)
		}
	}

	return diags
}
