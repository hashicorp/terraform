package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// Schemas is a container for various kinds of schema that Terraform needs
// during processing.
type Schemas struct {
	providers    map[string]*ProviderSchema
	provisioners map[string]*configschema.Block
}

// ProviderSchema returns the entire ProviderSchema object that was produced
// by the plugin for the given provider, or nil if no such schema is available.
//
// It's usually better to go use the more precise methods offered by type
// Schemas to handle this detail automatically.
func (ss *Schemas) ProviderSchema(typeName string) *ProviderSchema {
	if ss.providers == nil {
		return nil
	}
	return ss.providers[typeName]
}

// ProviderConfig returns the schema for the provider configuration of the
// given provider type, or nil if no such schema is available.
func (ss *Schemas) ProviderConfig(typeName string) *configschema.Block {
	ps := ss.ProviderSchema(typeName)
	if ps == nil {
		return nil
	}
	return ps.Provider
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
func (ss *Schemas) ResourceTypeConfig(providerType string, resourceType string) *configschema.Block {
	ps := ss.ProviderSchema(providerType)
	if ps == nil || ps.ResourceTypes == nil {
		return nil
	}

	return ps.ResourceTypes[resourceType]
}

// DataSourceConfig returns the schema for the configuration of a given
// data source belonging to a given provider type, or nil of no such
// schema is available.
//
// In many cases the provider type is inferrable from the data source name,
// but this is not always true because users can override the provider for
// a resource using the "provider" meta-argument. Therefore it's important to
// always pass the correct provider name, even though it many cases it feels
// redundant.
func (ss *Schemas) DataSourceConfig(providerType string, dataSource string) *configschema.Block {
	ps := ss.ProviderSchema(providerType)
	if ps == nil || ps.DataSources == nil {
		return nil
	}

	return ps.DataSources[dataSource]
}

// ProvisionerConfig returns the schema for the configuration of a given
// provisioner, or nil of no such schema is available.
func (ss *Schemas) ProvisionerConfig(name string) *configschema.Block {
	return ss.provisioners[name]
}

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
		providers:    map[string]*ProviderSchema{},
		provisioners: map[string]*configschema.Block{},
	}
	var diags tfdiags.Diagnostics

	newDiags := loadProviderSchemas(schemas.providers, config, state, components)
	diags = diags.Append(newDiags)
	newDiags = loadProvisionerSchemas(schemas.provisioners, config, components)
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
		}

		for t, r := range resp.ResourceTypes {
			s.ResourceTypes[t] = r.Block
		}

		for t, d := range resp.DataSources {
			s.DataSources[t] = d.Block
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

// ProviderSchema represents the schema for a provider's own configuration
// and the configuration for some or all of its resources and data sources.
//
// The completeness of this structure depends on how it was constructed.
// When constructed for a configuration, it will generally include only
// resource types and data sources used by that configuration.
type ProviderSchema struct {
	Provider      *configschema.Block
	ResourceTypes map[string]*configschema.Block
	DataSources   map[string]*configschema.Block
}

// ProviderSchemaRequest is used to describe to a ResourceProvider which
// aspects of schema are required, when calling the GetSchema method.
type ProviderSchemaRequest struct {
	ResourceTypes []string
	DataSources   []string
}
