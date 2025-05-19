// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Cloud) opApply(stopCtx, cancelCtx context.Context, op *backendrun.Operation, w *tfe.Workspace) (*tfe.Run, error) {
	log.Printf("[INFO] cloud: starting Apply operation")

	var diags tfdiags.Diagnostics

	// We should remove the `CanUpdate` part of this test, but for now
	// (to remain compatible with tfe.v2.1) we'll leave it in here.
	if !w.Permissions.CanUpdate && !w.Permissions.CanQueueApply {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Insufficient rights to apply changes",
			"The provided credentials have insufficient rights to apply changes. In order "+
				"to apply changes at least write permissions on the workspace are required.",
		))
		return nil, diags.Err()
	}

	if w.VCSRepo != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Apply not allowed for workspaces with a VCS connection",
			"A workspace that is connected to a VCS requires the VCS-driven workflow "+
				"to ensure that the VCS remains the single source of truth.",
		))
		return nil, diags.Err()
	}

	if b.ContextOpts != nil && b.ContextOpts.Parallelism != defaultParallelism {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Custom parallelism values are currently not supported",
			fmt.Sprintf("%s does not support setting a custom parallelism ", b.appName)+
				"value at this time.",
		))
	}

	if op.PlanFile.IsLocal() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Applying a saved local plan is not supported",
			fmt.Sprintf("%s can apply a saved cloud plan, or create a new plan when ", b.appName)+
				`configuration is present. It cannot apply a saved local plan.`,
		))
	}

	if !op.HasConfig() && op.PlanMode != plans.DestroyMode {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files found",
			`Apply requires configuration to be present. Applying without a configuration `+
				`would mark everything for destruction, which is normally not what is desired. `+
				`If you would like to destroy everything, please run 'terraform destroy' which `+
				`does not require any configuration files.`,
		))
	}

	// Return if there are any errors.
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	var r *tfe.Run
	var err error
	cp, hasSavedPlanFile := op.PlanFile.Cloud()
	if hasSavedPlanFile {
		log.Printf("[TRACE] Loading saved cloud plan for apply")
		// Check hostname first, for a more actionable error than a generic 404 later
		if cp.Hostname != b.Hostname {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Saved plan is for a different hostname",
				fmt.Sprintf("The given saved plan refers to a run on %s, but the currently configured %s instance is %s.", cp.Hostname, b.appName, b.Hostname),
			))
			return r, diags.Err()
		}
		// Fetch the run referenced in the saved plan bookmark.
		r, err = b.client.Runs.ReadWithOptions(stopCtx, cp.RunID, &tfe.RunReadOptions{
			Include: []tfe.RunIncludeOpt{tfe.RunWorkspace},
		})

		if err != nil {
			return r, err
		}

		if r.Workspace.ID != w.ID {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Saved plan is for a different workspace",
				fmt.Sprintf("The given saved plan does not refer to a run in the current workspace (%s/%s), so it cannot currently be applied. For more details, view this run in a browser at:\n%s", w.Organization.Name, w.Name, runURL(b.Hostname, r.Workspace.Organization.Name, r.Workspace.Name, r.ID)),
			))
			return r, diags.Err()
		}

		if !r.Actions.IsConfirmable {
			url := runURL(b.Hostname, b.Organization, op.Workspace, r.ID)
			return r, unusableSavedPlanError(r.Status, url, b.appName)
		}

		// Since we're not calling plan(), we need to print a run header ourselves:
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(applySavedHeader, b.appName) + "\n")))
			b.CLI.Output(b.Colorize().Color(strings.TrimSpace(fmt.Sprintf(
				runHeader, b.Hostname, b.Organization, r.Workspace.Name, r.ID)) + "\n"))
		}
	} else {
		log.Printf("[TRACE] Running new cloud plan for apply")
		// Run the plan phase.
		r, err = b.plan(stopCtx, cancelCtx, op, w)

		if err != nil {
			return r, err
		}

		// This check is also performed in the plan method to determine if
		// the policies should be checked, but we need to check the values
		// here again to determine if we are done and should return.
		if !r.HasChanges || r.Status == tfe.RunCanceled || r.Status == tfe.RunErrored {
			return r, nil
		}

		// Retrieve the run to get its current status.
		r, err = b.client.Runs.Read(stopCtx, r.ID)
		if err != nil {
			return r, b.generalError("Failed to retrieve run", err)
		}

		// Return if the run cannot be confirmed.
		if !op.AutoApprove && !r.Actions.IsConfirmable {
			return r, nil
		}

		mustConfirm := (op.UIIn != nil && op.UIOut != nil) && !op.AutoApprove

		if mustConfirm && b.input {
			opts := &terraform.InputOpts{Id: "approve"}

			if op.PlanMode == plans.DestroyMode {
				opts.Query = "\nDo you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
				opts.Description = "Terraform will destroy all your managed infrastructure, as shown above.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			} else {
				opts.Query = "\nDo you want to perform these actions in workspace \"" + op.Workspace + "\"?"
				opts.Description = "Terraform will perform the actions described above.\n" +
					"Only 'yes' will be accepted to approve."
			}

			err = b.confirm(stopCtx, op, opts, r, "yes")
			if err != nil && err != errRunApproved {
				return r, err
			}
		} else if mustConfirm && !b.input {
			return r, errApplyNeedsUIConfirmation
		} else {
			// If we don't need to ask for confirmation, insert a blank
			// line to separate the ouputs.
			if b.CLI != nil {
				b.CLI.Output("")
			}
		}
	}

	// Do the apply!
	// If we have a saved plan file, we proceed to apply the run without confirmation
	// regardless of the value of AutoApprove.
	if (!op.AutoApprove || hasSavedPlanFile) && err != errRunApproved {
		if err = b.client.Runs.Apply(stopCtx, r.ID, tfe.RunApplyOptions{}); err != nil {
			return r, b.generalError("Failed to approve the apply command", err)
		}
	}

	// Retrieve the run to get task stages.
	// Task Stages are calculated upfront so we only need to call this once for the run.
	taskStages, err := b.runTaskStages(stopCtx, b.client, r.ID)
	if err != nil {
		return r, err
	}

	if stage, ok := taskStages[tfe.PreApply]; ok {
		if err := b.waitTaskStage(stopCtx, cancelCtx, op, r, stage.ID, "Pre-apply Tasks"); err != nil {
			return r, err
		}
	}

	r, err = b.waitForRun(stopCtx, cancelCtx, op, "apply", r, w)
	if err != nil {
		return r, err
	}

	err = b.renderApplyLogs(stopCtx, r)
	if err != nil {
		return r, err
	}

	return r, nil
}

