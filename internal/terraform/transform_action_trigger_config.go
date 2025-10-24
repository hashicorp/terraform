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

type ActionTriggerConfigTransformer struct {
	Config        *configs.Config
	ActionTargets []addrs.Targetable
	Operation     walkOperation

	queryPlanMode bool

	ConcreteActionTriggerNodeFunc ConcreteActionTriggerNodeFunc
	CreateNodesAsAfter            bool
}

func (t *ActionTriggerConfigTransformer) Transform(g *Graph) error {
	// We don't want to run if we are using the query plan mode or have targets in place
	if (t.Operation != walkPlan && t.Operation != walkApply) || t.queryPlanMode || len(t.ActionTargets) > 0 {
		return nil
	}

	return t.transform(g, t.Config)
}

func (t *ActionTriggerConfigTransformer) transform(g *Graph, config *configs.Config) error {
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

func (t *ActionTriggerConfigTransformer) transformSingle(g *Graph, config *configs.Config) error {
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
		priorBeforeNodes := []dag.Vertex{}
		priorAfterNodes := []dag.Vertex{}
		for i, at := range r.Managed.ActionTriggers {
			containsBeforeEvent := false
			containsAfterEvent := false
			for _, event := range at.Events {
				switch event {
				case configs.BeforeCreate, configs.BeforeUpdate:
					containsBeforeEvent = true
				case configs.AfterCreate, configs.AfterUpdate:
					containsAfterEvent = true
				}
			}

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
						// nothing to do, these will get evaluated later
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

				abstract := &nodeAbstractActionTriggerExpand{
					Addr:   configAction,
					Config: actionConfig,
					lifecycleActionTrigger: &lifecycleActionTrigger{
						events:                  at.Events,
						resourceAddress:         resourceAddr,
						actionExpr:              action.Expr,
						actionTriggerBlockIndex: i,
						actionListIndex:         j,
						invokingSubject:         action.Expr.Range().Ptr(),
						conditionExpr:           at.Condition,
					},
				}

				// If CreateNodesAsAfter is set we want all nodes to run after the resource
				// If not we want expansion nodes only to exist if they are being used
				if !t.CreateNodesAsAfter && containsBeforeEvent {
					nat := t.ConcreteActionTriggerNodeFunc(abstract, RelativeActionTimingBefore)
					g.Add(nat)

					// We want to run before the resource nodes
					for _, node := range resourceNode {
						g.Connect(dag.BasicEdge(node, nat))
					}

					// We want to run after all prior nodes
					for _, priorNode := range priorBeforeNodes {
						g.Connect(dag.BasicEdge(nat, priorNode))
					}
					priorBeforeNodes = append(priorBeforeNodes, nat)
				}

				if t.CreateNodesAsAfter || containsAfterEvent {
					nat := t.ConcreteActionTriggerNodeFunc(abstract, RelativeActionTimingAfter)
					g.Add(nat)

					// We want to run after the resource nodes
					for _, node := range resourceNode {
						g.Connect(dag.BasicEdge(nat, node))
					}

					// We want to run after all prior nodes
					for _, priorNode := range priorAfterNodes {
						g.Connect(dag.BasicEdge(nat, priorNode))
					}
					priorAfterNodes = append(priorAfterNodes, nat)
				}
			}
		}
	}

	return nil
}
