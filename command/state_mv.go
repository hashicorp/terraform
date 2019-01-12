package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/states"
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

	var dryRun bool
	cmdFlags := c.Meta.defaultFlagSet("state mv")
	cmdFlags.BoolVar(&dryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&backupPathOut, "backup-out", "-", "backup")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
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
	stateFromMgr, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(stateFromMgr, "state-mv"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking source state: %s", err))
			return 1
		}
		defer stateLocker.Unlock(nil)
	}

	if err := stateFromMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh source state: %s", err))
		return 1
	}

	stateFrom := stateFromMgr.State()
	if stateFrom == nil {
		c.Ui.Error(fmt.Sprintf(errStateNotFound))
		return 1
	}

	// Read the destination state
	stateToMgr := stateFromMgr
	stateTo := stateFrom

	if statePathOut != "" {
		c.statePath = statePathOut
		c.backupPath = backupPathOut

		stateToMgr, err = c.State()
		if err != nil {
			c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
			return 1
		}

		if c.stateLock {
			stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
			if err := stateLocker.Lock(stateToMgr, "state-mv"); err != nil {
				c.Ui.Error(fmt.Sprintf("Error locking destination state: %s", err))
				return 1
			}
			defer stateLocker.Unlock(nil)
		}

		if err := stateToMgr.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to refresh destination state: %s", err))
			return 1
		}

		stateTo = stateToMgr.State()
		if stateTo == nil {
			stateTo = states.NewState()
		}
	}

	// Filter what we are moving.
	results, err := c.filter(stateFrom, []string{args[0]})
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateFilter, err))
		return cli.RunResultHelp
	}

	// If we have no results, exit early as we're not going to do anything.
	if len(results) == 0 {
		if dryRun {
			c.Ui.Output("Would have moved nothing.")
		} else {
			c.Ui.Output("No matching objects found.")
		}
		return 0
	}

	prefix := "Move"
	if dryRun {
		prefix = "Would move"
	}

	var moved int
	ssFrom := stateFrom.SyncWrapper()
	for _, result := range c.moveableResult(results) {
		switch addrFrom := result.Address.(type) {
		case addrs.ModuleInstance:
			search, err := addrs.ParseModuleInstanceStr(args[0])
			if err != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, err))
				return 1
			}
			addrTo, err := addrs.ParseModuleInstanceStr(args[1])
			if err != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, err))
				return 1
			}

			if len(search) < len(addrFrom) {
				addrTo = append(addrTo, addrFrom[len(search):]...)
			}

			if stateTo.Module(addrTo) != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, "destination module already exists"))
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), addrTo.String()))
			if !dryRun {
				ssFrom.RemoveModule(addrFrom)

				// Update the address before adding it to the state.
				m := result.Value.(*states.Module)
				m.Addr = addrTo
				stateTo.Modules[addrTo.String()] = m
			}

		case addrs.AbsResource:
			addrTo, err := addrs.ParseAbsResourceStr(args[1])
			if err != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, err))
				return 1
			}

			if addrFrom.Resource.Type != addrTo.Resource.Type {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "resource types do not match"))
				return 1
			}
			if stateTo.Module(addrTo.Module) == nil {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "destination module does not exist"))
				return 1
			}
			if stateTo.Resource(addrTo) != nil {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "destination resource already exists"))
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), addrTo.String()))
			if !dryRun {
				ssFrom.RemoveResource(addrFrom)

				// Update the address before adding it to the state.
				rs := result.Value.(*states.Resource)
				rs.Addr = addrTo.Resource
				stateTo.Module(addrTo.Module).Resources[addrTo.Resource.String()] = rs
			}

		case addrs.AbsResourceInstance:
			addrTo, err := addrs.ParseAbsResourceInstanceStr(args[1])
			if err != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, err))
				return 1
			}

			if stateTo.Module(addrTo.Module) == nil {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "destination module does not exist"))
				return 1
			}
			if stateTo.Resource(addrTo.ContainingResource()) == nil {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "destination resource does not exist"))
				return 1
			}
			if stateTo.ResourceInstance(addrTo) != nil {
				c.Ui.Error(fmt.Sprintf(
					errStateMv, "destination resource instance already exists"))
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), args[1]))
			if !dryRun {
				ssFrom.ForgetResourceInstanceAll(addrFrom)
				ssFrom.RemoveResourceIfEmpty(addrFrom.ContainingResource())

				rs := stateTo.Resource(addrTo.ContainingResource())
				rs.Instances[addrTo.Resource.Key] = result.Value.(*states.ResourceInstance)
			}
		}
	}

	if dryRun {
		if moved == 0 {
			c.Ui.Output("Would have moved nothing.")
		}
		return 0 // This is as far as we go in dry-run mode
	}

	// Write the new state
	if err := stateToMgr.WriteState(stateTo); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}
	if err := stateToMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	// Write the old state if it is different
	if stateTo != stateFrom {
		if err := stateFromMgr.WriteState(stateFrom); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
			return 1
		}
		if err := stateFromMgr.PersistState(); err != nil {
			c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
			return 1
		}
	}

	if moved == 0 {
		c.Ui.Output("No matching objects found.")
	} else {
		c.Ui.Output(fmt.Sprintf("Successfully moved %d object(s).", moved))
	}
	return 0
}

// moveableResult takes the result from a filter operation and returns what
// object(s) to move. The reason we do this is because in the module case
// we must add the list of all modules returned versus just the root module.
func (c *StateMvCommand) moveableResult(results []*states.FilterResult) []*states.FilterResult {
	result := results[:1]

	if len(results) > 1 {
		// If a state module then we should add the full list of modules.
		if _, ok := result[0].Address.(addrs.ModuleInstance); ok {
			for _, r := range results[1:] {
				if _, ok := r.Address.(addrs.ModuleInstance); ok {
					result = append(result, r)
				}
			}
		}
	}

	return result
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

  -dry-run            If set, prints out what would've been moved but doesn't
                      actually move anything.

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

  -lock=true          Lock the state files when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

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

const errStateMv = `Error moving state: %s

Please ensure your addresses and state paths are valid. No
state was persisted. Your existing states are untouched.`

const errStateMvPersist = `Error saving the state: %s

The state wasn't saved properly. If the error happening after a partial
write occurred, a backup file will have been created. Otherwise, the state
is in the same state it was when the operation started.`
