package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/digraph"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// GraphCommand is a Command implementation that takes a Terraform
// configuration and outputs the dependency tree in graphical form.
type GraphCommand struct {
	TFConfig *terraform.Config
	Ui       cli.Ui
}

func (c *GraphCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The graph command expects one argument.\n")
		cmdFlags.Usage()
		return 1
	}

	conf, err := config.Load(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading config: %s", err))
		return 1
	}

	g, err := terraform.Graph(&terraform.GraphOpts{
		Config:    conf,
		Providers: c.TFConfig.Providers,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error creating graph: %s", err))
		return 1
	}

	nodes := make([]digraph.Node, len(g.Nouns))
	for i, n := range g.Nouns {
		nodes[i] = n
	}
	digraph.GenerateDot(nodes, os.Stdout)

	return 0
}

func (c *GraphCommand) Help() string {
	helpText := `
Usage: terraform graph [options] PATH

  Outputs the visual graph of Terraform resources. If the path given is
  the path to a configuration, the dependency graph of the resources are
  shown. If the path is a plan file, then the dependency graph of the
  plan itself is shown.

`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Output visual graph of Terraform resources"
}
