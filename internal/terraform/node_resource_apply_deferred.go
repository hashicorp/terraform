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
// Note, that it does not implement Execute, as deferred instances are not
// executed during the apply phase.
type nodeApplyableDeferredInstance struct {
	*NodeAbstractResourceInstance

	Reason    providers.DeferredReason
	ChangeSrc *plans.ResourceInstanceChangeSrc
}

func (n *nodeApplyableDeferredInstance) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	change, err := n.ChangeSrc.Decode(n.Schema.ImpliedType())
	if err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to decode ", fmt.Sprintf("Terraform failed to decode a deferred change: %v\n\nThis is a bug in Terraform; please report it!", err)))
	}

	ctx.Deferrals().ReportResourceInstanceDeferred(n.Addr, n.Reason, change)
	return nil
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
	change, err := n.ChangeSrc.Decode(n.Schema.ImpliedType())
	if err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to decode ", fmt.Sprintf("Terraform failed to decode a deferred change: %v\n\nThis is a bug in Terraform; please report it!", err)))
	}

	ctx.Deferrals().ReportResourceExpansionDeferred(n.PartialAddr, change)
	return nil
}
