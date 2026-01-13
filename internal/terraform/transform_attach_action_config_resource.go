// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// GraphNodeAttachResourceConfig is an interface that must be implemented by nodes
// that want resource configurations attached.
type GraphNodeAttachActionConfig interface {
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

func (t *AttachActionConfigTransformer) Transform(g *Graph) error {
	allConfigActions := make(map[string]*configs.Action)
	// get all the action configs from the config
	t.Config.DeepEach(func(c *configs.Config) {
		actions := c.Module.Actions
		for key, config := range actions {
			allConfigActions[key] = config
		}
	})

	// Go through and find GraphNodeAttachActionConfig nodes
	for _, v := range g.Vertices() {
		an, ok := v.(GraphNodeAttachActionConfig)
		if !ok {
			continue
		}

		actions := an.ActionAddrs()
		for _, action := range actions {
			actionCfg := allConfigActions[action.String()]
			if actionCfg == nil {
				return fmt.Errorf("[ERROR] action configuration not found for action %s", action.String())
			}
			an.AttachActionConfig(action, actionCfg)
		}
	}
	return nil
}
