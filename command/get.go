package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config/module"
)

// GetCommand is a Command implementation that takes a Terraform
// configuration and downloads all the modules.
type GetCommand struct {
	Meta
}

func (c *GetCommand) Run(args []string) int {
	var update bool

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("get", flag.ContinueOnError)
	cmdFlags.BoolVar(&update, "update", false, "update")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var path string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The get command expects one argument.\n")
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

	mode := module.GetModeGet
	if update {
		mode = module.GetModeUpdate
	}

	_, _, err := c.Context(contextOpts{
		Path:    path,
		GetMode: mode,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading Terraform: %s", err))
		return 1
	}

	return 0
}

func (c *GetCommand) Help() string {
	helpText := `
Usage: terraform get [options] PATH

  Downloads and installs modules needed for the configuration given by
  PATH.

  This recursively downloads all modules needed, such as modules
  imported by modules imported by the root and so on. If a module is
  already downloaded, it will not be redownloaded or checked for updates
  unless the -update flag is specified.

Options:

  -update=false       If true, modules already downloaded will be checked
                      for updates and updated if necessary.

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *GetCommand) Synopsis() string {
	return "Download and install modules for the configuration"
}
