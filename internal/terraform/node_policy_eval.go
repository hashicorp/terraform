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

	resourceConfigMap := n.resourceConfigMap(config)
	getResourceConfig := func(config *configs.Config, addr addrs.ConfigResource) hcl.Body {
		mod, ok := resourceConfigMap.GetOk(addr.Module)
		if !ok {
			return nil
		}
		resource, ok := mod.GetOk(addr)
		if !ok {
			return nil
		}
		return resource
	}

	// Now we read the state and plan changes to build the policy resource map.
	// These are the resources that will be available during the callback evaluation.
	if walkOp == walkApply {
		resourceAddrs := state.Lock().AllManagedResourceInstanceObjectAddrs()
		state.Unlock()
		for _, resourceAddr := range resourceAddrs {
			addr := resourceAddr.ResourceInstance
			resource := state.Resource(addr.AffectedAbsResource())
			if resource == nil {
				// TODO: Should we return an error here instead?
				continue
			}
			resourceInstance := resource.Instance(addr.Resource.Key)
			if resourceInstance == nil {
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

			policyGraph.resourceMap.Put(addr, &PolicyResource{
				Addr:       resourceAddr.ResourceInstance,
				ConfigBody: getResourceConfig(config, addr.ConfigResource()),
				Schema:     resourceSchema.Body,
				Value:      decoded.Value,
			})
		}

	} else {
		for change := range plans.AllInstances(changes) {

			_, schema, err := getProvider(ctx, change.ProviderAddr)
			if err != nil {
				diags = diags.Append(err)
				continue
			}
			resourceSchema := schema.SchemaForResourceAddr(change.Addr.Resource.Resource)

			policyGraph.resourceMap.Put(change.Addr, &PolicyResource{
				Addr:       change.Addr,
				ConfigBody: getResourceConfig(config, change.Addr.ConfigResource()),
				Schema:     resourceSchema.Body,
				Value:      change.After,
			})
		}
	}

	return diags
}

// resourceConfigMap returns a map of resource configurations for each module,
// so that resource configs can be looked up at constant time.
func (n *nodePolicyEval) resourceConfigMap(config *configs.Config) addrs.Map[addrs.Module, addrs.Map[addrs.ConfigResource, hcl.Body]] {
	ret := addrs.MakeMap[addrs.Module, addrs.Map[addrs.ConfigResource, hcl.Body]]()
	for addr, config := range config.AllResources() {
		if moduleMap, ok := ret.GetOk(addr.Module); ok {
			moduleMap.Put(addr, config.Config)
		} else {
			moduleMap := addrs.MakeMap[addrs.ConfigResource, hcl.Body]()
			moduleMap.Put(addr, config.Config)
			ret.Put(addr.Module, moduleMap)
		}
	}
	return ret
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
