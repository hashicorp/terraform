package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// ApplyGraphBuilder implements GraphBuilder and is responsible for building
// a graph for applying a Terraform diff.
//
// Because the graph is built from the diff (vs. the config or state),
// this helps ensure that the apply-time graph doesn't modify any resources
// that aren't explicitly in the diff. There are other scenarios where the
// diff can be deviated, so this is just one layer of protection.
type ApplyGraphBuilder struct {
	// Module is the root module for the graph to build.
	Module *module.Tree

	// Diff is the diff to apply.
	Diff *Diff

	// State is the current state
	State *State

	// Providers is the list of providers supported.
	Providers []string

	// Provisioners is the list of provisioners supported.
	Provisioners []string
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Build(path []string) (*Graph, error) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
	}).Build(path)
}

// See GraphBuilder
func (b *ApplyGraphBuilder) Steps() []GraphTransformer {
	// Custom factory for creating providers.
	providerFactory := func(name string, path []string) GraphNodeProvider {
		return &NodeApplyableProvider{
			NameValue: name,
			PathValue: path,
		}
	}

	steps := []GraphTransformer{
		// Creates all the nodes represented in the diff.
		&DiffTransformer{
			Diff:   b.Diff,
			Module: b.Module,
			State:  b.State,
		},

		// Create all the providers
		&MissingProviderTransformer{Providers: b.Providers, Factory: providerFactory},
		&ProviderTransformer{},
		&ParentProviderTransformer{},

		// Provisioner-related transformations
		&MissingProvisionerTransformer{Provisioners: b.Provisioners},
		&ProvisionerTransformer{},

		// Connect references so ordering is correct
		&ReferenceTransformer{},

		// Attach the configurations
		&AttachConfigTransformer{Module: b.Module},

		// Single root
		&RootTransformer{},
	}

	return steps
}
