// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// GraphNodeAttachResourceConfig is an interface that must be implemented by nodes
// that want resource configurations attached.
type GraphNodeAttachResourceConfig interface {
	GraphNodeConfigResource

	// Sets the configuration, either to a present resource block or to
	// a "removed" block commemorating a resource that has since been
	// removed. Callers should always leave at least one of these
	// arguments set to nil.
	AttachResourceConfig(*configs.Resource, *configs.Removed)
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

		if r := config.Module.ResourceByAddr(addr.Resource); r != nil {
			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) config from %#v", dag.VertexName(v), v, r.DeclRange)
			arn.AttachResourceConfig(r, nil)
			if gnapmc, ok := v.(GraphNodeAttachProviderMetaConfigs); ok {
				log.Printf("[TRACE] AttachResourceConfigTransformer: attaching provider meta configs to %s", dag.VertexName(v))
				if config.Module.ProviderMetas != nil {
					gnapmc.AttachProviderMetaConfigs(config.Module.ProviderMetas)
				} else {
					log.Printf("[TRACE] AttachResourceConfigTransformer: no provider meta configs available to attach to %s", dag.VertexName(v))
				}
			}
		}

		for _, r := range config.Module.Removed {
			crAddr, ok := r.From.RelSubject.(addrs.ConfigResource)
			if !ok {
				// Not for a resource at all, so can't possibly match
				continue
			}
			rAddr := crAddr.Resource
			if rAddr != addr.Resource {
				// Not the same resource
				continue
			}

			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) removed block from %#v", dag.VertexName(v), v, r.DeclRange)

			// Validation ensures that there can't be both a resource/data block
			// and a removed block referring to the same configuration, so
			// we can assume that this isn't clobbering a non-removed resource
			// configuration we already attached above.
			arn.AttachResourceConfig(nil, r)
		}

		for _, r := range config.Module.EphemeralResources {
			rAddr := r.Addr()

			if rAddr != addr.Resource {
				// Not the same resource
				continue
			}

			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) config from %#v", dag.VertexName(v), v, r.DeclRange)
			arn.AttachResourceConfig(r, nil)

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
