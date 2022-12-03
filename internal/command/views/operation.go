package views

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Operation interface {
	Interrupted()
	FatalInterrupt()
	Stopping()
	Cancelled(planMode plans.Mode)

	EmergencyDumpState(stateFile *statefile.File) error

	PlannedChange(change *plans.ResourceInstanceChangeSrc)
	Plan(plan *plans.Plan, schemas *terraform.Schemas)
	PlanNextStep(planPath string)

	CheckStatusChanges(old, new *states.CheckResults)
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewOperation(vt arguments.ViewType, inAutomation bool, view *View) Operation {
	switch vt {
	case arguments.ViewHuman:
		return &OperationHuman{view: view, inAutomation: inAutomation}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type OperationHuman struct {
	view *View

	// inAutomation indicates that commands are being run by an
	// automated system rather than directly at a command prompt.
	//
	// This is a hint not to produce messages that expect that a user can
	// run a follow-up command, perhaps because Terraform is running in
	// some sort of workflow automation tool that abstracts away the
	// exact commands that are being run.
	inAutomation bool
}

var _ Operation = (*OperationHuman)(nil)

func (v *OperationHuman) Interrupted() {
	v.view.streams.Println(format.WordWrap(interrupted, v.view.outputColumns()))
}

func (v *OperationHuman) FatalInterrupt() {
	v.view.streams.Eprintln(format.WordWrap(fatalInterrupt, v.view.errorColumns()))
}

func (v *OperationHuman) Stopping() {
	v.view.streams.Println("Stopping operation...")
}

func (v *OperationHuman) Cancelled(planMode plans.Mode) {
	switch planMode {
	case plans.DestroyMode:
		v.view.streams.Println("Destroy cancelled.")
	default:
		v.view.streams.Println("Apply cancelled.")
	}
}

func (v *OperationHuman) EmergencyDumpState(stateFile *statefile.File) error {
	stateBuf := new(bytes.Buffer)
	jsonErr := statefile.Write(stateFile, stateBuf)
	if jsonErr != nil {
		return jsonErr
	}
	v.view.streams.Eprintln(stateBuf)
	return nil
}

func (v *OperationHuman) Plan(plan *plans.Plan, schemas *terraform.Schemas) {
	renderPlan(plan, schemas, v.view)
}

func (v *OperationHuman) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
	// PlannedChange is primarily for machine-readable output in order to
	// get a per-resource-instance change description. We don't use it
	// with OperationHuman because the output of Plan already includes the
	// change details for all resource instances.
}

// PlanNextStep gives the user some next-steps, unless we're running in an
// automation tool which is presumed to provide its own UI for further actions.
func (v *OperationHuman) PlanNextStep(planPath string) {
	if v.inAutomation {
		return
	}
	v.view.outputHorizRule()

	if planPath == "" {
		v.view.streams.Print(
			"\n" + strings.TrimSpace(format.WordWrap(planHeaderNoOutput, v.view.outputColumns())) + "\n",
		)
	} else {
		v.view.streams.Printf(
			"\n"+strings.TrimSpace(format.WordWrap(planHeaderYesOutput, v.view.outputColumns()))+"\n",
			planPath, planPath,
		)
	}
}

func (v *OperationHuman) CheckStatusChanges(old, new *states.CheckResults) {
	type UICheckResult struct {
		Module          addrs.ModuleInstance
		NewStatus       checks.Status
		Message         string
		FailureMessages []string
	}
	var uiResults []UICheckResult

	// statusPairs tracks all of the checkable objects that we can see
	// in at least one of "old" and "new", capturing both the old and new
	// status together for each one so that we can more easily produce
	// a summary of how the statuses have changed.
	statusPairs := addrs.MakeMap[addrs.Checkable, [2]checks.Status]()
	if new != nil {
		for _, elem := range new.ConfigResults.Elems {
			for _, elem := range elem.Value.ObjectResults.Elems {
				oldResult := old.GetObjectResult(elem.Key)
				newResult := elem.Value
				var statusPair [2]checks.Status
				if oldResult != nil {
					statusPair[0] = oldResult.Status
				}
				statusPair[1] = newResult.Status
				statusPairs.Put(elem.Key, statusPair)
			}
		}
	}
	if old != nil {
		for _, elem := range old.ConfigResults.Elems {
			for _, elem := range elem.Value.ObjectResults.Elems {
				if statusPairs.Has(elem.Key) {
					continue
				}
				oldResult := elem.Value
				newResult := new.GetObjectResult(elem.Key)
				var statusPair [2]checks.Status
				statusPair[0] = oldResult.Status
				if newResult != nil {
					statusPair[1] = newResult.Status
				}
				statusPairs.Put(elem.Key, statusPair)
			}
		}
	}

	for _, elem := range statusPairs.Elems {
		objAddr := elem.Key
		modAddr := objAddr.ContainingModuleInstance()
		statusPair := elem.Value
		var newResult *states.CheckResultObject
		if new != nil {
			newResult = new.GetObjectResult(objAddr) // might still be nil if newStatus is StatusUnknown
		}
		oldStatus, newStatus := statusPair[0], statusPair[1]

		switch newStatus {
		case checks.StatusPass:
			switch oldStatus {
			case checks.StatusFail:
				uiResults = append(uiResults, UICheckResult{
					Module:    modAddr,
					NewStatus: newStatus,
					Message:   fmt.Sprintf("%s has been fixed", objAddr),
				})
			}
		case checks.StatusFail:
			switch oldStatus {
			case checks.StatusFail:
				uiResults = append(uiResults, UICheckResult{
					Module:          modAddr,
					NewStatus:       newStatus,
					Message:         fmt.Sprintf("%s is still failing", objAddr),
					FailureMessages: newResult.FailureMessages,
				})
			default:
				uiResults = append(uiResults, UICheckResult{
					Module:          modAddr,
					NewStatus:       newStatus,
					Message:         fmt.Sprintf("%s failed", objAddr),
					FailureMessages: newResult.FailureMessages,
				})
			}
		case checks.StatusError:
			uiResults = append(uiResults, UICheckResult{
				Module:    modAddr,
				NewStatus: newStatus,
				Message:   fmt.Sprintf("%s has invalid conditions, so has not been completely checked", objAddr),
				// We might still have some failure messages if we had a
				// mixture of both failing and errored conditions.
				FailureMessages: newResult.FailureMessages,
			})
		case checks.StatusUnknown:
			// NOTE: newResult might be nil in this branch!

			switch oldStatus {
			case checks.StatusFail:
				uiResults = append(uiResults, UICheckResult{
					Module:    modAddr,
					NewStatus: newStatus,
					Message:   fmt.Sprintf("%s was failing and will be re-checked after apply", objAddr),
				})
			}
		}
	}

	// We sort first by module to group together all of the results for
	// a particular module, and then by message within each module.
	//
	// TODO: The goal of grouping by module here is so that we can include
	// group headings to avoid repeating the same long absolute check address
	// multiple times and instead just show module-local addresses in our
	// messages, but we don't yet have a way to extract just the local part
	// of a checkable address so we're redundantly including the module
	// address in the status messages too.
	sort.Slice(uiResults, func(i, j int) bool {
		if !uiResults[i].Module.Equal(uiResults[j].Module) {
			return uiResults[i].Module.Less(uiResults[j].Module)
		}
		return uiResults[i].Message < uiResults[j].Message
	})

	if len(uiResults) > 0 {
		v.view.streams.Println("\nStatus of checks:")
		for _, uiResult := range uiResults {
			bullet := "-"
			switch uiResult.NewStatus {
			case checks.StatusFail, checks.StatusError:
				bullet = v.view.colorize.Color("[red]-")
			case checks.StatusPass:
				bullet = v.view.colorize.Color("[green]-")
			case checks.StatusUnknown:
				bullet = v.view.colorize.Color("[yellow]-")
			}
			v.view.streams.Printf(
				"%s %s\n",
				bullet,
				uiResult.Message,
			)
			for _, msg := range uiResult.FailureMessages {
				v.view.streams.Printf(
					"    * %s\n",
					msg,
				)
			}
		}
	}
}

func (v *OperationHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type OperationJSON struct {
	view *JSONView
}

var _ Operation = (*OperationJSON)(nil)

func (v *OperationJSON) Interrupted() {
	v.view.Log(interrupted)
}

func (v *OperationJSON) FatalInterrupt() {
	v.view.Log(fatalInterrupt)
}

func (v *OperationJSON) Stopping() {
	v.view.Log("Stopping operation...")
}

func (v *OperationJSON) Cancelled(planMode plans.Mode) {
	switch planMode {
	case plans.DestroyMode:
		v.view.Log("Destroy cancelled")
	default:
		v.view.Log("Apply cancelled")
	}
}

func (v *OperationJSON) EmergencyDumpState(stateFile *statefile.File) error {
	stateBuf := new(bytes.Buffer)
	jsonErr := statefile.Write(stateFile, stateBuf)
	if jsonErr != nil {
		return jsonErr
	}
	v.view.StateDump(stateBuf.String())
	return nil
}

// Log a change summary and a series of "planned" messages for the changes in
// the plan.
func (v *OperationJSON) Plan(plan *plans.Plan, schemas *terraform.Schemas) {
	for _, dr := range plan.DriftedResources {
		// In refresh-only mode, we output all resources marked as drifted,
		// including those which have moved without other changes. In other plan
		// modes, move-only changes will be included in the planned changes, so
		// we skip them here.
		if dr.Action != plans.NoOp || plan.UIMode == plans.RefreshOnlyMode {
			v.view.ResourceDrift(json.NewResourceInstanceChange(dr))
		}
	}

	cs := &json.ChangeSummary{
		Operation: json.OperationPlanned,
	}
	for _, change := range plan.Changes.Resources {
		if change.Action == plans.Delete && change.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
			// Avoid rendering data sources on deletion
			continue
		}
		switch change.Action {
		case plans.Create:
			cs.Add++
		case plans.Delete:
			cs.Remove++
		case plans.Update:
			cs.Change++
		case plans.CreateThenDelete, plans.DeleteThenCreate:
			cs.Add++
			cs.Remove++
		}

		if change.Action != plans.NoOp || !change.Addr.Equal(change.PrevRunAddr) {
			v.view.PlannedChange(json.NewResourceInstanceChange(change))
		}
	}

	v.view.ChangeSummary(cs)

	var rootModuleOutputs []*plans.OutputChangeSrc
	for _, output := range plan.Changes.Outputs {
		if !output.Addr.Module.IsRoot() {
			continue
		}
		rootModuleOutputs = append(rootModuleOutputs, output)
	}
	if len(rootModuleOutputs) > 0 {
		v.view.Outputs(json.OutputsFromChanges(rootModuleOutputs))
	}
}

func (v *OperationJSON) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
	if change.Action == plans.Delete && change.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
		// Avoid rendering data sources on deletion
		return
	}
	v.view.PlannedChange(json.NewResourceInstanceChange(change))
}

// PlanNextStep does nothing for the JSON view as it is a hook for user-facing
// output only applicable to human-readable UI.
func (v *OperationJSON) PlanNextStep(planPath string) {
}

func (v *OperationJSON) CheckStatusChanges(old, new *states.CheckResults) {
}

func (v *OperationJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

const fatalInterrupt = `
Two interrupts received. Exiting immediately. Note that data loss may have occurred.
`

const interrupted = `
Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...
`

const planHeaderNoOutput = `
Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
`

const planHeaderYesOutput = `
Saved the plan to: %s

To perform exactly these actions, run the following command to apply:
    terraform apply %q
`
