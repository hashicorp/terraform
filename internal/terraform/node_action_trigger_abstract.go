// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
)

// ConcreteActionTriggerNodeFunc is a callback type used to convert an
// abstract action trigger to a concrete one of some type.
type ConcreteActionTriggerNodeFunc func(*nodeAbstractActionTriggerExpand) dag.Vertex

type nodeAbstractActionTriggerExpand struct {
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
	_ GraphNodeReferencer = (*nodeAbstractActionTriggerExpand)(nil)
)

func (n *nodeAbstractActionTriggerExpand) Name() string {
	triggeredBy := "triggered by "
	if n.lifecycleActionTrigger != nil {
		triggeredBy += n.lifecycleActionTrigger.resourceAddress.String()
	} else {
		triggeredBy += "unknown"
	}

	return fmt.Sprintf("%s %s", n.Addr.String(), triggeredBy)
}

func (n *nodeAbstractActionTriggerExpand) ModulePath() addrs.Module {
	return n.Addr.Module
}

func (n *nodeAbstractActionTriggerExpand) References() []*addrs.Reference {
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

func (n *nodeAbstractActionTriggerExpand) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
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

func (n *nodeAbstractActionTriggerExpand) Provider() (provider addrs.Provider) {
	return n.Config.Provider
}

func (n *nodeAbstractActionTriggerExpand) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}
