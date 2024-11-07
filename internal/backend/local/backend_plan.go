// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/genconfig"
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
	op *backendrun.Operation,
	runningOp *backendrun.RunningOperation) {

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

	if len(op.GenerateConfigOut) > 0 {
		if op.PlanMode != plans.NormalMode {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid generate-config-out flag",
				"Config can only be generated during a normal plan operation, and not during a refresh-only or destroy plan."))
			op.ReportResult(runningOp, diags)
			return
		}

		diags = diags.Append(genconfig.ValidateTargetFile(op.GenerateConfigOut))
		if diags.HasErrors() {
			op.ReportResult(runningOp, diags)
			return
		}
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
			runningOp.Result = backendrun.OperationFailure
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
		runningOp.Result = backendrun.OperationFailure
		return
	}
	log.Printf("[INFO] backend/local: plan operation completed")

	// NOTE: We intentionally don't stop here on errors because we always want
	// to try to present a partial plan report and, if the user chose to,
	// generate a partial saved plan file for external analysis.
	diags = diags.Append(planDiags)

	// Even if there are errors we need to handle anything that may be
	// contained within the plan, so only exit if there is no data at all.
	if plan == nil {
		runningOp.PlanEmpty = true
		op.ReportResult(runningOp, diags)
		return
	}

	// Record whether this plan includes any side-effects that could be applied.
	runningOp.PlanEmpty = !plan.Applyable

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

	// Render the plan, if we produced one.
	// (This might potentially be a partial plan with Errored set to true)
	schemas, moreDiags := lr.Core.Schemas(lr.Config, lr.InputState)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	// Write out any generated config, before we render the plan.
	wroteConfig, moreDiags := maybeWriteGeneratedConfig(plan, op.GenerateConfigOut)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		op.ReportResult(runningOp, diags)
		return
	}

	op.View.Plan(plan, schemas)

	// If we've accumulated any diagnostics along the way then we'll show them
	// here just before we show the summary and next steps. This can potentially
	// include errors, because we intentionally try to show a partial plan
	// above even if Terraform Core encountered an error partway through
	// creating it.
	op.ReportResult(runningOp, diags)

	if !runningOp.PlanEmpty {
		if wroteConfig {
			op.View.PlanNextStep(op.PlanOutPath, op.GenerateConfigOut)
		} else {
			op.View.PlanNextStep(op.PlanOutPath, "")
		}
	}
}

func maybeWriteGeneratedConfig(plan *plans.Plan, out string) (wroteConfig bool, diags tfdiags.Diagnostics) {
	if genconfig.ShouldWriteConfig(out) {
		diags := genconfig.ValidateTargetFile(out)
		if diags.HasErrors() {
			return false, diags
		}

		var writer io.Writer
		for _, c := range plan.Changes.Resources {
			change := genconfig.Change{
				Addr:            c.Addr.String(),
				GeneratedConfig: c.GeneratedConfig,
			}
			if c.Importing != nil {
				change.ImportID = c.Importing.ID
			}

			var moreDiags tfdiags.Diagnostics
			writer, wroteConfig, moreDiags = change.MaybeWriteConfig(writer, out)
			if moreDiags.HasErrors() {
				return false, diags.Append(moreDiags)
			}
		}
	}

	if wroteConfig {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Config generation is experimental",
			"Generating configuration during import is currently experimental, and the generated configuration format may change in future versions."))
	}

	return wroteConfig, diags
}
