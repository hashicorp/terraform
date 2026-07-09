// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/zclconf/go-cty/cty"
)

// nodeQueryResourcePolicy is a node in the policy subgraph that evaluates
// policy for a single resource discovered during a list block walk. Like
// nodeResourcePolicy, it runs as part of nodePolicyEval's DynamicExpand.
type nodeQueryResourcePolicy struct {
	// ResourceAddr is the address of this discovered resource instance.
	ResourceAddr addrs.AbsResourceInstance
	// ProviderAddr is the resolved provider for the originating list block.
	ProviderAddr addrs.AbsProviderConfig
	// GeneratedConfig is the provider-generated cty object for this resource.
	GeneratedConfig cty.Value
	// ResourceConfig is the list block config, used for diagnostic source locations.
	ResourceConfig *configs.Resource

	// Identity and ListBlockAddr are passed to the policy client in a following revision.
	// Correlates policy results to UI rows.
	Identity cty.Value
	// Groups results by the originating list block.
	ListBlockAddr addrs.AbsResourceInstance
}

func (n *nodeQueryResourcePolicy) Name() string {
	return n.ResourceAddr.String() + " (query policy evaluation)"
}

// TODO(CORE-6): implement Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics
