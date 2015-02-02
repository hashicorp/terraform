package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// GraphBuilder is an interface that can be implemented and used with
// Terraform to build the graph that Terraform walks.
type GraphBuilder interface {
	// Build builds the graph for the given module path. It is up to
	// the interface implementation whether this build should expand
	// the graph or not.
	Build(path []string) (*Graph, error)
}

// BuiltinGraphBuilder is responsible for building the complete graph that
// Terraform uses for execution. It is an opinionated builder that defines
// the step order required to build a complete graph as is used and expected
// by Terraform.
//
// If you require a custom graph, you'll have to build it up manually
// on your own by building a new GraphBuilder implementation.
type BuiltinGraphBuilder struct {
	// Root is the root module of the graph to build.
	Root *module.Tree

	// State is the global state. The proper module states will be looked
	// up by graph path.
	State *State

	// Providers is the list of providers supported.
	Providers []string
}

// Build builds the graph according to the steps returned by Steps.
func (b *BuiltinGraphBuilder) Build(path []string) (*Graph, error) {
	g := &Graph{Path: path}
	for _, step := range b.Steps() {
		if err := step.Transform(g); err != nil {
			return g, err
		}
	}

	return g, nil
}

// Steps returns the ordered list of GraphTransformers that must be executed
// to build a complete graph.
func (b *BuiltinGraphBuilder) Steps() []GraphTransformer {
	return []GraphTransformer{
		&ConfigTransformer{Module: b.Root},
		&OrphanTransformer{State: b.State, Module: b.Root},
		&TaintedTransformer{State: b.State},
		&MissingProviderTransformer{Providers: b.Providers},
		&ProviderTransformer{},
		&PruneProviderTransformer{},
	}
}
