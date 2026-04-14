// Copyright IBM Corp. 2014, 2026
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

type RelativeActionTiming = string

const (
	RelativeActionTimingBefore = "before"
	RelativeActionTimingAfter  = "after"
)

// ConcreteActionTriggerNodeFunc is a callback type used to convert an
// abstract action trigger to a concrete one of some type.
type ConcreteActionTriggerNodeFunc func(*nodeAbstractActionTrigger, RelativeActionTiming) dag.Vertex

type nodeAbstractActionTrigger struct {
	Addr             addrs.ConfigAction
	resolvedProvider addrs.AbsProviderConfig
	Config           *configs.Action

	triggerConfig *actionTriggerConfig
}

type actionTriggerConfig struct {
	resourceAddress         addrs.ConfigResource
	events                  []configs.ActionTriggerEvent
	actionTriggerBlockIndex int
	actionListIndex         int
	invokingSubject         *hcl.Range
	actionExpr              hcl.Expression
	conditionExpr           hcl.Expression
}

func (at *actionTriggerConfig) Name() string {
	return fmt.Sprintf("%s.lifecycle.action_trigger[%d].actions[%d]", at.resourceAddress.String(), at.actionTriggerBlockIndex, at.actionListIndex)
}

var (
	_ GraphNodeReferencer       = (*nodeAbstractActionTrigger)(nil)
	_ GraphNodeProviderConsumer = (*nodeAbstractActionTrigger)(nil)
	_ GraphNodeModulePath       = (*nodeAbstractActionTrigger)(nil)
)

func (n *nodeAbstractActionTrigger) Name() string {
	return fmt.Sprintf("%s triggered by %s", n.Addr.String(), n.triggerConfig.resourceAddress.String())
}

func (n *nodeAbstractActionTrigger) ModulePath() addrs.Module {
	return n.Addr.Module
}

func (n *nodeAbstractActionTrigger) References() []*addrs.Reference {
	var refs []*addrs.Reference
	refs = append(refs, &addrs.Reference{
		Subject: n.Addr.Action,
	})

	if n.triggerConfig != nil {
		refs = append(refs, &addrs.Reference{
			Subject: n.triggerConfig.resourceAddress.Resource,
		})

		conditionRefs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.triggerConfig.conditionExpr)
		refs = append(refs, conditionRefs...)
	}

	return refs
}

func (n *nodeAbstractActionTrigger) Provider() ProviderRef {
	if n.resolvedProvider.Provider.Type != "" {
		return ProviderRef{
			addr:     n.resolvedProvider,
			resolved: true,
		}
	}

	return ProviderRef{
		addr: addrs.AbsProviderConfig{
			Provider: n.Config.Provider,
			Module:   n.ModulePath(),
			Alias:    n.Config.ProviderConfigAddr().Alias,
		},
	}
}

func (n *nodeAbstractActionTrigger) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}
