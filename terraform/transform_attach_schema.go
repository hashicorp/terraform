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

// GraphNodeAttachProvisionerSchema is an interface implemented by node types
// that need one or more provisioner schemas attached.
type GraphNodeAttachProvisionerSchema interface {
	ProvisionedBy() []string

	// SetProvisionerSchema is called during transform for each provisioner
	// type returned from ProvisionedBy, providing the configuration schema
	// for each provisioner in turn. The implementer should save these for
	// later use in evaluating provisioner configuration blocks.
	AttachProvisionerSchema(name string, schema *configschema.Block)
}

// AttachSchemaTransformer finds nodes that implement
// GraphNodeAttachResourceSchema, GraphNodeAttachProviderConfigSchema, or
// GraphNodeAttachProvisionerSchema, looks up the needed schemas for each
// and then passes them to a method implemented by the node.
type AttachSchemaTransformer struct {
	GraphNodeProvisionerConsumer
	Components contextComponentFactory
}

func (t *AttachSchemaTransformer) Transform(g *Graph) error {
	if t.Components == nil {
		// Should never happen with a reasonable caller, but we'll return a
		// proper error here anyway so that we'll fail gracefully.
		return fmt.Errorf("AttachSchemaTransformer used with nil Components")
	}

	err := t.attachProviderSchemas(g)
	if err != nil {
		return err
	}
	err = t.attachProvisionerSchemas(g)
	if err != nil {
		return err
	}

	return nil
}

func (t *AttachSchemaTransformer) attachProviderSchemas(g *Graph) error {

	// First we'll figure out which provider types we need to fetch schemas for.
	needProviders := make(map[string]struct{})
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeAttachResourceSchema:
			providerAddr, _ := tv.ProvidedBy()
			log.Printf("[TRACE] AttachSchemaTransformer: %q (%T) needs %s", dag.VertexName(v), v, providerAddr)
			needProviders[providerAddr.ProviderConfig.Type] = struct{}{}
		case GraphNodeAttachProviderConfigSchema:
			providerAddr := tv.ProviderAddr()
			log.Printf("[TRACE] AttachSchemaTransformer: %q (%T) needs %s", dag.VertexName(v), v, providerAddr)
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

func (t *AttachSchemaTransformer) attachProvisionerSchemas(g *Graph) error {

	// First we'll figure out which provisioners we need to fetch schemas for.
	needProvisioners := make(map[string]struct{})
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeAttachProvisionerSchema:
			names := tv.ProvisionedBy()
			log.Printf("[TRACE] AttachSchemaTransformer: %q (%T) provisioned by %s", dag.VertexName(v), v, names)
			for _, name := range names {
				needProvisioners[name] = struct{}{}
			}
		}
	}

	// Now we'll fetch each one. This requires us to temporarily instantiate
	// them, though this is not a full bootstrap since we don't yet have
	// configuration information; the provisioners will be re-instantiated and
	// properly configured during the graph walk.
	schemas := make(map[string]*configschema.Block)
	for name := range needProvisioners {
		log.Printf("[TRACE] AttachSchemaTransformer: retrieving schema for provisioner %q", name)
		provisioner, err := t.Components.ResourceProvisioner(name, "early/"+name)
		if err != nil {
			return fmt.Errorf("failed to instantiate provisioner %q to obtain schema: %s", name, err)
		}

		schema, err := provisioner.GetConfigSchema()
		if err != nil {
			return fmt.Errorf("failed to retrieve schema from provisioner %q: %s", name, err)
		}
		schemas[name] = schema

		if closer, ok := provisioner.(ResourceProvisionerCloser); ok {
			closer.Close()
		}
	}

	// Finally we'll once again visit all of the vertices and attach to
	// them the schemas we found for them.
	for _, v := range g.Vertices() {
		switch tv := v.(type) {
		case GraphNodeAttachProvisionerSchema:
			names := tv.ProvisionedBy()
			for _, name := range names {
				schema := schemas[name]
				if schema != nil {
					log.Printf("[TRACE] AttachSchemaTransformer: attaching provisioner %q schema to %s", name, dag.VertexName(v))
					tv.AttachProvisionerSchema(name, schema)
				} else {
					log.Printf("[ERROR] AttachSchemaTransformer: No schema available for provisioner %q on %q", name, dag.VertexName(v))
				}
			}
		}
	}

	return nil
}
