package command

import (
	"strings"

	"github.com/hashicorp/terraform/tfdiags"
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

	path = c.normalizePath(path)

	diags := getModules(&c.Meta, path, update)
	c.showDiagnostics(diags)
	if diags.HasErrors() {
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

  -update             Check already-downloaded modules for available updates
                      and install the newest versions available.

  -no-color           Disable text coloring in the output.

`
	return strings.TrimSpace(helpText)
}

func (c *GetCommand) Synopsis() string {
	return "Download and install modules for the configuration"
}

func getModules(m *Meta, path string, upgrade bool) tfdiags.Diagnostics {
	hooks := uiModuleInstallHooks{
		Ui:             m.Ui,
		ShowLocalPaths: true,
	}
	return m.installModules(path, upgrade, hooks)
}
