package local

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func (b *Local) opPlan(
	stopCtx context.Context,
	cancelCtx context.Context,
	op *backend.Operation,
	runningOp *backend.RunningOperation) {

	log.Printf("[INFO] backend/local: starting Plan operation")

	var diags tfdiags.Diagnostics

	if op.PlanFile != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't re-plan a saved plan",
			"The plan command was given a saved plan file as its input. This command generates "+
				"a new plan, and so it requires a configuration directory as its argument.",
		))
		op.ReportResult(runningOp, diags)
		return
	}

	// Local planning requires a config, unless we're planning to destroy.
	if op.PlanMode != plans.DestroyMode && !op.HasConfig() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No configuration files",
			"Plan requires configuration to be present. Planning without a configuration would "+
				"mark everything for destruction, which is normally not what is desired. If you "+
				"would like to destroy everything, run plan with the -destroy option. Otherwise, "+
				"create a Terraform configuration file (.tf file) and try again.",
		))
		op.ReportResult(runningOp, diags)
		return
	}

	if b.ContextOpts == nil {
		b.ContextOpts = new(terraform.ContextOpts)
	}

	// Get our context
	lr, configSnap, opState, ctxDiags := b.localRun(op)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}
	// the state was locked during succesfull context creation; unlock the state
	// when the operation completes
	defer func() {
		diags := op.StateLocker.Unlock()
		if diags.HasErrors() {
			op.View.Diagnostics(diags)
			runningOp.Result = backend.OperationFailure
		}
	}()

	// Since planning doesn't immediately change the persisted state, the
	// resulting state is always just the input state.
	runningOp.State = lr.InputState

	// Perform the plan in a goroutine so we can be interrupted
	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer logging.PanicHandler()
		defer close(doneCh)
		log.Printf("[INFO] backend/local: plan calling Plan")
		plan, planDiags = lr.Core.Plan(lr.Config, lr.InputState, lr.PlanOpts)
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, lr.Core, opState, op.View) {
		// If we get in here then the operation was cancelled, which is always
		// considered to be a failure.
		log.Printf("[INFO] backend/local: plan operation was force-cancelled by interrupt")
		runningOp.Result = backend.OperationFailure
		return
	}
	log.Printf("[INFO] backend/local: plan operation completed")

	diags = diags.Append(planDiags)
	if planDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	// Record whether this plan includes any side-effects that could be applied.
	runningOp.PlanEmpty = !plan.CanApply()

	// Save the plan to disk
	if path := op.PlanOutPath; path != "" {
		if op.PlanOutBackend == nil {
			// This is always a bug in the operation caller; it's not valid
			// to set PlanOutPath without also setting PlanOutBackend.
			diags = diags.Append(fmt.Errorf(
				"PlanOutPath set without also setting PlanOutBackend (this is a bug in Terraform)"),
			)
			op.ReportResult(runningOp, diags)
			return
		}
		plan.Backend = *op.PlanOutBackend

		// We may have updated the state in the refresh step above, but we
		// will freeze that updated state in the plan file for now and
		// only write it if this plan is subsequently applied.
		plannedStateFile := statemgr.PlannedStateUpdate(opState, plan.PriorState)

		// We also include a file containing the state as it existed before
		// we took any action at all, but this one isn't intended to ever
		// be saved to the backend (an equivalent snapshot should already be
		// there) and so we just use a stub state file header in this case.
		// NOTE: This won't be exactly identical to the latest state snapshot
		// in the backend because it's still been subject to state upgrading
		// to make it consumable by the current Terraform version, and
		// intentionally doesn't preserve the header info.
		prevStateFile := &statefile.File{
			State: plan.PrevRunState,
		}

		log.Printf("[INFO] backend/local: writing plan output to: %s", path)
		err := planfile.Create(path, planfile.CreateArgs{
			ConfigSnapshot:       configSnap,
			PreviousRunStateFile: prevStateFile,
			StateFile:            plannedStateFile,
			Plan:                 plan,
			DependencyLocks:      op.DependencyLocks,
		})
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to write plan file",
				fmt.Sprintf("The plan file could not be written: %s.", err),
			))
			op.ReportResult(runningOp, diags)
			return
		}
	}

	// Render the plan
	schemas, moreDiags := lr.Core.Schemas(lr.Config, lr.InputState)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}
	op.View.Plan(plan, schemas)
	op.View.CheckStatusChanges(lr.InputState.CheckResults, plan.Checks)

	// If we've accumulated any warnings along the way then we'll show them
	// here just before we show the summary and next steps. If we encountered
	// errors then we would've returned early at some other point above.
	op.View.Diagnostics(diags)

	if !runningOp.PlanEmpty {
		op.View.PlanNextStep(op.PlanOutPath)
	}
}
