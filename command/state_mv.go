package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// StateMvCommand is a Command implementation that shows a single resource.
type StateMvCommand struct {
	StateMeta
}

func (c *StateMvCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	// We create two metas to track the two states
	var backupPathOut, statePathOut string

	cmdFlags := c.Meta.flagSet("state mv")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	cmdFlags.StringVar(&backupPathOut, "backup-out", "-", "backup")
	cmdFlags.StringVar(&statePathOut, "state-out", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("Exactly two arguments expected.\n")
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

	// Filter what we're moving
	filter := &terraform.StateFilter{State: stateFromReal}
	results, err := filter.Filter(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return cli.RunResultHelp
	}
	if len(results) == 0 {
		c.Ui.Output(fmt.Sprintf("Item to move doesn't exist: %s", args[0]))
		return 1
	}

	// Get the item to add to the state
	add := c.addableResult(results)

	// Do the actual move
	if err := stateFromReal.Remove(args[0]); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return 1
	}

	if err := stateToReal.Add(args[0], args[1], add); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMv, err))
		return 1
	}

	// Write the new state
	if err := stateTo.WriteState(stateToReal); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
		return 1
	}

	if err := stateTo.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
		return 1
	}

	// Write the old state if it is different
	if stateTo != stateFrom {
		if err := stateFrom.WriteState(stateFromReal); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
			return 1
		}

		if err := stateFrom.PersistState(); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateMvPersist, err))
			return 1
		}
	}

	c.Ui.Output(fmt.Sprintf(
		"Moved %s to %s", args[0], args[1]))
	return 0
}

// addableResult takes the result from a filter operation and returns what to
// call State.Add with. The reason we do this is because in the module case
// we must add the list of all modules returned versus just the root module.
func (c *StateMvCommand) addableResult(results []*terraform.StateFilterResult) interface{} {
	switch v := results[0].Value.(type) {
	case *terraform.ModuleState:
		// If a module state then we should add the full list of modules
		result := []*terraform.ModuleState{v}
		if len(results) > 1 {
			for _, r := range results[1:] {
				if ms, ok := r.Value.(*terraform.ModuleState); ok {
					result = append(result, ms)
				}
			}
		}

		return result

	case *terraform.ResourceState:
		// If a resource state with more than one result, it has a multi-count
		// and we need to add all of them.
		result := []*terraform.ResourceState{v}
		if len(results) > 1 {
			for _, r := range results[1:] {
				rs, ok := r.Value.(*terraform.ResourceState)
				if !ok {
					continue
				}

				if rs.Type == v.Type {
					result = append(result, rs)
				}
			}
		}

		// If we only have one item, add it directly
		if len(result) == 1 {
			return result[0]
		}

		return result

	default:
		// By default just add the first result
		return v
	}
}

func (c *StateMvCommand) Help() string {
	helpText := `
Usage: terraform state mv [options] SOURCE DESTINATION

 This command will move an item matched by the address given to the
 destination address. This command can also move to a destination address
 in a completely different state file.

 This can be used for simple resource renaming, moving items to and from
 a module, moving entire modules, and more. And because this command can also
 move data to a completely new state, it can also be used for refactoring
 one configuration into multiple separately managed Terraform configurations.

 This command will output a backup copy of the state prior to saving any
 changes. The backup cannot be disabled. Due to the destructive nature
 of this command, backups are required.

 If you're moving an item to a different state file, a backup will be created
 for each state file.

Options:

  -backup=PATH        Path where Terraform should write the backup for the original
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the statefile with
                      a ".backup" extension.

  -backup-out=PATH    Path where Terraform should write the backup for the destination
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the destination state
                      file with a backup extension. This only needs
                      to be specified if -state-out is set to a different path
                      than -state.

  -state=PATH         Path to the source state file. Defaults to the configured
                      backend, or "terraform.tfstate"

  -state-out=PATH     Path to the destination state file to write to. If this
                      isn't specified, the source state file will be used. This
                      can be a new or existing path.

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
