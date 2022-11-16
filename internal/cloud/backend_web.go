package cloud

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/command/webcommand"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// WebURLForObject implements command.WebCommandURLProvider, which makes
// this backend support the "terraform web" command.
func (b *Cloud) WebURLForObject(ctx context.Context, workspaceName string, targetObj webcommand.TargetObject) (*url.URL, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// The Terraform Cloud API doesn't currently return any information about
	// the web UI URLs corresponding to API objects, so for now we'll be
	// using hard-coded URL patterns here. These URL patterns will need to be
	// preserved (e.g. by redirects) if the web UI URL design changes in future.

	remoteWorkspaceName := b.getRemoteWorkspaceName(workspaceName)
	baseURL := &url.URL{
		Scheme: "https",
		Host:   b.hostname,
		Path:   "/app/",
	}
	ws, err := b.client.Workspaces.Read(ctx, b.organization, remoteWorkspaceName)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to fetch Terraform Cloud workspace",
			fmt.Sprintf("Error reading Terraform Cloud workspace %q: %s.", remoteWorkspaceName, err),
		))
		return nil, diags
	}

	switch targetObj {

	case webcommand.TargetObjectCurrentWorkspace:
		return baseURL.JoinPath(
			ws.Organization.Name,
			"workspaces", ws.Name,
		), diags

	case webcommand.TargetObjectLatestRun:
		if ws.CurrentRun == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No Current Run for Workspace",
				fmt.Sprintf("Terraform Cloud workspace %q does not have a current run.", ws.Name),
			))
			return nil, diags
		}
		return baseURL.JoinPath(
			ws.Organization.Name,
			"workspaces", ws.Name,
			"runs", ws.CurrentRun.ID,
		), diags
	}

	if targetObj, ok := targetObj.(webcommand.TargetObjectRun); ok {
		run, err := b.client.Runs.ReadWithOptions(ctx, targetObj.RunID, &tfe.RunReadOptions{
			Include: []tfe.RunIncludeOpt{
				"workspace",
			},
		})
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to Fetch Requested Run",
				fmt.Sprintf("Cannot read Terraform Cloud run %q: %s.", targetObj.RunID, err),
			))
			return nil, diags
		}
		if run.Workspace.ID != ws.ID {
			// If the user specified a run from elsewhere then we'll do what
			// they asked but also warn about it in case that was an accident
			// and they end up confused about where they ended up.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Run Belongs to Another Workspace",
				fmt.Sprintf("The selected run belongs to workspace %q in organization %q, which is not the currently-selected workspace.", run.Workspace.Name, run.Workspace.Organization.Name),
			))
		}
		return baseURL.JoinPath(
			run.Workspace.Organization.Name,
			"workspaces", run.Workspace.Name,
			"runs", run.ID,
		), diags
	}

	// NOTE: This is a fallback for robustness but we should typcially
	// avoid entering this fallback by ensuring that the cases above
	// always cover all of the possible values of arguments.WebTargetObject.
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Cannot view the selected object",
		fmt.Sprintf("The Terraform Cloud integration cannot open %s in your browser.", targetObj.UIDescription()),
	))
	return nil, diags
}
