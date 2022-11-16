package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/webcommand"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/posener/complete"
)

type WebCommand struct {
	Meta
}

func (c *WebCommand) Run(rawArgs []string) int {
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseWeb(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("web")
		return 1
	}

	// The backend is responsible for deciding the specific URL to open.
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	urlProvider, ok := b.(webcommand.URLProvider)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Command requires Terraform Cloud",
			"This command is available only for root modules which include a 'cloud' block for Terraform Cloud.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	workspaceName, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot determine current workspace",
			fmt.Sprintf("Failed to determine the currently-selected workspace: %s.", err),
		))
		c.showDiagnostics(diags)
		return 1
	}

	ctx, cancel := c.InterruptibleContext()
	defer cancel()
	url, moreDiags := urlProvider.WebURLForObject(ctx, workspaceName, args.TargetObject)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	launchBrowserManually := false
	if c.BrowserLauncher != nil {
		// NOTE: On some platforms we launch URLs by running external helper
		// commands that then in turn know how to launch the appropriate
		// browser. Those commands tend to produce output on stderr if they
		// fail, so if err isn't nil here then it's typical for there to
		// already be some arbitrary chatter on stderr describing that problem
		// in a platform-specific way.
		err := c.BrowserLauncher.OpenURL(url.String())
		if err != nil {
			// Assume we're on a platform where opening a browser isn't possible.
			launchBrowserManually = true
		}
	} else {
		launchBrowserManually = true
	}

	if launchBrowserManually {
		c.Ui.Output(fmt.Sprintf(
			"Terraform cannot automatically launch a web browser on this system.\n\nThe following is the URL for %s:\n    %s\n",
			args.TargetObject.UIDescription(),
			url.String(),
		))
	} else {
		c.Ui.Output(fmt.Sprintf(
			"Terraform is attempting to open %s in your browser:\n    %s\n",
			args.TargetObject.UIDescription(),
			url.String(),
		))
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *WebCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (c *WebCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-latest-run": complete.PredictNothing,
		"-run":        complete.PredictAnything,
	}
}

func (c *WebCommand) Help() string {
	helpText := `
Usage: terraform [global options] web [options]

  Launches your web browser to view a web UI representation of a selected
  object relevant to your current context.

  With no options at all the selected object is your currently-selected
  workspace. Use one of the following options to select a different object:

    -latest-run  The most recent run in the currently-selected workspace.

    -run=ID      The run with the given run ID from the currently-selected
                 workspace, if any.

  This command is available only when using Terraform Cloud, with the "cloud"
  block in your root module.
`
	return strings.TrimSpace(helpText)
}

func (c *WebCommand) Synopsis() string {
	return "View something in your web browser"
}
