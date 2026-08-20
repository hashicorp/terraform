// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	policyGraph := ctx.PolicyGraph()
	if policyGraph == nil {
		log.Printf("[DEBUG] policyGraph is nil")
		return nil, nil
	}
	// Close the changes/state objects to prevent writes during policy evaluation.
	// This is safe to do because policy evaluation is the final step in the plan/apply process.
	// If any future nodes attempt to write to these states, they will panic.
	ctx.Changes().Close()
	ctx.State().Close()

	_, span := tracer().Start(ctx.StopCtx(), "terraform.policy.evaluate")
	g := policyGraph.evalGraph(span)
	return g, nil
}

// AlwaysRun implements [dag.AlwaysRunVertex] so that the policy evaluation
// can proceed even if some resource instance nodes evaluated with error diagnostics.
func (n *nodePolicyEval) AlwaysRun() {}

func (n *nodePolicyEval) Execute(ctx EvalContext, walkOp walkOperation) tfdiags.Diagnostics {
	policyGraph := ctx.PolicyGraph()
	if policyGraph == nil {
		log.Printf("[DEBUG] policyGraph is nil")
		return nil
	}

	// Close the changes/state objects to prevent writes during policy evaluation.
	// This is safe to do because policy evaluation is the final step in the plan/apply process.
	// If any future nodes attempt to write to these states, they will panic.
	ctx.Changes().Close()
	ctx.State().Close()

	var diags tfdiags.Diagnostics
	state := ctx.State()
	config := ctx.Config()
	changes := ctx.Changes()

	// Now we read the state and plan changes to build the policy resource map.
	// These are the resources that will be available during the callback evaluation.
	if walkOp == walkApply {
		for _, resourceAddr := range state.Lock().AllManagedResourceInstanceObjectAddrs() {
			addr := resourceAddr.ResourceInstance
			change := changes.GetResourceInstanceChange(addr, addrs.NotDeposed)
			_, schema, err := getProvider(ctx, change.ProviderAddr)
			if err != nil {
				diags = diags.Append(err)
				continue
			}
			resourceSchema := schema.SchemaForResourceAddr(addr.Resource.Resource)

			resource := state.ResourceInstance(addr)
			if resource.Current == nil {
				// TODO: Should we return an error here instead?
				continue
			}
			decoded, err := resource.Current.Decode(resourceSchema)
			if err != nil {
				diags = diags.Append(err)
				continue
			}

			policyGraph.resourceMap.Put(addr, &PolicyResource{
				Addr:   resourceAddr.ResourceInstance,
				Body:   getResourceConfig(config, addr),
				Schema: resourceSchema.Body,
				Value:  decoded.Value,
			})
		}
		state.Unlock()

	} else {
		for change := range plans.AllInstances(changes) {

			_, schema, err := getProvider(ctx, change.ProviderAddr)
			if err != nil {
				diags = diags.Append(err)
				continue
			}
			resourceSchema := schema.SchemaForResourceAddr(change.Addr.Resource.Resource)

			policyGraph.resourceMap.Put(change.Addr, &PolicyResource{
				Addr:   change.Addr,
				Body:   getResourceConfig(config, change.Addr),
				Schema: resourceSchema.Body,
				Value:  change.After,
			})
		}
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

func getResourceConfig(config *configs.Config, addr addrs.AbsResourceInstance) hcl.Body {
	var rscConfig hcl.Body
	for config := range config.AllModules() {
		if rsc := config.Module.ResourceByAddr(addr.Resource.Resource); rsc != nil {
			if rsc.Config != nil {
				rscConfig = rsc.Config
			}
			break
		}
	}
	return rscConfig
}
