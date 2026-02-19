// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerApplyExpand struct {
	*nodeAbstractActionTriggerExpand

	actionInvocationInstances []*plans.ActionInvocationInstanceSrc
	relativeTiming            RelativeActionTiming
}

var (
	_ GraphNodeDynamicExpandable = (*nodeActionTriggerApplyExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionTriggerApplyExpand)(nil)
	_ GraphNodeProviderConsumer  = (*nodeActionTriggerApplyExpand)(nil)
	_ GraphNodeModulePath        = (*nodeActionTriggerApplyExpand)(nil)
)

func (n *nodeActionTriggerApplyExpand) Name() string {
	return fmt.Sprintf("%s (apply - %s)", n.nodeAbstractActionTriggerExpand.Name(), n.relativeTiming)
}

func (n *nodeActionTriggerApplyExpand) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics

	if n.lifecycleActionTrigger == nil {
		panic("Only actions triggered by plan and apply are supported")
	}

	invocationMap := map[*plans.ActionInvocationInstanceSrc]*nodeActionTriggerApplyInstance{}
	// We already planned the action invocations, so we can just add them to the graph
	for _, ai := range n.actionInvocationInstances {
		node := &nodeActionTriggerApplyInstance{
			ActionInvocation:   ai,
			resolvedProvider:   n.resolvedProvider,
			ActionTriggerRange: n.lifecycleActionTrigger.invokingSubject.Ptr(),
			ConditionExpr:      n.lifecycleActionTrigger.conditionExpr,
		}
		g.Add(node)
		invocationMap[ai] = node
	}

	for _, ai := range n.actionInvocationInstances {
		node := invocationMap[ai]
		others := ai.FilterLaterActionInvocations(n.actionInvocationInstances)
		for _, other := range others {
			otherNode := invocationMap[other]
			g.Connect(dag.BasicEdge(otherNode, node))
		}
	}

	addRootNodeToGraph(&g)
	return &g, diags
}

func (n *nodeActionTriggerApplyExpand) SetActionInvocationInstances(instances []*plans.ActionInvocationInstanceSrc) {
	n.actionInvocationInstances = instances
}

func (n *nodeActionTriggerApplyExpand) References() []*addrs.Reference {
	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: n.Addr.Action,
	})

	if n.lifecycleActionTrigger != nil {
		conditionRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.lifecycleActionTrigger.conditionExpr)
		refs = append(refs, conditionRefs...)
	}

	return refs
}
