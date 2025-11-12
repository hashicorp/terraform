// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/plans"
)

type nodeActionTriggerApplyInstance struct {
	ActionInvocation   *plans.ActionInvocationInstanceSrc
	resolvedProvider   addrs.AbsProviderConfig
	ActionTriggerRange *hcl.Range
	ConditionExpr      hcl.Expression
}

var (
	_ GraphNodeReferencer       = (*nodeActionTriggerApplyInstance)(nil)
	_ GraphNodeProviderConsumer = (*nodeActionTriggerApplyInstance)(nil)
	_ GraphNodeModulePath       = (*nodeActionTriggerApplyInstance)(nil)
)

func (n *nodeActionTriggerApplyInstance) Name() string {
	return n.ActionInvocation.Addr.String() + " (instance)"
}

func (n *nodeActionTriggerApplyInstance) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	return n.ActionInvocation.ProviderAddr, true
}

func (n *nodeActionTriggerApplyInstance) Provider() (provider addrs.Provider) {
	return n.ActionInvocation.ProviderAddr.Provider
}

func (n *nodeActionTriggerApplyInstance) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

func (n *nodeActionTriggerApplyInstance) References() []*addrs.Reference {
	var refs []*addrs.Reference

	refs = append(refs, &addrs.Reference{
		Subject: n.ActionInvocation.Addr.Action,
	})

	conditionRefs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRef, n.ConditionExpr)
	if refDiags.HasErrors() {
		panic(fmt.Sprintf("error parsing references in expression: %v", refDiags))
	}
	if conditionRefs != nil {
		refs = append(refs, conditionRefs...)
	}

	return refs
}

// GraphNodeModulePath
func (n *nodeActionTriggerApplyInstance) ModulePath() addrs.Module {
	return n.ActionInvocation.Addr.Module.Module()
}

// GraphNodeModuleInstance
func (n *nodeActionTriggerApplyInstance) Path() addrs.ModuleInstance {
	return n.ActionInvocation.Addr.Module
}
