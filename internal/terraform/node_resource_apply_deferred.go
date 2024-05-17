// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeApplyableDeferredInstance is a node that represents a deferred instance
// in the apply graph. This node is targetable and helps maintain the correct
// ordering of the apply graph.
//
// When executed during the apply phase, this transfers the planned change we
// got from the plan's deferrals *back* into the EvalContext's Deferred struct,
// so downstream references to deferred objects can get partial values (of
// varying quality).
type nodeApplyableDeferredInstance struct {
	*NodeAbstractResourceInstance

	Reason    providers.DeferredReason
	ChangeSrc *plans.ResourceInstanceChangeSrc
}

func (n *nodeApplyableDeferredInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	change, err := n.ChangeSrc.Decode(n.Schema.ImpliedType())
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to decode ", fmt.Sprintf("Terraform failed to decode a deferred change: %v\n\nThis is a bug in Terraform; please report it!", err)))
	}

	switch n.Addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		ctx.Deferrals().ReportResourceInstanceDeferred(n.Addr, n.Reason, change)
	case addrs.DataResourceMode:
		ctx.Deferrals().ReportDataSourceInstanceDeferred(n.Addr, n.Reason, change)
	}
	return diags
}

// nodeApplyableDeferredPartialInstance is a node that represents a deferred
// partial instance in the apply graph. This simply adds a method  to get the
// partial address on top of the regular behaviour of
// nodeApplyableDeferredInstance.
type nodeApplyableDeferredPartialInstance struct {
	*nodeApplyableDeferredInstance

	PartialAddr addrs.PartialExpandedResource
}

func (n *nodeApplyableDeferredPartialInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	change, err := n.ChangeSrc.Decode(n.Schema.ImpliedType())
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to decode ", fmt.Sprintf("Terraform failed to decode a deferred change: %v\n\nThis is a bug in Terraform; please report it!", err)))
	}

	switch n.Addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		ctx.Deferrals().ReportResourceExpansionDeferred(n.PartialAddr, change)
	case addrs.DataResourceMode:
		ctx.Deferrals().ReportDataSourceExpansionDeferred(n.PartialAddr, change)
	}
	return diags
}
