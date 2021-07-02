package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

// movedTransformer inserts a graph node for each "moved" block it finds
// in the configuration, and it also inserts any necessary dependency edges
// with any nodes representing the objects given in "from" and "to", to
// ensure the correct evaluation order for moved block validation.
//
// This node does nothing at all if MakeNode is nil, because this transformer
// is only relevant in the plan walk.
type movedTransformer struct {
	Config   *configs.Config
	MakeNode func(module addrs.Module, movedConfig *configs.Moved, allConfig *configs.Config) dag.Vertex
}

var _ GraphTransformer = (*movedTransformer)(nil)

func (t *movedTransformer) Transform(g *Graph) error {
	if t.MakeNode == nil {
		return nil // nothing to do
	}

	t.processModule(t.Config, g)

	return nil
}

func (t *movedTransformer) processModule(modCfg *configs.Config, g *Graph) {
	for _, mc := range modCfg.Module.Moved {
		node := t.MakeNode(modCfg.Path, mc, t.Config)
		g.Add(node)

		// If we already have any nodes representing the "from" address
		// then we'll also need some dependency edges. The moved node should
		// run after "from" because nodeMovedValidate relies on the working
		// state to observe the effect of module/resource expansion without
		// having to repeat it.
		//
		// Note that we're intentionally using the addresses exactly as
		// given in configuration here, as opposed to trying to resolve
		// where they'd end up after all of the moves, because moved nodes
		// exist only to validate whether moved statements are valid and
		// that happens in terms of the configuration as currently written,
		// not in terms of the state we might be applying the moves to.
		fromAddr := mc.From.ConfigMoveable(modCfg.Path)

		// We don't make any edges related to "to" because none of our
		// validation rules react to the configuration of that resource.
		// Note also that "from" and "to" can potentially refer to the
		// same resource (if moving between instances) and so it would
		// not be correct to make "to" depend on the move; that would
		// lead to cycles.

		for _, v := range g.Vertices() {
			if v, ok := v.(GraphNodeConfigResource); ok {
				vAddr := v.ResourceAddr()
				if vAddr.IncludedInMoveable(fromAddr) {
					g.Connect(dag.BasicEdge(node, v))
				}
			}
			if v, ok := v.(GraphNodeModulePath); ok {
				vAddr := v.ModulePath()
				if vAddr.IncludedInMoveable(fromAddr) {
					g.Connect(dag.BasicEdge(node, v))
				}
			}
		}
	}

	// Recursively visit all child modules too
	for _, cc := range modCfg.Children {
		t.processModule(cc, g)
	}
}

func moveEndpointIncludesResource(epAddr addrs.ConfigMoveable, rAddr addrs.ConfigResource) bool {
	switch epAddr := epAddr.(type) {
	case addrs.ConfigResource:
		return epAddr.Equal(rAddr)
	case addrs.Module:
		// A resource is in scope of a module move if the resource's module
		// path has the given module path as a prefix.
		rModAddr := rAddr.Module
		if len(rModAddr) < len(epAddr) {
			return false // epAddr can't possibly be a prefix, then
		}
		rModAddr = rModAddr[:len(epAddr)]
		return epAddr.Equal(rModAddr)
	default:
		// The above cases should cover all ConfigMovable types
		panic(fmt.Sprintf("unhandled move endpoint address type %T", epAddr))
	}
}

func moveEndpointIncludesModule(epAddr addrs.ConfigMoveable, mAddr addrs.Module) bool {
	switch epAddr := epAddr.(type) {
	case addrs.ConfigResource:
		// A specific resource address can never refer to a whole module.
		return false
	case addrs.Module:
		// A module is in the scope of a module move if it has the move
		// address as a prefix.
		if len(mAddr) < len(epAddr) {
			return false // epAddr can't possibly be a prefix, then
		}
		mAddr = mAddr[:len(epAddr)]
		return epAddr.Equal(mAddr)
	default:
		// The above cases should cover all ConfigMovable types
		panic(fmt.Sprintf("unhandled move endpoint address type %T", epAddr))
	}
}
