package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
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
	var checkVars bool

	cmdFlags := c.Meta.flagSet("validate")
	cmdFlags.BoolVar(&checkVars, "check-variables", true, "check-variables")
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

	rtnCode := c.validate(dir, checkVars)

	return rtnCode
}

func (c *ValidateCommand) Synopsis() string {
	return "Validates the Terraform files"
}

func (c *ValidateCommand) Help() string {
	helpText := `
Usage: terraform validate [options] [dir]

  Validate the terraform files in a directory. Validation includes a
  basic check of syntax as well as checking that all variables declared
  in the configuration are specified in one of the possible ways:

      -var foo=...
      -var-file=foo.vars
      TF_VAR_foo environment variable
      terraform.tfvars
      default value

  If dir is not specified, then the current directory will be used.

Options:

  -check-variables=true If set to true (default), the command will check
                        whether all required variables have been specified.

  -no-color             If specified, output won't contain any color.

  -var 'foo=bar'        Set a variable in the Terraform configuration. This
                        flag can be set multiple times.

  -var-file=foo         Set variables in the Terraform configuration from
                        a file. If "terraform.tfvars" is present, it will be
                        automatically loaded if this flag is not specified.
`
	return strings.TrimSpace(helpText)
}

func (c *ValidateCommand) validate(dir string, checkVars bool) int {
	cfg, err := config.LoadDir(dir)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}
	err = cfg.Validate()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	if checkVars {
		mod, err := c.Module(dir)
		if err != nil {
			c.showDiagnostics(err)
			return 1
		}

		opts := c.contextOpts()
		opts.Module = mod

		tfCtx, err := terraform.NewContext(opts)
		if err != nil {
			c.showDiagnostics(err)
			return 1
		}

		if !validateContext(tfCtx, c.Ui) {
			return 1
		}
	}

	return 0
}
