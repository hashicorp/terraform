// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeResourcePolicy is a node that evaluates a resource instance's policy.
type nodeResourcePolicy struct {
	ResourceAddr addrs.AbsResourceInstance
	ProviderAddr addrs.AbsProviderConfig
	Before       cty.Value
	After        cty.Value
	Action       plans.Action
}

var _ GraphNodeExecutable = (*nodeResourcePolicy)(nil)

func (n *nodeResourcePolicy) Name() string {
	return n.ResourceAddr.String() + " (policy evaluation)"
}

func (n *nodeResourcePolicy) Execute(ctx EvalContext, operation walkOperation) tfdiags.Diagnostics {
	return nil
}

func policyNodeFromChange(change *plans.ResourceInstanceChange) *nodeResourcePolicy {
	return &nodeResourcePolicy{
		ResourceAddr: change.Addr,
		ProviderAddr: change.ProviderAddr,
		Action:       change.Action,
		Before:       change.Before,
		After:        change.After,
	}
}
