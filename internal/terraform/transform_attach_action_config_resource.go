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
type GraphNodeAttachActionConfig interface {
	GraphNodeModulePath
	// Sets the action config for any actions returned by ActionAddrs()
	AttachActionConfig(addrs.ConfigAction, *configs.Action)

	// Actions referenced in the resource's lifecycle
	ActionAddrs() []addrs.ConfigAction
}

// AttachActionConfigTransformer goes through the graph and attaches
// action configuration structures to resource nodes that implement
// GraphNodeAttachActionConfig.
//
// The attached configuration structures are directly from the configuration.
// If they're going to be modified, a copy should be made.
type AttachActionConfigTransformer struct {
	Config *configs.Config // Config is the root node in the config tree
}

// @mildwonkey missing module?
func (t *AttachActionConfigTransformer) Transform(g *Graph) error {
	// Go through and find GraphNodeAttachActionConfig nodes
	for _, v := range g.Vertices() {
		an, ok := v.(GraphNodeAttachActionConfig)
		if !ok {
			continue
		}

		actions := an.ActionAddrs()
		for _, action := range actions {
			// Get the configuration.
			config := t.Config.Descendant(an.ModulePath())
			if config == nil {
				log.Printf("[TRACE] AttachActionConfigTransformer: %q (%T) has no configuration available", dag.VertexName(v), v)
				continue
			}
			if a := config.Module.ActionByAddr(action.Action); a != nil {
				log.Printf("[TRACE] AttachActionConfigTransformer: attaching to %q (%T) config from %#v", dag.VertexName(v), v, a.DeclRange)
				an.AttachActionConfig(action, a)
			}
		}
	}
	return nil
}
