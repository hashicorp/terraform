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

	// Get the orphans from our configuration. This will only get resources.
	orphans := state.Orphans(t.Config)
	if len(orphans) == 0 {
		return nil
	}

	// Go over each orphan and add it to the graph.
	for _, k := range orphans {
		g.ConnectTo(
			g.Add(&graphNodeOrphanResource{ResourceName: k}),
			state.Resources[k].Dependencies)
	}

	// TODO: modules

	return nil
}

// graphNodeOrphan is the graph vertex representing an orphan resource..
type graphNodeOrphanResource struct {
	ResourceName string
}

func (n *graphNodeOrphanResource) Name() string {
	return fmt.Sprintf("%s (orphan)", n.ResourceName)
}
