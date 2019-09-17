package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/jsonprovider"
	"github.com/hashicorp/terraform/tfdiags"
)

// ProvidersCommand is a Command implementation that prints out information
// about the providers used in the current configuration/state.
type ProvidersSchemaCommand struct {
	Meta
}

func (c *ProvidersSchemaCommand) Help() string {
	return providersSchemaCommandHelp
}

func (c *ProvidersSchemaCommand) Synopsis() string {
	return "Prints the schemas of the providers used in the configuration"
}

func (c *ProvidersSchemaCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("providers schema")
	var jsonOutput bool
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	if !jsonOutput {
		c.Ui.Error(
			"The `terraform providers schema` command requires the `-json` flag.\n")
		cmdFlags.Usage()
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
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

	// we expect that the config dir is the cwd
	cwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting cwd: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = cwd
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	schemas := ctx.Schemas()
	jsonSchemas, err := jsonprovider.Marshal(schemas)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to marshal provider schemas to json: %s", err))
		return 1
	}
	c.Ui.Output(string(jsonSchemas))

	return 0
}

const providersSchemaCommandHelp = `
Usage: terraform providers schema -json

  Prints out a json representation of the schemas for all providers used 
  in the current configuration.
`
