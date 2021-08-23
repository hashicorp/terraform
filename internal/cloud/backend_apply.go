package cloud

import (
	"bufio"
	"context"
	"io"
	"log"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Cloud) opApply(stopCtx, cancelCtx context.Context, op *backend.Operation, w *tfe.Workspace) (*tfe.Run, error) {
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
			`Terraform Cloud does not support setting a custom parallelism `+
				`value at this time.`,
		))
	}

	if op.PlanFile != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Applying a saved plan is currently not supported",
			`Terraform Cloud currently requires configuration to be present and `+
				`does not accept an existing saved plan as an argument at this time.`,
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

	// Run the plan phase.
	r, err := b.plan(stopCtx, cancelCtx, op, w)
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
		return r, generalError("Failed to retrieve run", err)
	}

	// Return if the run cannot be confirmed.
	if !op.AutoApprove && !r.Actions.IsConfirmable {
		return r, nil
	}

	mustConfirm := (op.UIIn != nil && op.UIOut != nil) && !op.AutoApprove

	if mustConfirm {
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
	} else {
		// If we don't need to ask for confirmation, insert a blank
		// line to separate the ouputs.
		if b.CLI != nil {
			b.CLI.Output("")
		}
	}

	if !op.AutoApprove && err != errRunApproved {
		if err = b.client.Runs.Apply(stopCtx, r.ID, tfe.RunApplyOptions{}); err != nil {
			return r, generalError("Failed to approve the apply command", err)
		}
	}

	r, err = b.waitForRun(stopCtx, cancelCtx, op, "apply", r, w)
	if err != nil {
		return r, err
	}

	logs, err := b.client.Applies.Logs(stopCtx, r.Apply.ID)
	if err != nil {
		return r, generalError("Failed to retrieve logs", err)
	}
	reader := bufio.NewReaderSize(logs, 64*1024)

	if b.CLI != nil {
		skip := 0
		for next := true; next; {
			var l, line []byte

			for isPrefix := true; isPrefix; {
				l, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						return r, generalError("Failed to read logs", err)
					}
					next = false
				}
				line = append(line, l...)
			}

			// Skip the first 3 lines to prevent duplicate output.
			if skip < 3 {
				skip++
				continue
			}

			if next || len(line) > 0 {
				b.CLI.Output(b.Colorize().Color(string(line)))
			}
		}
	}

	return r, nil
}

const applyDefaultHeader = `
[reset][yellow]Running apply in Terraform Cloud. Output will stream here. Pressing Ctrl-C
will cancel the remote apply if it's still pending. If the apply started it
will stop streaming the logs, but will not stop the apply running remotely.[reset]

Preparing the remote apply...
`
