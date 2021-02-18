package local

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
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
	if !op.Destroy && !op.HasConfig() {
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
	tfCtx, configSnap, opState, ctxDiags := b.context(op)
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
			op.ShowDiagnostics(diags)
			runningOp.Result = backend.OperationFailure
		}
	}()

	runningOp.State = tfCtx.State()

	// Perform the plan in a goroutine so we can be interrupted
	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		log.Printf("[INFO] backend/local: plan calling Plan")
		plan, planDiags = tfCtx.Plan()
	}()

	if b.opWait(doneCh, stopCtx, cancelCtx, tfCtx, opState, op.View) {
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
	runningOp.PlanEmpty = plan.Changes.Empty()

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
		plannedStateFile := statemgr.PlannedStateUpdate(opState, plan.State)

		log.Printf("[INFO] backend/local: writing plan output to: %s", path)
		err := planfile.Create(path, configSnap, plannedStateFile, plan)
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

	// Perform some output tasks
	if runningOp.PlanEmpty {
		op.View.PlanNoChanges()

		// Even if there are no changes, there still could be some warnings
		op.ShowDiagnostics(diags)
		return
	}

	// Render the plan
	op.View.Plan(plan, plan.State, tfCtx.Schemas())

	// If we've accumulated any warnings along the way then we'll show them
	// here just before we show the summary and next steps. If we encountered
	// errors then we would've returned early at some other point above.
	op.ShowDiagnostics(diags)

	op.View.PlanNextStep(op.PlanOutPath)
}
