package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/dag"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/config/configschema"
)

// GraphNodeAttachResourceSchema is an interface implemented by node types
// that need a resource schema attached.
type GraphNodeAttachResourceSchema interface {
	GraphNodeResource
	GraphNodeProviderConsumer

	AttachResourceSchema(*configschema.Block)
}

// GraphNodeAttachProviderConfigSchema is an interface implemented by node types
// that need a provider configuration schema attached.
type GraphNodeAttachProviderConfigSchema interface {
	GraphNodeProvider

	AttachProviderConfigSchema(*configschema.Block)
}

// AttachSchemaTransformer finds nodes that implement either
// GraphNodeAttachResourceSchema or GraphNodeAttachProviderConfigSchema, looks up
// the schema for each, and then passes it to a method implemented by the
// node.
type AttachSchemaTransformer struct {
	Components contextComponentFactory
}

func (t *AttachSchemaTransformer) Transform(g *Graph) error {

	// First we'll figure out which provider types we need to fetch schemas for.
	needProviders := make(map[string]struct{})
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeAttachResourceSchema:
			providerAddr, _ := tv.ProvidedBy()
			needProviders[providerAddr.ProviderConfig.Type] = struct{}{}
		case GraphNodeAttachProviderConfigSchema:
			providerAddr := tv.ProviderAddr()
			needProviders[providerAddr.ProviderConfig.Type] = struct{}{}
		}
	}

	// Now we'll fetch each one. This requires us to temporarily instantiate
	// them, though this is not a full bootstrap since we don't yet have
	// configuration information; the providers will be re-instantiated and
	// properly configured during the graph walk.
	schemas := make(map[string]*ProviderSchema)
	for typeName := range needProviders {
		log.Printf("[TRACE] AttachSchemaTransformer: retrieving schema for provider type %q", typeName)
		provider, err := t.Components.ResourceProvider(typeName, "early/"+typeName)
		if err != nil {
			return fmt.Errorf("failed to instantiate provider %q to obtain schema: %s", typeName, err)
		}

		// FIXME: The provider interface is currently awkward in that it
		// requires us to tell the provider which resources types and data
		// sources we need. In future this will change to just return
		// everything available, but for now we'll fake that by fetching all
		// of the available names and then requesting them.
		resourceTypes := provider.Resources()
		dataSources := provider.DataSources()
		resourceTypeNames := make([]string, len(resourceTypes))
		for i, o := range resourceTypes {
			resourceTypeNames[i] = o.Name
		}
		dataSourceNames := make([]string, len(dataSources))
		for i, o := range dataSources {
			dataSourceNames[i] = o.Name
		}

		schema, err := provider.GetSchema(&ProviderSchemaRequest{
			ResourceTypes: resourceTypeNames,
			DataSources:   dataSourceNames,
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve schema from provider %q: %s", typeName, err)
		}

		schemas[typeName] = schema

		if closer, ok := provider.(ResourceProviderCloser); ok {
			closer.Close()
		}
	}

	// Finally we'll once again visit all of the vertices and attach to
	// them the schemas we found for them.
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeAttachResourceSchema:
			addr := tv.ResourceAddr()
			mode := addr.Resource.Mode
			typeName := addr.Resource.Type
			providerAddr, _ := tv.ProvidedBy()
			var schema *configschema.Block
			providerSchema := schemas[providerAddr.ProviderConfig.Type]
			if providerSchema == nil {
				log.Printf("[ERROR] AttachSchemaTransformer: No schema available for %s because provider schema for %q is missing", addr, providerAddr.ProviderConfig.Type)
				continue
			}
			switch mode {
			case addrs.ManagedResourceMode:
				schema = providerSchema.ResourceTypes[typeName]
			case addrs.DataResourceMode:
				schema = providerSchema.DataSources[typeName]
			}
			if schema != nil {
				log.Printf("[TRACE] AttachSchemaTransformer: attaching schema to %s", dag.VertexName(v))
				tv.AttachResourceSchema(schema)
			} else {
				log.Printf("[ERROR] AttachSchemaTransformer: No schema available for %s", addr)
			}
		case GraphNodeAttachProviderConfigSchema:
			providerAddr := tv.ProviderAddr()
			providerSchema := schemas[providerAddr.ProviderConfig.Type]
			if providerSchema == nil {
				log.Printf("[ERROR] AttachSchemaTransformer: No schema available for %s because the whole provider schema is missing", providerAddr)
				continue
			}

			schema := schemas[providerAddr.ProviderConfig.Type].Provider

			if schema != nil {
				log.Printf("[TRACE] AttachSchemaTransformer: attaching schema to %s", dag.VertexName(v))
				tv.AttachProviderConfigSchema(schema)
			} else {
				log.Printf("[ERROR] AttachSchemaTransformer: No schema available for %s", providerAddr)
			}
		}
	}

	return nil
}
