package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.BoolVar(&verbose, "verbose", false, "verbose")
	cmdFlags.BoolVar(&drawCycles, "draw-cycles", false, "draw-cycles")
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

	ctx, _, err := c.Context(contextOpts{
		Path:      path,
		StatePath: "",
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading Terraform: %s", err))
		return 1
	}

	// Skip validation during graph generation - we want to see the graph even if
	// it is invalid for some reason.
	g, err := ctx.Graph(&terraform.ContextGraphOpts{
		Verbose:  verbose,
		Validate: false,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error creating graph: %s", err))
		return 1
	}

	graphStr, err := terraform.GraphDot(g, &terraform.GraphDotOpts{
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

  Outputs the visual dependency graph of Terraform resources according to
  configuration files in DIR (or the current directory if omitted).

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.

Options:

  -draw-cycles         Highlight any cycles in the graph with colored edges.
                       This helps when diagnosing cycle errors.

  -module-depth=n      The maximum depth to expand modules. By default this is
                       -1, which will expand resources within all modules.

  -verbose             Generate a verbose, "worst-case" graph, with all nodes
                       for potential operations in place.

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Create a visual graph of Terraform resources"
}
