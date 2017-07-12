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
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	// We create two metas to track the two states
	var meta1, meta2 Meta
	cmdFlags := c.Meta.flagSet("state mv")
	cmdFlags.StringVar(&meta1.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&meta1.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&meta2.backupPath, "backup-out", "-", "backup")
	cmdFlags.StringVar(&meta2.statePath, "state-out", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("Exactly two arguments expected.\n")
		return cli.RunResultHelp
	}

	// Copy the `-state` flag for output if we weren't given a custom one
	if meta2.statePath == "" {
		meta2.statePath = meta1.statePath
	}

	// Read the from state
	stateFrom, err := c.StateMeta.State(&meta1)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return cli.RunResultHelp
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
	if meta2.statePath != meta1.statePath {
		stateTo, err = c.StateMeta.State(&meta2)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
			return cli.RunResultHelp
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
Usage: terraform state mv [options] ADDRESS ADDRESS

  Move an item in the state to another location or to a completely different
  state file.

  This command is useful for module refactors (moving items into a module),
  configuration refactors (moving items to a completely different or new
  state file), or generally renaming of resources.

  This command creates a timestamped backup of the state on every invocation.
  This can't be disabled. Due to the destructive nature of this command,
  the backup is ensured by Terraform for safety reasons.

  If you're moving from one state file to a different state file, a backup
  will be created for each state file.

Options:

  -backup=PATH        Path where Terraform should write the backup for the original
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the statefile with
                      a backup extension.

  -backup-out=PATH    Path where Terraform should write the backup for the destination
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the destination state
                      file with a backup extension. This only needs
                      to be specified if -state-out is set to a different path
                      than -state.

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
