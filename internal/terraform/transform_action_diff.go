// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
)

// ActionDiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type ActionDiffTransformer struct {
	Changes *plans.ChangesSrc
	Config  *configs.Config
}

func (t *ActionDiffTransformer) Transform(g *Graph) error {
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
		actionConfig, ok := actionConfigNodes.GetOk(ai.Addr.ConfigAction())
		if !ok {
			return fmt.Errorf("no action config node found for action trigger %s", ai.ActionTrigger)
		}

		actionTriggerInstance := &actionTriggerApplyInstance{
			ActionInvocation: ai,
			actionNode:       actionConfig,
		}

		switch actionTrigger := ai.ActionTrigger.(type) {
		case *plans.ResourceActionTrigger:
			resourceInstances, ok := resourceInstanceNodes.GetOk(actionTrigger.TriggeringResourceAddr)
			if !ok {
				return fmt.Errorf("no resource node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

			foundNode := false

			// Add the action triggers to their instance nodes.
			for _, resourceInstance := range resourceInstances {
				invoker, ok := resourceInstance.(GraphNodeActionInvoker)
				if !ok {
					return fmt.Errorf("node %s type %T is not a GraphNodeActionInvoker", resourceInstance.ResourceInstanceAddr(), resourceInstance)
				}

				switch resourceInstance.(type) {
				case GraphNodeDestroyer:
					if !actionTrigger.ActionTriggerEvent.IsDestroy() {
						// we may have both create and destroy nodes under ths
						// same address, so make sure only destroy action triggers
						// are attached to destroyer nodes
						continue
					}

					// A destroy node might need to reevaluate the action in the
					// case of ephemeral values, so store the caller's change in
					// case it's needed later. We can't decode the change
					// however, so leave that to the caller to handle.
					changeSrc := t.Changes.ResourceInstance(actionTrigger.TriggeringResourceAddr)
					log.Printf("[DEBUG] ActionDiffTransformer: storing action destroy change src for %s", actionTrigger.TriggeringResourceAddr)
					actionTriggerInstance.callerChange = changeSrc

					invoker.AttachActionApplyTrigger(actionTriggerInstance)
					foundNode = true

				default:
					if actionTrigger.ActionTriggerEvent.IsDestroy() {
						continue
					}

					invoker.AttachActionApplyTrigger(actionTriggerInstance)
					foundNode = true
				}
			}

			if !foundNode {
				return fmt.Errorf("no resource node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

		case *plans.InvokeActionTrigger:
			// Add nodes for each action invocation
			node := &nodeActionInvokeApplyInstance{actionTriggerInstance}
			g.Add(node)
		}
	}

	return nil
}
