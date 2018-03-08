package terraform

import (
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// DestroyPlanGraphBuilder implements GraphBuilder and is responsible for
// planning a pure-destroy.
//
// Planning a pure destroy operation is simple because we can ignore most
// ordering configuration and simply reverse the state.
type DestroyPlanGraphBuilder struct {
	// Config is the configuration for the graph to build.
	Config *configs.Config

	// State is the current state
	State *State

	// Targets are resources to target
	Targets []string

	// Validate will do structural validation of the graph.
	Validate bool
}

// See GraphBuilder
func (b *DestroyPlanGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: b.Validate,
		Name:     "DestroyPlanGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *DestroyPlanGraphBuilder) Steps() []GraphTransformer {
	concreteResource := func(a *NodeAbstractResource) dag.Vertex {
		return &NodePlanDestroyableResource{
			NodeAbstractResource: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the nodes represented in the state.
		&StateTransformer{
			Concrete: concreteResource,
			State:    b.State,
		},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Module: b.Module},

		// Destruction ordering. We require this only so that
		// targeting below will prune the correct things.
		&DestroyEdgeTransformer{Module: b.Module, State: b.State},

		// Target. Note we don't set "Destroy: true" here since we already
		// created proper destroy ordering.
		&TargetsTransformer{Targets: b.Targets},

		// Single root
		&RootTransformer{},
	}

	return steps
}
