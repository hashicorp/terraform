package terraform

import (
	"bytes"
	"bufio"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/depgraph"
	"github.com/hashicorp/terraform/digraph"
)

// GraphDotOpts are options for turning a graph into dot format.
type GraphDotOpts struct {
	// ModuleDepth is the depth of modules to expand. Zero is no expansion,
	// one expands the first set of modules, etc. If this is set to -1, then
	// all modules are expanded.
	ModuleDepth int

	// Depth is an internal track of what depth we're at within
	// the graph, used to control indentation and other such things.
	depth int
}

// GraphDot returns the dot formatting of a visual representation of
// the given Terraform graph.
func GraphDot(g *depgraph.Graph, opts *GraphDotOpts) string {
	buf := new(bytes.Buffer)

	if opts.depth == 0 {
		buf.WriteString("digraph {\n")
	}

	// Determine and add the title
	// graphDotTitle(buf, g)

	// Add all the resource.
	graphDotAddResources(buf, g)

	// Add all the resource providers
	graphDotAddResourceProviders(buf, g)

	// Add all the modules
	graphDotAddModules(buf, g, opts)

	if opts.depth == 0 {
		buf.WriteString("}\n")
	}

	return buf.String()
}

func graphDotAddRoot(buf *bytes.Buffer, n *depgraph.Noun) {
	buf.WriteString(fmt.Sprintf("\t\"%s\" [shape=circle];\n", "root"))

	for _, e := range n.Edges() {
		target := e.Tail()
		buf.WriteString(fmt.Sprintf(
			"\t\"%s\" -> \"%s\";\n",
			"root",
			target))
	}
}

func graphDotAddModules(buf *bytes.Buffer, g *depgraph.Graph, opts *GraphDotOpts) {
	for _, n := range g.Nouns {
		_, ok := n.Meta.(*GraphNodeModule)
		if !ok {
			continue
		}

		if opts.ModuleDepth == opts.depth {
			// We're not expanding, so just add the module on its own
			graphDotAddModuleSingle(buf, n, opts)
		} else {
			// We're expanding
			graphDotAddModuleExpand(buf, n, opts)
		}
	}
}

func graphDotAddModuleExpand(
	buf *bytes.Buffer, n *depgraph.Noun, opts *GraphDotOpts) {
	m := n.Meta.(*GraphNodeModule)
	tab := strings.Repeat("\t", opts.depth+1)

	// Wrap ourselves in a subgraph
	buf.WriteString(fmt.Sprintf("%ssubgraph \"%s\" {\n", tab, n.Name))
	defer buf.WriteString(fmt.Sprintf("%s}\n", tab))

	// Graph the subgraph just as we would any other graph
	subOpts := *opts
	subOpts.depth++
	subStr := GraphDot(m.Graph, &subOpts)

	// Tab all the lines of the subgraph
	s := bufio.NewScanner(strings.NewReader(subStr))
	for s.Scan() {
		buf.WriteString(fmt.Sprintf("%s%s\n", tab, s.Text()))
	}
}

func graphDotAddModuleSingle(
	buf *bytes.Buffer, n *depgraph.Noun, opts *GraphDotOpts) {
	tab := strings.Repeat("\t", opts.depth+1)

	//m := n.Meta.(*GraphNodeModule)

	// Create this node.
	buf.WriteString(fmt.Sprintf("%s\"%s\" [\n", tab, n))
	buf.WriteString(fmt.Sprintf("%s\tshape=component\n", tab))
	buf.WriteString(fmt.Sprintf("%s];\n", tab))

	for _, e := range n.Edges() {
		target := e.Tail()
		buf.WriteString(fmt.Sprintf(
			"%s\"%s\" -> \"%s\";\n",
			tab,
			n,
			target))
	}
}

