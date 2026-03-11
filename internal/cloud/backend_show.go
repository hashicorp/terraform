// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"fmt"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
	"github.com/hashicorp/terraform/internal/plans"
)

// ShowPlanForRun downloads the JSON plan output for the specified cloud run
// (either the redacted or unredacted format, per the caller's request), and
// returns it in a cloudplan.RemotePlanJSON wrapper struct (along with various
// metadata required by terraform show). It's intended for use by the terraform
// show command, in order to format and display a saved cloud plan.
func (b *Cloud) ShowPlanForRun(ctx context.Context, runID, runHostname string, redacted bool) (*cloudplan.RemotePlanJSON, error) {
	var jsonBytes []byte
	mode := plans.NormalMode
	var opts []plans.Quality

	// Bail early if wrong hostname
	if runHostname != b.Hostname {
		return nil, fmt.Errorf("hostname for run (%s) does not match the configured cloud integration (%s)", runHostname, b.Hostname)
	}

	// Get run and plan
	r, err := b.client.Runs.ReadWithOptions(ctx, runID, &tfe.RunReadOptions{Include: []tfe.RunIncludeOpt{tfe.RunPlan, tfe.RunWorkspace}})
	if err == tfe.ErrResourceNotFound {
		return nil, fmt.Errorf("couldn't read information for cloud run %s; make sure you've run `terraform login` and that you have permission to view the run", runID)
	} else if err != nil {
		return nil, fmt.Errorf("couldn't read information for cloud run %s: %w", runID, err)
	}

	// Sort out the run mode
	if r.IsDestroy {
		mode = plans.DestroyMode
	} else if r.RefreshOnly {
		mode = plans.RefreshOnlyMode
	}

	// Check that the plan actually finished
	switch r.Plan.Status {
	case tfe.PlanErrored:
		// Errored plans might still be displayable, but we want to mention it to the renderer.
		opts = append(opts, plans.Errored)
	case tfe.PlanFinished:
		// Good to go, but alert the renderer if it has no changes.
		if !r.Plan.HasChanges {
			opts = append(opts, plans.NoChanges)
		}
	default:
		// Bail, we can't use this.
		err = fmt.Errorf("can't display a cloud plan that is currently %s", r.Plan.Status)
		return nil, err
	}

	// Fetch the json plan!
	if redacted {
		jsonBytes, err = readRedactedPlan(ctx, b.client.BaseURL(), b.Token, r.Plan.ID)
	} else {
		jsonBytes, err = b.client.Plans.ReadJSONOutput(ctx, r.Plan.ID)
	}
	if err == tfe.ErrResourceNotFound {
		if redacted {
			return nil, fmt.Errorf("couldn't read plan data for cloud run %s; make sure you've run `terraform login` and that you have permission to view the run", runID)
		} else {
			return nil, fmt.Errorf("couldn't read unredacted JSON plan data for cloud run %s; make sure you've run `terraform login` and that you have admin permissions on the workspace", runID)
		}
	} else if err != nil {
		return nil, fmt.Errorf("couldn't read plan data for cloud run %s: %w", runID, err)
	}

	// Format a run header and footer
	header := strings.TrimSpace(fmt.Sprintf(runHeader, b.Hostname, b.Organization, r.Workspace.Name, r.ID))
	footer := strings.TrimSpace(statusFooter(r.Status, r.Actions.IsConfirmable, r.Workspace.Locked))

	out := &cloudplan.RemotePlanJSON{
		JSONBytes: jsonBytes,
		Redacted:  redacted,
		Mode:      mode,
		Qualities: opts,
		RunHeader: header,
		RunFooter: footer,
	}

	return out, nil
}

func statusFooter(status tfe.RunStatus, isConfirmable, locked bool) string {
	statusText := strings.ReplaceAll(string(status), "_", " ")
	statusColor := "red"
	statusNote := "not confirmable"
	if isConfirmable {
		statusColor = "green"
		statusNote = "confirmable"
	}
	lockedColor := "green"
	lockedText := "unlocked"
	if locked {
		lockedColor = "red"
		lockedText = "locked"
	}
	return fmt.Sprintf(statusFooterText, statusColor, statusText, statusNote, lockedColor, lockedText)
}

const statusFooterText = `
[reset][%s]Run status: %s (%s)[reset]
[%s]Workspace is %s[reset]
`
