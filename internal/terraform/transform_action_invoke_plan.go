// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ActionInvokePlanTransformer struct {
	Config          *configs.Config
	ActionTargets   []addrs.TargetPattern
	ResourceTargets []addrs.TargetPattern
	Operation       walkOperation

	queryPlanMode bool
}

func (t *ActionInvokePlanTransformer) Transform(g *Graph) error {
	if t.Operation != walkPlan || t.queryPlanMode || len(t.ActionTargets) == 0 {
		return nil
	}

	// we need to track the targets we've made use of so we can report back to
	// the user when some target is not found.
	targetSet := addrs.MakeSet(t.ActionTargets...)

	// We could be invoking an action which is triggered from a resource,
	// requiring us to resolve `caller`. In that case the calling resource
	// invokes the action rather than a standalone invoke node.
	//
	// calledActions maps the action addresses to the callers, so we can lookup
	// the resource below.
	calledActions := addrs.MakeMap[addrs.ConfigAction, []GraphNodeActionCaller]()
	for v := range g.VerticesSeq() {
		caller, ok := v.(GraphNodeActionCaller)
		if !ok {
			continue
		}

		for _, target := range t.ActionTargets {
			targetConfig, ok := target.ConfigAddr().(addrs.ConfigAction)
			if !ok {
				continue
			}
			for _, callee := range caller.ActionCalls() {
				if targetConfig.Equal(callee) {
					callers := calledActions.Get(callee)
					callers = append(callers, caller)
					// this resource invokes the calling node
					calledActions.Put(callee, callers)
				}
			}
		}
	}

	for v := range g.VerticesSeq() {
		actionNode, ok := v.(*NodeActionConfig)
		if !ok {
			continue
		}

		for _, target := range t.ActionTargets {
			targetConfig, ok := target.ConfigAddr().(addrs.ConfigAction)
			if !ok || !targetConfig.Equal(actionNode.Addr) {
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

			g.Add(&nodeActionInvokeExpand{
				Target:          target,
				ResourceTargets: t.ResourceTargets,
				Module:          actionNode.Addr.Module,
				ActionConfig:    actionNode,
				Callers:         resourceCallers,
			})
			targetSet.Remove(target)
		}
	}

	var diags tfdiags.Diagnostics
	for target := range targetSet.Iter() {
		diags = diags.Append(fmt.Errorf("invoke target %s not found", target))
	}

	return diags.ErrWithWarnings()
}
