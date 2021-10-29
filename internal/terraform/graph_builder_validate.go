package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
)

// validateGraphBuilder creates the graph for the validate operation.
//
// validateGraphBuilder is based on the PlanGraphBuilder. We do this so that
// we only have to validate what we'd normally plan anyways. The
// PlanGraphBuilder given will be modified so it shouldn't be used for anything
// else after calling this function.
func validateGraphBuilder(p *PlanGraphBuilder, opts *ValidateOpts) GraphBuilder {
	if opts == nil {
		opts = DefaultValidateOpts
	}

	// We're going to customize the concrete functions
	p.CustomConcrete = true

	// Set the provider to the normal provider. This will ask for input.
	p.ConcreteProvider = func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeApplyableProvider{
			NodeAbstractProvider: a,
		}
	}

	p.ConcreteResource = func(a *NodeAbstractResource) dag.Vertex {
		return &NodeValidatableResource{
			NodeAbstractResource: a,
			Hints:                opts.Hints,
		}
	}

	p.ConcreteModule = func(n *nodeExpandModule) dag.Vertex {
		return &nodeValidateModule{
			nodeExpandModule: *n,
		}
	}

	// We purposely don't set any other concrete types since they don't
	// require validation.

	return p
}
