package views

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

type Operation interface {
	Interrupted()
	FatalInterrupt()
	Stopping()
	Cancelled(planMode plans.Mode)

	EmergencyDumpState(stateFile *statefile.File) error

	PlanNoChanges()
	Plan(plan *plans.Plan, baseState *states.State, schemas *terraform.Schemas)
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

const fatalInterrupt = `
Two interrupts received. Exiting immediately. Note that data loss may have occurred.
`

const interrupted = `
Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...
`

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

func (v *OperationHuman) PlanNoChanges() {
	v.view.streams.Println("\n" + v.view.colorize.Color(strings.TrimSpace(planNoChanges)))
	v.view.streams.Println("\n" + strings.TrimSpace(format.WordWrap(planNoChangesDetail, v.view.outputColumns())))
}

func (v *OperationHuman) Plan(plan *plans.Plan, baseState *states.State, schemas *terraform.Schemas) {
	renderPlan(plan, baseState, schemas, v.view)
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

const planNoChanges = `
[reset][bold][green]No changes. Infrastructure is up-to-date.[reset][green]
`

const planNoChangesDetail = `
This means that Terraform did not detect any differences between your configuration and the remote system(s). As a result, there are no actions to take.
`

const planHeaderNoOutput = `
Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
`

const planHeaderYesOutput = `
Saved the plan to: %s

To perform exactly these actions, run the following command to apply:
    terraform apply %q
`
