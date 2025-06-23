// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonlist"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func NewQueryOperation(vt arguments.ViewType, inAutomation bool, view *View) Operation {
	switch vt {
	case arguments.ViewHuman:
		return &QueryOperationHuman{view: view, inAutomation: inAutomation}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type QueryOperationHuman struct {
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

var _ Operation = (*QueryOperationHuman)(nil)

func (v *QueryOperationHuman) Interrupted() {
	v.view.streams.Println(format.WordWrap(interrupted, v.view.outputColumns()))
}

func (v *QueryOperationHuman) FatalInterrupt() {
	v.view.streams.Eprintln(format.WordWrap(fatalInterrupt, v.view.errorColumns()))
}

func (v *QueryOperationHuman) Stopping() {
	v.view.streams.Println("Stopping operation...")
}

func (v *QueryOperationHuman) Cancelled(planMode plans.Mode) {
	v.view.streams.Println("Query cancelled.")
}

func (v *QueryOperationHuman) EmergencyDumpState(stateFile *statefile.File) error {
	return nil
}

func (v *QueryOperationHuman) Plan(plan *plans.Plan, schemas *terraform.Schemas) {
	results, err := jsonlist.MarshalForRenderer(plan, schemas)
	if err != nil {
		v.view.streams.Eprintf("Failed to marshal query results to json: %s", err)
		return
	}

	// TODO: Update to render list results
	renderer := jsonformat.Renderer{
		Colorize:            v.view.colorize,
		Streams:             v.view.streams,
		RunningInAutomation: v.inAutomation,
	}

	jplan := jsonformat.Plan{
		PlanFormatVersion:     jsonplan.FormatVersion,
		ProviderFormatVersion: jsonprovider.FormatVersion,
		QueryResults:          results,
		ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
	}

	renderer.RenderHumanList(jplan)
}

func (v *QueryOperationHuman) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
}

func (v *QueryOperationHuman) PlanNextStep(planPath string, genConfigPath string) {
}

func (v *QueryOperationHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type QueryOperationJSON struct {
	view *JSONView
}

var _ Operation = (*QueryOperationJSON)(nil)

func (v *QueryOperationJSON) Interrupted() {
	v.view.Log(interrupted)
}

func (v *QueryOperationJSON) FatalInterrupt() {
	v.view.Log(fatalInterrupt)
}

func (v *QueryOperationJSON) Stopping() {
	v.view.Log("Stopping operation...")
}

func (v *QueryOperationJSON) Cancelled(planMode plans.Mode) {
	v.view.Log("Query cancelled")
}

func (v *QueryOperationJSON) EmergencyDumpState(stateFile *statefile.File) error {
	return nil
}

func (v *QueryOperationJSON) Plan(plan *plans.Plan, schemas *terraform.Schemas) {
	// TODO: log operation updates as structured logging
	for _, query := range plan.Changes.Queries {
		v.view.QueryResult(json.NewQueryResults(query))
	}

}

func (v *QueryOperationJSON) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
}

func (v *QueryOperationJSON) PlanNextStep(planPath string, genConfigPath string) {
}

func (v *QueryOperationJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
