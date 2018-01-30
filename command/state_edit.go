package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// StateEditCommand is a Command implementation that edits a
// resource's property within a state file.
type StateEditCommand struct {
	StateMeta
}

func (c *StateEditCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	// We create two metas to track the two states
	var backupPathOut, statePathOut string

	cmdFlags := c.Meta.flagSet("state edit")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	cmdFlags.StringVar(&backupPathOut, "backup-out", "-", "backup")
	cmdFlags.StringVar(&statePathOut, "state-out", "", "path")
	// TODO: cmdFlags.BoolVar(something to control editing more than 1 resource at a time)
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 3 {
		c.Ui.Error("Exactly three arguments expected.\n")
		return cli.RunResultHelp
	}

	// Read the from state
	stateFrom, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	if err := stateFrom.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	stateFromReal := stateFrom.State()
	if stateFromReal == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	// Read the destination state
	stateTo := stateFrom
	stateToReal := stateFromReal

	if statePathOut != "" {
		c.statePath = statePathOut
		c.backupPath = backupPathOut
		stateTo, err = c.State()
		if err != nil {
			c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
			return 1
		}

		if err := stateTo.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
			return 1
		}

		stateToReal = stateTo.State()
		if stateToReal == nil {
			stateToReal = terraform.NewState()
		}
	}

	// Filter what we're going to edit
	filter := &terraform.StateFilter{State: stateFromReal}
	results, err := filter.Filter(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateEdit, err))
		return cli.RunResultHelp
	}
	if len(results) == 0 {
		c.Ui.Output(fmt.Sprintf("Item to move doesn't exist: %s", args[0]))
		return 1
	}

	instance, err := c.filterInstance(results)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if instance == nil {
		return 0
	}

	is := instance.Value.(*terraform.InstanceState)

	// Edit the attribute
	key := args[1]
	value := args[2]
	if key == "id" {
		is.ID = value
	}
	is.Attributes[key] = value

	// Write the new state
	if err := stateTo.WriteState(stateToReal); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateEditPersist, err))
		return 1
	}

	if err := stateTo.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateEditPersist, err))
		return 1
	}

	// Write the old state if it is different
	if stateTo != stateFrom {
		if err := stateFrom.WriteState(stateFromReal); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateEditPersist, err))
			return 1
		}

		if err := stateFrom.PersistState(); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateEditPersist, err))
			return 1
		}
	}

	c.Ui.Output(fmt.Sprintf(
		"Set %s[%s] to %s", args[0], args[1], args[2]))
	return 0

}

func (c *StateEditCommand) Help() string {
	helpText := `
Usage: terraform state edit [options] PATH PROPERTY VALUE

bla bla
`
	return strings.TrimSpace(helpText)
}

func (c *StateEditCommand) Synopsis() string {
	return "Edits the property of an item in the state"
}

const errStateEdit = `Error editing state: %[1]s

Please ensure your state path and property name are valid. No state
was persisted. Your existing states are untouched.`

const errStateEditPersist = `Error saving the state: %s

The state wasn't saved properly. If the error happening after a partial
write occurred, a backup file will have been created. Otherwise, the state
is in the same state it was when the operation started.`
