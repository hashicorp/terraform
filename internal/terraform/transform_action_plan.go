// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ActionPlanTransformer struct {
	Config    *configs.Config
	Operation walkOperation
}

func (t *ActionPlanTransformer) Transform(g *Graph) error {
	if t.Operation != walkPlan {
		return nil
	}
	return t.transform(g, t.Config)
}

func (t *ActionPlanTransformer) transform(g *Graph, config *configs.Config) error {
	// Add our resources
	if err := t.transformSingle(g, config); err != nil {
		return err
	}

	// Transform all the children without generating config.
	for _, c := range config.Children {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ActionPlanTransformer) transformSingle(g *Graph, config *configs.Config) error {
	for _, r := range config.Module.ManagedResources {
		for _, at := range r.Managed.ActionTriggers {
			for _, action := range at.Actions {
				ref, parseRefDiags := addrs.ParseRef(action.Traversal)
				if parseRefDiags != nil {
					return parseRefDiags.Err()
				}
				var instance addrs.AbsActionInstance

				switch ai := ref.Subject.(type) {
				case addrs.Action:
					instance = ai.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
				case addrs.ActionInstance:
					instance = ai.Absolute(addrs.RootModuleInstance)
				default:
					continue
				}
				nat := &nodeActionTriggerPlan{
					resourceAddress: r.Addr().InModule(addrs.RootModule),
					actionAddress:   instance,
					events:          at.Events,
				}

				g.Add(nat)
			}
		}
	}

	return nil
}
