package terraform

import (
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

// EvalGraphBuilder implements GraphBuilder and constructs a graph suitable
// for evaluating in-memory values (input variables, local values, output
// values) in the state without any other side-effects.
//
// This graph is used only in weird cases, such as the "terraform console"
// CLI command, where we need to evaluate expressions against the state
// without taking any other actions.
//
// The generated graph will include nodes for providers, resources, etc
// just to allow indirect dependencies to be resolved, but these nodes will
// not take any actions themselves since we assume that their parts of the
// state, if any, are already complete.
//
// Although the providers are never configured, they must still be available
// in order to obtain schema information used for type checking, etc.
type EvalGraphBuilder struct {
	// Config is the configuration tree.
	Config *configs.Config

	// State is the current state
	State *State

	// Components is a factory for the plug-in components (providers and
	// provisioners) available for use.
	Components contextComponentFactory
}

// See GraphBuilder
func (b *EvalGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
		Name:     "EvalGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *EvalGraphBuilder) Steps() []GraphTransformer {
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeEvalableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the data resources that aren't in the state. This will also
		// add any orphans from scaling in as destroy nodes.
		&ConfigTransformer{
			Concrete: nil, // just use the abstract type
			Config:   b.Config,
			Unique:   true,
		},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// Add root variables
		&RootVariableTransformer{Config: b.Config},

		// Must be before TransformProviders and ReferenceTransformer, since
		// schema is required to extract references from config.
		&AttachSchemaTransformer{Components: b.Components},

		TransformProviders(b.Components.ResourceProviders(), concreteProvider, b.Config),

		// Add the local values
		&LocalTransformer{Config: b.Config},

		// Add the outputs
		&OutputTransformer{Config: b.Config},

		// Add module variables
		&ModuleVariableTransformer{Config: b.Config},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Although we don't configure providers, we do still start them up
		// to get their schemas, and so we must shut them down again here.
		&CloseProviderTransformer{},

		// Single root
		&RootTransformer{},

		// Remove redundant edges to simplify the graph.
		&TransitiveReductionTransformer{},
	}

	return steps
}
