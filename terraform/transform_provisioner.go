package terraform

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeProvisioner is an interface that nodes that can be a provisioner
// must implement. The ProvisionerName returned is the name of the provisioner
// they satisfy.
type GraphNodeProvisioner interface {
	ProvisionerName() string
}

// GraphNodeProvisionerConsumer is an interface that nodes that require
// a provisioner must implement. ProvisionedBy must return the name of the
// provisioner to use.
type GraphNodeProvisionerConsumer interface {
	ProvisionedBy() []string
}

// ProvisionerTransformer is a GraphTransformer that maps resources to
// provisioners within the graph. This will error if there are any resources
// that don't map to proper resources.
type ProvisionerTransformer struct{}

func (t *ProvisionerTransformer) Transform(g *Graph) error {
	// Go through the other nodes and match them to provisioners they need
	var err error
	m := provisionerVertexMap(g)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvisionerConsumer); ok {
			for _, provisionerName := range pv.ProvisionedBy() {
				target := m[provisionerName]
				if target == nil {
					err = multierror.Append(err, fmt.Errorf(
						"%s: provisioner %s couldn't be found",
						dag.VertexName(v), provisionerName))
					continue
				}

				g.Connect(dag.BasicEdge(v, target))
			}
		}
	}

	return err
}

// MissingProvisionerTransformer is a GraphTransformer that adds nodes
// for missing provisioners into the graph. Specifically, it creates provisioner
// configuration nodes for all the provisioners that we support. These are
// pruned later during an optimization pass.
type MissingProvisionerTransformer struct {
	// Provisioners is the list of provisioners we support.
	Provisioners []string
}

func (t *MissingProvisionerTransformer) Transform(g *Graph) error {
	m := provisionerVertexMap(g)
	for _, p := range t.Provisioners {
		if _, ok := m[p]; ok {
			// This provisioner already exists as a configured node
			continue
		}

		// Add our own missing provisioner node to the graph
		g.Add(&graphNodeMissingProvisioner{ProvisionerNameValue: p})
	}

	return nil
}

// PruneProvisionerTransformer is a GraphTransformer that prunes all the
// provisioners that aren't needed from the graph. A provisioner is unneeded if
// no resource or module is using that provisioner.
type PruneProvisionerTransformer struct{}

func (t *PruneProvisionerTransformer) Transform(g *Graph) error {
	for _, v := range g.Vertices() {
		// We only care about the provisioners
		if _, ok := v.(GraphNodeProvisioner); !ok {
			continue
		}

		// Does anything depend on this? If not, then prune it.
		if s := g.UpEdges(v); s.Len() == 0 {
			g.Remove(v)
		}
	}

	return nil
}

type graphNodeMissingProvisioner struct {
	ProvisionerNameValue string
}

func (n *graphNodeMissingProvisioner) Name() string {
	return fmt.Sprintf("provisioner.%s", n.ProvisionerNameValue)
}

// GraphNodeEvalable impl.
func (n *graphNodeMissingProvisioner) EvalTree() EvalNode {
	return &EvalInitProvisioner{Name: n.ProvisionerNameValue}
}

func (n *graphNodeMissingProvisioner) ProvisionerName() string {
	return n.ProvisionerNameValue
}

func provisionerVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvisioner); ok {
			m[pv.ProvisionerName()] = v
		}
	}

	return m
}
