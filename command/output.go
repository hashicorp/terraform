package command

import (
	"flag"
	"fmt"
	"strings"
)

// OutputCommand is a Command implementation that reads an output
// from a Terraform state and prints it.
type OutputCommand struct {
	Meta
}

func (c *OutputCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("output", flag.ContinueOnError)
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 || args[0] == "" {
		c.Ui.Error(
			"The output command expects exactly one argument with the name\n" +
				"of an output variable.\n")
		cmdFlags.Usage()
		return 1
	}
	name := args[0]

	stateStore, err := c.Meta.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading state: %s", err))
		return 1
	}

	state := stateStore.State()
	if len(state.RootModule().Outputs) == 0 {
		c.Ui.Error(fmt.Sprintf(
			"The state file has no outputs defined. Define an output\n" +
				"in your configuration with the `output` directive and re-run\n" +
				"`terraform apply` for it to become available."))
		return 1
	}
	v, ok := state.RootModule().Outputs[name]
	if !ok {
		c.Ui.Error(fmt.Sprintf(
			"The output variable requested could not be found in the state\n" +
				"file. If you recently added this to your configuration, be\n" +
				"sure to run `terraform apply`, since the state won't be updated\n" +
				"with new output variables until that command is run."))
		return 1
	}

	c.Ui.Output(v)
	return 0
}

func (c *OutputCommand) Help() string {
	helpText := `
Usage: terraform output [options] NAME

  Reads an output variable from a Terraform state file and prints
  the value.

Options:

  -state=path      Path to the state file to read. Defaults to
                   "terraform.tfstate".

`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Read an output from a state file"
}
