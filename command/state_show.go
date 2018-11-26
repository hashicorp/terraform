package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/states"
	"github.com/mitchellh/cli"
)

// StateShowCommand is a Command implementation that shows a single resource.
type StateShowCommand struct {
	Meta
	StateMeta
}

func (c *StateShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("state show")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("Exactly one argument expected.\n")
		return cli.RunResultHelp
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Check if the address can be parsed
	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	if addrDiags.HasErrors() {
		c.Ui.Error(fmt.Sprintf(errParsingAddress, args[0]))
		return 1
	}

	// We expect the config dir to always be the cwd
	cwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting cwd: %s", err))
		return 1
	}

	// Build the operation (required to get the schemas)
	opReq := c.Operation(b)
	opReq.ConfigDir = cwd

	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing config loader: %s", err))
		return 1
	}

	// Get the context (required to get the schemas)
	ctx, _, ctxDiags := local.Context(opReq)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(ctxDiags)
		return 1
	}

	// Get the schemas from the context
	schemas := ctx.Schemas()

	// Get the state
	env := c.Workspace()
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	is := state.ResourceInstance(addr)
	if !is.HasCurrent() {
		c.Ui.Error(errNoInstanceFound)
		return 1
	}

	singleInstance := states.NewState()
	singleInstance.EnsureModule(addr.Module).SetResourceInstanceCurrent(
		addr.Resource,
		is.Current,
		addr.Resource.Resource.DefaultProviderConfig().Absolute(addr.Module),
	)

	output := format.State(&format.StateOpts{
		State:   singleInstance,
		Color:   c.Colorize(),
		Schemas: schemas,
	})
	c.Ui.Output(output[strings.Index(output, "#"):])

	return 0
}

func (c *StateShowCommand) Help() string {
	helpText := `
Usage: terraform state show [options] ADDRESS

  Shows the attributes of a resource in the Terraform state.

  This command shows the attributes of a single resource in the Terraform
  state. The address argument must be used to specify a single resource.
  You can view the list of available resources with "terraform state list".

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

`
	return strings.TrimSpace(helpText)
}

func (c *StateShowCommand) Synopsis() string {
	return "Show a resource in the state"
}

const errNoInstanceFound = `No instance found for the given address!

This command requires that the address references one specific instance.
To view the available instances, use "terraform state list". Please modify 
the address to reference a specific instance.`

const errParsingAddress = `Error parsing instance address: %s

This command requires that the address references one specific instance.
To view the available instances, use "terraform state list". Please modify 
the address to reference a specific instance.`
