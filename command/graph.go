package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.BoolVar(&verbose, "verbose", false, "verbose")
	cmdFlags.BoolVar(&drawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graphTypeStr, "type", "", "type")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var path string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The graph command expects one argument.\n")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		path = args[0]
	} else {
		var err error
		path, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	ctx, planFile, err := c.Context(contextOpts{
		Path:      path,
		StatePath: "",
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading Terraform: %s", err))
		return 1
	}

	// Determine the graph type
	graphType := terraform.GraphTypePlan
	if planFile {
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
                 legacy.

`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Create a visual graph of Terraform resources"
}
