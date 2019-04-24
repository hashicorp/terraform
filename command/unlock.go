package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/states/statemgr"

	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
)

// UnlockCommand is a cli.Command implementation that manually unlocks
// the state.
type UnlockCommand struct {
	Meta
}

func (c *UnlockCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var force bool
	cmdFlags := c.Meta.defaultFlagSet("force-unlock")
	cmdFlags.BoolVar(&force, "force", false, "force")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) == 0 {
		c.Ui.Error("unlock requires a lock id argument")
		return cli.RunResultHelp
	}

	lockID := args[0]
	args = args[1:]

	// assume everything is initialized. The user can manually init if this is
	// required.
	configPath, err := ModulePath(args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	env := c.Workspace()
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	_, isLocal := stateMgr.(*statemgr.Filesystem)

	if !force {
		// Forcing this doesn't do anything, but doesn't break anything either,
		// and allows us to run the basic command test too.
		if isLocal {
			c.Ui.Error("Local state cannot be unlocked by another process")
			return 1
		}

		desc := "Terraform will remove the lock on the remote state.\n" +
			"This will allow local Terraform commands to modify this state, even though it\n" +
			"may be still be in use. Only 'yes' will be accepted to confirm."

		v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
			Id:          "force-unlock",
			Query:       "Do you really want to force-unlock?",
			Description: desc,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error asking for confirmation: %s", err))
			return 1
		}
		if v != "yes" {
			c.Ui.Output("force-unlock cancelled.")
			return 1
		}
	}

	if err := stateMgr.Unlock(lockID); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to unlock state: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(strings.TrimSpace(outputUnlockSuccess)))
	return 0
}

func (c *UnlockCommand) Help() string {
	helpText := `
Usage: terraform force-unlock LOCK_ID [DIR]

  Manually unlock the state for the defined configuration.

  This will not modify your infrastructure. This command removes the lock on the
  state for the current configuration. The behavior of this lock is dependent
  on the backend being used. Local state files cannot be unlocked by another
  process.

Options:

  -force                 Don't ask for input for unlock confirmation.
`
	return strings.TrimSpace(helpText)
}

func (c *UnlockCommand) Synopsis() string {
	return "Manually unlock the terraform state"
}

const outputUnlockSuccess = `
[reset][bold][green]Terraform state has been successfully unlocked![reset][green]

The state has been unlocked, and Terraform commands should now be able to
obtain a new lock on the remote state.
`
