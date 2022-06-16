package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ChecksCommand is a Command implementation that shows the current check
// statuses from the latest state snapshot.
type ChecksCommand struct {
	Meta
}

func (c *ChecksCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse and validate command line arguments
	args, diags := arguments.ParseChecks(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("output")
		return 1
	}

	view := views.NewChecks(arguments.ViewHuman, c.View)

	// Fetch data from state
	checkResults, moreDiags := c.CheckResults()
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Render the view
	moreDiags = view.CurrentResults(checkResults, views.ChecksResultOptions{
		PreferShowAll: args.ShowAll,
	})
	diags = diags.Append(moreDiags)

	view.Diagnostics(diags)

	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *ChecksCommand) CheckResults() (*states.CheckResults, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		return nil, diags
	}

	// Get the state
	stateStore, err := b.StateMgr(env)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	if err := stateStore.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		return nil, diags
	}

	state := stateStore.State()
	if state == nil {
		state = states.NewState()
	}

	return state.CheckResults, nil
}

func (c *ChecksCommand) Help() string {
	helpText := `
Usage: terraform [global options] checks [options] [NAME]

  Reads the latest state snapshot for the current workspace and shows
  the statuses of any checks that were evaluated during the most
  recent apply operation.

  By default this skips showing passing checks. Use the '-all' option
  to include all checks regardless of status.

  This command requires direct access to the latest state snapshot
  via the configured backend.

Options:

  -all             Show all checks regardless of status. By default,
                   this command omits any passing checks.

  -no-color        If specified, output won't contain any color.
`
	return strings.TrimSpace(helpText)
}

func (c *ChecksCommand) Synopsis() string {
	return "Show the status of checks from the most recent run"
}
