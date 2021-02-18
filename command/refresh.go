package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	Meta
}

func (c *RefreshCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.extendedFlagSet("refresh")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	diags := c.parseTargetFlags()
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	backendConfig, configDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	c.showDiagnostics(diags)
	diags = nil

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = configPath
	opReq.Hooks = []terraform.Hook{c.uiHook()}
	opReq.ShowDiagnostics = c.showDiagnostics
	opReq.Type = backend.OperationTypeRefresh
	opReq.View = views.NewOperation(arguments.ViewHuman, c.RunningInAutomation, c.View)

	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	{
		var moreDiags tfdiags.Diagnostics
		opReq.Variables, moreDiags = c.collectVariableValues()
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	op, err := c.RunOperation(b, opReq)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}
	if op.Result != backend.OperationSuccess {
		return op.Result.ExitStatus()
	}

	if op.State != nil {
		outputValues := op.State.RootModule().OutputValues
		if len(outputValues) > 0 {
			c.Ui.Output(c.Colorize().Color("[reset][bold][green]\nOutputs:\n\n"))
			view := views.NewOutput(arguments.ViewHuman, c.View)
			view.Output("", outputValues)
		}
	}

	return op.Result.ExitStatus()
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options]

  Update the state file of your infrastructure with metadata that matches
  the physical resources they are tracking.

  This will not modify your infrastructure, but it can modify your
  state file to update metadata. This metadata might cause new changes
  to occur when you generate a plan or call apply next.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -compact-warnings   If Terraform produces any warnings that are not
                      accompanied by errors, show them in a more compact form
                      that includes only the summary messages.

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
	return "Update the state to match remote systems"
}
