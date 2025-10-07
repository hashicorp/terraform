// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerPlanExpand struct {
	*nodeAbstractActionTriggerExpand
}

var (
	_ GraphNodeDynamicExpandable = (*nodeActionTriggerPlanExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionTriggerPlanExpand)(nil)
)

func (n *nodeActionTriggerPlanExpand) Name() string {
	return fmt.Sprintf("%s (plan)", n.nodeAbstractActionTriggerExpand.Name())
}

func (n *nodeActionTriggerPlanExpand) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics

	if n.lifecycleActionTrigger == nil {
		panic("Only actions triggered by plan and apply are supported")
	}

	expander := ctx.InstanceExpander()

	// The possibility of partial-expanded modules and resources is guarded by a
	// top-level option for the whole plan, so that we can preserve mainline
	// behavior for the modules runtime. So, we currently branch off into an
	// entirely-separate codepath in those situations, at the expense of
	// duplicating some of the logic for behavior this method would normally
	// handle.
	if ctx.Deferrals().DeferralAllowed() {
		pem := expander.UnknownModuleInstances(n.Addr.Module, false)

		for _, moduleAddr := range pem {
			actionAddr := moduleAddr.Action(n.Addr.Action)
			resourceAddr := moduleAddr.Resource(n.lifecycleActionTrigger.resourceAddress.Resource)

			// And add a node to the graph for this action.
			g.Add(&NodeActionTriggerPartialExpanded{
				addr:             actionAddr,
				config:           n.Config,
				resolvedProvider: n.resolvedProvider,
				lifecycleActionTrigger: &lifecycleActionTriggerPartialExpanded{
					resourceAddress:         resourceAddr,
					events:                  n.lifecycleActionTrigger.events,
					actionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
					actionListIndex:         n.lifecycleActionTrigger.actionListIndex,
					invokingSubject:         n.lifecycleActionTrigger.invokingSubject,
				},
			})
		}
	}

	// First we expand the module
	moduleInstances := expander.ExpandModule(n.lifecycleActionTrigger.resourceAddress.Module, false)
	for _, module := range moduleInstances {
		_, keys, _ := expander.ResourceInstanceKeys(n.lifecycleActionTrigger.resourceAddress.Absolute(module))
		for _, key := range keys {
			absResourceInstanceAddr := n.lifecycleActionTrigger.resourceAddress.Absolute(module).Instance(key)

			// The n.Addr was derived from the ActionRef hcl.Expression referenced inside the resource's lifecycle block, and has not yet been
			// expanded or fully evaluated, so we will do that now.
			// Grab the instance key, necessary if the action uses [count.index] or [each.key]
			repData := instances.RepetitionData{}
			switch k := key.(type) {
			case addrs.IntKey:
				repData.CountIndex = k.Value()
			case addrs.StringKey:
				repData.EachKey = k.Value()
				repData.EachValue = cty.DynamicVal
			}

			ref, evalActionDiags := evaluateActionExpression(n.lifecycleActionTrigger.actionExpr, repData)
			diags = append(diags, evalActionDiags...)
			if diags.HasErrors() {
				continue
			}

			// The reference is either an action or action instance
			var actionAddr addrs.AbsActionInstance
			switch sub := ref.Subject.(type) {
			case addrs.Action:
				actionAddr = sub.Absolute(module).Instance(addrs.NoKey)
			case addrs.ActionInstance:
				actionAddr = sub.Absolute(module)
			}

			node := nodeActionTriggerPlanInstance{
				actionAddress:    actionAddr,
				resolvedProvider: n.resolvedProvider,
				actionConfig:     n.Config,
				lifecycleActionTrigger: &lifecycleActionTriggerInstance{
					resourceAddress:         absResourceInstanceAddr,
					events:                  n.lifecycleActionTrigger.events,
					actionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
					actionListIndex:         n.lifecycleActionTrigger.actionListIndex,
					invokingSubject:         n.lifecycleActionTrigger.invokingSubject,
					conditionExpr:           n.lifecycleActionTrigger.conditionExpr,
				},
			}

			g.Add(&node)
		}
	}

	addRootNodeToGraph(&g)
	return &g, diags
}
