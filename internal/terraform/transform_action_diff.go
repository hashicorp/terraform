// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

// ActionDiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type ActionDiffTransformer struct {
	Changes *plans.ChangesSrc
	Config  *configs.Config

	// Skip, when true, causes the transformer to be a no-op (e.g. during test runs).
	Skip bool
}

func (t *ActionDiffTransformer) Transform(g *Graph) error {
	if t.Skip {
		return nil
	}
	resourceInstanceNodes := addrs.MakeMap[addrs.AbsResourceInstance, []GraphNodeResourceInstance]()
	actionConfigNodes := addrs.MakeMap[addrs.ConfigAction, *NodeActionConfig]()

	// collect all the instance nodes, any of which could have action triggers
	for _, v := range g.Vertices() {
		switch v := v.(type) {
		case GraphNodeResourceInstance:
			instances := resourceInstanceNodes.Get(v.ResourceInstanceAddr())
			resourceInstanceNodes.Put(v.ResourceInstanceAddr(), append(instances, v))
		case *NodeActionConfig:
			actionConfigNodes.Put(v.ActionAddr(), v)
		}
	}

	for _, ai := range t.Changes.ActionInvocations {
		switch actionTrigger := ai.ActionTrigger.(type) {
		case *plans.ResourceActionTrigger:
			resourceInstances, ok := resourceInstanceNodes.GetOk(actionTrigger.TriggeringResourceAddr)
			if !ok {
				return fmt.Errorf("no resource node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

			actionConfig, ok := actionConfigNodes.GetOk(ai.Addr.ConfigAction())
			if !ok {
				return fmt.Errorf("no action config node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

			foundNode := false

			// Add the action triggers to their instance nodes.
			for _, resourceInstance := range resourceInstances {
				invoker, ok := resourceInstance.(GraphNodeActionInvoker)
				if !ok {
					return fmt.Errorf("node %s type %T is not a GraphNodeActionInvoker", resourceInstance.ResourceInstanceAddr(), resourceInstance)
				}

				if actionTrigger.ActionTriggerEvent.IsDestroy() {
					// we may have both create and destroy nodes under ths same
					// address, so match make sure the destroy action trigger is
					// only attached to a destroyer node
					if _, ok := resourceInstance.(GraphNodeDestroyer); !ok {
						continue
					}
				}

				invoker.AttachActionApplyTrigger(&actionTriggerApplyInstance{
					ActionInvocation: ai,
					actionNode:       actionConfig,
				})
				foundNode = true
			}
			if !foundNode {
				return fmt.Errorf("no resource node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}
		case *plans.InvokeActionTrigger:
			actionConfig, ok := actionConfigNodes.GetOk(ai.Addr.ConfigAction())
			if !ok {
				panic(fmt.Sprintf("FIXME: missing action for invoke: %s", ai.Addr))
			}

			// Add nodes for each action invocation
			node := &nodeActionInvokeApplyInstance{
				&actionTriggerApplyInstance{
					ActionInvocation: ai,
					actionNode:       actionConfig,
				},
			}
			g.Add(node)
		}
	}

	return nil
}
