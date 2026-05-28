// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluateActionExpression expands the hcl.Expression from a resource's list
// action_trigger.actions. Note that if the action uses count or for_each, it's
// using the resource instances count/for_each.
func evaluateActionExpression(expr hcl.Expression, repData instances.RepetitionData) (addrs.ActionInstance, tfdiags.Diagnostics) {
	ref, diags := evalSemiStaticExpr(expr, repData)
	if diags.HasErrors() {
		return addrs.ActionInstance{}, diags
	}

	var actionInst addrs.ActionInstance
	switch sub := ref.Subject.(type) {
	case addrs.Action:
		actionInst = sub.Instance(addrs.NoKey)
	case addrs.ActionInstance:
		actionInst = sub
	default:
		panic(fmt.Sprintf("unknown action address type: %T", sub))
	}

	return actionInst, diags
}
