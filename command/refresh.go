package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	Meta
}

func (c *RefreshCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("refresh")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", 0, "parallelism")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	// Load the module
	mod, diags := c.Module(configPath)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Type = backend.OperationTypeRefresh
	opReq.Module = mod

	// Perform the operation
	op, err := b.Operation(context.Background(), opReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error starting operation: %s", err))
		return 1
	}

	// Wait for the operation to complete or an interrupt to occur
	select {
	case <-c.ShutdownCh:
		// gracefully stop the operation
		op.Stop()

		// Notify the user
		c.Ui.Output(outputInterrupt)

		// Still get the result, since there is still one
		select {
		case <-c.ShutdownCh:
			c.Ui.Error(
				"Two interrupts received. Exiting immediately. Note that data\n" +
					"loss may have occurred.")

			// cancel the operation completely
			op.Cancel()

			// the operation should return asap
			// but timeout just in case
			select {
			case <-op.Done():
			case <-time.After(5 * time.Second):
			}

			return 1

		case <-op.Done():
		}
	case <-op.Done():
		if err := op.Err; err != nil {
			diags = diags.Append(err)
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	// Output the outputs
	if outputs := outputsAsString(op.State, terraform.RootModulePath, nil, true); outputs != "" {
		c.Ui.Output(c.Colorize().Color(outputs))
	}

	return 0
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options] [dir]

  Update the state file of your infrastructure with metadata that matches
  the physical resources they are tracking.

  This will not modify your infrastructure, but it can modify your
  state file to update metadata. This metadata might cause new changes
  to occur when you generate a plan or call apply next.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -input=true         Ask for input for variables if not directly set.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" or any ".auto.tfvars"
                      files are present, they will be automatically loaded.

`
	return strings.TrimSpace(helpText)
}

func (c *RefreshCommand) Synopsis() string {
	return "Update local state file against real resources"
}
