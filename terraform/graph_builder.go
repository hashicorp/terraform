package terraform

import (
	"log"

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

// BasicGraphBuilder is a GraphBuilder that builds a graph out of a
// series of transforms and (optionally) validates the graph is a valid
// structure.
type BasicGraphBuilder struct {
	Steps    []GraphTransformer
	Validate bool
}

func (b *BasicGraphBuilder) Build(path []string) (*Graph, error) {
	g := &Graph{Path: path}
	for _, step := range b.Steps {
		if err := step.Transform(g); err != nil {
			return g, err
		}

		log.Printf(
			"[TRACE] Graph after step %T:\n\n%s",
			step, g.StringWithNodeTypes())
	}

	// Validate the graph structure
	if b.Validate {
		if err := g.Validate(); err != nil {
			log.Printf("[ERROR] Graph validation failed. Graph:\n\n%s", g.String())
			return nil, err
		}
	}

	return g, nil
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

	// Diff is the diff. The proper module diffs will be looked up.
	Diff *Diff

	// State is the global state. The proper module states will be looked
	// up by graph path.
	State *State

	// Providers is the list of providers supported.
	Providers []string

	// Provisioners is the list of provisioners supported.
	Provisioners []string

	// Targets is the user-specified list of resources to target.
	Targets []string

	// Destroy is set to true when we're in a `terraform destroy` or a
	// `terraform plan -destroy`
	Destroy bool

	// Determines whether the GraphBuilder should perform graph validation before
	// returning the Graph. Generally you want this to be done, except when you'd
	// like to inspect a problematic graph.
	Validate bool

	// Verbose is set to true when the graph should be built "worst case",
	// skipping any prune steps. This is used for early cycle detection during
	// Validate and for manual inspection via `terraform graph -verbose`.
	Verbose bool
}

// Build builds the graph according to the steps returned by Steps.
func (b *BuiltinGraphBuilder) Build(path []string) (*Graph, error) {
	basic := &BasicGraphBuilder{
		Steps:    b.Steps(path),
		Validate: b.Validate,
	}

	return basic.Build(path)
}

// Steps returns the ordered list of GraphTransformers that must be executed
// to build a complete graph.
func (b *BuiltinGraphBuilder) Steps(path []string) []GraphTransformer {
	steps := []GraphTransformer{
		// Create all our resources from the configuration and state
		&ConfigTransformer{Module: b.Root},
		&OrphanTransformer{
			State:  b.State,
			Module: b.Root,
		},

		// Output-related transformations
		&AddOutputOrphanTransformer{State: b.State},

		// Provider-related transformations
		&MissingProviderTransformer{Providers: b.Providers},
		&ProviderTransformer{},
		&DisableProviderTransformer{},

		// Provisioner-related transformations
		&MissingProvisionerTransformer{Provisioners: b.Provisioners},
		&ProvisionerTransformer{},

		// Run our vertex-level transforms
		&VertexTransformer{
			Transforms: []GraphVertexTransformer{
				// Expand any statically expanded nodes, such as module graphs
				&ExpandTransform{
					Builder: b,
				},
			},
		},

		// Flatten stuff
		&FlattenTransformer{},

		// Make sure all the connections that are proxies are connected through
		&ProxyTransformer{},
	}

	// If we're on the root path, then we do a bunch of other stuff.
	// We don't do the following for modules.
	if len(path) <= 1 {
		steps = append(steps,
			// Optionally reduces the graph to a user-specified list of targets and
			// their dependencies.
			&TargetsTransformer{Targets: b.Targets, Destroy: b.Destroy},

			// Prune the providers. This must happen only once because flattened
			// modules might depend on empty providers.
			&PruneProviderTransformer{},

			// Create the destruction nodes
			&DestroyTransformer{FullDestroy: b.Destroy},
			b.conditional(&conditionalOpts{
				If:   func() bool { return !b.Destroy },
				Then: &CreateBeforeDestroyTransformer{},
			}),
			b.conditional(&conditionalOpts{
				If:   func() bool { return !b.Verbose },
				Then: &PruneDestroyTransformer{Diff: b.Diff, State: b.State},
			}),

			// Remove the noop nodes
			&PruneNoopTransformer{Diff: b.Diff, State: b.State},

			// Insert nodes to close opened plugin connections
			&CloseProviderTransformer{},
			&CloseProvisionerTransformer{},

			// Perform the transitive reduction to make our graph a bit
			// more sane if possible (it usually is possible).
			&TransitiveReductionTransformer{},
		)
	}

	// Make sure we have a single root
	steps = append(steps, &RootTransformer{})

	// Remove nils
	for i, s := range steps {
		if s == nil {
			steps = append(steps[:i], steps[i+1:]...)
		}
	}

	return steps
}

type conditionalOpts struct {
	If   func() bool
	Then GraphTransformer
}

func (b *BuiltinGraphBuilder) conditional(o *conditionalOpts) GraphTransformer {
	if o.If != nil && o.Then != nil && o.If() {
		return o.Then
	}
	return nil
}
