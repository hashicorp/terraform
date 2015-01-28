package terraform

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// ProviderTransformer is a GraphTransformer that maps resources to
// providers within the graph. This will error if there are any resources
// that don't map to proper resources.
type ProviderTransformer struct{}

func (t *ProviderTransformer) Transform(g *Graph) error {
	// First, build a map of the providers
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvider); ok {
			m[pv.ProviderName()] = v
		}
	}

	// Go through the other nodes and match them to providers they need
	var err error
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
