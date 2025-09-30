// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerPlanExpand struct {
	Addr             addrs.ConfigAction
	resolvedProvider addrs.AbsProviderConfig
	Config           *configs.Action

	lifecycleActionTrigger *lifecycleActionTrigger
}
type lifecycleActionTrigger struct {
	resourceAddress         addrs.ConfigResource
	events                  []configs.ActionTriggerEvent
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
	actionExpr              hcl.Expression
	conditionExpr           hcl.Expression
}

func (at *lifecycleActionTrigger) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ GraphNodeDynamicExpandable = (*nodeActionTriggerPlanExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionTriggerPlanExpand)(nil)
	_ GraphNodeReferenceable     = (*nodeActionTriggerPlanExpand)(nil)
)

func (n *nodeActionTriggerPlanExpand) Name() string {
	triggeredBy := "triggered by "
	if n.lifecycleActionTrigger != nil {
		triggeredBy += n.lifecycleActionTrigger.resourceAddress.String()
	} else {
		triggeredBy += "unknown"
	}

	return fmt.Sprintf("%s %s", n.Addr.String(), triggeredBy)
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

func (n *nodeActionTriggerPlanExpand) ModulePath() addrs.Module {
	return n.Addr.Module
}

func (n *nodeActionTriggerPlanExpand) References() []*addrs.Reference {
	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: n.Addr.Action,
	})

	if n.lifecycleActionTrigger != nil {
		refs = append(refs, &addrs.Reference{
			Subject: n.lifecycleActionTrigger.resourceAddress.Resource,
		})

		conditionRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.lifecycleActionTrigger.conditionExpr)
		refs = append(refs, conditionRefs...)
	}

	return refs
}

func (n *nodeActionTriggerPlanExpand) ReferenceableAddrs() []addrs.Referenceable {
	// the action triggers themselves aren't directly referenceable, but during
	// planning we do want to force any dependents on the resource to wait for
	// any triggered actions to be planned in case the action results in the
	// original resource being deferred. therefore, we expose the address of the
	// underlying resource as being the referenceable address for this node.
	return []addrs.Referenceable{n.lifecycleActionTrigger.resourceAddress.Resource}
}

func (n *nodeActionTriggerPlanExpand) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	if n.resolvedProvider.Provider.Type != "" {
		return n.resolvedProvider, true
	}

	// Since we always have a config, we can use it
	relAddr := n.Config.ProviderConfigAddr()
	return addrs.LocalProviderConfig{
		LocalName: relAddr.LocalName,
		Alias:     relAddr.Alias,
	}, false
}

func (n *nodeActionTriggerPlanExpand) Provider() (provider addrs.Provider) {
	return n.Config.Provider
}

func (n *nodeActionTriggerPlanExpand) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}
