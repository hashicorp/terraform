// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import "github.com/hashicorp/terraform/internal/tfdiags"

// nodePolicyEval is a node that evaluates policy for all resource instances
// after they have been planned or applied,
// so that the complete resource graph state is available to the policy engine.
type nodePolicyEval struct{}

var _ GraphNodeDynamicExpandable = (*nodePolicyEval)(nil)

func (n *nodePolicyEval) Name() string {
	return "(evaluate policies)"
}

func (n *nodePolicyEval) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	return nil, nil
}
