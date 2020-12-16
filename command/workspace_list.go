package command

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceListCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceListCommand) Run(args []string) int {
	args = c.Meta.process(args)
	workspaceCommandShowWarning(c.Ui, c.Colorize(), c.LegacyName)

	cmdFlags := c.Meta.defaultFlagSet("workspace list")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	configPath, err := ModulePath(args)
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

	// This command will not write state
	c.ignoreRemoteBackendVersionConflict(b)

	states, err := b.Workspaces()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	env, isOverridden := c.WorkspaceOverridden()

	var out bytes.Buffer
	for _, s := range states {
		if s == env {
			out.WriteString("* ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(s + "\n")
	}

	c.Ui.Output(out.String())

	if isOverridden {
		c.Ui.Output(envIsOverriddenNote)
	}

	return 0
}

func (c *WorkspaceListCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("")
}

func (c *WorkspaceListCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceListCommand) Help() string {
	helpText := `
Usage: terraform state available

  Prints a list of all of the named states available in the
  currently-configured backend.

  In the typical situation where a configuration has only one associated state,
  that single state is named "default".
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List all states for the current configuration"
}
