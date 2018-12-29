package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/dag"
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
	Schemas *Schemas
}

func (t *AttachSchemaTransformer) Transform(g *Graph) error {
	if t.Schemas == nil {
		// Should never happen with a reasonable caller, but we'll return a
		// proper error here anyway so that we'll fail gracefully.
		return fmt.Errorf("AttachSchemaTransformer used with nil Schemas")
	}

	for _, v := range g.Vertices() {

		if tv, ok := v.(GraphNodeAttachResourceSchema); ok {
			addr := tv.ResourceAddr()
			mode := addr.Resource.Mode
			typeName := addr.Resource.Type
			providerAddr, _ := tv.ProvidedBy()
			providerType := providerAddr.ProviderConfig.Type

			var schema *configschema.Block
			switch mode {
			case addrs.ManagedResourceMode:
				schema = t.Schemas.ResourceTypeConfig(providerType, typeName)
			case addrs.DataResourceMode:
				schema = t.Schemas.DataSourceConfig(providerType, typeName)
			}
			if schema == nil {
				log.Printf("[ERROR] AttachSchemaTransformer: No resource schema available for %s", addr)
				continue
			}
			log.Printf("[TRACE] AttachSchemaTransformer: attaching resource schema to %s", dag.VertexName(v))
			tv.AttachResourceSchema(schema)
		}

		if tv, ok := v.(GraphNodeAttachProviderConfigSchema); ok {
			providerAddr := tv.ProviderAddr()
			schema := t.Schemas.ProviderConfig(providerAddr.ProviderConfig.Type)
			if schema == nil {
				log.Printf("[ERROR] AttachSchemaTransformer: No provider config schema available for %s", providerAddr)
				continue
			}
			log.Printf("[TRACE] AttachSchemaTransformer: attaching provider config schema to %s", dag.VertexName(v))
			tv.AttachProviderConfigSchema(schema)
		}

		if tv, ok := v.(GraphNodeAttachProvisionerSchema); ok {
			names := tv.ProvisionedBy()
			for _, name := range names {
				schema := t.Schemas.ProvisionerConfig(name)
				if schema == nil {
					log.Printf("[ERROR] AttachSchemaTransformer: No schema available for provisioner %q on %q", name, dag.VertexName(v))
					continue
				}
				log.Printf("[TRACE] AttachSchemaTransformer: attaching provisioner %q config schema to %s", name, dag.VertexName(v))
				tv.AttachProvisionerSchema(name, schema)
			}
		}
	}

	return nil
}