func (b *Cloud) renderApplyLogs(ctx context.Context, run *tfe.Run) error {
	logs, err := b.client.Applies.Logs(ctx, run.Apply.ID)
	if err != nil {
		return err
	}

	if b.CLI != nil {
		reader := bufio.NewReaderSize(logs, 64*1024)
		skip := 0

		for next := true; next; {
			var l, line []byte
			var err error

			for isPrefix := true; isPrefix; {
				l, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						return b.generalError("Failed to read logs", err)
					}
					next = false
				}

				line = append(line, l...)
			}

			// Apply logs show the same Terraform info logs as shown in the plan logs
			// (which contain version and os/arch information), we therefore skip to prevent duplicate output.
			if skip < 3 {
				skip++
				continue
			}

			if next || len(line) > 0 {
				log := &jsonformat.JSONLog{}
				if err := json.Unmarshal(line, log); err != nil {
					// If we can not parse the line as JSON, we will simply
					// print the line. This maintains backwards compatibility for
					// users who do not wish to enable structured output in their
					// workspace.
					b.CLI.Output(string(line))
					continue
				}

				if b.renderer != nil {
					// Otherwise, we will print the log
					err := b.renderer.RenderLog(log)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func runURL(hostname, orgName, wsName, runID string) string {
	return fmt.Sprintf("https://%s/app/%s/%s/runs/%s", hostname, orgName, wsName, runID)
}

func unusableSavedPlanError(status tfe.RunStatus, url, appName string) error {
	var diags tfdiags.Diagnostics
	var summary, reason string

	switch status {
	case tfe.RunApplied:
		summary = "Saved plan is already applied"
		reason = "The given plan file was already successfully applied, and cannot be applied again."
	case tfe.RunApplying, tfe.RunApplyQueued, tfe.RunConfirmed:
		summary = "Saved plan is already confirmed"
		reason = "The given plan file is already being applied, and cannot be applied again."
	case tfe.RunCanceled:
		summary = "Saved plan is canceled"
		reason = fmt.Sprintf("The given plan file can no longer be applied because the run was canceled via the %s UI or API.", appName)
	case tfe.RunDiscarded:
		summary = "Saved plan is discarded"
		reason = fmt.Sprintf("The given plan file can no longer be applied; either another run was applied first, or a user discarded it via the %s UI or API.", appName)
	case tfe.RunErrored:
		summary = "Saved plan is errored"
		reason = "The given plan file refers to a plan that had errors and did not complete successfully. It cannot be applied."
	case tfe.RunPlannedAndFinished:
		// Note: planned and finished can also indicate a plan-only run, but
		// terraform plan can't create a saved plan for a plan-only run, so we
		// know it's no-changes in this case.
		summary = "Saved plan has no changes"
		reason = "The given plan file contains no changes, so it cannot be applied."
	case tfe.RunPolicyOverride:
		summary = "Saved plan requires policy override"
		reason = "The given plan file has soft policy failures, and cannot be applied until a user with appropriate permissions overrides the policy check."
	default:
		summary = "Saved plan cannot be applied"
		reason = fmt.Sprintf("%s cannot apply the given plan file. This may mean the plan and checks have not yet completed, or may indicate another problem.", appName)
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		summary,
		fmt.Sprintf("%s For more details, view this run in a browser at:\n%s", reason, url),
	))
	return diags.Err()
}

const applyDefaultHeader = `
[reset][yellow]Running apply in %s. Output will stream here. Pressing Ctrl-C
will cancel the remote apply if it's still pending. If the apply started it
will stop streaming the logs, but will not stop the apply running remotely.[reset]

Preparing the remote apply...
`

const applySavedHeader = `
[reset][yellow]Running apply in %s. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the apply running remotely.[reset]

Preparing the remote apply...
`
