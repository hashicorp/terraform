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
	applyNodes := addrs.MakeMap[addrs.ConfigResource, *nodeExpandApplyableResource]()
	for _, vs := range g.Vertices() {
		applyableResource, ok := vs.(*nodeExpandApplyableResource)
		if !ok {
			continue
		}

		applyNodes.Put(applyableResource.Addr, applyableResource)
	}

	invocationMap := map[*plans.ActionInvocationInstanceSrc]*nodeActionTriggerApply{}
	for _, action := range t.Changes.ActionInvocations {
		// Add nodes for each action invocation
		node := &nodeActionTriggerApply{
			ActionInvocation: action,
		}

		// If the action invocations is triggered within the lifecycle of a resource
		// we want to add information about the source location to the apply node
		if at, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger); ok {
			moduleInstance := t.Config.DescendantForInstance(at.TriggeringResourceAddr.Module)
			if moduleInstance == nil {
				panic(fmt.Sprintf("Could not find module instance for resource %s in config", at.TriggeringResourceAddr.String()))
			}

			resourceInstance := moduleInstance.Module.ResourceByAddr(at.TriggeringResourceAddr.Resource.Resource)
			if resourceInstance == nil {
				panic(fmt.Sprintf("Could not find resource instance for resource %s in config", at.TriggeringResourceAddr.String()))
			}

			triggerBlock := resourceInstance.Managed.ActionTriggers[at.ActionTriggerBlockIndex]
			if triggerBlock == nil {
				panic(fmt.Sprintf("Could not find action trigger block %d for resource %s in config", at.ActionTriggerBlockIndex, at.TriggeringResourceAddr.String()))
			}

			act := triggerBlock.Actions[at.ActionsListIndex]
			node.ActionTriggerRange = &act.Range
		}

		g.Add(node)
		invocationMap[action] = node

		// Add edge to triggering resource
		if lat, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger); ok {
			// Add edges for lifecycle action triggers
			resourceNode, ok := applyNodes.GetOk(lat.TriggeringResourceAddr.ConfigResource())
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
		}
	}

	// Find all dependencies between action invocations
	for _, action := range t.Changes.ActionInvocations {
		if _, ok := action.ActionTrigger.(*plans.LifecycleActionTrigger); !ok {
			// only add dependencies between lifecycle actions. invoke actions
			// all act independently.
			continue
		}

		node := invocationMap[action]
		others := action.FilterLaterActionInvocations(t.Changes.ActionInvocations)
		for _, other := range others {
			otherNode := invocationMap[other]
			g.Connect(dag.BasicEdge(otherNode, node))
		}
	}
	return nil
}
