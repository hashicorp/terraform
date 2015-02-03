package terraform

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeProvider is an interface that nodes that can be a provider
// must implement. The ProviderName returned is the name of the provider
// they satisfy.
type GraphNodeProvider interface {
	ProviderName() string
}

// GraphNodeProviderConsumer is an interface that nodes that require
// a provider must implement. ProvidedBy must return the name of the provider
// to use.
type GraphNodeProviderConsumer interface {
	ProvidedBy() string
}

// ProviderTransformer is a GraphTransformer that maps resources to
// providers within the graph. This will error if there are any resources
// that don't map to proper resources.
type ProviderTransformer struct{}

func (t *ProviderTransformer) Transform(g *Graph) error {
	// Go through the other nodes and match them to providers they need
	var err error
	m := providerVertexMap(g)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProviderConsumer); ok {
			target := m[pv.ProvidedBy()]
			if target == nil {
				err = multierror.Append(err, fmt.Errorf(
					"%s: provider %s couldn't be found",
					dag.VertexName(v), pv.ProvidedBy()))
				continue
			}

			g.Connect(dag.BasicEdge(v, target))
		}
	}

	return err
}

// MissingProviderTransformer is a GraphTransformer that adds nodes
// for missing providers into the graph. Specifically, it creates provider
// configuration nodes for all the providers that we support. These are
// pruned later during an optimization pass.
type MissingProviderTransformer struct {
	// Providers is the list of providers we support.
	Providers []string
}

func (t *MissingProviderTransformer) Transform(g *Graph) error {
	m := providerVertexMap(g)
	for _, p := range t.Providers {
		if _, ok := m[p]; ok {
			// This provider already exists as a configured node
			continue
		}

		// Add our own missing provider node to the graph
		g.Add(&graphNodeMissingProvider{ProviderNameValue: p})
	}

	return nil
}

// PruneProviderTransformer is a GraphTransformer that prunes all the
// providers that aren't needed from the graph. A provider is unneeded if
// no resource or module is using that provider.
type PruneProviderTransformer struct{}

func (t *PruneProviderTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about the providers
		if _, ok := v.(GraphNodeProvider); !ok {
			continue
		}

		// Does anything depend on this? If not, then prune it.
		if s := g.UpEdges(v); s.Len() == 0 {
			g.Remove(v)
		}
	}

	return nil
}

type graphNodeMissingProvider struct {
	ProviderNameValue string
}

func (n *graphNodeMissingProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.ProviderNameValue)
}

// GraphNodeEvalable impl.
func (n *graphNodeMissingProvider) EvalTree() EvalNode {
	return &EvalInitProvider{Name: n.ProviderNameValue}
}

func (n *graphNodeMissingProvider) ProviderName() string {
	return n.ProviderNameValue
}

func providerVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvider); ok {
			m[pv.ProviderName()] = v
		}
	}

	return m
}
