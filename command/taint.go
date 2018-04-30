package command

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/terraform"
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

	// Require the one argument for the resource to taint
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The taint command expects exactly one argument.")
		cmdFlags.Usage()
		return 1
	}

	name := args[0]
	if module == "" {
		module = "root"
	} else {
		module = "root." + module
	}

	rsk, err := terraform.ParseResourceStateKey(name)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to parse resource name: %s", err))
		return 1
	}

	if !rsk.Mode.Taintable() {
		c.Ui.Error(fmt.Sprintf("Resource '%s' cannot be tainted", name))
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// Get the state
	env := c.Workspace()
	st, err := b.State(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(st, "taint"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
		defer stateLocker.Unlock(nil)
	}

	if err := st.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Get the actual state structure
	s := st.State()
	if s.Empty() {
		if allowMissing {
			return c.allowMissingExit(name, module)
		}

		c.Ui.Error(fmt.Sprintf(
			"The state is empty. The most common reason for this is that\n" +
				"an invalid state file path was given or Terraform has never\n " +
				"been run for this infrastructure. Infrastructure must exist\n" +
				"for it to be tainted."))
		return 1
	}

	// Get the ModuleState where we will taint. This is provided in a legacy
	// string form that doesn't support module instance keys, so we'll shim
	// it here.
	modPath := addrs.Module(strings.Split(module, ".")).UnkeyedInstanceShim()
	mod := s.ModuleByPath(modPath)
	if mod == nil {
		if allowMissing {
			return c.allowMissingExit(name, module)
		}

		c.Ui.Error(fmt.Sprintf(
			"The module %s could not be found. There is nothing to taint.",
			module))
		return 1
	}

	// If there are no resources in this module, it is an error
	if len(mod.Resources) == 0 {
		if allowMissing {
			return c.allowMissingExit(name, module)
		}

		c.Ui.Error(fmt.Sprintf(
			"The module %s has no resources. There is nothing to taint.",
			module))
		return 1
	}

	// Get the resource we're looking for
	rs, ok := mod.Resources[name]
	if !ok {
		if allowMissing {
			return c.allowMissingExit(name, module)
		}

		c.Ui.Error(fmt.Sprintf(
			"The resource %s couldn't be found in the module %s.",
			name,
			module))
		return 1
	}

	// Taint the resource
	rs.Taint()

	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := st.WriteState(s); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}
	if err := st.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf(
		"The resource %s in the module %s has been marked as tainted!",
		name, module))
	return 0
}

func (c *TaintCommand) Help() string {
	helpText := `
Usage: terraform taint [options] name

  Manually mark a resource as tainted, forcing a destroy and recreate
  on the next plan/apply.

  This will not modify your infrastructure. This command changes your
  state to mark a resource as tainted so that during the next plan or
  apply, that resource will be destroyed and recreated. This command on
  its own will not modify infrastructure. This command can be undone by
  reverting the state backup file that is created.

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

func (c *TaintCommand) Synopsis() string {
	return "Manually mark a resource for recreation"
}

func (c *TaintCommand) allowMissingExit(name, module string) int {
	c.Ui.Output(fmt.Sprintf(
		"The resource %s in the module %s was not found, but\n"+
			"-allow-missing is set, so we're exiting successfully.",
		name, module))
	return 0
}
