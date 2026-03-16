// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
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
	// The hook for individual query blocks do not display any output when the results are empty,
	// so we will display a grouped warning message here for the empty queries.
	emptyBlocks := []string{}
	for _, query := range plan.Changes.Queries {
		pSchema := schemas.ProviderSchema(query.ProviderAddr.Provider)
		addr := query.Addr
		schema := pSchema.ListResourceTypes[addr.Resource.Resource.Type]

		results, err := query.Decode(schema)
		if err != nil {
			v.view.streams.Eprintln(err)
			continue
		}

		data := results.Results.Value.GetAttr("data")
		if data.LengthInt() == 0 {
			emptyBlocks = append(emptyBlocks, addr.String())
		}

	}

	if len(emptyBlocks) > 0 {
		msg := fmt.Sprintf(v.view.colorize.Color("[bold][yellow]Warning:[reset][bold] list block(s) [%s] returned 0 results.\n"), strings.Join(emptyBlocks, ", "))
		v.view.streams.Println(format.WordWrap(msg, v.view.outputColumns()))
	}
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
}

func (v *QueryOperationJSON) PlannedChange(change *plans.ResourceInstanceChangeSrc) {
}

func (v *QueryOperationJSON) PlanNextStep(planPath string, genConfigPath string) {
}

func (v *QueryOperationJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
