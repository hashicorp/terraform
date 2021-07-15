package views

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	if err := v.resourceDrift(plan.PrevRunState, plan.PriorState, schemas); err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(err)
		v.Diagnostics(diags)
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

		if change.Action != plans.NoOp {
			v.view.PlannedChange(json.NewResourceInstanceChange(change))
		}
	}

	v.view.ChangeSummary(cs)
}

func (v *OperationJSON) resourceDrift(oldState, newState *states.State, schemas *terraform.Schemas) error {
	if newState.ManagedResourcesEqual(oldState) {
		// Nothing to do, because we only detect and report drift for managed
		// resource instances.
		return nil
	}
	var changes []*json.ResourceInstanceChange
	for _, ms := range oldState.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				// Drift reporting is only for managed resources
				continue
			}

			provider := rs.ProviderConfig.Provider
			for key, oldIS := range rs.Instances {
				if oldIS.Current == nil {
					// Not interested in instances that only have deposed objects
					continue
				}
				addr := rs.Addr.Instance(key)
				newIS := newState.ResourceInstance(addr)

				schema, _ := schemas.ResourceTypeConfig(
					provider,
					addr.Resource.Resource.Mode,
					addr.Resource.Resource.Type,
				)
				if schema == nil {
					return fmt.Errorf("no schema found for %s (in provider %s)", addr, provider)
				}
				ty := schema.ImpliedType()

				oldObj, err := oldIS.Current.Decode(ty)
				if err != nil {
					return fmt.Errorf("failed to decode previous run data for %s: %s", addr, err)
				}

				var newObj *states.ResourceInstanceObject
				if newIS != nil && newIS.Current != nil {
					newObj, err = newIS.Current.Decode(ty)
					if err != nil {
						return fmt.Errorf("failed to decode refreshed data for %s: %s", addr, err)
					}
				}

				var oldVal, newVal cty.Value
				oldVal = oldObj.Value
				if newObj != nil {
					newVal = newObj.Value
				} else {
					newVal = cty.NullVal(ty)
				}

				if oldVal.RawEquals(newVal) {
					// No drift if the two values are semantically equivalent
					continue
				}

				// We can only detect updates and deletes as drift.
				action := plans.Update
				if newVal.IsNull() {
					action = plans.Delete
				}

				change := &plans.ResourceInstanceChangeSrc{
					Addr: addr,
					ChangeSrc: plans.ChangeSrc{
						Action: action,
					},
				}
				changes = append(changes, json.NewResourceInstanceChange(change))
			}
		}
	}

	// Sort the change structs lexically by address to give stable output
	sort.Slice(changes, func(i, j int) bool { return changes[i].Resource.Addr < changes[j].Resource.Addr })

	for _, change := range changes {
		v.view.ResourceDrift(change)
	}

	return nil
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
