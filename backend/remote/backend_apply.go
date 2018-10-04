package remote

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Remote) opApply(stopCtx, cancelCtx context.Context, op *backend.Operation) (*tfe.Run, error) {
	log.Printf("[INFO] backend/remote: starting Apply operation")

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(stopCtx, b.organization, op.Workspace)
	if err != nil {
		return nil, generalError("error retrieving workspace", err)
	}

	if !w.Permissions.CanUpdate {
		return nil, fmt.Errorf(strings.TrimSpace(
			fmt.Sprintf(applyErrNoUpdateRights, b.hostname, b.organization, op.Workspace)))
	}

	if w.VCSRepo != nil {
		return nil, fmt.Errorf(strings.TrimSpace(applyErrVCSNotSupported))
	}

	if op.Plan != nil {
		return nil, fmt.Errorf(strings.TrimSpace(applyErrPlanNotSupported))
	}

	if op.Targets != nil {
		return nil, fmt.Errorf(strings.TrimSpace(applyErrTargetsNotSupported))
	}

	if (op.Module == nil || op.Module.Config().Dir == "") && !op.Destroy {
		return nil, fmt.Errorf(strings.TrimSpace(applyErrNoConfig))
	}

	// Run the plan phase.
	r, err := b.plan(stopCtx, cancelCtx, op, w)
	if err != nil {
		return r, err
	}

	// Retrieve the run to get its current status.
	r, err = b.client.Runs.Read(stopCtx, r.ID)
	if err != nil {
		return r, generalError("error retrieving run", err)
	}

	// Return if there are no changes or the run errored. We return
	// without an error, even if the run errored, as the error is
	// already displayed by the output of the remote run.
	if !r.HasChanges || r.Status == tfe.RunErrored {
		return r, nil
	}

	// Check any configured sentinel policies.
	if len(r.PolicyChecks) > 0 {
		err = b.checkPolicy(stopCtx, cancelCtx, op, r)
		if err != nil {
			return r, err
		}
	}

	// Retrieve the run to get its current status.
	r, err = b.client.Runs.Read(stopCtx, r.ID)
	if err != nil {
		return r, generalError("error retrieving run", err)
	}

	// Return if the run cannot be confirmed.
	if !r.Actions.IsConfirmable {
		return r, nil
	}

	// Since we already checked the permissions before creating the run
	// this should never happen. But it doesn't hurt to keep this in as
	// a safeguard for any unexpected situations.
	if !r.Permissions.CanApply {
		// Make sure we discard the run if possible.
		if r.Actions.IsDiscardable {
			err = b.client.Runs.Discard(stopCtx, r.ID, tfe.RunDiscardOptions{})
			if err != nil {
				if op.Destroy {
					return r, generalError("error disarding destroy", err)
				}
				return r, generalError("error disarding apply", err)
			}
		}
		return r, fmt.Errorf(strings.TrimSpace(
			fmt.Sprint(applyErrNoApplyRights, b.hostname, b.organization, op.Workspace)))
	}

	hasUI := op.UIIn != nil && op.UIOut != nil
	mustConfirm := hasUI &&
		((op.Destroy && (!op.DestroyForce && !op.AutoApprove)) || (!op.Destroy && !op.AutoApprove))
	if mustConfirm {
		opts := &terraform.InputOpts{Id: "approve"}

		if op.Destroy {
			opts.Query = "\nDo you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
			opts.Description = "Terraform will destroy all your managed infrastructure, as shown above.\n" +
				"There is no undo. Only 'yes' will be accepted to confirm."
		} else {
			opts.Query = "\nDo you want to perform these actions in workspace \"" + op.Workspace + "\"?"
			opts.Description = "Terraform will perform the actions described above.\n" +
				"Only 'yes' will be accepted to approve."
		}

		if err = b.confirm(stopCtx, op, opts, r, "yes"); err != nil {
			return r, err
		}
	} else {
		if b.CLI != nil {
			// Insert a blank line to separate the ouputs.
			b.CLI.Output("")
		}
	}

	err = b.client.Runs.Apply(stopCtx, r.ID, tfe.RunApplyOptions{})
	if err != nil {
		return r, generalError("error approving the apply command", err)
	}

	logs, err := b.client.Applies.Logs(stopCtx, r.Apply.ID)
	if err != nil {
		return r, generalError("error retrieving logs", err)
	}
	scanner := bufio.NewScanner(logs)

	skip := 0
	for scanner.Scan() {
		// Skip the first 3 lines to prevent duplicate output.
		if skip < 3 {
			skip++
			continue
		}
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(scanner.Text()))
		}
	}
	if err := scanner.Err(); err != nil {
		return r, generalError("error reading logs", err)
	}

	return r, nil
}

