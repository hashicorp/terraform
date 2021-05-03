package local

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states"
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
			op.View.Diagnostics(diags)
			runningOp.Result = backend.OperationFailure
		}
	}()

	// TEMP: We'll keep a snapshot of the original state, prior to any
	// refreshing as a temporary way to approximate detecting and reporting
	// changes during refresh, until we've integrated that properly into
	// the plan model.
	initialState := tfCtx.State().DeepCopy()

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

	refreshFoundChanges := tempRefreshReporting(initialState, plan.State, op.View)

	// Record whether this plan includes any side-effects that could be applied.
	runningOp.PlanEmpty = plan.Changes.Empty() && !refreshFoundChanges

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
		op.View.Diagnostics(diags)
		return
	}

	// Render the plan
	op.View.Plan(plan, plan.State, tfCtx.Schemas())

	// If we've accumulated any warnings along the way then we'll show them
	// here just before we show the summary and next steps. If we encountered
	// errors then we would've returned early at some other point above.
	op.View.Diagnostics(diags)

	op.View.PlanNextStep(op.PlanOutPath)
}

// tempRefreshReporting is a temporary placeholder for what will hopefully be
// a better-integrated and more user-friendly report of any changes detected
// as a result of refreshing existing managed resources.
//
// For now it just prints out a developer-oriented summary of what it found
// and returns true only if there is at least one resource instance difference
// which a user might therefore want to save as part of a new state snapshot.
func tempRefreshReporting(baseState, priorState *states.State, view views.Operation) bool {
	if baseState == nil || priorState == nil {
		return false
	}
	changes := false
	for _, bms := range baseState.Modules {
		for _, brs := range bms.Resources {
			if brs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue // only managed resources can "drift"
			}
			prs := priorState.Resource(brs.Addr)
			if prs == nil {
				// Refreshing detected that the remote object has been deleted
				var diags tfdiags.Diagnostics
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"(Prototype-only refresh result reporting)",
					fmt.Sprintf("Apparently %s has been deleted outside of Terraform.", brs.Addr),
				))
				view.Diagnostics(diags)
				changes = true
				continue
			}
			if !prs.Equal(brs) {
				// Refreshing detected that the remote object has changed.
				var diags tfdiags.Diagnostics
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"(Prototype-only refresh result reporting)",
					fmt.Sprintf("Apparently %s has been changed outside of Terraform.", brs.Addr),
				))
				view.Diagnostics(diags)
				changes = true
				continue
			}
		}
	}
	return changes
}
