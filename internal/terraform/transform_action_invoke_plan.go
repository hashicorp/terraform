// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

type ActionInvokePlanTransformer struct {
	Config        *configs.Config
	ActionTargets []addrs.Targetable
	Operation     walkOperation

	queryPlanMode bool
}

func (t *ActionInvokePlanTransformer) Transform(g *Graph) error {
	if t.Operation != walkPlan || t.queryPlanMode || len(t.ActionTargets) == 0 {
		return nil
	}

	// We could be invoking an action which is triggered from a resource,
	// requiring us to resolve `caller`. In that case the calling resource
	// invokes the action rather than a standalone invoke node.
	//
	// calledActions maps the action addresses to the callers, so we can lookup
	// the resource below.
	calledActions := addrs.MakeMap[addrs.ConfigAction, []GraphNodeActionCaller]()
	for _, v := range g.Vertices() {
		caller, ok := v.(GraphNodeActionCaller)
		if !ok {
			continue
		}

		for _, target := range t.ActionTargets {
			for _, callee := range caller.ActionCalls() {
				if target.TargetContains(callee) {
					callers := calledActions.Get(callee)
					callers = append(callers, caller)
					// this resource invokes the calling node
					calledActions.Put(callee, callers)
				}
			}
		}
	}

	for _, v := range g.Vertices() {
		actionNode, ok := v.(*NodeActionConfig)
		if !ok {
			continue
		}

		for _, target := range t.ActionTargets {
			if !target.TargetContains(actionNode.Addr) {
				continue
			}

			// These wil be all resources calling an the action if it uses
			// "caller". We need to record these so standalone invoke nodes know
			// how to evaluate the configuration. We will also end up creating
			// separate invoke nodes for every caller.
			var resourceCallers []addrs.ConfigResource

			// first check if the action is using "caller" at all. If not just
			// continue with the standalone invoke path below
			if callers, ok := calledActions.GetOk(actionNode.Addr); ok {
				for _, ref := range actionNode.References() {
					if ref.Subject == addrs.Caller {
						for _, caller := range callers {
							resourceCallers = append(resourceCallers, caller.ResourceAddr())
						}
						break
					}
				}
			}

			// we need to create the invoke node in the correct module scope for each target
			var instAddr addrs.AbsActionInstance

			switch target := target.(type) {
			case addrs.AbsActionInstance:
				instAddr = target
			case addrs.AbsAction:
				instAddr = target.Instance(addrs.NoKey)
			default:
				panic(fmt.Sprintf("invalid action addr: %#v", target))
			}

			g.Add(&nodeActionInvokeExpand{
				Target:       target,
				Module:       actionNode.Addr.Module,
				Addr:         instAddr,
				ActionConfig: actionNode,
				Callers:      resourceCallers,
			})
		}
	}

	return nil
}
