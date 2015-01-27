package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// OrphanTransformer is a GraphTransformer that adds orphans to the
// graph. This transformer adds both resource and module orphans.
type OrphanTransformer struct {
	State  *ModuleState
	Config *config.Config
}

func (t *OrphanTransformer) Transform(g *dag.Graph) error {
	// Get the orphans from our configuration. This will only get resources.
	orphans := t.State.Orphans(t.Config)
	if len(orphans) == 0 {
		return nil
	}

	// Go over each orphan and add it to the graph.
	for _, k := range orphans {
		v := g.Add(&graphNodeOrphanResource{ResourceName: k})
		GraphConnectDeps(g, v, t.State.Resources[k].Dependencies)
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
