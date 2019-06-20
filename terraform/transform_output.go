package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
)

// OutputTransformer is a GraphTransformer that adds a graph node
// for each output value declared in the configuration or the state.
//
// Don't use this when building an "apply" graph: use OutputChangesTransformer
// instead.
type OutputTransformer struct {
	Config *configs.Config
	State  *states.State

	// If Concrete is not nil, Transform will call it to obtain a value
	// of a concrete subtype for a node representing an output that
	// is declared in the configuration.
	//
	// If Concrete is nil, the NodeRefreshableOutput type is used by default.
	Concrete func(abstract *NodeAbstractOutput) dag.Vertex

	// If ConcreteOrphan is not nil, Transform will call it to obtain a value
	// of a concrete subtype for a node representing an output that is
	// recorded in the state but not declared in the configuration.
	//
	// If ConcreteOrphan is nil, orphaned outputs are not added to the graph
	// at all.
	ConcreteOrphan func(abstract *NodeAbstractOutput) dag.Vertex
}

func (t *OutputTransformer) Transform(g *Graph) error {
	err := t.transformConfig(g, t.Config)
	if err != nil {
		return err
	}

	if t.State == nil || t.ConcreteOrphan == nil {
		return nil // nothing else to do
	}

	var rootMod *configs.Module
	if t.Config != nil {
		rootMod = t.Config.Module
	}
	return t.transformRootOrphans(g, rootMod, t.State.RootModule())
}

func (t *OutputTransformer) transformConfig(g *Graph, c *configs.Config) error {
	if c == nil {
		// If we have no config then there can be no outputs.
		return nil
	}

	// Transform all the children. We must do this first because
	// we can reference module outputs and they must show up in the
	// reference map.
	for _, cc := range c.Children {
		if err := t.transformConfig(g, cc); err != nil {
			return err
		}
	}

	// Our addressing system distinguishes between modules and module instances,
	// but we're not yet ready to make that distinction here (since we don't
	// support "count"/"for_each" on modules) and so we just do a naive
	// transform of the module path into a module instance path, assuming that
	// no keys are in use. This should be removed when "count" and "for_each"
	// are implemented for modules.
	path := c.Path.UnkeyedInstanceShim()

	for _, o := range c.Module.Outputs {
		addr := path.OutputValue(o.Name)

		abstract := &NodeAbstractOutput{
			Addr:   addr,
			Config: o,
		}
		var node dag.Vertex
		if t.Concrete != nil {
			node = t.Concrete(abstract)
		} else {
			node = &NodeRefreshableOutput{
				NodeAbstractOutput: abstract,
			}
		}
		g.Add(node)
	}

	return nil
}

func (t *OutputTransformer) transformRootOrphans(g *Graph, mc *configs.Module, ms *states.Module) error {
	// We only retain root module output values in state snapshots, so
	// therefore orphans can only exist there. Output values can always
	// be recalculated on subsequent runs, so the root outputs are only there
	// so they can be read by terraform_remote_state, "terraform output", etc.

	for n := range ms.OutputValues {
		if mc != nil {
			if _, exists := mc.Outputs[n]; exists {
				continue // ignore outputs still present in config
			}
		}

		abstract := &NodeAbstractOutput{
			Addr:   addrs.OutputValue{Name: n}.Absolute(addrs.RootModuleInstance),
			Config: nil, // orphan nodes have no config
		}
		g.Add(t.ConcreteOrphan(abstract))
	}

	return nil
}

// OutputChangesTransformer is a GraphTransformer that adds a graph node
// for each output value mentioned in a changeset.
//
// This is a specialized variant of OutputTransformer used only for building
// "apply" graphs.
type OutputChangesTransformer struct {
	Config  *configs.Config
	Changes *plans.Changes

	// If Concrete is not nil, Transform will call it to obtain a value
	// of a concrete subtype for a node representing each output.
	//
	// The configuration for a given output may be null if it has been removed
	// from configuration already. (In that case, we are presumably dealing
	// with a "Delete" change.)
	//
	// If Concrete is nil, the abstract node itself is added to the graph.
	Concrete func(abstract *NodeAbstractOutput, action plans.Action) dag.Vertex
}

func (t *OutputChangesTransformer) Transform(g *Graph) error {
	for _, changeSrc := range t.Changes.Outputs {
		addr := changeSrc.Addr

		// We'll try to find a corresponding declaration in the configuration,
		// but it's okay if one isn't present.
		var cfg *configs.Output
		if mc := t.Config.DescendentForInstance(addr.Module); mc != nil {
			cfg = mc.Module.Outputs[addr.OutputValue.Name]
		}

		abstract := &NodeAbstractOutput{
			Addr:   addr,
			Config: cfg,
		}
		var node dag.Vertex
		if t.Concrete != nil {
			node = t.Concrete(abstract, changeSrc.Action)
		} else {
			node = abstract
		}
		g.Add(node)
	}
	return nil
}
