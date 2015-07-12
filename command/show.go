package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
}

func (c *ShowCommand) Run(args []string) int {
	var moduleDepth int

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("show", flag.ContinueOnError)
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The show command expects at most one argument with the path\n" +
				"to a Terraform state or plan file.\n")
		cmdFlags.Usage()
		return 1
	}

	var planErr, stateErr error
	var path string
	var plan *terraform.Plan
	var state *terraform.State
	if len(args) > 0 {
		path = args[0]
		f, err := os.Open(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading file: %s", err))
			return 1
		}
		defer f.Close()

		plan, err = terraform.ReadPlan(f)
		if err != nil {
			if _, err := f.Seek(0, 0); err != nil {
				c.Ui.Error(fmt.Sprintf("Error reading file: %s", err))
				return 1
			}

			plan = nil
			planErr = err
		}
		if plan == nil {
			state, err = terraform.ReadState(f)
			if err != nil {
				stateErr = err
			}
		}
	} else {
		stateOpts := c.StateOpts()
		stateOpts.RemoteCacheOnly = true
		result, err := State(stateOpts)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error reading state: %s", err))
			return 1
		}
		state = result.State.State()
		if state == nil {
			c.Ui.Output("No state.")
			return 0
		}
	}

	if plan == nil && state == nil {
		c.Ui.Error(fmt.Sprintf(
			"Terraform couldn't read the given file as a state or plan file.\n"+
				"The errors while attempting to read the file as each format are\n"+
				"shown below.\n\n"+
				"State read error: %s\n\nPlan read error: %s",
			stateErr,
			planErr))
		return 1
	}

	if plan != nil {
		c.Ui.Output(FormatPlan(&FormatPlanOpts{
			Plan:        plan,
			Color:       c.Colorize(),
			ModuleDepth: moduleDepth,
		}))
		return 0
	}

	c.Ui.Output(FormatState(&FormatStateOpts{
		State:       state,
		Color:       c.Colorize(),
		ModuleDepth: moduleDepth,
	}))
	return 0
}

func (c *ShowCommand) Help() string {
	helpText := `
Usage: terraform show [options] [path]

  Reads and outputs a Terraform state or plan file in a human-readable
  form. If no path is specified, the current state will be shown.

Options:

  -module-depth=n     Specifies the depth of modules to show in the output.
                      By default this is zero. -1 will expand all.

  -no-color           If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *ShowCommand) Synopsis() string {
	return "Inspect Terraform state or plan"
}
