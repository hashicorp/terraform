package command

import (
	"bufio"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/wrappedstreams"
	"github.com/hashicorp/terraform/repl"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/mitchellh/cli"
)

// ConsoleCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ConsoleCommand struct {
	Meta
}

func (c *ConsoleCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("console")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
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
	local, ok := b.(backend.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
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
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	defer func() {
		err := opReq.StateLocker.Unlock(nil)
		if err != nil {
			c.Ui.Error(err.Error())
		}
	}()

	// Setup the UI so we can output directly to stdout
	ui := &cli.BasicUi{
		Writer:      wrappedstreams.Stdout(),
		ErrorWriter: wrappedstreams.Stderr(),
	}

	// Before we can evaluate expressions, we must compute and populate any
	// derived values (input variables, local values, output values)
	// that are not stored in the persistent state.
	scope, scopeDiags := ctx.Eval(addrs.RootModuleInstance)
	diags = diags.Append(scopeDiags)
	if scope == nil {
		// scope is nil if there are errors so bad that we can't even build a scope.
		// Otherwise, we'll try to eval anyway.
		c.showDiagnostics(diags)
		return 1
	}
	if diags.HasErrors() {
		diags = diags.Append(tfdiags.SimpleWarning("Due to the problems above, some expressions may produce unexpected results."))
	}

	// Before we become interactive we'll show any diagnostics we encountered
	// during initialization, and then afterwards the driver will manage any
	// further diagnostics itself.
	c.showDiagnostics(diags)

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
	scanner := bufio.NewScanner(wrappedstreams.Stdin())
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
Usage: terraform console [options] [DIR]

  Starts an interactive console for experimenting with Terraform
  interpolations.

  This will open an interactive console that you can use to type
  interpolations into and inspect their values. This command loads the
  current state. This lets you explore and test interpolations before
  using them in future configurations.

  This command will never modify your state.

  DIR can be set to a directory with a Terraform state to load. By
  default, this will default to the current working directory.

Options:

  -state=path            Path to read state. Defaults to "terraform.tfstate"

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" or any ".auto.tfvars"
                         files are present, they will be automatically loaded.


`
	return strings.TrimSpace(helpText)
}

func (c *ConsoleCommand) Synopsis() string {
	return "Interactive console for Terraform interpolations"
}
