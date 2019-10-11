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
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	envCommandShowWarning(c.Ui, c.LegacyName)

	cmdFlags := c.Meta.defaultFlagSet("workspace list")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	var diags tfdiags.Diagnostics

	project, moreDiags := c.findCurrentProject()
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	workspaces := project.AllWorkspaceAddrs()
	currentWorkspace, isOverridden := c.WorkspaceOverridden()

	var out bytes.Buffer
	for _, addr := range workspaces {
		if addr == currentWorkspace {
			out.WriteString("* ")
		} else {
			out.WriteString("  ")
		}
		out.WriteString(addr.StringCompact() + "\n")
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
Usage: terraform workspace list [DIR]

  List Terraform workspaces.

`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceListCommand) Synopsis() string {
	return "List Workspaces"
}
