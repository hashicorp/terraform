// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

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
	actionConfigs := addrs.MakeMap[addrs.ConfigAction, *configs.Action]()
	for _, a := range config.Module.Actions {
		actionConfigs.Put(a.Addr().InModule(config.Path), a)
	}

	for _, r := range config.Module.ManagedResources {
		for i, at := range r.Managed.ActionTriggers {
			for j, action := range at.Actions {
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
					// This should have been caught during validation
					panic(fmt.Sprintf("unexpected action address %T", ai))
				}

				actionConfig, ok := actionConfigs.GetOk(instance.ConfigAction())
				if !ok {
					// This should have been caught during validation
					panic(fmt.Sprintf("actionConfig not found for %s", instance))
				}

				nat := &nodeActionTriggerPlan{
					actionAddress: instance,
					actionConfig:  actionConfig,
					lifecycleActionTrigger: &lifecycleActionTrigger{
						events:                  at.Events,
						resourceAddress:         r.Addr().InModule(config.Path),
						actionTriggerBlockIndex: i,
						actionListIndex:         j,
						invokingSubject:         action.Traversal.SourceRange().Ptr(),
					},
				}

				g.Add(nat)
			}
		}
	}

	return nil
}
