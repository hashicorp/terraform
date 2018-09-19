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

func (b *Remote) opApply(stopCtx, cancelCtx context.Context, op *backend.Operation) error {
	log.Printf("[INFO] backend/remote: starting Apply operation")

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(stopCtx, b.organization, op.Workspace)
	if err != nil {
		return generalError("error retrieving workspace", err)
	}

	if w.VCSRepo != nil {
		return fmt.Errorf(strings.TrimSpace(applyErrVCSNotSupported))
	}

	if op.Plan != nil {
		return fmt.Errorf(strings.TrimSpace(applyErrPlanNotSupported))
	}

	if op.Targets != nil {
		return fmt.Errorf(strings.TrimSpace(applyErrTargetsNotSupported))
	}

	if (op.Module == nil || op.Module.Config().Dir == "") && !op.Destroy {
		return fmt.Errorf(strings.TrimSpace(planErrNoConfig))
	}

	r, err := b.plan(stopCtx, cancelCtx, op, w)
	if err != nil {
		return err
	}

	if len(r.PolicyChecks) > 0 {
		err = b.checkPolicy(stopCtx, cancelCtx, op, r)
		if err != nil {
			return err
		}
	}

	hasUI := op.UIOut != nil && op.UIIn != nil
	mustConfirm := hasUI && (op.Destroy && (!op.DestroyForce && !op.AutoApprove))
	if mustConfirm {
		opts := &terraform.InputOpts{Id: "approve"}

		if op.Destroy {
			opts.Query = "Do you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
			opts.Description = "Terraform will destroy all your managed infrastructure, as shown above.\n" +
				"There is no undo. Only 'yes' will be accepted to confirm."
		} else {
			opts.Query = "Do you want to perform these actions in workspace \"" + op.Workspace + "\"?"
			opts.Description = "Terraform will perform the actions described above.\n" +
				"Only 'yes' will be accepted to approve."
		}

		if err = b.confirm(stopCtx, op, opts, r); err != nil {
			return err
		}
	}

	err = b.client.Runs.Apply(stopCtx, r.ID, tfe.RunApplyOptions{})
	if err != nil {
		return generalError("error approving the apply command", err)
	}

	logs, err := b.client.Applies.Logs(stopCtx, r.Apply.ID)
	if err != nil {
		return generalError("error retrieving logs", err)
	}
	scanner := bufio.NewScanner(logs)

	for scanner.Scan() {
		if b.CLI != nil {
			b.CLI.Output(b.Colorize().Color(scanner.Text()))
		}
	}
	if err := scanner.Err(); err != nil {
		return generalError("error reading logs", err)
	}

	return nil
}

func (b *Remote) checkPolicy(stopCtx, cancelCtx context.Context, op *backend.Operation, r *tfe.Run) error {
	for _, pc := range r.PolicyChecks {
		logs, err := b.client.PolicyChecks.Logs(stopCtx, pc.ID)
		if err != nil {
			return generalError("error retrieving policy check logs", err)
		}
		scanner := bufio.NewScanner(logs)

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
			b.CLI.Output(b.Colorize().Color("\n" + msgPrefix + ":\n"))
		}

		for scanner.Scan() {
			if b.CLI != nil {
				b.CLI.Output(b.Colorize().Color(scanner.Text()))
			}
		}
		if err := scanner.Err(); err != nil {
			return generalError("error reading logs", err)
		}

		pc, err := b.client.PolicyChecks.Read(stopCtx, pc.ID)
		if err != nil {
			return generalError("error retrieving policy check", err)
		}

		switch pc.Status {
		case tfe.PolicyPasses:
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
			Query:       "Do you want to override the failed policy check?",
			Description: "Only 'yes' will be accepted to override.",
		}

		if err = b.confirm(stopCtx, op, opts, r); err != nil {
			return err
		}
	}

	return nil
}

func (b *Remote) confirm(stopCtx context.Context, op *backend.Operation, opts *terraform.InputOpts, r *tfe.Run) error {
	v, err := op.UIIn.Input(opts)
	if err != nil {
		return fmt.Errorf("Error asking %s: %v", opts.Id, err)
	}
	if v != "yes" {
		// Make sure we discard the run.
		err = b.client.Runs.Discard(stopCtx, r.ID, tfe.RunDiscardOptions{})
		if err != nil {
			if op.Destroy {
				return generalError("error disarding destroy", err)
			}
			return generalError("error disarding apply", err)
		}

		// Even if the run was disarding successfully, we still
		// return an error as the apply command was cancelled.
		if op.Destroy {
			return errors.New("Destroy cancelled.")
		}
		return errors.New("Apply cancelled.")
	}

	return nil
}

const applyErrVCSNotSupported = `
Apply not allowed for workspaces with a VCS connection!

A workspace that is connected to a VCS requires the VCS based workflow
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

const applyDefaultHeader = `
[reset][yellow]Running apply in the remote backend. Output will stream here. Pressing Ctrl-C
will stop streaming the logs, but will not stop the apply running remotely.
To view this run in a browser, visit:
https://%s/app/%s/%s/runs/%s[reset]

Waiting for the apply to start...
`