func (b *Remote) checkPolicy(stopCtx, cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	if b.CLI != nil {
		b.CLI.Output("\n------------------------------------------------------------------------\n")
	}
	for _, pc := range r.PolicyChecks {
		logs, err := b.client.PolicyChecks.Logs(stopCtx, pc.ID)
		if err != nil {
			return generalError("error retrieving policy check logs", err)
		}
		scanner := bufio.NewScanner(logs)

		// Retrieve the policy check to get its current status.
		pc, err := b.client.PolicyChecks.Read(stopCtx, pc.ID)
		if err != nil {
			return generalError("error retrieving policy check", err)
		}

		var msgPrefix string
		switch pc.Scope {
		case tfe.PolicyScopeOrganization:
			msgPrefix = "Organization policy check"
		case tfe.PolicyScopeWorkspace:
			msgPrefix = "Workspace policy check"
		default:
			msgPrefix = fmt.Sprintf("Unknown policy check (%s)", pc.Scope)
		}

		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(msgPrefix + ":\n"))
		}

		for scanner.Scan() {
			if b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(scanner.Text()))
			}
		}
		if err := scanner.Err(); err != nil {
			return generalError("error reading logs", err)
		}

		switch pc.Status {
		case tfe.PolicyPasses:
			if b.CLI != nil {
				b.CLI.Output("\n------------------------------------------------------------------------")
			}
			continue
		case tfe.PolicyErrored:
			return fmt.Errorf(msgPrefix + " errored.")
		case tfe.PolicyHardFailed:
			return fmt.Errorf(msgPrefix + " hard failed.")
		case tfe.PolicySoftFailed:
			if op.UIOut == nil || op.UIIn == nil ||
				!pc.Actions.IsOverridable || !pc.Permissions.CanOverride {
				return fmt.Errorf(msgPrefix + " soft failed.")
			}
		default:
			return fmt.Errorf("Unknown or unexpected policy state: %s", pc.Status)
		}

		opts := &terraform.InputOpts{
			Id:          "override",
			Query:       "\nDo you want to override the soft failed policy check?",
			Description: "Only 'override' will be accepted to override.",
		}

		if err = b.confirm(stopCtx, op, opts, r, "override"); err != nil {
			return err
		}

		if _, err = b.client.PolicyChecks.Override(stopCtx, pc.ID); err != nil {
			return generalError("error overriding policy check", err)
		}

		if b.CLI != nil {
			b.CLI.Output("------------------------------------------------------------------------")
		}
	}

	return nil
}

func (b *Remote) confirm(stopCtx context.Context, op *backend.Operation, opts *terraform.InputOpts, r *tfe.Run, keyword string) error {
	v, err := op.UIIn.Input(opts)
	if err != nil {
		return fmt.Errorf("Error asking %s: %v", opts.Id, err)
	}
	if v != keyword {
		// Retrieve the run again to get its current status.
		r, err = b.client.Runs.Read(stopCtx, r.ID)
		if err != nil {
			return generalError("error retrieving run", err)
		}

		// Make sure we discard the run if possible.
		if r.Actions.IsDiscardable {
			err = b.client.Runs.Discard(stopCtx, r.ID, tfe.RunDiscardOptions{})
			if err != nil {
				if op.Destroy {
					return generalError("error disarding destroy", err)
				}
				return generalError("error disarding apply", err)
			}
		}

		// Even if the run was disarding successfully, we still
		// return an error as the apply command was cancelled.
		if op.Destroy {
			return errors.New("Destroy discarded.")
		}
		return errors.New("Apply discarded.")
	}

	return nil
}

const applyErrNoUpdateRights = `
Insufficient rights to apply changes!

[reset][yellow]The provided credentials have insufficient rights to apply changes. In order
to apply changes at least write permissions on the workspace are required. To
queue a run that can be approved by someone else, please use the 'Queue Plan'
button in the web UI:
https://%s/app/%s/%s/runs[reset]
`

const applyErrVCSNotSupported = `
Apply not allowed for workspaces with a VCS connection.

A workspace that is connected to a VCS requires the VCS-driven workflow
to ensure that the VCS remains the single source of truth.
`

const applyErrPlanNotSupported = `
Applying a saved plan is currently not supported!

The "remote" backend currently requires configuration to be present
and does not accept an existing saved plan as an argument at this time.
`

const applyErrTargetsNotSupported = `
Resource targeting is currently not supported!

The "remote" backend does not support resource targeting at this time.
`

const applyErrNoConfig = `
No configuration files found!

Apply requires configuration to be present. Applying without a configuration
would mark everything for destruction, which is normally not what is desired.
If you would like to destroy everything, please run 'terraform destroy' which
does not require any configuration files.
`

const applyErrNoApplyRights = `
Insufficient rights to approve the pending changes!

[reset][yellow]There are pending changes, but the provided credentials have insufficient rights
to approve them. The run will be discarded to prevent it from blocking the queue
waiting for external approval. To queue a run that can be approved by someone
else, please use the 'Queue Plan' button in the web UI:
https://%s/app/%s/%s/runs[reset]
`

const applyDefaultHeader = `
[reset][yellow]Running apply in the remote backend. Output will stream here. Pressing Ctrl-C
will cancel the remote apply if its still pending. If the apply started it
will stop streaming the logs, but will not stop the apply running remotely.
To view this run in a browser, visit:
https://%s/app/%s/%s/runs/%s[reset]

Waiting for the apply to start...
`
