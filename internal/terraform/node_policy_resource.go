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
	"github.com/hashicorp/terraform/internal/states"
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

	// DeposedKey is the deposed key of the change the node evaluates, which
	// apply allocates for the delete half of a create-before-destroy
	// replace; NotDeposed for current objects.
	DeposedKey states.DeposedKey

	// OldObject is set for the delete half of a planned replace, which
	// deletes the instance's old object.
	OldObject bool

	// Refreshed is whether Before and After are the object's refreshed state,
	// rather than a change's, as in a refresh-only plan.
	Refreshed bool
}

var _ GraphNodeExecutable = (*nodeResourcePolicy)(nil)

func (n *nodeResourcePolicy) Name() string {
	return n.ResourceAddr.String() + " (policy evaluation)"
}

// Execute evaluates the resource instance's policies and reports the result
// through the PolicyResult hook.
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

	pg := ctx.PolicyGraph()
	if pg == nil {
		pg = newPolicySubgraph()
	}
	spec := n.objectSpec(pg, policyOperation)
	attrs, prior := n.After, n.Before
	switch {
	case policyOperation == proto.Operation_CREATE || n.Refreshed:
		prior = cty.NilVal
	case policyOperation == proto.Operation_DELETE:
		attrs = cty.NilVal
	}
	body := schema.SchemaForResourceAddr(n.ResourceAddr.Resource.Resource).Body
	obj := pg.callbackObject(spec, attrs, prior, body)

	var relations []*proto.Relation
	if r := pg.walkRelations(); r != nil {
		val := n.After
		if policyOperation == proto.Operation_DELETE {
			val = n.Before
		}
		val, _ = val.UnmarkDeep()
		relations = r.subject(&policyObject{
			id:       obj.Identifier(),
			addr:     n.ResourceAddr.String(),
			typ:      policyTypeKey{source: providerAddr.Provider.String(), typ: n.ResourceAddr.Resource.Resource.Type},
			op:       policyOperation,
			provider: providerAddr.String(),
			value:    val,
		})
	}

	result := evaluatePolicies(ctx, n.ResourceAddr, resourceConfig, n.After, n.Before, obj.Metadata, obj, callbacks, relations)
	if !result.Empty() {
		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PolicyResult(n.ResourceAddr.String(), result)
		})
		diags = diags.Append(hookErr)
	}
	return diags
}

// objectSpec says which object of the resource instance the node evaluates
// (contract §3.4). In an apply, the deposed object of a create-before-destroy
// replace, whose key the apply allocates, is the replace's old object, and a
// deposed object with a planned change keeps its deposed key.
func (n *nodeResourcePolicy) objectSpec(pg *policySubgraph, op proto.Operation) policyObjectSpec {
	spec := policyObjectSpec{addr: n.ResourceAddr, provider: n.ProviderAddr, op: op, old: n.OldObject}
	if op != proto.Operation_DELETE || pg.run == nil || !pg.run.Apply {
		return spec
	}
	if n.DeposedKey != states.NotDeposed && pg.appliedChange(n.ResourceAddr, n.DeposedKey) != nil {
		spec.deposed = n.DeposedKey
		return spec
	}
	if change := pg.appliedChange(n.ResourceAddr, states.NotDeposed); change != nil {
		spec.old = change.Action == plans.DeleteThenCreate || change.Action == plans.CreateThenDelete
	}
	return spec
}

func policyOperationForAction(action plans.Action) (proto.Operation, bool) {
	switch action {
	case plans.Create:
		return proto.Operation_CREATE, true
	case plans.Delete:
		return proto.Operation_DELETE, true
	case plans.NoOp:
		return proto.Operation_NO_OP, true
	case plans.Update:
		return proto.Operation_UPDATE, true
	default:
		return 0, false
	}
}

// policyNodesFromChange creates a nodeResourcePolicy from a ResourceInstanceChange.
// For replace operations, two nodes are created: one for the create operation
// and one for the delete operation.
func policyNodesFromChange(change *plans.ResourceInstanceChange) []*nodeResourcePolicy {
	switch change.Action {
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		return []*nodeResourcePolicy{
			{
				ResourceAddr: change.Addr,
				ProviderAddr: change.ProviderAddr,
				Action:       plans.Create,
				Before:       cty.NilVal,
				After:        change.After,
			},
			{
				ResourceAddr: change.Addr,
				ProviderAddr: change.ProviderAddr,
				Action:       plans.Delete,
				Before:       change.Before,
				After:        cty.NilVal,
				OldObject:    true,
			},
		}
	case plans.ForgetThenCreate, plans.CreateThenForget:
		return []*nodeResourcePolicy{
			{
				ResourceAddr: change.Addr,
				ProviderAddr: change.ProviderAddr,
				Action:       plans.Create,
				Before:       cty.NilVal,
				After:        change.After,
			},
		}
	default:
	}
	return []*nodeResourcePolicy{
		{
			ResourceAddr: change.Addr,
			ProviderAddr: change.ProviderAddr,
			Action:       change.Action,
			Before:       change.Before,
			After:        change.After,
		},
	}
}
