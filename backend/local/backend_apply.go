package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

func (b *Local) opApply(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {
	log.Printf("[INFO] backend/local: starting Apply operation")

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.Plan == nil && op.Module == nil && !op.Destroy {
		runningOp.Err = fmt.Errorf(strings.TrimSpace(applyErrNoConfig))
		return
	}

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.Module == nil {
		op.Module = module.NewEmptyTree()
	}

	// Setup our count hook that keeps track of resource changes
	countHook := new(CountHook)
	stateHook := new(StateHook)
	if b.ContextOpts == nil {
		b.ContextOpts = new(terraform.ContextOpts)
	}
	old := b.ContextOpts.Hooks
	defer func() { b.ContextOpts.Hooks = old }()
	b.ContextOpts.Hooks = append(b.ContextOpts.Hooks, countHook, stateHook)

	// Get our context
	tfCtx, opState, err := b.context(op)
	if err != nil {
		runningOp.Err = err
		return
	}

	// Setup the state
	runningOp.State = tfCtx.State()

	// If we weren't given a plan, then we refresh/plan
	if op.Plan == nil {
		// If we're refreshing before apply, perform that
		if op.PlanRefresh {
			log.Printf("[INFO] backend/local: apply calling Refresh")
			_, err := tfCtx.Refresh()
			if err != nil {
				runningOp.Err = errwrap.Wrapf("Error refreshing state: {{err}}", err)
				return
			}
		}

		// Perform the plan
		log.Printf("[INFO] backend/local: apply calling Plan")
		plan, err := tfCtx.Plan()
		if err != nil {
			runningOp.Err = errwrap.Wrapf("Error running plan: {{err}}", err)
			return
		}

		dispPlan := format.NewPlan(plan)
		trivialPlan := dispPlan.Empty()
		hasUI := op.UIOut != nil && op.UIIn != nil
		mustConfirm := hasUI && ((op.Destroy && (!op.DestroyForce && !op.AutoApprove)) || (!op.Destroy && !op.AutoApprove && !trivialPlan))
		if mustConfirm {
			var desc, query string
			if op.Destroy {
				if op.Workspace != "default" {
					query = "Do you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you really want to destroy all resources?"
				}
				desc = "Terraform will destroy all your managed infrastructure, as shown above.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			} else {
				if op.Workspace != "default" {
					query = "Do you want to perform these actions in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you want to perform these actions?"
				}
				desc = "Terraform will perform the actions described above.\n" +
					"Only 'yes' will be accepted to approve."
			}

			if !trivialPlan {
				// Display the plan of what we are going to apply/destroy.
				b.renderPlan(dispPlan)
				b.CLI.Output("")
			}

			v, err := op.UIIn.Input(&terraform.InputOpts{
				Id:          "approve",
				Query:       query,
				Description: desc,
			})
			if err != nil {
				runningOp.Err = errwrap.Wrapf("Error asking for approval: {{err}}", err)
				return
			}
			if v != "yes" {
				if op.Destroy {
					runningOp.Err = errors.New("Destroy cancelled.")
				} else {
					runningOp.Err = errors.New("Apply cancelled.")
				}
				return
			}
		}
	}

	// Setup our hook for continuous state updates
	stateHook.State = opState

	// Start the apply in a goroutine so that we can be interrupted.
	var applyState *terraform.State
	var applyErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		_, applyErr = tfCtx.Apply()
		// we always want the state, even if apply failed
		applyState = tfCtx.State()
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState) {
		return
	}

	// Store the final state
	runningOp.State = applyState

	// Persist the state
	if err := opState.WriteState(applyState); err != nil {
		runningOp.Err = b.backupStateForError(applyState, err)
		return
	}
	if err := opState.PersistState(); err != nil {
		runningOp.Err = b.backupStateForError(applyState, err)
		return
	}

	if applyErr != nil {
		runningOp.Err = fmt.Errorf(
			"Error applying plan:\n\n"+
				"%s\n\n"+
				"Terraform does not automatically rollback in the face of errors.\n"+
				"Instead, your Terraform state file has been partially updated with\n"+
				"any resources that successfully completed. Please address the error\n"+
				"above and apply again to incrementally change your infrastructure.",
			multierror.Flatten(applyErr))
		return
	}

	// If we have a UI, output the results
	if b.CLI != nil {
		if op.Destroy {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset][bold][green]\n"+
					"Destroy complete! Resources: %d destroyed.",
				countHook.Removed)))
		} else {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset][bold][green]\n"+
					"Apply complete! Resources: %d added, %d changed, %d destroyed.",
				countHook.Added,
				countHook.Changed,
				countHook.Removed)))
		}

		// only show the state file help message if the state is local.
		if (countHook.Added > 0 || countHook.Changed > 0) && b.StateOutPath != "" {
			b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
				"[reset]\n"+
					"The state of your infrastructure has been saved to the path\n"+
					"below. This state is required to modify and destroy your\n"+
					"infrastructure, so keep it safe. To inspect the complete state\n"+
					"use the `terraform show` command.\n\n"+
					"State path: %s",
				b.StateOutPath)))
		}
	}
}

