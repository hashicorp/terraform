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
		switch actionTrigger := ai.ActionTrigger.(type) {
		case *plans.ResourceActionTrigger:
			atns, ok := resourceInstanceNodes.GetOk(actionTrigger.TriggeringResourceAddr)
			if !ok {
				return fmt.Errorf("no resource node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

			foundNode := false

			actionConfig, ok := actionConfigNodes.GetOk(ai.Addr.ConfigAction())
			if !ok {
				return fmt.Errorf("no action config node found for action trigger %s", actionTrigger.TriggeringResourceAddr)
			}

			// Add the action triggers to their instance nodes.
			for _, atn := range atns {
				if actionTrigger.ActionTriggerEvent.IsDestroy() {
					// FIXME: hard-coded types!
					if n, ok := atn.(*NodeDestroyResourceInstance); ok {
						fmt.Printf("FOUND DEST INST")
						n.actionApplyTriggers = append(n.actionApplyTriggers, &actionTriggerApplyInstance{
							ActionInvocation: ai,
							actionNode:       actionConfig,
						})
						foundNode = true

					}
					continue
				}

				if n, ok := atn.(*NodeApplyableResourceInstance); ok {
					n.actionApplyTriggers = append(n.actionApplyTriggers, &actionTriggerApplyInstance{
						ActionInvocation: ai,
						actionNode:       actionConfig,
					})
					foundNode = true
				}
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
