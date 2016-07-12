package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// RemoteOutputCommand is a Command implementation that is used to
// read a Terraform remote state.
type RemoteOutputCommand struct {
	Meta
}

// Run runs the terraform remote output command.
func (c *RemoteOutputCommand) Run(args []string) int {
	config := make(map[string]string)
	var module, backend string

	args = c.Meta.process(args, false)
	cmdFlags := flag.NewFlagSet("remote output", flag.ContinueOnError)
	cmdFlags.StringVar(&backend, "backend", "atlas", "backend")
	cmdFlags.Var((*FlagKV)(&config), "backend-config", "config")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("\nError parsing CLI flags: %s", err))
		return 1
	}

	name, index, err := parseOutputNameIndex(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Lowercase the type
	backend = strings.ToLower(backend)

	state, err := getState(backend, config)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	mod, err := moduleFromState(state, module)
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

// getState loads a Terraform state, provided a backend and config.
func getState(backend string, config map[string]string) (*terraform.State, error) {
	client, err := remote.NewClient(backend, config)
	if err != nil {
		return nil, err
	}

	s := &remote.State{Client: client}
	if err := s.RefreshState(); err != nil {
		return nil, err
	}

	return s.State(), nil
}

// Help displays the help text for the terraform remote output command.
func (c *RemoteOutputCommand) Help() string {
	helpText := `
Usage: terraform remote output [options] [NAME]

  Reads an output variable from the specified Terraform remote state. Does
  not read or alter your existing configruation, and can be used without
  any remote state configured.
  
  If NAME is not specified, all outputs are printed.

Options:

  -backend=Atlas         Specifies the type of remote backend. See
                         "terraform remote config -help" for a list of
                         supported backends. Defaults to Atlas.

  -config="k=v"          Specifies configuration for the remote storage
                         backend. This can be specified multiple times.

  -no-color              If specified, output won't contain any color.

  -module=name           If specified, returns the outputs for a
                         specific module.

`
	return strings.TrimSpace(helpText)
}

// Synopsis displays the synopsis text for the terraform remote output command.
func (c *RemoteOutputCommand) Synopsis() string {
	return "Reads a Terraform remote state"
}
