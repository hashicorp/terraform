package command

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RPCPlugin is a Command implementation that causes Terraform to behave as
// an RPCPlugin server which allows its caller to interact directly with
// Terraform Core, largely bypassing our CLI layer except for its handing
// of working directory artifacts like the local plugin cache.
type RPCPluginCommand struct {
	Meta
}

func (c *RPCPluginCommand) Help() string {
	return rpcPluginCommandHelp
}

func (c *RPCPluginCommand) Synopsis() string {
	return "Run as an RPC server accessible by the parent process"
}

func (c *RPCPluginCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("rpcplugin")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command line options: %s\n", err.Error()))
		return 1
	}

	ctx, ctxCancel := c.InterruptibleContext()
	defer ctxCancel()

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var diags tfdiags.Diagnostics

	if !rpcapi.RunningAsPlugin(ctx) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Not a user-facing command",
			`The "rpcplugin" command is for internal use by other wrapper programs that understand its RPC protocol, and cannot be used directly from a shell.`,
		))
		c.showDiagnostics(diags)
		return 1
	}

	coreOpts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to prepare Terraform Core runtime",
			fmt.Sprintf("Could not instantiate the Terraform Core runtime: %s.", err),
		))
		c.showDiagnostics(diags)
		return 1
	}

	err = rpcapi.Serve(ctx, rpcapi.ServeOpts{
		GetCoreOpts: func() *terraform.ContextOpts {
			return coreOpts
		},
	})
	if err != nil { // Should _always_ have an error if Serve returns
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to start RPC server",
			fmt.Sprintf("Could not start the Terraform Core RPC server: %s.", err),
		))
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

const rpcPluginCommandHelp = `
Usage: terraform [global options] rpcplugin

  A plumbing command that makes Terraform behave as an RPC server to the
  program that calls it.

  This is not intended for running directly from a shell. It's useful only
  for special client programs that understand how to negotiate an RPC channel
  and make requests to it.

  The behavior of the "rpcplugin" mode is EXPERIMENTAL and subject to breaking
  changes or total removal even in patch releases.
`
