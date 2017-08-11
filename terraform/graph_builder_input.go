package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// InputGraphBuilder creates the graph for the input operation.
//
// Unlike other graph builders, this is a function since it currently modifies
// and is based on the PlanGraphBuilder. The PlanGraphBuilder passed in will be
// modified and should not be used for any other operations.
func InputGraphBuilder(p *PlanGraphBuilder) GraphBuilder {
	// convert this to an InputPlan
	p.Input = true

	// We're going to customize the concrete functions
	p.CustomConcrete = true

	// Set the provider to the normal provider. This will ask for input.
	p.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	// We purposely don't set any more concrete fields since the remainder
	// should be no-ops.

	return p
}
