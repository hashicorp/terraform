package views

import (
	"fmt"

	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
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
func NewApply(vt arguments.ViewType, destroy bool, runningInAutomation bool, view *View) Apply {
	switch vt {
	case arguments.ViewHuman:
		return &ApplyHuman{
			View:         *view,
			destroy:      destroy,
			inAutomation: runningInAutomation,
			countHook:    &countHook{},
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

// The ApplyHuman implementation renders human-readable text logs, suitable for
// a scrolling terminal.
type ApplyHuman struct {
	View

	destroy      bool
	inAutomation bool

	countHook *countHook
}

var _ Apply = (*ApplyHuman)(nil)

func (v *ApplyHuman) ResourceCount(stateOutPath string) {
	if v.destroy {
		v.streams.Printf(
			v.colorize.Color("[reset][bold][green]\nDestroy complete! Resources: %d destroyed.\n"),
			v.countHook.Removed,
		)
	} else {
		v.streams.Printf(
			v.colorize.Color("[reset][bold][green]\nApply complete! Resources: %d added, %d changed, %d destroyed.\n"),
			v.countHook.Added,
			v.countHook.Changed,
			v.countHook.Removed,
		)
	}
	if (v.countHook.Added > 0 || v.countHook.Changed > 0) && stateOutPath != "" {
		v.streams.Printf("\n%s\n\n", format.WordWrap(stateOutPathPostApply, v.View.outputColumns()))
		v.streams.Printf("State path: %s\n", stateOutPath)
	}
}

func (v *ApplyHuman) Outputs(outputValues map[string]*states.OutputValue) {
	if len(outputValues) > 0 {
		v.streams.Print(v.colorize.Color("[reset][bold][green]\nOutputs:\n\n"))
		NewOutput(arguments.ViewHuman, &v.View).Output("", outputValues)
	}
}

func (v *ApplyHuman) Operation() Operation {
	return NewOperation(arguments.ViewHuman, v.inAutomation, &v.View)
}

func (v *ApplyHuman) Hooks() []terraform.Hook {
	return []terraform.Hook{
		v.countHook,
		NewUiHook(&v.View),
	}
}

func (v *ApplyHuman) HelpPrompt() {
	command := "apply"
	if v.destroy {
		command = "destroy"
	}
	v.View.HelpPrompt(command)
}

const stateOutPathPostApply = "The state of your infrastructure has been saved to the path below. This state is required to modify and destroy your infrastructure, so keep it safe. To inspect the complete state use the `terraform show` command."
