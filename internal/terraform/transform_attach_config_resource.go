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
	// Collect removed blocks to attach to any resources. These are collected
	// independently because removed blocks may live in a parent module of the
	// resource referenced.
	removed := addrs.MakeMap[addrs.ConfigResource, *configs.Removed]()

	t.Config.DeepEach(func(c *configs.Config) {
		for _, rem := range c.Module.Removed {
			resAddr, ok := rem.From.RelSubject.(addrs.ConfigResource)
			if !ok {
				// Not for a resource at all, so can't possibly match.
				// Non-resource removed targets have nothing to attach.
				continue
			}
			removed.Put(resAddr, rem)
		}
	})

	// Go through and find GraphNodeAttachResource
	for _, v := range g.Vertices() {
		// Only care about GraphNodeAttachResource implementations
		arn, ok := v.(GraphNodeAttachResourceConfig)
		if !ok {
			continue
		}

		// Determine what we're looking for
		addr := arn.ResourceAddr()

		// Check for a removed block first, since that would preclude any resource config.
		if remCfg, ok := removed.GetOk(addr); ok {
			log.Printf("[TRACE] AttachResourceConfigTransformer: attaching to %q (%T) removed block from %#v", dag.VertexName(v), v, remCfg.DeclRange)
			arn.AttachResourceConfig(nil, remCfg)
		}

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
	}

	return nil
}
