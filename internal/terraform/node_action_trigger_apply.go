// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeActionTriggerApply struct {
	ActionInvocation *plans.ActionInvocationInstanceSrc
	resolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeExecutable = (*nodeActionTriggerApply)(nil)
	_ GraphNodeReferencer = (*nodeActionTriggerApply)(nil)
)

func (n *nodeActionTriggerApply) Name() string {
	return "action_apply_" + n.ActionInvocation.Addr.String()
}

func (n *nodeActionTriggerApply) Execute(ctx EvalContext, wo walkOperation) tfdiags.Diagnostics {
	_, diags := invokeActions(ctx, []*plans.ActionInvocationInstanceSrc{n.ActionInvocation})
	return diags
}

func (n *nodeActionTriggerApply) ProvidedBy() (addr addrs.ProviderConfig, exact bool) {
	return n.ActionInvocation.ProviderAddr, true

}

func (n *nodeActionTriggerApply) Provider() (provider addrs.Provider) {
	return n.ActionInvocation.ProviderAddr.Provider
}

func (n *nodeActionTriggerApply) SetProvider(config addrs.AbsProviderConfig) {
	n.resolvedProvider = config
}

func (n *nodeActionTriggerApply) References() []*addrs.Reference {
	var refs []*addrs.Reference

	fmt.Printf("\n\t n.ActionInvocation.Addr.Action.Action --> %#v \n", n.ActionInvocation.Addr.Action.Action)

	refs = append(refs, &addrs.Reference{
		Subject: n.ActionInvocation.Addr.Action.Action,
	})

	return refs
}

// GraphNodeModulePath
func (n *nodeActionTriggerApply) ModulePath() addrs.Module {
	return addrs.RootModule
}
