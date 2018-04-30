package terraform

import (
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
)

// FlatConfigTransformer is a GraphTransformer that adds the configuration
// to the graph. The module used to configure this transformer must be
// the root module.
//
// This transform adds the nodes but doesn't connect any of the references.
// The ReferenceTransformer should be used for that.
//
// NOTE: In relation to ConfigTransformer: this is a newer generation config
// transformer. It puts the _entire_ config into the graph (there is no
// "flattening" step as before).
type FlatConfigTransformer struct {
	Concrete ConcreteResourceNodeFunc // What to turn resources into

	Config *configs.Config
}

func (t *FlatConfigTransformer) Transform(g *Graph) error {
	// We have nothing to do if there is no configuration.
	if t.Config == nil {
		return nil
	}

	return t.transform(g, t.Config)
}

func (t *FlatConfigTransformer) transform(g *Graph, config *configs.Config) error {
	// If we have no configuration then there's nothing to do.
	if config == nil {
		return nil
	}

	// Transform all the children.
	for _, c := range config.Children {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	module := config.Module
	// For now we assume that each module call produces only one module
	// instance with no key, since we don't yet support "count" and "for_each"
	// on modules.
	// FIXME: As part of supporting "count" and "for_each" on modules, rework
	// this so that we'll "expand" the module call first and then create graph
	// nodes for each module instance separately.
	instPath := config.Path.UnkeyedInstanceShim()

	for _, r := range module.ManagedResources {
		addr := r.Addr().Absolute(instPath)
		abstract := &NodeAbstractResource{
			Addr:   addr,
			Config: r,
		}
		// Grab the address for this resource
		var node dag.Vertex = abstract
		if f := t.Concrete; f != nil {
			node = f(abstract)
		}

		g.Add(node)
	}

	return nil
}
