package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/cli"
)

// StateMvCommand is a Command implementation that shows a single resource.
type StateMvCommand struct {
	StateMeta
}

func (c *StateMvCommand) Run(args []string) int {
	args = c.Meta.process(args)
	// We create two metas to track the two states
	var backupPathOut, statePathOut string

	var dryRun bool
	cmdFlags := c.Meta.ignoreRemoteVersionFlagSet("state mv")
	cmdFlags.BoolVar(&dryRun, "dry-run", false, "dry run")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.StringVar(&backupPathOut, "backup-out", "-", "backup")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	cmdFlags.StringVar(&statePathOut, "state-out", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
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
		stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
		if diags := stateLocker.Lock(stateFromMgr, "state-mv"); diags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
		defer func() {
			if diags := stateLocker.Unlock(); diags.HasErrors() {
				c.showDiagnostics(diags)
			}
		}()
	}

	if err := stateFromMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh source state: %s", err))
		return 1
	}

	stateFrom := stateFromMgr.State()
	if stateFrom == nil {
		c.Ui.Error(errStateNotFound)
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
			stateLocker := clistate.NewLocker(c.stateLockTimeout, views.NewStateLocker(arguments.ViewHuman, c.View))
			if diags := stateLocker.Lock(stateToMgr, "state-mv"); diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}
			defer func() {
				if diags := stateLocker.Unlock(); diags.HasErrors() {
					c.showDiagnostics(diags)
				}
			}()
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

	var diags tfdiags.Diagnostics
	sourceAddr, moreDiags := c.lookupSingleStateObjectAddr(stateFrom, args[0])
	diags = diags.Append(moreDiags)
	destAddr, moreDiags := c.lookupSingleStateObjectAddr(stateFrom, args[1])
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	prefix := "Move"
	if dryRun {
		prefix = "Would move"
	}

	const msgInvalidSource = "Invalid source address"
	const msgInvalidTarget = "Invalid target address"

	var moved int
	ssFrom := stateFrom.SyncWrapper()
	sourceAddrs := c.sourceObjectAddrs(stateFrom, sourceAddr)
	if len(sourceAddrs) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			msgInvalidSource,
			fmt.Sprintf("Cannot move %s: does not match anything in the current state.", sourceAddr),
		))
	}
	for _, rawAddrFrom := range sourceAddrs {
		switch addrFrom := rawAddrFrom.(type) {
		case addrs.ModuleInstance:
			search := sourceAddr.(addrs.ModuleInstance)
			addrTo, ok := destAddr.(addrs.ModuleInstance)
			if !ok {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidTarget,
					fmt.Sprintf("Cannot move %s to %s: the target must also be a module.", addrFrom, destAddr),
				))
				c.showDiagnostics(diags)
				return 1
			}

			if len(search) < len(addrFrom) {
				n := make(addrs.ModuleInstance, 0, len(addrTo)+len(addrFrom)-len(search))
				n = append(n, addrTo...)
				n = append(n, addrFrom[len(search):]...)
				addrTo = n
			}

			if stateTo.Module(addrTo) != nil {
				c.Ui.Error(fmt.Sprintf(errStateMv, "destination module already exists"))
				return 1
			}

			ms := ssFrom.Module(addrFrom)
			if ms == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidSource,
					fmt.Sprintf("The current state does not contain %s.", addrFrom),
				))
				c.showDiagnostics(diags)
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), addrTo.String()))
			if !dryRun {
				ssFrom.RemoveModule(addrFrom)

				// Update the address before adding it to the state.
				ms.Addr = addrTo
				stateTo.Modules[addrTo.String()] = ms
			}

		case addrs.AbsResource:
			addrTo, ok := destAddr.(addrs.AbsResource)
			if !ok {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidTarget,
					fmt.Sprintf("Cannot move %s to %s: the source is a whole resource (not a resource instance) so the target must also be a whole resource.", addrFrom, destAddr),
				))
				c.showDiagnostics(diags)
				return 1
			}
			diags = diags.Append(c.validateResourceMove(addrFrom, addrTo))

			if stateTo.Resource(addrTo) != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidTarget,
					fmt.Sprintf("Cannot move to %s: there is already a resource at that address in the current state.", addrTo),
				))
			}

			rs := ssFrom.Resource(addrFrom)
			if rs == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidSource,
					fmt.Sprintf("The current state does not contain %s.", addrFrom),
				))
			}

			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), addrTo.String()))
			if !dryRun {
				ssFrom.RemoveResource(addrFrom)

				// Update the address before adding it to the state.
				rs.Addr = addrTo
				stateTo.EnsureModule(addrTo.Module).Resources[addrTo.Resource.String()] = rs
			}

		case addrs.AbsResourceInstance:
			addrTo, ok := destAddr.(addrs.AbsResourceInstance)
			if !ok {
				ra, ok := destAddr.(addrs.AbsResource)
				if !ok {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						msgInvalidTarget,
						fmt.Sprintf("Cannot move %s to %s: the target must also be a resource instance.", addrFrom, destAddr),
					))
					c.showDiagnostics(diags)
					return 1
				}
				addrTo = ra.Instance(addrs.NoKey)
			}

			diags = diags.Append(c.validateResourceMove(addrFrom.ContainingResource(), addrTo.ContainingResource()))

			if stateTo.Module(addrTo.Module) == nil {
				// moving something to a mew module, so we need to ensure it exists
				stateTo.EnsureModule(addrTo.Module)
			}
			if stateTo.ResourceInstance(addrTo) != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidTarget,
					fmt.Sprintf("Cannot move to %s: there is already a resource instance at that address in the current state.", addrTo),
				))
			}

			is := ssFrom.ResourceInstance(addrFrom)
			if is == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					msgInvalidSource,
					fmt.Sprintf("The current state does not contain %s.", addrFrom),
				))
			}

			if diags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}

			moved++
			c.Ui.Output(fmt.Sprintf("%s %q to %q", prefix, addrFrom.String(), args[1]))
			if !dryRun {
				fromResourceAddr := addrFrom.ContainingResource()
				fromResource := ssFrom.Resource(fromResourceAddr)
				fromProviderAddr := fromResource.ProviderConfig
				ssFrom.ForgetResourceInstanceAll(addrFrom)
				ssFrom.RemoveResourceIfEmpty(fromResourceAddr)

				rs := stateTo.Resource(addrTo.ContainingResource())
				if rs == nil {
					// If we're moving to an address without an index then that
					// suggests the user's intent is to establish both the
					// resource and the instance at the same time (since the
					// address covers both). If there's an index in the
					// target then allow creating the new instance here.
					resourceAddr := addrTo.ContainingResource()
					stateTo.SyncWrapper().SetResourceProvider(
						resourceAddr,
						fromProviderAddr, // in this case, we bring the provider along as if we were moving the whole resource
					)
					rs = stateTo.Resource(resourceAddr)
				}

				rs.Instances[addrTo.Resource.Key] = is
			}
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				msgInvalidSource,
				fmt.Sprintf("Cannot move %s: Terraform doesn't know how to move this object.", rawAddrFrom),
			))
		}

		// Look for any dependencies that may be effected and
		// remove them to ensure they are recreated in full.
		for _, mod := range stateTo.Modules {
			for _, res := range mod.Resources {
				for _, ins := range res.Instances {
					if ins.Current == nil {
						continue
					}

					for _, dep := range ins.Current.Dependencies {
						// check both directions here, since we may be moving
						// an instance which is in a resource, or a module
						// which can contain a resource.
						if dep.TargetContains(rawAddrFrom) || rawAddrFrom.TargetContains(dep) {
							ins.Current.Dependencies = nil
							break
						}
					}
				}
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

	c.showDiagnostics(diags)

	if moved == 0 {
		c.Ui.Output("No matching objects found.")
	} else {
		c.Ui.Output(fmt.Sprintf("Successfully moved %d object(s).", moved))
	}
	return 0
}

// sourceObjectAddrs takes a single source object address and expands it to
// potentially multiple objects that need to be handled within it.
//
// In particular, this handles the case where a module is requested directly:
// if it has any child modules, then they must also be moved. It also resolves
// the ambiguity that an index-less resource address could either be a resource
// address or a resource instance address, by making a decision about which
// is intended based on the current state of the resource in question.
func (c *StateMvCommand) sourceObjectAddrs(state *states.State, matched addrs.Targetable) []addrs.Targetable {
	var ret []addrs.Targetable

	switch addr := matched.(type) {
	case addrs.ModuleInstance:
		for _, mod := range state.Modules {
			if len(mod.Addr) < len(addr) {
				continue // can't possibly be our selection or a child of it
			}
			if !mod.Addr[:len(addr)].Equal(addr) {
				continue
			}
			ret = append(ret, mod.Addr)
		}
	case addrs.AbsResource:
		// If this refers to a resource without "count" or "for_each" set then
		// we'll assume the user intended it to be a resource instance
		// address instead, to allow for requests like this:
		//   terraform state mv aws_instance.foo aws_instance.bar[1]
		// That wouldn't be allowed if aws_instance.foo had multiple instances
		// since we can't move multiple instances into one.
		if rs := state.Resource(addr); rs != nil {
			if _, ok := rs.Instances[addrs.NoKey]; ok {
				ret = append(ret, addr.Instance(addrs.NoKey))
			} else {
				ret = append(ret, addr)
			}
		}
	default:
		ret = append(ret, matched)
	}

	return ret
}

func (c *StateMvCommand) validateResourceMove(addrFrom, addrTo addrs.AbsResource) tfdiags.Diagnostics {
	const msgInvalidRequest = "Invalid state move request"

	var diags tfdiags.Diagnostics
	if addrFrom.Resource.Mode != addrTo.Resource.Mode {
		switch addrFrom.Resource.Mode {
		case addrs.ManagedResourceMode:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				msgInvalidRequest,
				fmt.Sprintf("Cannot move %s to %s: a managed resource can be moved only to another managed resource address.", addrFrom, addrTo),
			))
		case addrs.DataResourceMode:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				msgInvalidRequest,
				fmt.Sprintf("Cannot move %s to %s: a data resource can be moved only to another data resource address.", addrFrom, addrTo),
			))
		default:
			// In case a new mode is added in future, this unhelpful error is better than nothing.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				msgInvalidRequest,
				fmt.Sprintf("Cannot move %s to %s: cannot change resource mode.", addrFrom, addrTo),
			))
		}
	}
	if addrFrom.Resource.Type != addrTo.Resource.Type {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			msgInvalidRequest,
			fmt.Sprintf("Cannot move %s to %s: resource types don't match.", addrFrom, addrTo),
		))
	}
	return diags
}

func (c *StateMvCommand) Help() string {
	helpText := `
Usage: terraform [global options] state mv [options] SOURCE DESTINATION

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

  -dry-run                If set, prints out what would've been moved but doesn't
                          actually move anything.

  -lock=false             Don't hold a state lock during the operation. This is
                          dangerous if others might concurrently run commands
                          against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -ignore-remote-version  A rare option used for the remote backend only. See
                          the remote backend documentation for more information.

  -state, state-out, and -backup are legacy options supported for the local
  backend only. For more information, see the local backend's documentation.

`
	return strings.TrimSpace(helpText)
}

func (c *StateMvCommand) Synopsis() string {
	return "Move an item in the state"
}

const errStateMv = `Error moving state: %s

Please ensure your addresses and state paths are valid. No
state was persisted. Your existing states are untouched.`
