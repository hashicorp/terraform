package command

import (
	"bytes"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/terraform"
)

// GraphCommand is a Command implementation that takes a Terraform
// configuration and outputs the dependency tree in graphical form.
type GraphCommand struct {
	Meta
}

func (c *GraphCommand) Run(args []string) int {
	var moduleDepth int
	var verbose bool
	var drawCycles bool
	var graphTypeStr string

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.BoolVar(&verbose, "verbose", false, "verbose")
	cmdFlags.BoolVar(&drawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graphTypeStr, "type", "", "type")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check if the path is a plan
	plan, err := c.Plan(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if plan != nil {
		// Reset for backend loading
		configPath = ""
	}

	var diags tfdiags.Diagnostics

	// Load the module
	var mod *module.Tree
	if plan == nil {
		var modDiags tfdiags.Diagnostics
		mod, modDiags = c.Module(configPath)
		diags = diags.Append(modDiags)
		if modDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
		Plan:   plan,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Building a graph may require config module to be present, even if it's
	// empty.
	if mod == nil && plan == nil {
		mod = module.NewEmptyTree()
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Module = mod
	opReq.Plan = plan

	// Get the context
	ctx, _, err := local.Context(opReq)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Determine the graph type
	graphType := terraform.GraphTypePlan
	if plan != nil {
		graphType = terraform.GraphTypeApply
	}

	if graphTypeStr != "" {
		v, ok := terraform.GraphTypeMap[graphTypeStr]
		if !ok {
			c.Ui.Error(fmt.Sprintf("Invalid graph type requested: %s", graphTypeStr))
			return 1
		}

		graphType = v
	}

	// Skip validation during graph generation - we want to see the graph even if
	// it is invalid for some reason.
	g, err := ctx.Graph(graphType, &terraform.ContextGraphOpts{
		Verbose:  verbose,
		Validate: false,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error creating graph: %s", err))
		return 1
	}

	graphStr := c.dotGraph(g, drawCycles)

	if diags.HasErrors() {
		// For this command we only show diagnostics if there are errors,
		// because printing out naked warnings could upset a naive program
		// consuming our dot output.
		c.showDiagnostics(diags)
		return 1
	}

	c.Ui.Output(graphStr)

	return 0
}

func (c *GraphCommand) dotGraph(g *terraform.Graph, drawCycles bool) string {
	nodes := map[dag.Vertex]*graphCommandNode{}
	moduleNodes := map[string][]*graphCommandNode{}
	var startNodes []*graphCommandNode
	var endNodes []*graphCommandNode

	for _, v := range g.Vertices() {
		//fmt.Printf("vertex %#v\n", v)
		node := c.node(v)
		if node == nil {
			continue
		}
		nodes[node.Vertex] = node
		moduleNodes[node.ModuleAddr] = append(moduleNodes[node.ModuleAddr], node)

		if g.UpEdges(v).Len() == 0 {
			endNodes = append(endNodes, node)
		}
		if g.DownEdges(v).Len() == 0 {
			startNodes = append(startNodes, node)
		}
	}

	moduleAddrs := make([]string, 0, len(moduleNodes))
	for name := range moduleNodes {
		moduleAddrs = append(moduleAddrs, name)
	}
	sort.Strings(moduleAddrs)

	buf := &bytes.Buffer{}

	fmt.Fprintln(buf, "digraph G {")
	fmt.Fprintln(buf, `  compound = "true";`)
	fmt.Fprintln(buf, `  newrank = "true";`)
	fmt.Fprintln(buf, `  rankdir = "RL";`)
	fmt.Fprintln(buf, `  bgcolor = "lightgrey";`)
	fmt.Fprintln(buf, `  style = "solid";`)
	fmt.Fprintln(buf, `  penwidth = "0.5";`)
	fmt.Fprintln(buf, `  pad = "0.1";`)
	fmt.Fprintln(buf, `  nodesep = "0.35";`)
	fmt.Fprintln(buf, `  graph [fontname="helvetica"];`)
	fmt.Fprintln(buf, `  node [fontname="helvetica", style="filled", fillcolor="honeydew", penwidth="1.0", margin="0.05,0.0"];`)
	fmt.Fprintln(buf, `  edge [fontname="helvetica", minlen="2", dir="back"];`)
	fmt.Fprintln(buf, `  before [label="", shape="point"];`)
	fmt.Fprintln(buf, `  after [label="", shape="point"];`)

	for _, addr := range moduleAddrs {
		nodes := moduleNodes[addr]
		fmt.Fprintf(buf, "  subgraph \"cluster_%s\" {\n", addr)
		fmt.Fprintf(buf, "    label = \"%s\";\n", addr)
		fmt.Fprintln(buf, `    labelloc = "t";`)
		fmt.Fprintln(buf, `    labeljust = "c";`)
		fmt.Fprintln(buf, `    bgcolor = "white";`)
		for _, node := range nodes {
			fmt.Fprintf(buf, "    \"%s:%s\" [label=%q, shape=%q];\n", node.ModuleAddr, node.Label, node.Label, node.Shape)
		}
		fmt.Fprintln(buf, "  }")
	}

	for _, node := range startNodes {
		fmt.Fprintf(buf, "  \"%s:%s\" -> before;\n", node.ModuleAddr, node.Label)
	}
	for _, node := range endNodes {
		fmt.Fprintf(buf, "  after -> \"%s:%s\"\n", node.ModuleAddr, node.Label)
	}

	fmt.Fprintln(buf, "}\n")

	return buf.String()
}

type graphCommandNode struct {
	Vertex     dag.Vertex
	Label      string
	ModuleAddr string
	Shape      string
}

// node returns a graphCommandNode if the given vertex is an interesting one
// to render in our graph, or nil to ignore the node for rendering.
func (c *GraphCommand) node(v dag.Vertex) *graphCommandNode {
	switch tv := v.(type) {

	case terraform.GraphNodeResource:
		addr := tv.ResourceAddr()

		// Pull out just the module part of the address to get our ModuleAddr
		moduleAddr := addr.Copy()
		moduleAddr.Type = ""
		moduleAddr.Name = ""
		moduleAddr.Mode = config.ManagedResourceMode

		// Pull out just the local part of the address to get our Label
		localAddr := addr.Copy()
		localAddr.Path = nil

		var shape string
		switch addr.Mode {
		case config.DataResourceMode:
			shape = "octagon"
		default: // assumed to be config.ManagedResourcemode
			shape = "box"
		}

		return &graphCommandNode{
			Vertex:     v,
			Label:      localAddr.String(),
			ModuleAddr: moduleAddr.String(),
			Shape:      shape,
		}

	case *terraform.NodeRootVariable:
		return &graphCommandNode{
			Vertex:     v,
			Label:      tv.Name(),
			ModuleAddr: "",
			Shape:      "oval",
		}

	case *terraform.NodeApplyableModuleVariable:
		return &graphCommandNode{
			Vertex:     v,
			Label:      fmt.Sprintf("var.%s", tv.Config.Name),
			ModuleAddr: c.friendlyModulePath(tv.PathValue),
			Shape:      "oval",
		}

	case *terraform.NodeApplyableOutput:
		return &graphCommandNode{
			Vertex:     v,
			Label:      fmt.Sprintf("output.%s", tv.Config.Name),
			ModuleAddr: c.friendlyModulePath(tv.PathValue),
			Shape:      "oval",
		}

	default:
		return nil
	}
}

func (c *GraphCommand) friendlyModulePath(path []string) string {
	if len(path) == 0 {
		// should never happen
		return ""
	}

	subModules := path[1:]
	buf := &bytes.Buffer{}
	for i, name := range subModules {
		if i == 0 {
			buf.WriteString("module.")
		} else {
			buf.WriteString(".module.")
		}
		buf.WriteString(name)
	}
	return buf.String()
}

func (c *GraphCommand) Help() string {
	helpText := `
Usage: terraform graph [options] [DIR]

  Outputs the visual execution graph of Terraform resources according to
  configuration files in DIR (or the current directory if omitted).

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.

  The -type flag can be used to control the type of graph shown. Terraform
  creates different graphs for different operations. See the options below
  for the list of types supported. The default type is "plan" if a
  configuration is given, and "apply" if a plan file is passed as an
  argument.

Options:

  -draw-cycles   Highlight any cycles in the graph with colored edges.
                 This helps when diagnosing cycle errors.

  -no-color      If specified, output won't contain any color.

  -type=plan     Type of graph to output. Can be: plan, plan-destroy, apply,
                 validate, input, refresh.


`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Create a visual graph of Terraform resources"
}
