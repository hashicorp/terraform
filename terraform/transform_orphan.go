package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// OrphanTransformer is a GraphTransformer that adds orphans to the
// graph. This transformer adds both resource and module orphans.
type OrphanTransformer struct {
	// State is the global state. We require the global state to
	// properly find module orphans at our path.
	State *State

	// Config is just the configuration of our current module.
	Config *config.Config
}

func (t *OrphanTransformer) Transform(g *Graph) error {
	state := t.State.ModuleByPath(g.Path)
	if state == nil {
		// If there is no state for our module, there can't be any orphans
		return nil
	}

	// Go over each resource orphan and add it to the graph.
	for _, k := range state.Orphans(t.Config) {
		g.ConnectTo(
			g.Add(&graphNodeOrphanResource{ResourceName: k}),
			state.Resources[k].Dependencies)
	}

	// Go over each module orphan and add it to the graph
	for _, path := range t.State.ModuleOrphans(g.Path, t.Config) {
		g.Add(&graphNodeOrphanModule{Path: path})
	}

	return nil
}

// graphNodeOrphanModule is the graph vertex representing an orphan resource..
type graphNodeOrphanModule struct {
	Path []string
}

func (n *graphNodeOrphanModule) Name() string {
	return fmt.Sprintf("module.%s (orphan)", n.Path[len(n.Path)-1])
}

// graphNodeOrphanResource is the graph vertex representing an orphan resource..
type graphNodeOrphanResource struct {
	ResourceName string
}

func (n *graphNodeOrphanResource) Name() string {
	return fmt.Sprintf("%s (orphan)", n.ResourceName)
}
