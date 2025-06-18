// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/repl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ConsoleCommand is a Command implementation that starts an interactive
// console that can be used to try expressions with the current config.
type ConsoleCommand struct {
	Meta
}

func (c *ConsoleCommand) Run(args []string) int {
	args = c.Meta.process(args)
	var evalFromPlan bool
	cmdFlags := c.Meta.extendedFlagSet("console")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "use state locking")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.BoolVar(&evalFromPlan, "plan", false, "evaluate from plan")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command line flags: %s\n", err.Error()))
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	configPath = c.Meta.normalizePath(configPath)

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
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

	// We require a local backend
	local, ok := b.(backendrun.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Build the operation
	opReq := c.Operation(b, arguments.ViewHuman)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
	opReq.AllowUnsetVariables = true // we'll just evaluate them as unknown
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
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

	// Get the context
	lr, _, ctxDiags := local.LocalRun(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Successfully creating the context can result in a lock, so ensure we release it
	defer func() {
		diags := opReq.StateLocker.Unlock()
		if diags.HasErrors() {
			c.showDiagnostics(diags)
		}
	}()

	// Set up the UI so we can output directly to stdout
	ui := &cli.BasicUi{
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	var scope *lang.Scope
	if evalFromPlan {
		var planDiags tfdiags.Diagnostics
		_, scope, planDiags = lr.Core.PlanAndEval(lr.Config, lr.InputState, lr.PlanOpts)
		diags = diags.Append(planDiags)
	} else {
		evalOpts := &terraform.EvalOpts{}
		if lr.PlanOpts != nil {
			// the LocalRun type is built primarily to support the main operations,
			// so the variable values end up in the "PlanOpts" even though we're
			// not actually making a plan.
			evalOpts.SetVariables = lr.PlanOpts.SetVariables
		}

		// Before we can evaluate expressions, we must compute and populate any
		// derived values (input variables, local values, output values)
		// that are not stored in the persistent state.
		var scopeDiags tfdiags.Diagnostics
		scope, scopeDiags = lr.Core.Eval(lr.Config, lr.InputState, addrs.RootModuleInstance, evalOpts)
		diags = diags.Append(scopeDiags)
	}
	if scope == nil {
		// scope is nil if there are errors so bad that we can't even build a scope.
		// Otherwise, we'll try to eval anyway.
		c.showDiagnostics(diags)
		return 1
	}

	// set the ConsoleMode to true so any available console-only functions included.
	scope.ConsoleMode = true

	// Before we become interactive we'll show any diagnostics we encountered
	// during initialization, and then afterwards the driver will manage any
	// further diagnostics itself.
	if diags.HasErrors() {
		// showDiagnostics is designed to always render warnings first, but
		// for this command we have one special warning that should always
		// appear after everything else, to increase the chances that the
		// user will notice it before they become confused by an incomplete
		// expression result.
		c.showDiagnostics(diags)
		diags = nil
		diags = diags.Append(tfdiags.SimpleWarning("Due to the problems above, some expressions may produce unexpected results."))
	}
	c.showDiagnostics(diags)
	diags = nil

	// IO Loop
	session := &repl.Session{
		Scope: scope,
	}

	// Determine if stdin is a pipe. If so, we evaluate directly.
	if c.StdinPiped() {
		return c.modePiped(session, ui)
	}

	return c.modeInteractive(session, ui)
}

func (c *ConsoleCommand) modePiped(session *repl.Session, ui cli.Ui) int {
	var lastResult string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		result, exit, diags := session.Handle(strings.TrimSpace(scanner.Text()))
		if diags.HasErrors() {
			// In piped mode we'll exit immediately on error.
			c.showDiagnostics(diags)
			return 1
		}
		if exit {
			return 0
		}

		// Store the last result
		lastResult = result
	}

	// Output the final result
	ui.Output(lastResult)

	return 0
}

func (c *ConsoleCommand) Help() string {
	helpText := `
Usage: terraform [global options] console [options]

  Starts an interactive console for experimenting with Terraform
  interpolations.

  This will open an interactive console that you can use to type
  interpolations into and inspect their values. This command loads the
  current state. This lets you explore and test interpolations before
  using them in future configurations.

  This command will never modify your state.

Options:

  -state=path       Legacy option for the local backend only. See the local
                    backend's documentation for more information.

  -lock=false       Don't hold a state lock during the operation. This is
					dangerous if others might concurrently run commands
					against the same workspace.

  -plan             Create a new plan (as if running "terraform plan") and
                    then evaluate expressions against its planned state,
                    instead of evaluating against the current state.
                    You can use this to inspect the effects of configuration
                    changes that haven't been applied yet.

  -var 'foo=bar'    Set a variable in the Terraform configuration. This
                    flag can be set multiple times.

  -var-file=foo     Set variables in the Terraform configuration from
                    a file. If "terraform.tfvars" or any ".auto.tfvars"
                    files are present, they will be automatically loaded.
`
	return strings.TrimSpace(helpText)
}

func (c *ConsoleCommand) Synopsis() string {
	return "Try Terraform expressions at an interactive command prompt"
}
