// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeResourcePolicy is a node that evaluates a resource instance's policy.
// The node is not part of the main graph, but is executed as part of the
// policy subgraph of nodePolicyEval.
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
	var diags tfdiags.Diagnostics
	client := ctx.PolicyClient()
	config := ctx.Config()

	if client == nil {
		log.Printf("[DEBUG] No policy client configured, skipping policy evaluation")
		return nil
	}
	if config == nil {
		log.Printf("[DEBUG] No configuration available, skipping policy evaluation")
		return nil
	}

	providerAddr := n.ProviderAddr
	provider, schema, err := getProvider(ctx, providerAddr)
	if err != nil {
		return diags.Append(err)
	}

	modCfg := config.DescendantForInstance(n.ResourceAddr.Module)

	policyOperation, ok := policyOperationForAction(n.Action)
	if !ok {
		log.Printf("[DEBUG] Unsupported plan action for policies %q, skipping policy evaluation", n.Action)
		return nil
	}

	meta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
		ProviderType: providerAddr.Provider.Type,
		Operation:    policyOperation,
		ModulePath:   n.ResourceAddr.Module.String(),
	}

	// the module config may be nil if the module call has been removed from the configuration
	// in this case, we are fine with a nil resource config, as that would only mean
	// that we do not have the terraform config's source information in the diagnostics.
	var resourceConfig *configs.Resource
	if modCfg != nil {
		resourceConfig = modCfg.Module.ResourceByAddr(n.ResourceAddr.Resource.Resource)
	}

	callbacks := callback.Functions{
		GetResources:  getResourcesForPolicyCallback(ctx, operation, provider, schema, config),
		GetDataSource: getDataSourceForPolicyCallback(ctx, provider, schema),
	}

	result := evaluatePolicies(ctx, n.ResourceAddr, resourceConfig, n.After, n.Before, meta, callbacks)
	ctx.PolicyResults().AddResource(n.ResourceAddr, result, resourceConfig)
	return diags
}

func policyOperationForAction(action plans.Action) (proto.Operation, bool) {
	switch action {
	case plans.Create:
		return proto.Operation_CREATE, true
	case plans.Delete:
		return proto.Operation_DELETE, true
	case plans.NoOp:
		return proto.Operation_NO_OP, true
	case plans.Update,
		plans.DeleteThenCreate,
		plans.CreateThenDelete,
		plans.CreateThenForget:
		return proto.Operation_UPDATE, true
	default:
		return 0, false
	}
}

// policyNodeFromChange creates a nodeResourcePolicy from a ResourceInstanceChange.
func policyNodeFromChange(change *plans.ResourceInstanceChange) *nodeResourcePolicy {
	return &nodeResourcePolicy{
		ResourceAddr: change.Addr,
		ProviderAddr: change.ProviderAddr,
		Action:       change.Action,
		Before:       change.Before,
		After:        change.After,
	}
}
