package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/tfdiags"
)

// ValidateCommand is a Command implementation that validates the terraform files
type ValidateCommand struct {
	Meta
}

const defaultPath = "."

func (c *ValidateCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("validate")
	cmdFlags.Usage = func() {
		c.Ui.Error(c.Help())
	}
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()

	var dirPath string
	if len(args) == 1 {
		dirPath = args[0]
	} else {
		dirPath = "."
	}
	dir, err := filepath.Abs(dirPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Unable to locate directory %v\n", err.Error()))
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	rtnCode := c.validate(dir)

	return rtnCode
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [dir]

  Validate the configuration files in a directory, referring only to the
  configuration and not accessing any remote services such as remote state,
  provider APIs, etc.

  Validate runs checks that verify whether a configuration is
  internally-consistent, regardless of any provided variables or existing
  state. It is thus primarily useful for general verification of reusable
  modules, including correctness of attribute names and value types.

  To verify configuration in the context of a particular run (a particular
  target workspace, operation variables, etc), use the following command
  instead:
      terraform plan -validate-only

  It is safe to run this command automatically, for example as a post-save
  check in a text editor or as a test step for a re-usable module in a CI
  system.

  Validation requires an initialized working directory with any referenced
  plugins and modules installed. To initialize a working directory for
  validation without accessing any configured remote backend, use:
      terraform init -backend=false

  If dir is not specified, then the current directory will be used.

Options:

  -no-color    If specified, output won't contain any color.
`
	return strings.TrimSpace(helpText)
}

func (c *ValidateCommand) validate(dir string) int {
	var diags tfdiags.Diagnostics

	_, cfgDiags := c.loadConfig(dir)
	diags = diags.Append(cfgDiags)

	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// TODO: run a validation walk once terraform.NewContext is updated
	// to support new-style configuration.
	/* old implementation of validation....
	mod, modDiags := c.Module(dir)
	diags = diags.Append(modDiags)
	if modDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	opts := c.contextOpts()
	opts.Module = mod

	tfCtx, err := terraform.NewContext(opts)
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	diags = diags.Append(tfCtx.Validate())
	*/

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	return 0
}
