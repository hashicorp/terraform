// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// Identity is the identity cty object from the list response element. It is
	// converted to a map[string]string and attached to every EvaluationResponse so
	// downstream consumers (UI, cloud backend) can correlate results to rows.
	Identity cty.Value
	// ListBlockAddr is the AbsResourceInstance address of the originating list
	// block. It is attached to every EvaluationResponse to group results.
	ListBlockAddr addrs.AbsResourceInstance
}

var _ GraphNodeExecutable = (*nodeQueryResourcePolicy)(nil)

func (n *nodeQueryResourcePolicy) Name() string {
	return n.ResourceAddr.String() + " (query policy evaluation)"
}

// Execute evaluates policy for a single discovered resource from a query list block.
// It calls evaluatePolicies via the PolicyClient, passing the synthetic address and
// generated config. The method annotates every resulting EvaluationResponse with both
// a structured identity map (map[string]string) and the list block address string so
// downstream UI and cloud-backend consumers can correlate results to rows.
//
// Unlike nodeResourcePolicy.Execute, the result is always emitted through the hook
// regardless of whether the policy passed. Query policy consumers need a hook event
// for every evaluated resource — including passing ones — so that downstream
// aggregators can include passing resources in summary records. The !result.Empty()
// gate used by nodeResourcePolicy is correct for plan/apply (where passing resources
// carry no actionable information), but wrong here because query consumers need a
// complete row for every discovered resource.
func (n *nodeQueryResourcePolicy) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	client := ctx.PolicyClient()
	config := ctx.Config()

	if client == nil {
		log.Printf("[DEBUG] No policy client configured, skipping query policy evaluation")
		return nil
	}
	if config == nil {
		log.Printf("[DEBUG] No configuration available, skipping query policy evaluation")
		return nil
	}

	// Acquire the policy semaphore to limit concurrent policy evaluations.
	// The nil-config guard above must remain before this acquire so that a
	// missing config does not consume a semaphore slot for no work.
	policySem := ctx.PolicySemaphore()
	if policySem != nil {
		policySem.Acquire()
		defer policySem.Release()
	}

	providerAddr := n.ProviderAddr
	provider, schema, err := getProvider(ctx, providerAddr)
	if err != nil {
		return diags.Append(err)
	}

	modCfg := config.DescendantForInstance(n.ResourceAddr.Module)

	// Query resources are always evaluated as CREATE operations since they
	// represent discovered resources that don't exist in the configuration.
	meta := &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
		ProviderType: providerAddr.Provider.Type,
		Operation:    proto.Operation_CREATE,
		ModulePath:   n.ResourceAddr.Module.String(),
	}

	// The resource config may be nil if the list block has been removed from
	// the configuration. In that case we proceed without source information
	// in diagnostics.
	var resourceConfig *configs.Resource
	if modCfg != nil {
		resourceConfig = modCfg.Module.ResourceByAddr(n.ResourceAddr.Resource.Resource)
	}

	callbacks := callback.Functions{
		GetResources:  getResourcesForPolicyCallback(ctx, op, provider, schema, config),
		GetDataSource: getDataSourceForPolicyCallback(ctx, provider, schema),
	}

	// Evaluate policies with the generated config as the "after" state
	// and null as the "before" state (since these are discovered resources).
	// evaluatePolicies already applies WithLocalRange internally; do not reapply.
	result := evaluatePolicies(ctx, n.ResourceAddr, resourceConfig, n.GeneratedConfig, cty.NullVal(n.GeneratedConfig.Type()), meta, callbacks)

	// Annotate the result with query correlation metadata so that downstream
	// consumers (UI, cloud backend) can identify which list-block row this
	// result belongs to.
	result = result.WithQueryMetadata(ctyIdentityToStringMap(n.Identity), n.ListBlockAddr.String())

	// Always emit for query policy nodes. WithQueryMetadata always sets
	// ListBlockAddr to a non-empty string, so every result has identity.
	// Query nodes always emit so downstream aggregators can include passing
	// resources in summary records; the !result.Empty() gate must not be
	// applied here.
	hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PolicyResult(n.ResourceAddr.String(), result)
	})
	diags = diags.Append(hookErr)

	return diags
}

// ctyIdentityToStringMap converts a cty object value that represents a resource
// identity into a map[string]string suitable for diagnostic annotation. Attributes
// whose values are not known or not strings are omitted. A null or invalid value
// returns nil.
//
// NOTE: this function can return nil even for a valid, non-null object — e.g. when all
// attributes are unknown or non-string (numeric IDs, unknown values during plan).
// Callers that consume this map (e.g. downstream aggregators correlating rows) must
// treat a nil return as "identity not available" rather than as an error.
func ctyIdentityToStringMap(val cty.Value) map[string]string {
	if val == cty.NilVal || val.IsNull() || !val.IsKnown() || !val.Type().IsObjectType() {
		return nil
	}
	attrs := val.AsValueMap()
	if len(attrs) == 0 {
		return nil
	}
	result := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if !v.IsKnown() || v.IsNull() || v.Type() != cty.String {
			continue
		}
		result[k] = v.AsString()
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
