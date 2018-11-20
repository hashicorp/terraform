package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/states"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/tfdiags"
)

// TaintCommand is a cli.Command implementation that manually taints
// a resource, marking it for recreation.
type TaintCommand struct {
	Meta
}

func (c *TaintCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var allowMissing bool
	var module string
	cmdFlags := c.Meta.flagSet("taint")
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

	// Require the one argument for the resource to taint
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The taint command expects exactly one argument.")
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

	if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
		c.Ui.Error(fmt.Sprintf("Resource instance %s cannot be tainted", addr))
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
	env := c.Workspace()
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(stateMgr, "taint"); err != nil {
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

	obj.Status = states.ObjectTainted
	ss.SetResourceInstanceCurrent(addr, obj, rs.ProviderConfig)

	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}
	if err := stateMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("Resource instance %s has been marked as tainted.", addr))
	return 0
}

func (c *TaintCommand) Help() string {
	helpText := `
Usage: terraform taint [options] <address>

  Manually mark a resource as tainted, forcing a destroy and recreate
  on the next plan/apply.

  This will not modify your infrastructure. This command changes your
  state to mark a resource as tainted so that during the next plan or
  apply that resource will be destroyed and recreated. This command on
  its own will not modify infrastructure. This command can be undone
  using the "terraform untaint" command with the same address.

  The address is in the usual resource address syntax, as shown in
  the output from other commands, such as:
    aws_instance.foo
    aws_instance.bar[1]
    module.foo.module.bar.aws_instance.baz

Options:

  -allow-missing      If specified, the command will succeed (exit code 0)
                      even if the resource is missing.

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *TaintCommand) Synopsis() string {
	return "Manually mark a resource for recreation"
}

func (c *TaintCommand) allowMissingExit(name addrs.AbsResourceInstance) int {
	c.showDiagnostics(tfdiags.Sourceless(
		tfdiags.Warning,
		"No such resource instance",
		"Resource instance %s was not found, but this is not an error because -allow-missing was set.",
	))
	return 0
}
