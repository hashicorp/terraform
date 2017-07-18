package command

import (
	"flag"
	"fmt"
	"strings"

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

	// Load the module
	var mod *module.Tree
	if plan == nil {
		mod, err = c.Module(configPath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load root config module: %s", err))
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

	graphStr, err := terraform.GraphDot(g, &dag.DotOpts{
		DrawCycles: drawCycles,
		MaxDepth:   moduleDepth,
		Verbose:    verbose,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error converting graph: %s", err))
		return 1
	}

	c.Ui.Output(graphStr)

	return 0
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
