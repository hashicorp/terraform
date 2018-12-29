package command

import (
	"fmt"
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

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("get")
	cmdFlags.BoolVar(&update, "update", false, "update")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	path, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	mode := module.GetModeGet
	if update {
		mode = module.GetModeUpdate
	}

	if err := getModules(&c.Meta, path, mode); err != nil {
		c.Ui.Error(err.Error())
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

func getModules(m *Meta, path string, mode module.GetMode) error {
	mod, err := module.NewTreeModule("", path)
	if err != nil {
		return fmt.Errorf("Error loading configuration: %s", err)
	}

	err = mod.Load(m.moduleStorage(m.DataDir(), mode))
	if err != nil {
		return fmt.Errorf("Error loading modules: %s", err)
	}

	return nil
}
