package command

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
)

// StateRmCommand is a Command implementation that shows a single resource.
type StateRmCommand struct {
	Meta
	StateMeta
}

func (c *StateRmCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("state show")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	if len(args) < 1 {
		c.Ui.Error("At least one resource address is required.")
		return 1
	}

	state, err := c.StateMeta.State(&c.Meta)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return cli.RunResultHelp
	}
	if err := state.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	stateReal := state.State()
	if stateReal == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	if err := stateReal.Remove(args...); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRm, err))
		return 1
	}

	if err := state.WriteState(stateReal); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	if err := state.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	c.Ui.Output("Item removal successful.")
	return 0
}

func (c *StateRmCommand) Help() string {
	helpText := `
Usage: terraform state rm [options] ADDRESS...

  Remove one or more items from the Terraform state.

  This command removes one or more items from the Terraform state based
  on the address given. You can view and list the available resources
  with "terraform state list".

  This command creates a timestamped backup of the state on every invocation.
  This can't be disabled. Due to the destructive nature of this command,
  the backup is ensured by Terraform for safety reasons.

Options:

  -backup=PATH        Path where Terraform should write the backup
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the statefile with
                      a backup extension.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

`
	return strings.TrimSpace(helpText)
}

func (c *StateRmCommand) Synopsis() string {
	return "Remove an item from the state"
}

const errStateRm = `Error removing items from the state: %s

The state was not saved. No items were removed from the persisted
state. No backup was created since no modification occurred. Please
resolve the issue above and try again.`

const errStateRmPersist = `Error saving the state: %s

The state was not saved. No items were removed from the persisted
state. No backup was created since no modification occurred. Please
resolve the issue above and try again.`
