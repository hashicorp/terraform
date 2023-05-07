// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The Apply view is used for the apply command.
type Apply interface {
	ResourceCount(stateOutPath string)
	Outputs(outputValues map[string]*states.OutputValue)

	Operation() Operation
	Hooks() []terraform.Hook

	Diagnostics(diags tfdiags.Diagnostics)
	HelpPrompt()
}

// NewApply returns an initialized Apply implementation for the given ViewType.
func NewApply(vt arguments.ViewType, destroy bool, view *View) Apply {
	switch vt {
	case arguments.ViewJSON:
		return &ApplyJSON{
			view:      NewJSONView(view),
			destroy:   destroy,
			countHook: &countHook{},
		}
	case arguments.ViewHuman:
		return &ApplyHuman{
			view:         view,
			destroy:      destroy,
			inAutomation: view.RunningInAutomation(),
			countHook:    &countHook{},
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The ApplyHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type ApplyHuman struct {
	view *View

	destroy      bool
	inAutomation bool

	countHook *countHook
}

var _ Apply = (*ApplyHuman)(nil)

func (v *ApplyHuman) ResourceCount(stateOutPath string) {
	if v.destroy {
		v.view.streams.Printf(
			v.view.colorize.Color("[reset][bold][green]\nDestroy complete! Resources: %d destroyed.\n"),
			v.countHook.Removed,
		)
	} else if v.countHook.Imported > 0 {
		v.view.streams.Printf(
			v.view.colorize.Color("[reset][bold][green]\nApply complete! Resources: %d imported, %d added, %d changed, %d destroyed.\n"),
			v.countHook.Imported,
			v.countHook.Added,
			v.countHook.Changed,
			v.countHook.Removed,
		)
	} else {
		v.view.streams.Printf(
			v.view.colorize.Color("[reset][bold][green]\nApply complete! Resources: %d added, %d changed, %d destroyed.\n"),
			v.countHook.Added,
			v.countHook.Changed,
			v.countHook.Removed,
		)
	}
	if (v.countHook.Added > 0 || v.countHook.Changed > 0) && stateOutPath != "" {
		v.view.streams.Printf("\n%s\n\n", format.WordWrap(stateOutPathPostApply, v.view.outputColumns()))
		v.view.streams.Printf("State path: %s\n", stateOutPath)
	}
}

func (v *ApplyHuman) Outputs(outputValues map[string]*states.OutputValue) {
	if len(outputValues) > 0 {
		v.view.streams.Print(v.view.colorize.Color("[reset][bold][green]\nOutputs:\n\n"))
		NewOutput(arguments.ViewHuman, v.view).Output("", outputValues)
	}
}

func (v *ApplyHuman) Operation() Operation {
	return NewOperation(arguments.ViewHuman, v.inAutomation, v.view)
}

func (v *ApplyHuman) Hooks() []terraform.Hook {
	return []terraform.Hook{
		v.countHook,
		NewUiHook(v.view),
	}
}

func (v *ApplyHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ApplyHuman) HelpPrompt() {
	command := "apply"
	if v.destroy {
		command = "destroy"
	}
	v.view.HelpPrompt(command)
}

const stateOutPathPostApply = "The state of your infrastructure has been saved to the path below. This state is required to modify and destroy your infrastructure, so keep it safe. To inspect the complete state use the `terraform show` command."

// The ApplyJSON implementation renders streaming JSON logs, suitable for
// integrating with other software.
type ApplyJSON struct {
	view *JSONView

	destroy bool

	countHook *countHook
}

var _ Apply = (*ApplyJSON)(nil)

func (v *ApplyJSON) ResourceCount(stateOutPath string) {
	operation := json.OperationApplied
	if v.destroy {
		operation = json.OperationDestroyed
	}
	v.view.ChangeSummary(&json.ChangeSummary{
		Add:       v.countHook.Added,
		Change:    v.countHook.Changed,
		Remove:    v.countHook.Removed,
		Import:    v.countHook.Imported,
		Operation: operation,
	})
}

func (v *ApplyJSON) Outputs(outputValues map[string]*states.OutputValue) {
	outputs, diags := json.OutputsFromMap(outputValues)
	if diags.HasErrors() {
		v.Diagnostics(diags)
	} else {
		v.view.Outputs(outputs)
	}
}

func (v *ApplyJSON) Operation() Operation {
	return &OperationJSON{view: v.view}
}

func (v *ApplyJSON) Hooks() []terraform.Hook {
	return []terraform.Hook{
		v.countHook,
		newJSONHook(v.view),
	}
}

func (v *ApplyJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ApplyJSON) HelpPrompt() {
}
