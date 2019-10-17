package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/tfdiags"
	"github.com/posener/complete"
)

type WorkspaceShowCommand struct {
	Meta
}

func (c *WorkspaceShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("workspace show")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	var diags tfdiags.Diagnostics

	projectMgr, moreDiags := c.findCurrentProjectManager()
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	workspaceAddr := c.Workspace()
	workspace, moreDiags := projectMgr.LoadWorkspace(workspaceAddr)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	depAddrs := projectMgr.Project().WorkspaceDependencies(workspaceAddr)

	fmt.Printf("Workspace %s\n\n", workspaceAddr.StringCompact())
	fmt.Printf("Configuration Root: %s\n", workspace.ConfigDir())
	fmt.Printf("Input Variables:\n")
	for addr, val := range workspace.InputVariables() {
		fmt.Printf("  %s = %#v\n", addr.Name, val)
	}
	fmt.Printf("Dependencies:\n")
	for _, addr := range depAddrs {
		fmt.Printf("  %s\n", addr)
	}
	fmt.Println("")

	return 0
}

func (c *WorkspaceShowCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *WorkspaceShowCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceShowCommand) Help() string {
	helpText := `
Usage: terraform workspace show

  Show the name of the current workspace.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceShowCommand) Synopsis() string {
	return "Show the name of the current workspace"
}
