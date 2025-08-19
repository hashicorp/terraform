// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
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

	// Transform all the children.
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

	resourceNodes := addrs.MakeMap[addrs.ConfigResource, []GraphNodeConfigResource]()
	for _, node := range g.Vertices() {
		rn, ok := node.(GraphNodeConfigResource)
		if !ok {
			continue
		}
		// We ignore any instances that _also_ implement
		// GraphNodeResourceInstance, since in the unlikely event that they
		// do exist we'd probably end up creating cycles by connecting them.
		if _, ok := node.(GraphNodeResourceInstance); ok {
			continue
		}

		rAddr := rn.ResourceAddr()
		resourceNodes.Put(rAddr, append(resourceNodes.Get(rAddr), rn))
	}

	for _, r := range config.Module.ManagedResources {
		priorNodes := []*nodeActionTriggerPlanExpand{}
		for i, at := range r.Managed.ActionTriggers {
			for j, action := range at.Actions {
				refs, parseRefDiags := langrefs.ReferencesInExpr(addrs.ParseRef, action.Expr)
				if parseRefDiags != nil {
					return parseRefDiags.Err()
				}

				var configAction addrs.ConfigAction

				for _, ref := range refs {
					switch a := ref.Subject.(type) {
					case addrs.Action:
						configAction = a.InModule(config.Path)
					case addrs.ActionInstance:
						configAction = a.Action.InModule(config.Path)
					case addrs.CountAttr, addrs.ForEachAttr:
						// nothing to do, these will get evaluate later
					default:
						// This should have been caught during validation
						panic(fmt.Sprintf("unexpected action address %T", a))
					}
				}

				actionConfig, ok := actionConfigs.GetOk(configAction)
				if !ok {
					// This should have been caught during validation
					panic(fmt.Sprintf("action config not found for %s", configAction))
				}

				resourceAddr := r.Addr().InModule(config.Path)
				resourceNode, ok := resourceNodes.GetOk(resourceAddr)
				if !ok {
					panic(fmt.Sprintf("Could not find node for %s", resourceAddr))
				}

				nat := &nodeActionTriggerPlanExpand{
					Addr:   configAction,
					Config: actionConfig,
					lifecycleActionTrigger: &lifecycleActionTrigger{
						events:                  at.Events,
						resourceAddress:         resourceAddr,
						actionTriggerBlockIndex: i,
						actionListIndex:         j,
						invokingSubject:         action.Expr.Range().Ptr(),
					},
				}

				g.Add(nat)

				// We always want to plan after the resource is done planning
				for _, node := range resourceNode {
					g.Connect(dag.BasicEdge(nat, node))
				}

				// We want to plan after all prior nodes
				for _, priorNode := range priorNodes {
					g.Connect(dag.BasicEdge(nat, priorNode))
				}
				priorNodes = append(priorNodes, nat)
			}
		}
	}

	return nil
}
