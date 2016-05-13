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

	var module, remoteBackend string
	var remoteState bool
	var backendConfig map[string]string
	cmdFlags := flag.NewFlagSet("output", flag.ContinueOnError)
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.BoolVar(&remoteState, "remote", false, "remote")
	cmdFlags.StringVar(&remoteBackend, "remote-backend", "atlas", "remote-backend")
	cmdFlags.Var((*FlagKV)(&backendConfig), "remote-config", "remote-config")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	name, index, err := parseOutputNameIndex(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	stateStore, err := c.Meta.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading state: %s", err))
		return 1
	}
	mod, err := moduleFromState(stateStore.State(), module)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var out string
	if name != "" {
		out, err = singleOutputAsString(mod, name, index)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	} else {
		out = allOutputsAsString(mod, nil, false)
	}

	c.Ui.Output(out)

	return 0
}

func (c *OutputCommand) Help() string {
	helpText := `
Usage: terraform output [options] [NAME]

  Reads an output variable from a Terraform state file, or remote state,
  and prints the value. If NAME is not specified, all outputs are printed.

Options:

  -state=path            Path to the state file to read. Defaults to
                         "terraform.tfstate".

  -no-color              If specified, output won't contain any color.

  -module=name           If specified, returns the outputs for a
                         specific module.

`
	return strings.TrimSpace(helpText)
}

func (c *OutputCommand) Synopsis() string {
	return "Read an output from a state file"
}
