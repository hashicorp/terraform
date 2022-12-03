package local

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// test hook called between plan+apply during opApply
var testHookStopPlanApply func()

func (b *Local) opApply(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {
	log.Printf("[INFO] backend/local: starting Apply operation")

	var diags, moreDiags tfdiags.Diagnostics

	// If we have a nil module at this point, then set it to an empty tree
	// to avoid any potential crashes.
	if op.PlanFile == nil && op.PlanMode != plans.DestroyMode && !op.HasConfig() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files",
			"Apply requires configuration to be present. Applying without a configuration "+
				"would mark everything for destruction, which is normally not what is desired. "+
				"If you would like to destroy everything, run 'terraform destroy' instead.",
		))
		op.ReportResult(runningOp, diags)
		return
	}

	stateHook := new(StateHook)
	op.Hooks = append(op.Hooks, stateHook)

	// Get our context
	lr, _, opState, contextDiags := b.localRun(op)
	diags = diags.Append(contextDiags)
	if contextDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}
	// the state was locked during successful context creation; unlock the state
	// when the operation completes
	defer func() {
		diags := op.StateLocker.Unlock()
		if diags.HasErrors() {
			op.View.Diagnostics(diags)
			runningOp.Result = backend.OperationFailure
		}
	}()

	// We'll start off with our result being the input state, and replace it
	// with the result state only if we eventually complete the apply
	// operation.
	runningOp.State = lr.InputState

	schemas, moreDiags := lr.Core.Schemas(lr.Config, lr.InputState)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	var plan *plans.Plan
	// If we weren't given a plan, then we refresh/plan
	if op.PlanFile == nil {
		// Perform the plan
		log.Printf("[INFO] backend/local: apply calling Plan")
		plan, moreDiags = lr.Core.Plan(lr.Config, lr.InputState, lr.PlanOpts)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			op.ReportResult(runningOp, diags)
			return
		}

		trivialPlan := !plan.CanApply()
		hasUI := op.UIOut != nil && op.UIIn != nil
		mustConfirm := hasUI && !op.AutoApprove && !trivialPlan
		op.View.Plan(plan, schemas)
		op.View.CheckStatusChanges(lr.InputState.CheckResults, plan.Checks)

		if testHookStopPlanApply != nil {
			testHookStopPlanApply()
		}

		// Check if we've been stopped before going through confirmation, or
		// skipping confirmation in the case of -auto-approve.
		// This can currently happen if a single stop request was received
		// during the final batch of resource plan calls, so no operations were
		// forced to abort, and no errors were returned from Plan.
		if stopCtx.Err() != nil {
			diags = diags.Append(errors.New("execution halted"))
			runningOp.Result = backend.OperationFailure
			op.ReportResult(runningOp, diags)
			return
		}

		if mustConfirm {
			var desc, query string
			switch op.PlanMode {
			case plans.DestroyMode:
				if op.Workspace != "default" {
					query = "Do you really want to destroy all resources in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you really want to destroy all resources?"
				}
				desc = "Terraform will destroy all your managed infrastructure, as shown above.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			case plans.RefreshOnlyMode:
				if op.Workspace != "default" {
					query = "Would you like to update the Terraform state for \"" + op.Workspace + "\" to reflect these detected changes?"
				} else {
					query = "Would you like to update the Terraform state to reflect these detected changes?"
				}
				desc = "Terraform will write these changes to the state without modifying any real infrastructure.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm."
			default:
				if op.Workspace != "default" {
					query = "Do you want to perform these actions in workspace \"" + op.Workspace + "\"?"
				} else {
					query = "Do you want to perform these actions?"
				}
				desc = "Terraform will perform the actions described above.\n" +
					"Only 'yes' will be accepted to approve."
			}

			// We'll show any accumulated warnings before we display the prompt,
			// so the user can consider them when deciding how to answer.
			if len(diags) > 0 {
				op.View.Diagnostics(diags)
				diags = nil // reset so we won't show the same diagnostics again later
			}

			v, err := op.UIIn.Input(stopCtx, &terraform.InputOpts{
				Id:          "approve",
				Query:       "\n" + query,
				Description: desc,
			})
			if err != nil {
				diags = diags.Append(fmt.Errorf("error asking for approval: %w", err))
				op.ReportResult(runningOp, diags)
				return
			}
			if v != "yes" {
				op.View.Cancelled(op.PlanMode)
				runningOp.Result = backend.OperationFailure
				return
			}
		}
	} else {
		plan = lr.Plan
		for _, change := range plan.Changes.Resources {
			if change.Action != plans.NoOp {
				op.View.PlannedChange(change)
			}
		}
	}

	// Set up our hook for continuous state updates
	stateHook.StateMgr = opState

	// Start the apply in a goroutine so that we can be interrupted.
	var applyState *states.State
	var applyDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer logging.PanicHandler()
		defer close(doneCh)
		log.Printf("[INFO] backend/local: apply calling Apply")
		applyState, applyDiags = lr.Core.Apply(plan, lr.Config)
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, lr.Core, opState, op.View) {
		return
	}
	diags = diags.Append(applyDiags)

	// Even on error with an empty state, the state value should not be nil.
	// Return early here to prevent corrupting any existing state.
	if diags.HasErrors() && applyState == nil {
		log.Printf("[ERROR] backend/local: apply returned nil state")
		op.ReportResult(runningOp, diags)
		return
	}
	op.View.CheckStatusChanges(lr.InputState.CheckResults, applyState.CheckResults)

	// Store the final state
	runningOp.State = applyState
	err := statemgr.WriteAndPersist(opState, applyState, schemas)
	if err != nil {
		// Export the state file from the state manager and assign the new
		// state. This is needed to preserve the existing serial and lineage.
		stateFile := statemgr.Export(opState)
		if stateFile == nil {
			stateFile = &statefile.File{}
		}
		stateFile.State = applyState

		diags = diags.Append(b.backupStateForError(stateFile, err, op.View))
		op.ReportResult(runningOp, diags)
		return
	}

	if applyDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	// If we've accumulated any warnings along the way then we'll show them
	// here just before we show the summary and next steps. If we encountered
	// errors then we would've returned early at some other point above.
	op.View.Diagnostics(diags)
}

