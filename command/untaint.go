package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
)

// UntaintCommand is a cli.Command implementation that manually untaints
// a resource, marking it as primary and ready for service.
type UntaintCommand struct {
	Meta
}

func (c *UntaintCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var allowMissing bool
	var module string
	cmdFlags := c.Meta.flagSet("untaint")
	cmdFlags.BoolVar(&allowMissing, "allow-missing", false, "module")
	cmdFlags.StringVar(&module, "module", "", "module")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var diags tfdiags.Diagnostics

	// Require the one argument for the resource to untaint
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The untaint command expects exactly one argument.")
		cmdFlags.Usage()
		return 1
	}

	if module != "" {
		c.Ui.Error("The -module option is no longer used. Instead, include the module path in the main resource address, like \"module.foo.module.bar.null_resource.baz\".")
		return 1
	}

	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	diags = diags.Append(addrDiags)
	if addrDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Get the state
	workspace := c.Workspace()
	stateMgr, err := b.StateMgr(workspace)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(stateMgr, "untaint"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
		defer stateLocker.Unlock(nil)
	}

	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Get the actual state structure
	state := stateMgr.State()
	if state.Empty() {
		if allowMissing {
			return c.allowMissingExit(addr)
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No such resource instance",
			"The state currently contains no resource instances whatsoever. This may occur if the configuration has never been applied or if it has recently been destroyed.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	ss := state.SyncWrapper()

	// Get the resource and instance we're going to taint
	rs := ss.Resource(addr.ContainingResource())
	is := ss.ResourceInstance(addr)
	if is == nil {
		if allowMissing {
			return c.allowMissingExit(addr)
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No such resource instance",
			fmt.Sprintf("There is no resource instance in the state with the address %s. If the resource configuration has just been added, you must run \"terraform apply\" once to create the corresponding instance(s) before they can be tainted.", addr),
		))
		c.showDiagnostics(diags)
		return 1
	}

	obj := is.Current
	if obj == nil {
		if len(is.Deposed) != 0 {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance",
				fmt.Sprintf("Resource instance %s is currently part-way through a create_before_destroy replacement action. Run \"terraform apply\" to complete its replacement before tainting it.", addr),
			))
		} else {
			// Don't know why we're here, but we'll produce a generic error message anyway.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No such resource instance",
				fmt.Sprintf("Resource instance %s does not currently have a remote object associated with it, so it cannot be tainted.", addr),
			))
		}
		c.showDiagnostics(diags)
		return 1
	}

	if obj.Status != states.ObjectTainted {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource instance is not tainted",
			fmt.Sprintf("Resource instance %s is not currently tainted, and so it cannot be untainted.", addr),
		))
		c.showDiagnostics(diags)
		return 1
	}
	obj.Status = states.ObjectReady
	ss.SetResourceInstanceCurrent(addr, obj, rs.ProviderConfig)

	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}
	if err := stateMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Resource instance %s has been successfully untainted.", addr))
	return 0
}

func (c *UntaintCommand) Help() string {
	helpText := `
Usage: terraform untaint [options] name

  Manually unmark a resource as tainted, restoring it as the primary
  instance in the state.  This reverses either a manual 'terraform taint'
  or the result of provisioners failing on a resource.

  This will not modify your infrastructure. This command changes your
  state to unmark a resource as tainted.  This command can be undone by
  reverting the state backup file that is created, or by running
  'terraform taint' on the resource.

Options:

  -allow-missing      If specified, the command will succeed (exit code 0)
                      even if the resource is missing.

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -module=path        The module path where the resource lives. By
                      default this will be root. Child modules can be specified
                      by names. Ex. "consul" or "consul.vpc" (nested modules).

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *UntaintCommand) Synopsis() string {
	return "Manually unmark a resource as tainted"
}

func (c *UntaintCommand) allowMissingExit(name addrs.AbsResourceInstance) int {
	c.showDiagnostics(tfdiags.Sourceless(
		tfdiags.Warning,
		"No such resource instance",
		"Resource instance %s was not found, but this is not an error because -allow-missing was set.",
	))
	return 0
}
