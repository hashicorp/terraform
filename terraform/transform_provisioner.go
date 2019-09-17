package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeProvisioner is an interface that nodes that can be a provisioner
// must implement. The ProvisionerName returned is the name of the provisioner
// they satisfy.
type GraphNodeProvisioner interface {
	ProvisionerName() string
}

// GraphNodeCloseProvisioner is an interface that nodes that can be a close
// provisioner must implement. The CloseProvisionerName returned is the name
// of the provisioner they satisfy.
type GraphNodeCloseProvisioner interface {
	CloseProvisionerName() string
}

// GraphNodeProvisionerConsumer is an interface that nodes that require
// a provisioner must implement. ProvisionedBy must return the names of the
// provisioners to use.
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
			for _, p := range pv.ProvisionedBy() {
				if m[p] == nil {
					err = multierror.Append(err, fmt.Errorf(
						"%s: provisioner %s couldn't be found",
						dag.VertexName(v), p))
					continue
				}

				log.Printf("[TRACE] ProvisionerTransformer: %s is provisioned by %s (%q)", dag.VertexName(v), p, dag.VertexName(m[p]))
				g.Connect(dag.BasicEdge(v, m[p]))
			}
		}
	}

	return err
}

// MissingProvisionerTransformer is a GraphTransformer that adds nodes
// for missing provisioners into the graph.
type MissingProvisionerTransformer struct {
	// Provisioners is the list of provisioners we support.
	Provisioners []string
}

func (t *MissingProvisionerTransformer) Transform(g *Graph) error {
	// Create a set of our supported provisioners
	supported := make(map[string]struct{}, len(t.Provisioners))
	for _, v := range t.Provisioners {
		supported[v] = struct{}{}
	}

	// Get the map of provisioners we already have in our graph
	m := provisionerVertexMap(g)

	// Go through all the provisioner consumers and make sure we add
	// that provisioner if it is missing.
	for _, v := range g.Vertices() {
		pv, ok := v.(GraphNodeProvisionerConsumer)
		if !ok {
			continue
		}

		for _, p := range pv.ProvisionedBy() {
			if _, ok := m[p]; ok {
				// This provisioner already exists as a configure node
				continue
			}

			if _, ok := supported[p]; !ok {
				// If we don't support the provisioner type, we skip it.
				// Validation later will catch this as an error.
				continue
			}

			// Build the vertex
			var newV dag.Vertex = &NodeProvisioner{
				NameValue: p,
			}

			// Add the missing provisioner node to the graph
			m[p] = g.Add(newV)
			log.Printf("[TRACE] MissingProviderTransformer: added implicit provisioner %s, first implied by %s", p, dag.VertexName(v))
		}
	}

	return nil
}

// CloseProvisionerTransformer is a GraphTransformer that adds nodes to the
// graph that will close open provisioner connections that aren't needed
// anymore. A provisioner connection is not needed anymore once all depended
// resources in the graph are evaluated.
type CloseProvisionerTransformer struct{}

func (t *CloseProvisionerTransformer) Transform(g *Graph) error {
	m := closeProvisionerVertexMap(g)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeProvisionerConsumer); ok {
			for _, p := range pv.ProvisionedBy() {
				source := m[p]

				if source == nil {
					// Create a new graphNodeCloseProvisioner and add it to the graph
					source = &graphNodeCloseProvisioner{ProvisionerNameValue: p}
					g.Add(source)

					// Make sure we also add the new graphNodeCloseProvisioner to the map
					// so we don't create and add any duplicate graphNodeCloseProvisioners.
					m[p] = source
				}

				g.Connect(dag.BasicEdge(source, v))
			}
		}
	}

	return nil
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

func closeProvisionerVertexMap(g *Graph) map[string]dag.Vertex {
	m := make(map[string]dag.Vertex)
	for _, v := range g.Vertices() {
		if pv, ok := v.(GraphNodeCloseProvisioner); ok {
			m[pv.CloseProvisionerName()] = v
		}
	}

	return m
}

type graphNodeCloseProvisioner struct {
	ProvisionerNameValue string
}

func (n *graphNodeCloseProvisioner) Name() string {
	return fmt.Sprintf("provisioner.%s (close)", n.ProvisionerNameValue)
}

// GraphNodeEvalable impl.
func (n *graphNodeCloseProvisioner) EvalTree() EvalNode {
	return &EvalCloseProvisioner{Name: n.ProvisionerNameValue}
}

func (n *graphNodeCloseProvisioner) CloseProvisionerName() string {
	return n.ProvisionerNameValue
}
