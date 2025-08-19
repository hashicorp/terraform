// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerPlanExpand struct {
	Addr             addrs.ConfigAction
	resolvedProvider addrs.AbsProviderConfig
	Config           *configs.Action

	lifecycleActionTrigger *lifecycleActionTrigger
}

type lifecycleActionTrigger struct {
	resourceAddress addrs.ConfigResource
	events          []configs.ActionTriggerEvent
	//condition       hcl.Expression
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
}

func (at *lifecycleActionTrigger) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ GraphNodeDynamicExpandable = (*nodeActionTriggerPlanExpand)(nil)
	_ GraphNodeReferencer        = (*nodeActionTriggerPlanExpand)(nil)
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
	// First we expand the module
	moduleInstances := expander.ExpandModule(n.lifecycleActionTrigger.resourceAddress.Module, false)

	for _, module := range moduleInstances {
		actionAddr := n.Addr.Action.Absolute(module)

		_, keys, _ := expander.ResourceInstanceKeys(n.lifecycleActionTrigger.resourceAddress.Absolute(module))
		for _, key := range keys {
			absResourceInstanceAddr := n.lifecycleActionTrigger.resourceAddress.Absolute(module).Instance(key)

			// this is the action referenced inside the resource's lifecycle block, so it uses the resource's key.
			absActionAddr := actionAddr.Instance(key)

			node := nodeActionTriggerPlanInstance{
				actionAddress:    absActionAddr,
				resolvedProvider: n.resolvedProvider,
				actionConfig:     n.Config,
				lifecycleActionTrigger: &lifecycleActionTriggerInstance{
					resourceAddress:         absResourceInstanceAddr,
					events:                  n.lifecycleActionTrigger.events,
					actionTriggerBlockIndex: n.lifecycleActionTrigger.actionTriggerBlockIndex,
					actionListIndex:         n.lifecycleActionTrigger.actionListIndex,
					invokingSubject:         n.lifecycleActionTrigger.invokingSubject,
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
	}

	return refs
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
