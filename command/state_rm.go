package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
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

	cmdFlags := c.Meta.flagSet("state show")
	var dryRun bool
	cmdFlags.BoolVar(&dryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	var diags tfdiags.Diagnostics

	if len(args) < 1 {
		c.Ui.Error("At least one resource address is required.")
		return 1
	}

	stateMgr, err := c.State()
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

	toRemove := make([]addrs.AbsResourceInstance, len(args))
	for i, rawAddr := range args {
		addr, moreDiags := addrs.ParseAbsResourceInstanceStr(rawAddr)
		diags = diags.Append(moreDiags)
		toRemove[i] = addr
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We will first check that all of the instances are present, so we can
	// either remove all of them successfully or make no change at all.
	// (If we're in dry run mode, this is also where we print out what
	// we would've done.)
	var currentCount, deposedCount int
	var dryRunBuf bytes.Buffer
	for _, addr := range toRemove {
		is := state.ResourceInstance(addr)
		if is == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance in state",
				fmt.Sprintf("There is no resource instance in the current state with the address %s.", addr),
			))
			continue
		}
		if is.Current != nil {
			currentCount++
		}
		deposedCount += len(is.Deposed)
		if dryRun {
			if is.Current != nil {
				fmt.Fprintf(&dryRunBuf, "Would remove %s\n", addr)
			}
			for k := range is.Deposed {
				fmt.Fprintf(&dryRunBuf, "Would remove %s deposed object %s\n", addr, k)
			}
		}
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	if dryRun {
		c.Ui.Output(fmt.Sprintf("%s\nWould've removed %d current and %d deposed objects, without -dry-run.", dryRunBuf.String(), currentCount, deposedCount))
		return 0 // This is as far as we go in dry-run mode
	}

	// Now we will actually remove them. Due to our validation above, we should
	// succeed in removing every one.
	// We'll use the "SyncState" wrapper to do this not because we're doing
	// any concurrent work here (we aren't) but because it guarantees to clean
	// up any leftover empty module we might leave behind.
	ss := state.SyncWrapper()
	for _, addr := range toRemove {
		ss.ForgetResourceInstanceAll(addr)
	}

	switch {
	case currentCount == 0:
		c.Ui.Output(fmt.Sprintf("Removed %d deposed objects.", deposedCount))
	case deposedCount == 0:
		c.Ui.Output(fmt.Sprintf("Removed %d objects.", currentCount))
	default:
		c.Ui.Output(fmt.Sprintf("Removed %d current and %d deposed objects.", currentCount, deposedCount))
	}

	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	if err := stateMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	c.Ui.Output("Updated state written successfully.")
	return 0
}

func (c *StateRmCommand) Help() string {
	helpText := `
Usage: terraform state rm [options] ADDRESS...

  Remove one or more items from the Terraform state.

  This command removes one or more resource instances from the Terraform state
  based on the addresses given. You can view and list the available instances
  with "terraform state list".

  This command creates a timestamped backup of the state on every invocation.
  This can't be disabled. Due to the destructive nature of this command,
  the backup is ensured by Terraform for safety reasons.

Options:

  -dry-run            If set, prints out what would've been removed but
                      doesn't actually remove anything.

  -backup=PATH        Path where Terraform should write the backup
                      state. This can't be disabled. If not set, Terraform
                      will write it to the same path as the statefile with
                      a backup extension.

  -state=PATH         Path to the source state file. Defaults to the configured
                      backend, or "terraform.tfstate"

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
