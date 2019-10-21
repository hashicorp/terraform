package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
)

// StateListCommand is a Command implementation that lists the resources
// within a state file.
type StateListCommand struct {
	Meta
	StateMeta
}

func (c *StateListCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	var statePath string
	cmdFlags := c.Meta.defaultFlagSet("state list")
	cmdFlags.StringVar(&statePath, "state", "", "path")
	lookupId := cmdFlags.String("id", "", "Restrict output to paths with a resource having the specified ID.")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	if statePath != "" {
		c.Meta.statePath = statePath
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// Get the state
	env := c.Workspace()
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	var addrs []addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics
	if len(args) == 0 {
		addrs, diags = c.lookupAllResourceInstanceAddrs(state)
	} else {
		addrs, diags = c.lookupResourceInstanceAddrs(state, args...)
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	for _, addr := range addrs {
		if is := state.ResourceInstance(addr); is != nil {
			if *lookupId == "" || *lookupId == states.LegacyInstanceObjectID(is.Current) {
				c.Ui.Output(addr.String())
			}
		}
	}

	c.showDiagnostics(diags)

	return 0
}

func (c *StateListCommand) Help() string {
	helpText := `
Usage: terraform state list [options] [address...]

  List resources in the Terraform state.

  This command lists resource instances in the Terraform state. The address
  argument can be used to filter the instances by resource or module. If
  no pattern is given, all resource instances are listed.

  The addresses must either be module addresses or absolute resource
  addresses, such as:
      aws_instance.example
      module.example
      module.example.module.child
      module.example.aws_instance.example

  An error will be returned if any of the resources or modules given as
  filter addresses do not exist in the state.

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default, Terraform
                      will consult the state of the currently-selected
                      workspace.

  -id=ID              Filters the results to include only instances whose
                      resource types have an attribute named "id" whose value
                      equals the given id string.

`
	return strings.TrimSpace(helpText)
}

func (c *StateListCommand) Synopsis() string {
	return "List resources in the state"
}

const errStateFilter = `Error filtering state: %[1]s

Please ensure that all your addresses are formatted properly.`

const errStateLoadingState = `Error loading the state: %[1]s

Please ensure that your Terraform state exists and that you've
configured it properly. You can use the "-state" flag to point
Terraform at another state file.`

const errStateNotFound = `No state file was found!

State management commands require a state file. Run this command
in a directory where Terraform has been run or use the -state flag
to point the command to a specific state location.`
