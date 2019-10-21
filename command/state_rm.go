package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
)

// StateRmCommand is a Command implementation that shows a single resource.
type StateRmCommand struct {
	StateMeta
}

func (c *StateRmCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	var dryRun bool
	cmdFlags := c.Meta.defaultFlagSet("state rm")
	cmdFlags.BoolVar(&dryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) < 1 {
		c.Ui.Error("At least one address is required.\n")
		return cli.RunResultHelp
	}

	// Get the state
	stateMgr, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(stateMgr, "state-rm"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
		defer stateLocker.Unlock(nil)
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

	// This command primarily works with resource instances, though it will
	// also clean up any modules and resources left empty by actions it takes.
	var addrs []addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics
	for _, addrStr := range args {
		moreAddrs, moreDiags := c.lookupResourceInstanceAddr(state, true, addrStr)
		addrs = append(addrs, moreAddrs...)
		diags = diags.Append(moreDiags)
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	prefix := "Removed "
	if dryRun {
		prefix = "Would remove "
	}

	var isCount int
	ss := state.SyncWrapper()
	for _, addr := range addrs {
		isCount++
		c.Ui.Output(prefix + addr.String())
		if !dryRun {
			ss.ForgetResourceInstanceAll(addr)
			ss.RemoveResourceIfEmpty(addr.ContainingResource())
		}
	}

	if dryRun {
		if isCount == 0 {
			c.Ui.Output("Would have removed nothing.")
		}
		return 0 // This is as far as we go in dry-run mode
	}

	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}
	if err := stateMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	if len(diags) > 0 {
		c.showDiagnostics(diags)
	}

	if isCount == 0 {
		c.Ui.Output("No matching resource instances found.")
	} else {
		c.Ui.Output(fmt.Sprintf("Successfully removed %d resource instance(s).", isCount))
	}
	return 0
}

func (c *StateRmCommand) Help() string {
	helpText := `
Usage: terraform state rm [options] ADDRESS...

  Remove one or more items from the Terraform state, causing Terraform to
  "forget" those items without first destroying them in the remote system.

  This command removes one or more resource instances from the Terraform state
  based on the addresses given. You can view and list the available instances
  with "terraform state list".

  If you give the address of an entire module then all of the instances in
  that module and any of its child modules will be removed from the state.

  If you give the address of a resource that has "count" or "for_each" set,
  all of the instances of that resource will be removed from the state.

Options:

  -dry-run            If set, prints out what would've been removed but
                      doesn't actually remove anything.

  -backup=PATH        Path where Terraform should write the backup
                      state.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -state=PATH         Path to the state file to update. Defaults to the current
                      workspace state.

`
	return strings.TrimSpace(helpText)
}

func (c *StateRmCommand) Synopsis() string {
	return "Remove instances from the state"
}

const errStateRm = `Error removing items from the state: %s

The state was not saved. No items were removed from the persisted
state. No backup was created since no modification occurred. Please
resolve the issue above and try again.`

const errStateRmPersist = `Error saving the state: %s

The state was not saved. No items were removed from the persisted
state. No backup was created since no modification occurred. Please
resolve the issue above and try again.`
