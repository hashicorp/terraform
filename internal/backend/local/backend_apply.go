// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
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
	op *backendrun.Operation,
	runningOp *backendrun.RunningOperation) {
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
			runningOp.Result = backendrun.OperationFailure
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
	// stateHook uses schemas for when it periodically persists state to the
	// persistent storage backend.
	stateHook.Schemas = schemas
	stateHook.PersistInterval = time.Duration(op.StatePersistInterval) * time.Second

	var plan *plans.Plan
	combinedPlanApply := false
	// If we weren't given a plan, then we refresh/plan
	if op.PlanFile == nil {
		combinedPlanApply = true
		// Perform the plan
		log.Printf("[INFO] backend/local: apply calling Plan")
		plan, moreDiags = lr.Core.Plan(lr.Config, lr.InputState, lr.PlanOpts)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			// If Terraform Core generated a partial plan despite the errors
			// then we'll make a best effort to render it. Terraform Core
			// promises that if it returns a non-nil plan along with errors
			// then the plan won't necessarily contain all of the needed
			// actions but that any it does include will be properly-formed.
			// plan.Errored will be true in this case, which our plan
			// renderer can rely on to tailor its messaging.
			if plan != nil && (len(plan.Changes.Resources) != 0 || len(plan.Changes.Outputs) != 0) {
				op.View.Plan(plan, schemas)
			}
			op.ReportResult(runningOp, diags)
			return
		}

		trivialPlan := !plan.Applyable
		hasUI := op.UIOut != nil && op.UIIn != nil
		mustConfirm := hasUI && !op.AutoApprove && !trivialPlan
		op.View.Plan(plan, schemas)

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
			runningOp.Result = backendrun.OperationFailure
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
				runningOp.Result = backendrun.OperationFailure
				return
			}
		} else {
			// If we didn't ask for confirmation from the user, and they have
			// included any failing checks in their configuration, then they
			// will see a very confusing output after the apply operation
			// completes. This is because all the diagnostics from the plan
			// operation will now be shown alongside the diagnostics from the
			// apply operation. For check diagnostics, the plan output is
			// irrelevant and simple noise after the same set of checks have
			// been executed again during the apply stage. As such, we are going
			// to remove all diagnostics marked as check diagnostics at this
			// stage, so we will only show the user the check results from the
			// apply operation.
			//
			// Note, if we did ask for approval then we would have displayed the
			// plan check results at that point which is useful as the user can
			// use them to make a decision about whether to apply the changes.
			// It's just that if we didn't ask for approval then showing the
			// user the checks from the plan alongside the checks from the apply
			// is needlessly confusing.
			var filteredDiags tfdiags.Diagnostics
			for _, diag := range diags {
				if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok && rule.Container.CheckableKind() == addrs.CheckableCheck {
					continue
				}
				filteredDiags = filteredDiags.Append(diag)
			}
			diags = filteredDiags
		}
	} else {
		plan = lr.Plan
		if plan.Errored {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Cannot apply incomplete plan",
				"Terraform encountered an error when generating this plan, so it cannot be applied.",
			))
			op.ReportResult(runningOp, diags)
			return
		}
		for _, change := range plan.Changes.Resources {
			if change.Action != plans.NoOp {
				op.View.PlannedChange(change)
			}
		}
	}

	// Set up our hook for continuous state updates
	stateHook.StateMgr = opState

	applyTimeValues := make(terraform.InputValues, plan.ApplyTimeVariables.Len())

	// In a combined plan/apply run, getting the context already gathers the interactive
	// input, therefore we need to make sure to pass the ephemeral variables to the applyOpts.
	if combinedPlanApply {
		for varName, v := range lr.PlanOpts.SetVariables {
			decl, ok := lr.Config.Module.Variables[varName]
			if !ok {
				continue // This should never happen, but we'll ignore it if it does.
			}

			if v.SourceType == terraform.ValueFromInput && decl.Ephemeral {
				applyTimeValues[varName] = v
			}
		}
	}

	if len(op.Variables) != 0 {
		// Undeclared variables cause warnings during plan, but will show up
		// again here during apply. Their handling is tricky though, because it
		// depends on how they were declared, and is subject to compatibility
		// constraints. Collect any suspect values as we go, and then use the
		// same parsing logic from the plan to generate the diagnostics.
		undeclaredVariables := map[string]backendrun.UnparsedVariableValue{}

		parsedVars, _ := backendrun.ParseVariableValues(op.Variables, lr.Config.Module.Variables)

		for varName := range op.Variables {
			parsedVar, parsed := parsedVars[varName]

			decl, ok := lr.Config.Module.Variables[varName]
			if !ok || !parsed {
				// We'll try to parse this and handle diagnostics for missing
				// variables with ParseUndeclaredVariableValues after.
				undeclaredVariables[varName] = op.Variables[varName]
				continue
			}

			var rng *hcl.Range
			if parsedVar.HasSourceRange() {
				rng = parsedVar.SourceRange.ToHCL().Ptr()
			}

			// If the var is declared as ephemeral in config, go ahead and handle it
			if decl.Ephemeral {
				// Determine whether this is an apply-time variable, i.e. an
				// ephemeral variable that was set (non-null) during the
				// planning phase.
				applyTimeVar := false
				for avName := range plan.ApplyTimeVariables.All() {
					if varName == avName {
						applyTimeVar = true
					}
				}

				// If this isn't an apply-time variable, it's not valid to
				// set it during apply.
				if !applyTimeVar {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Ephemeral variable was not set during planning",
						Detail: fmt.Sprintf(
							"The ephemeral input variable %q was not set during the planning phase, and so must remain unset during the apply phase.",
							varName,
						),
						Subject: rng,
					})
					continue
				}

				// If this is an apply-time variable, the user must supply a
				// value during apply: it can't be null.
				if applyTimeVar && parsedVar.Value.IsNull() {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Ephemeral variable must be set for apply",
						Detail: fmt.Sprintf(
							"The ephemeral input variable %q was set during the planning phase, and so must be set again during the apply phase.",
							varName,
						),
					})
					continue
				}

				// If we get here, we are in possession of a non-null
				// ephemeral apply-time input variable, and need only pass
				// its value on to the ApplyOpts.
				applyTimeValues[varName] = parsedVar
			} else {
				// If a non-ephemeral variable is set differently between plan and apply, we should emit a diagnostic.
				plannedVariableValue, ok := plan.VariableValues[varName]
				if !ok {
					// We'll catch this with ParseUndeclaredVariableValues after
					undeclaredVariables[varName] = op.Variables[varName]
					continue
				}

				plannedVar, err := plannedVariableValue.Decode(cty.DynamicPseudoType)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Could not decode variable value from plan",
						Detail:   fmt.Sprintf("The variable %s could not be decoded from the plan. %s. This is a bug in Terraform, please report it.", varName, err),
						Subject:  rng,
					})
				} else {
					// The user can't override the planned variables, so we
					// error when possible to avoid confusion. If the parsed
					// variables comes from an auto-file however, it's not input
					// directly by the user so we have to ignore it.
					if parsedVar.Value.Equals(plannedVar).False() && parsedVar.SourceType != terraform.ValueFromAutoFile {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Can't change variable when applying a saved plan",
							Detail:   fmt.Sprintf("The variable %s cannot be set using the -var and -var-file options when applying a saved plan file, because a saved plan includes the variable values that were set when it was created. The saved plan specifies %s as the value whereas during apply the value %s was %s. To declare an ephemeral variable which is not saved in the plan file, use ephemeral = true.", varName, tfdiags.CompactValueStr(parsedVar.Value), tfdiags.CompactValueStr(plannedVar), parsedVar.SourceType.DiagnosticLabel()),
							Subject:  rng,
						})
					}
				}
			}

		}
		_, undeclaredDiags := backendrun.ParseUndeclaredVariableValues(undeclaredVariables, map[string]*configs.Variable{})
		// always add hard errors here, and add warnings if we're not in a
		// combined op which just emitted those same warnings already.
		if undeclaredDiags.HasErrors() || !combinedPlanApply {
			diags = diags.Append(undeclaredDiags)
		}

		if diags.HasErrors() {
			op.ReportResult(runningOp, diags)
			return
		}
	}

	// Start the apply in a goroutine so that we can be interrupted.
	var applyState *states.State
	var applyDiags tfdiags.Diagnostics
	doneCh := make(chan struct{})
	go func() {
		defer logging.PanicHandler()
		defer close(doneCh)

		log.Printf("[INFO] backend/local: apply calling Apply")
		applyState, applyDiags = lr.Core.Apply(plan, lr.Config, &terraform.ApplyOpts{
			SetVariables: applyTimeValues,
		})
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