func graphDotAddResources(buf *bytes.Buffer, g *depgraph.Graph) {
	// Determine if we have diffs. If we do, then we're graphing a
	// plan, which alters our graph a bit.
	hasDiff := false
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.Resource.Diff != nil && !rn.Resource.Diff.Empty() {
			hasDiff = true
			break
		}
	}

	var edgeBuf bytes.Buffer
	// Do all the non-destroy resources
	buf.WriteString("\tsubgraph {\n")
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.Resource.Diff != nil && rn.Resource.Diff.Destroy {
			continue
		}

		// If we have diffs then we're graphing a plan. If we don't have
		// have a diff on this resource, don't graph anything, since the
		// plan wouldn't do anything to this resource.
		if hasDiff {
			if rn.Resource.Diff == nil || rn.Resource.Diff.Empty() {
				continue
			}
		}

		// Determine the colors. White = no change, yellow = change,
		// green = create. Destroy is in the next section.
		var color, fillColor string
		if rn.Resource.Diff != nil && !rn.Resource.Diff.Empty() {
			if rn.Resource.State != nil && rn.Resource.State.ID != "" {
				color = "#FFFF00"
				fillColor = "#FFFF94"
			} else {
				color = "#00FF00"
				fillColor = "#9EFF9E"
			}
		}

		// Create this node.
		buf.WriteString(fmt.Sprintf("\t\t\"%s\" [\n", n))
		buf.WriteString("\t\t\tshape=box\n")
		if color != "" {
			buf.WriteString("\t\t\tstyle=filled\n")
			buf.WriteString(fmt.Sprintf("\t\t\tcolor=\"%s\"\n", color))
			buf.WriteString(fmt.Sprintf("\t\t\tfillcolor=\"%s\"\n", fillColor))
		}
		buf.WriteString("\t\t];\n")

		// Build up all the edges in a separate buffer so they're not in the
		// subgraph.
		for _, e := range n.Edges() {
			target := e.Tail()
			edgeBuf.WriteString(fmt.Sprintf(
				"\t\"%s\" -> \"%s\";\n",
				n,
				target))
		}
	}
	buf.WriteString("\t}\n\n")
	if edgeBuf.Len() > 0 {
		buf.WriteString(edgeBuf.String())
		buf.WriteString("\n")
	}

	// Do all the destroy resources
	edgeBuf.Reset()
	buf.WriteString("\tsubgraph {\n")
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.Resource.Diff == nil || !rn.Resource.Diff.Destroy {
			continue
		}

		buf.WriteString(fmt.Sprintf(
			"\t\t\"%s\" [shape=box,style=filled,color=\"#FF0000\",fillcolor=\"#FF9494\"];\n", n))

		for _, e := range n.Edges() {
			target := e.Tail()
			edgeBuf.WriteString(fmt.Sprintf(
				"\t\"%s\" -> \"%s\";\n",
				n,
				target))
		}
	}
	buf.WriteString("\t}\n\n")
	if edgeBuf.Len() > 0 {
		buf.WriteString(edgeBuf.String())
		buf.WriteString("\n")
	}

	// Handle the meta resources
	edgeBuf.Reset()
	for _, n := range g.Nouns {
		_, ok := n.Meta.(*GraphNodeResourceMeta)
		if !ok {
			continue
		}

		// Determine which edges to add
		var edges []digraph.Edge
		if hasDiff {
			for _, e := range n.Edges() {
				rn, ok := e.Tail().(*depgraph.Noun).Meta.(*GraphNodeResource)
				if !ok {
					continue
				}
				if rn.Resource.Diff == nil || rn.Resource.Diff.Empty() {
					continue
				}
				edges = append(edges, e)
			}
		} else {
			edges = n.Edges()
		}

		// Do not draw if we have no edges
		if len(edges) == 0 {
			continue
		}

		for _, e := range edges {
			target := e.Tail()
			edgeBuf.WriteString(fmt.Sprintf(
				"\t\"%s\" -> \"%s\";\n",
				n,
				target))
		}
	}
	if edgeBuf.Len() > 0 {
		buf.WriteString(edgeBuf.String())
		buf.WriteString("\n")
	}
}

func graphDotAddResourceProviders(buf *bytes.Buffer, g *depgraph.Graph) {
	var edgeBuf bytes.Buffer
	buf.WriteString("\tsubgraph {\n")
	for _, n := range g.Nouns {
		_, ok := n.Meta.(*GraphNodeResourceProvider)
		if !ok {
			continue
		}

		// Create this node.
		buf.WriteString(fmt.Sprintf("\t\t\"%s\" [\n", n))
		buf.WriteString("\t\t\tshape=diamond\n")
		buf.WriteString("\t\t];\n")

		// Build up all the edges in a separate buffer so they're not in the
		// subgraph.
		for _, e := range n.Edges() {
			target := e.Tail()
			edgeBuf.WriteString(fmt.Sprintf(
				"\t\"%s\" -> \"%s\";\n",
				n,
				target))
		}
	}
	buf.WriteString("\t}\n\n")
	if edgeBuf.Len() > 0 {
		buf.WriteString(edgeBuf.String())
		buf.WriteString("\n")
	}
}

func graphDotTitle(buf *bytes.Buffer, g *depgraph.Graph) {
	// Determine if we have diffs. If we do, then we're graphing a
	// plan, which alters our graph a bit.
	hasDiff := false
	for _, n := range g.Nouns {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			continue
		}
		if rn.Resource.Diff != nil && !rn.Resource.Diff.Empty() {
			hasDiff = true
			break
		}
	}

	graphType := "Configuration"
	if hasDiff {
		graphType = "Plan"
	}
	title := fmt.Sprintf("Terraform %s Resource Graph", graphType)

	buf.WriteString(fmt.Sprintf("\tlabel=\"%s\\n\\n\\n\";\n", title))
	buf.WriteString("\tlabelloc=\"t\";\n\n")
}
