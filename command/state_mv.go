package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// StateMvCommand is a Command implementation that shows a single resource.
type StateMvCommand struct {
	Meta
	StateMeta
}

func (c *StateMvCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	var backupPath string
	cmdFlags := c.Meta.flagSet("state show")
	cmdFlags.StringVar(&backupPath, "backup", "", "backup")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("Exactly two arguments expected.\n")
		return cli.RunResultHelp
	}

	state, err := c.StateMeta.State(&c.Meta)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return cli.RunResultHelp
	}

	stateReal := state.State()
	if stateReal == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	filter := &terraform.StateFilter{State: stateReal}
	results, err := filter.Filter(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return cli.RunResultHelp
	}

	if err := stateReal.Remove(args[0]); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return 1
	}

	if err := stateReal.Add(args[0], args[1], results[0].Value); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return 1
	}

	if err := state.WriteState(stateReal); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
		return 1
	}

	if err := state.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf(
		"Moved %s to %s", args[0], args[1]))
	return 0
}

func (c *StateMvCommand) Help() string {
	helpText := `
Usage: terraform state mv [options] ADDRESS ADDRESS

  Move an item in the state to another location within the same state.

  This command is useful for module refactors (moving items into a module)
  or generally renaming of resources.

  This command creates a timestamped backup of the state on every invocation.
  This can't be disabled. Due to the destructive nature of this command,
  the backup is ensured by Terraform for safety reasons.

  This command can't currently move an item from one state file to a
  completely new state file, but this functionality will come in an update.

Options:

  -backup=PATH        Path where Terraform should write the backup
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the statefile with
                      a backup extension.

  -state=PATH         Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

  -state-out=PATH     Path to the destination state file to move the item
                      to. This defaults to the same statefile. This will
                      overwrite the destination state file.

`
	return strings.TrimSpace(helpText)
}

func (c *StateMvCommand) Synopsis() string {
	return "Move an item in the state"
}

const errStateMv = `Error moving state: %[1]s

Please ensure your addresses and state paths are valid. No
state was persisted. Your existing states are untouched.`

const errStateMvPersist = `Error saving the state: %s

The state wasn't saved properly. If the error happening after a partial
write occurred, a backup file will have been created. Otherwise, the state
is in the same state it was when the operation started.`
