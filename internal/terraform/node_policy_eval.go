// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/trace"
)

// nodePolicyEval is a node that completes the building of the policy graph,
// with incoming edges from the resource graph so that policy evaluation
// is performed only when the resource graph is complete.
type nodePolicyEval struct {
}

var _ GraphNodeDynamicExpandable = (*nodePolicyEval)(nil)
var _ dag.AlwaysRunVertex = (*nodePolicyEval)(nil)

func (n *nodePolicyEval) Name() string {
	return "(evaluate policies)"
}

func (n *nodePolicyEval) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	policyState := ctx.PolicyState()
	if policyState == nil {
		log.Printf("[DEBUG] policyState is nil")
		return nil, nil
	}
	// Close the changes/state objects to prevent writes during policy evaluation.
	// This is safe to do because policy evaluation is the final step in the plan/apply process.
	// If any future nodes attempt to write to these states, they will panic.
	ctx.Changes().Close()
	ctx.State().Close()

	_, span := tracer().Start(ctx.StopCtx(), "terraform.policy.evaluate")
	g := policyState.evalGraph(span)
	return g, nil
}

// AlwaysRun implements [dag.AlwaysRunVertex] so that the policy evaluation
// can proceed even if some resource instance nodes evaluated with error diagnostics.
func (n *nodePolicyEval) AlwaysRun() {}

func (n *nodePolicyEval) Execute(ctx EvalContext, walkOp walkOperation) tfdiags.Diagnostics {
	policyState := ctx.PolicyState()
	if policyState == nil {
		log.Printf("[DEBUG] policyState is nil")
		return nil
	}

	// Close the changes/state objects to prevent writes during policy evaluation.
	// This is safe to do because policy evaluation is the final step in the plan/apply process.
	// If any future nodes attempt to write to these states, they will panic.
	ctx.Changes().Close()
	ctx.State().Close()

	// prepare all resources from the state + changes
	diags := collectPolicyResources(ctx, walkOp, policyState)
	return diags
}

// collectPolicyResources prepares the policy resources
// before any policy evaluations or identifier callbacks can run.
func collectPolicyResources(ctx EvalContext, walkOp walkOperation, policyState *policyWalkState) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	state := ctx.State()
	changes := ctx.Changes()

	// Now we read the state and plan changes to build the policy resource map.
	// These are the resources that will be available during the callback evaluation.
	if walkOp == walkApply {
		resourceAddrs := state.Lock().AllManagedResourceInstanceObjectAddrs()
		state.Unlock()
		for _, resourceAddr := range resourceAddrs {
			addr := resourceAddr.ResourceInstance
			if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			resource := state.Resource(addr.AffectedAbsResource())
			if resource == nil {
				// TODO: Should we return an error here instead?
				continue
			}
			resourceInstance := resource.Instance(addr.Resource.Key)
			if resourceInstance == nil || resourceInstance.Current == nil {
				// TODO: Should we return an error here instead?
				continue
			}

			_, schema, err := getProvider(ctx, resource.ProviderConfig)
			if err != nil {
				diags = diags.Append(err)
				continue
			}
			resourceSchema := schema.SchemaForResourceAddr(addr.Resource.Resource)

			decoded, err := resourceInstance.Current.Decode(resourceSchema)
			if err != nil {
				diags = diags.Append(err)
				continue
			}

			value := decoded.Value
			if value == cty.NilVal {
				value = cty.UnknownVal(resourceSchema.Body.ImpliedType())
			}
			prior, action := decoded.Value, plans.NoOp
			if change, ok := policyState.applyChanges.GetOk(addr); ok {
				prior, action = change.Before, change.Action
			}
			policyState.addResource(&policyResource{
				Addr:         addr,
				ProviderAddr: resource.ProviderConfig,
				Value:        policy.CtyToPolicyValue(value),
				PriorValue:   policy.CtyToPolicyValue(prior),
				Action:       action,
			})
		}

	} else {
		for change := range plans.AllInstances(changes) {
			if change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}

			// exclude deletes
			if change.Action == plans.Delete {
				continue
			}

			_, schema, err := getProvider(ctx, change.ProviderAddr)
			if err != nil {
				diags = diags.Append(err)
				continue
			}
			resourceSchema := schema.SchemaForResourceAddr(change.Addr.Resource.Resource)
			value := change.After
			if value == cty.NilVal {
				value = cty.UnknownVal(resourceSchema.Body.ImpliedType())
			}
			policyResource := &policyResource{
				Addr:         change.Addr,
				ProviderAddr: change.ProviderAddr,
				Value:        policy.CtyToPolicyValue(value),
				PriorValue:   policy.CtyToPolicyValue(change.Before),
				Action:       change.Action,
			}
			policyState.addResource(policyResource)
		}
	}

	// mark deferred changes as unknown types
	for _, deferred := range ctx.Deferrals().GetDeferredChanges() {
		policyState.markUnknownResourceType(deferred.Change.Addr.Resource.Resource.Type)

		// TODO: Account for targeting, refresh-only plans, and resources
		// that failed to plan or were skipped after an upstream error.
	}

	return diags
}

// nodePolicyEvalFinish is a sentinel node appended to the policy subgraph that
// runs after every policy node and ends the policy-execution phase span. It
// must tolerate upstream failures so the span is still closed even if a policy
// node returned error diagnostics.
type nodePolicyEvalFinish struct {
	span trace.Span
}

var _ GraphNodeExecutable = (*nodePolicyEvalFinish)(nil)
var _ dag.AlwaysRunVertex = (*nodePolicyEvalFinish)(nil)

func (n *nodePolicyEvalFinish) Name() string {
	return "(policy evaluation complete)"
}

func (n *nodePolicyEvalFinish) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	n.span.End()
	return nil
}

// AlwaysRun implements [dag.AlwaysRunVertex] so that this node still executes
// even if dependencies errored
func (n *nodePolicyEvalFinish) AlwaysRun() {}