// backupStateForError is called in a scenario where we're unable to persist the
// state for some reason, and will attempt to save a backup copy of the state
// to local disk to help the user recover. This is a "last ditch effort" sort
// of thing, so we really don't want to end up in this codepath; we should do
// everything we possibly can to get the state saved _somewhere_.
func (b *Local) backupStateForError(applyState *terraform.State, err error) error {
	b.CLI.Error(fmt.Sprintf("Failed to save state: %s\n", err))

	local := &state.LocalState{Path: "errored.tfstate"}
	writeErr := local.WriteState(applyState)
	if writeErr != nil {
		b.CLI.Error(fmt.Sprintf(
			"Also failed to create local state file for recovery: %s\n\n", writeErr,
		))
		// To avoid leaving the user with no state at all, our last resort
		// is to print the JSON state out onto the terminal. This is an awful
		// UX, so we should definitely avoid doing this if at all possible,
		// but at least the user has _some_ path to recover if we end up
		// here for some reason.
		stateBuf := new(bytes.Buffer)
		jsonErr := terraform.WriteState(applyState, stateBuf)
		if jsonErr != nil {
			b.CLI.Error(fmt.Sprintf(
				"Also failed to JSON-serialize the state to print it: %s\n\n", jsonErr,
			))
			return errors.New(stateWriteFatalError)
		}

		b.CLI.Output(stateBuf.String())

		return errors.New(stateWriteConsoleFallbackError)
	}

	return errors.New(stateWriteBackedUpError)
}

const applyErrNoConfig = `
No configuration files found!

Apply requires configuration to be present. Applying without a configuration
would mark everything for destruction, which is normally not what is desired.
If you would like to destroy everything, please run 'terraform destroy' instead
which does not require any configuration files.
`

const stateWriteBackedUpError = `Failed to persist state to backend.

The error shown above has prevented Terraform from writing the updated state
to the configured backend. To allow for recovery, the state has been written
to the file "errored.tfstate" in the current working directory.

Running "terraform apply" again at this point will create a forked state,
making it harder to recover.

To retry writing this state, use the following command:
    terraform state push errored.tfstate
`

const stateWriteConsoleFallbackError = `Failed to persist state to backend.

The errors shown above prevented Terraform from writing the updated state to
the configured backend and from creating a local backup file. As a fallback,
the raw state data is printed above as a JSON object.

To retry writing this state, copy the state data (from the first { to the
last } inclusive) and save it into a local file called errored.tfstate, then
run the following command:
    terraform state push errored.tfstate
`

const stateWriteFatalError = `Failed to save state after apply.

A catastrophic error has prevented Terraform from persisting the state file
or creating a backup. Unfortunately this means that the record of any resources
created during this apply has been lost, and such resources may exist outside
of Terraform's management.

For resources that support import, it is possible to recover by manually
importing each resource using its id from the target system.

This is a serious bug in Terraform and should be reported.
`

const earlyStateWriteErrorFmt = `Error saving current state: %s

Terraform encountered an error attempting to save the state before canceling
the current operation. Once the operation is complete another attempt will be
made to save the final state.
`
