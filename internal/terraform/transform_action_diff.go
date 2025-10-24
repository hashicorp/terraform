// Copyright (c) HashiCorp, Inc.
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
	actionTriggerNodes := addrs.MakeMap[addrs.ConfigResource, []*nodeActionTriggerApplyExpand]()
	for _, vs := range g.Vertices() {
		if atn, ok := vs.(*nodeActionTriggerApplyExpand); ok {
			configResource := actionTriggerNodes.Get(atn.lifecycleActionTrigger.resourceAddress)
			actionTriggerNodes.Put(atn.lifecycleActionTrigger.resourceAddress, append(configResource, atn))
		}
	}

	for _, ai := range t.Changes.ActionInvocations {
		lat, ok := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
		if !ok {
			continue
		}
		isBefore := lat.ActionTriggerEvent == configs.BeforeCreate || lat.ActionTriggerEvent == configs.BeforeUpdate
		isAfter := lat.ActionTriggerEvent == configs.AfterCreate || lat.ActionTriggerEvent == configs.AfterUpdate

		atns, ok := actionTriggerNodes.GetOk(lat.TriggeringResourceAddr.ConfigResource())
		if !ok {
			return fmt.Errorf("no action trigger nodes found for resource %s", lat.TriggeringResourceAddr)
		}
		// We add the action invocations one by one
		for _, atn := range atns {
			beforeMatches := atn.relativeTiming == RelativeActionTimingBefore && isBefore
			afterMatches := atn.relativeTiming == RelativeActionTimingAfter && isAfter

			if (beforeMatches || afterMatches) && atn.lifecycleActionTrigger.actionTriggerBlockIndex == lat.ActionTriggerBlockIndex && atn.lifecycleActionTrigger.actionListIndex == lat.ActionsListIndex {
				atn.actionInvocationInstances = append(atn.actionInvocationInstances, ai)
			}
		}
	}

	return nil
}
