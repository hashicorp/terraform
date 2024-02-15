// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
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
	PlanNextStep(planPath string, genConfigPath string)

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
	outputs, changed, drift, attrs, err := jsonplan.MarshalForRenderer(plan, schemas)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal plan to json: %s", err)
		return
	}

	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.inAutomation,
	}

	jplan := jsonformat.Plan{
		PlanFormatVersion:     jsonplan.FormatVersion,
		ProviderFormatVersion: jsonprovider.FormatVersion,
		OutputChanges:         outputs,
		ResourceChanges:       changed,
		ResourceDrift:         drift,
		ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
		RelevantAttributes:    attrs,
	}

	// Side load some data that we can't extract from the JSON plan.
	var opts []plans.Quality
	if plan.Errored {
		opts = append(opts, plans.Errored)
	} else if !plan.Applyable {
		// FIXME: There might be other reasons for "non-applyable" in future,
		// so maybe we should check plan.Changes.IsEmpty here and use a more
		// generic fallback message if the plan doesn't seem to be empty.
		// There are currently no other cases though, so we'll keep it
		// simple for now.
		opts = append(opts, plans.NoChanges)
	}

	renderer.RenderHumanPlan(jplan, plan.UIMode, opts...)
}

func (v *OperationHuman) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
	// PlannedChange is primarily for machine-readable output in order to
	// get a per-resource-instance change description. We don't use it
	// with OperationHuman because the output of Plan already includes the
	// change details for all resource instances.
}

// PlanNextStep gives the user some next-steps, unless we're running in an
// automation tool which is presumed to provide its own UI for further actions.
func (v *OperationHuman) PlanNextStep(planPath string, genConfigPath string) {
	if v.inAutomation {
		return
	}
	v.view.outputHorizRule()

	if genConfigPath != "" {
		v.view.streams.Printf(
			format.WordWrap(
				"\n"+strings.TrimSpace(fmt.Sprintf(planHeaderGenConfig, genConfigPath)),
				v.view.outputColumns(),
			) + "\n")
	}

	if planPath == "" {
		v.view.streams.Print(
			format.WordWrap(
				"\n"+strings.TrimSpace(planHeaderNoOutput),
				v.view.outputColumns(),
			) + "\n",
		)
	} else {
		v.view.streams.Printf(
			format.WordWrap(
				"\n"+strings.TrimSpace(fmt.Sprintf(planHeaderYesOutput, planPath, planPath)),
				v.view.outputColumns(),
			) + "\n",
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

		if change.Importing != nil {
			cs.Import++
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

		if change.Action != plans.NoOp || !change.Addr.Equal(change.PrevRunAddr) || change.Importing != nil {
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
func (v *OperationJSON) PlanNextStep(planPath string, genConfigPath string) {
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

const planHeaderGenConfig = `
Terraform has generated configuration and written it to %s. Please review the configuration and edit it as necessary before adding it to version control.
`
