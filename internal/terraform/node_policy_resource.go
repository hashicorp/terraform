// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
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
	if modCfg == nil {
		return nil
	}

	attrs, _ := n.After.UnmarkDeep()
	priorAttrs, _ := n.Before.UnmarkDeep()

	var policyOperation proto.Operation
	switch action := n.Action; action {
	case plans.Create:
		policyOperation = proto.Operation_CREATE
	case plans.Delete:
		policyOperation = proto.Operation_DELETE
	case plans.Update,
		plans.DeleteThenCreate,
		plans.CreateThenDelete,
		plans.CreateThenForget:
		policyOperation = proto.Operation_UPDATE
	default:
		return nil
	}

	meta := &proto.ResourceMetadata{
		Type:         n.ResourceAddr.Resource.Resource.Type,
		ProviderType: providerAddr.Provider.Type,
		Operation:    policyOperation,
	}

	providerRef := ProviderRef{
		addr:     providerAddr,
		resolved: true,
	}

	metaVal, metaDiags := providerRef.getProviderMeta(ctx, n.ResourceAddr.Resource, modCfg.Module.ProviderMetas)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return diags
	}

	callbacks := callback.Functions{
		GetResources:  getResourcesForPolicyCallback(ctx, config),
		GetDataSource: getDataSourceForPolicyCallback(ctx, provider, schema, metaVal),
	}

	rscConfig := modCfg.Module.ResourceByAddr(n.ResourceAddr.Resource.Resource)
	result := evaluatePolicies(ctx, operation, n.ResourceAddr, rscConfig, client, attrs, priorAttrs, meta, callbacks)
	ctx.PolicyResults().AddResource(n.ResourceAddr, result, rscConfig)
	return diags
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
