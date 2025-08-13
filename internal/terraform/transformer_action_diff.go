// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
)

// ActionDiffTransformer is a GraphTransformer that adds graph nodes representing
// each of the resource changes described in the given Changes object.
type ActionDiffTransformer struct {
	Changes *plans.ChangesSrc
	Config  *configs.Config
}

func (t *ActionDiffTransformer) Transform(g *Graph) error {
	applyableNodes := addrs.MakeMap[addrs.AbsResource, *nodeExpandApplyableResource]()
	for _, vs := range g.Vertices() {
		applyableResource, ok := vs.(*nodeExpandApplyableResource)
		if !ok {
			continue
		}

		applyableNodes.Put(applyableResource.Addr.Absolute(addrs.RootModuleInstance), applyableResource)
	}

	for _, action := range t.Changes.ActionInvocations {
		// Add nodes for each action invocation

		node := &nodeActionTriggerApply{
			ActionInvocation: action,
		}
		fmt.Printf("\n\t nodeActionTriggerApply --> %#v \n", node)

		g.Add(node)

		// Add edges
		if lat, ok := action.ActionTrigger.(plans.LifecycleActionTrigger); ok {
			// Add edges for lifecycle action triggers
			resourceNode, ok := applyableNodes.GetOk(lat.TriggeringResourceAddr.ContainingResource())
			if !ok {
				panic("Could not find resource node for lifecycle action trigger")
			}

			switch lat.ActionTriggerEvent {
			case configs.BeforeCreate, configs.BeforeUpdate, configs.BeforeDestroy:
				g.Connect(dag.BasicEdge(resourceNode, node))
			case configs.AfterCreate, configs.AfterUpdate, configs.AfterDestroy:
				g.Connect(dag.BasicEdge(node, resourceNode))
			default:
				panic("Unknown event")
			}

			others := laterInvocationsfindLaterActionInvocation(t.Changes.ActionInvocations, action)
			for _, other := range others {
				g.Connect(dag.BasicEdge(other, node))
			}
		}
	}
	return nil
}

func laterInvocationsfindLaterActionInvocation(actionInvocations []*plans.ActionInvocationInstanceSrc, needle *plans.ActionInvocationInstanceSrc) []*plans.ActionInvocationInstanceSrc {
	needleLat := needle.ActionTrigger.(plans.LifecycleActionTrigger)

	var laterInvocations []*plans.ActionInvocationInstanceSrc
	for _, invocation := range actionInvocations {
		if lat, ok := invocation.ActionTrigger.(plans.LifecycleActionTrigger); ok {
			if lat.TriggeringResourceAddr.Equal(needleLat.TriggeringResourceAddr) && (lat.ActionTriggerBlockIndex > needleLat.ActionTriggerBlockIndex || lat.ActionTriggerBlockIndex == needleLat.ActionTriggerBlockIndex && lat.ActionsListIndex > needleLat.ActionsListIndex) {
				laterInvocations = append(laterInvocations, invocation)
			}
		}
	}
	return laterInvocations
}
