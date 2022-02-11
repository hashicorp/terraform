package componentseval

import (
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/tfcomponents/componentstree"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type componentGraphNode struct {
	GroupCallPath []ngaddrs.ComponentGroupCall
	ComponentCall ngaddrs.ComponentCall
}

func (n *componentGraphNode) String() string {
	var buf strings.Builder
	for _, step := range n.GroupCallPath {
		buf.WriteString("group.")
		buf.WriteString(step.Name)
		buf.WriteString("[*].")
	}
	buf.WriteString("component.")
	buf.WriteString(n.ComponentCall.Name)
	buf.WriteString("[*]")
	return buf.String()
}

// componentGraphRefNode is a node type we use as an intermediate step during
// component graph construction. These all get eliminated from the graph
// before we return, so the final graph contains only componentGraphNode
// instances with direct edges between them.
type componentGraphRefNode struct {
	Namespace []ngaddrs.ComponentGroupCall
	SelfAddr  addrs.Referenceable
	Refs      []*ngaddrs.Reference
}

// TODO: componentGraphRefNode should implement addrs.UniqueKeyer so we can
// make unique keys to build the refNodes maps below.

func newComponentGraph(root *componentstree.Node) (*dag.AcyclicGraph, tfdiags.Diagnostics) {
	graph := &dag.AcyclicGraph{}
	refNodes := make(map[addrs.UniqueKey]*componentGraphRefNode)
	diags := addComponentGraphNodes(graph, root, refNodes)
	return graph, diags
}

func addComponentGraphNodes(to *dag.AcyclicGraph, from *componentstree.Node, refNodes map[addrs.UniqueKey]*componentGraphRefNode) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// To start we add to the graph everything that can generate references,
	// and associate the component calls themselves only with the references
	// notes they directly generated. After we return, the next step will
	// be to connect the refNodes to each other based on their references,
	// which is to be done by our caller.

	//cfg := from.Config
	//namespace := from.CallPath

	for _, childNode := range from.Children {
		diags = diags.Append(
			addComponentGraphNodes(to, childNode, refNodes),
		)
	}
	return diags
}