// backupStateForError is called in a scenario where we're unable to persist the
// state for some reason, and will attempt to save a backup copy of the state
// to local disk to help the user recover. This is a "last ditch effort" sort
// of thing, so we really don't want to end up in this codepath; we should do
// everything we possibly can to get the state saved _somewhere_.
func (b *Local) backupStateForError(stateFile *statefile.File, err error, view views.Operation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to save state",
		fmt.Sprintf("Error saving state: %s", err),
	))

	local := statemgr.NewFilesystem("errored.tfstate")
	writeErr := local.WriteStateForMigration(stateFile, true)
	if writeErr != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create local state file",
			fmt.Sprintf("Error creating local state file for recovery: %s", writeErr),
		))

		// To avoid leaving the user with no state at all, our last resort
		// is to print the JSON state out onto the terminal. This is an awful
		// UX, so we should definitely avoid doing this if at all possible,
		// but at least the user has _some_ path to recover if we end up
		// here for some reason.
		if dumpErr := view.EmergencyDumpState(stateFile); dumpErr != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize state",
				fmt.Sprintf(stateWriteFatalErrorFmt, dumpErr),
			))
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to persist state to backend",
			stateWriteConsoleFallbackError,
		))
		return diags
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Failed to persist state to backend",
		stateWriteBackedUpError,
	))

	return diags
}

const stateWriteBackedUpError = `The error shown above has prevented Terraform from writing the updated state to the configured backend. To allow for recovery, the state has been written to the file "errored.tfstate" in the current working directory.

Running "terraform apply" again at this point will create a forked state, making it harder to recover.

To retry writing this state, use the following command:
    terraform state push errored.tfstate
`

const stateWriteConsoleFallbackError = `The errors shown above prevented Terraform from writing the updated state to
the configured backend and from creating a local backup file. As a fallback,
the raw state data is printed above as a JSON object.

To retry writing this state, copy the state data (from the first { to the last } inclusive) and save it into a local file called errored.tfstate, then run the following command:
    terraform state push errored.tfstate
`

const stateWriteFatalErrorFmt = `Failed to save state after apply.

Error serializing state: %s

A catastrophic error has prevented Terraform from persisting the state file or creating a backup. Unfortunately this means that the record of any resources created during this apply has been lost, and such resources may exist outside of Terraform's management.

For resources that support import, it is possible to recover by manually importing each resource using its id from the target system.

This is a serious bug in Terraform and should be reported.
`
