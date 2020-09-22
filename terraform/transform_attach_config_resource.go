package terraform

import (
	"log"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeAttachResourceConfig is an interface that must be implemented by nodes
// that want resource configurations attached.
type GraphNodeAttachResourceConfig interface {
	GraphNodeConfigResource

	// Sets the configuration
	AttachResourceConfig(*configs.Resource)
}

// AttachResourceConfigTransformer goes through the graph and attaches
// resource configuration structures to nodes that implement
// GraphNodeAttachManagedResourceConfig or GraphNodeAttachDataResourceConfig.
//
// The attached configuration structures are directly from the configuration.
// If they're going to be modified, a copy should be made.
type AttachResourceConfigTransformer struct {
	Config *configs.Config // Config is the root node in the config tree
}

func (t *AttachResourceConfigTransformer) Transform(g *Graph) error {

	// Go through and find GraphNodeAttachResource
	for _, v := range g.Vertices() {
		// Only care about GraphNodeAttachResource implementations
		arn, ok := v.(GraphNodeAttachResourceConfig)
		if !ok {
			continue
		}

		// Determine what we're looking for
		addr := arn.ResourceAddr()

		// Get the configuration.
		config := t.Config.Descendent(addr.Module)
		if config == nil {
			log.Printf("[TRACE] AttachResourceConfigTransformer: %q (%T) has no configuration available", dag.VertexName(v), v)
			continue
		}

		for _, r := range config.Module.ManagedResources {
			rAddr := r.Addr()

			if rAddr != addr.Resource {
				// Not the same resource
				continue
			}

			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) config from %s", dag.VertexName(v), v, r.DeclRange)
			arn.AttachResourceConfig(r)

			// attach the provider_meta info
			if gnapmc, ok := v.(GraphNodeAttachProviderMetaConfigs); ok {
				log.Printf("[TRACE] AttachResourceConfigTransformer: attaching provider meta configs to %s", dag.VertexName(v))
				if config == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no config set on the transformer for %s", dag.VertexName(v))
					continue
				}
				if config.Module == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no module in config for %s", dag.VertexName(v))
					continue
				}
				if config.Module.ProviderMetas == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no provider metas defined for %s", dag.VertexName(v))
					continue
				}
				gnapmc.AttachProviderMetaConfigs(config.Module.ProviderMetas)
			}
		}
		for _, r := range config.Module.DataResources {
			rAddr := r.Addr()

			if rAddr != addr.Resource {
				// Not the same resource
				continue
			}

			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) config from %#v", dag.VertexName(v), v, r.DeclRange)
			arn.AttachResourceConfig(r)

			// attach the provider_meta info
			if gnapmc, ok := v.(GraphNodeAttachProviderMetaConfigs); ok {
				log.Printf("[TRACE] AttachResourceConfigTransformer: attaching provider meta configs to %s", dag.VertexName(v))
				if config == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no config set on the transformer for %s", dag.VertexName(v))
					continue
				}
				if config.Module == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no module in config for %s", dag.VertexName(v))
					continue
				}
				if config.Module.ProviderMetas == nil {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no provider metas defined for %s", dag.VertexName(v))
					continue
				}
				gnapmc.AttachProviderMetaConfigs(config.Module.ProviderMetas)
			}
		}
	}

	return nil
}
