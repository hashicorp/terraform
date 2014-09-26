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

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	cmdFlags.IntVar(&moduleDepth, "module-depth", 0, "module-depth")
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

	g, err := ctx.Graph()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error creating graph: %s", err))
		return 1
	}

	opts := &terraform.GraphDotOpts{
		ModuleDepth: moduleDepth,
	}

	c.Ui.Output(terraform.GraphDot(g, opts))

	return 0
}

func (c *GraphCommand) Help() string {
	helpText := `
Usage: terraform graph [options] PATH

  Outputs the visual graph of Terraform resources. If the path given is
  the path to a configuration, the dependency graph of the resources are
  shown. If the path is a plan file, then the dependency graph of the
  plan itself is shown.

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.

Options:

  -module-depth=n      The maximum depth to expand modules. By default this is
                       zero, which will not expand modules at all.

`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Create a visual graph of Terraform resources"
}
