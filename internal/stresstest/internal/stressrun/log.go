package stressrun

import (
	"errors"
	"strings"
	"time"

	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressgen"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
)

// Log represents the high-level sequence of steps taken as part of evaluating
// a configuration series.
type Log []LogStep

// LogStep represents a single step in a log, capturing the input configuration,
// input variables, and state snapshot at the step, along with the fake
// remote objects that the "stressful" provider was tracking when the step
// completed.
type LogStep struct {
	Time          time.Time
	Message       string
	Status        LogStatus
	Config        *stressgen.Config
	StateSnapshot *states.State
	RemoteObjects map[string]cty.Value
}

// LogStatus describes the outcome of a particular log step.
type LogStatus rune

const (
	// StepSucceeded indicates that the step completed successfully, allowing
	// the run to proceed to subsequent steps.
	StepSucceeded LogStatus = 'S'

	// StepFailed indicates that the step failed, blocking the run from
	// proceeding to subsequent steps.
	StepFailed LogStatus = 'F'

	// StepBlocked indicates that a particular step wasn't started at all,
	// because an earlier failure blocked it.
	StepBlocked LogStatus = 'B'
)

// Successful returns true if all of the steps recorded in the log completed
// successfully.
func (log Log) Successful() bool {
	for _, step := range log {
		if step.Status != StepSucceeded {
			return false
		}
	}
	return true
}

// Successful returns true if the step completed successfully.
func (step LogStep) Successful() bool {
	// This is a bit silly given how simple this test is, but this function
	// is here just to make the LogStep API similar to the Log API.
	return step.Status == StepSucceeded
}

// Err returns a non-nil error if there's a failed step in the recieving log,
// or a nil error otherwise.
func (log Log) Err() error {
	for _, step := range log {
		switch step.Status {
		case StepFailed:
			return errors.New(step.Message)
		case StepBlocked:
			// Shouldn't get here because a blocked step should always
			// come after a failed step, but we'll handle this to be robust.
			return errors.New("blocked step without failed step")
		}
	}
	return nil
}

func renderDiagnostics(diags tfdiags.Diagnostics, sources map[string][]byte) string {
	var buf strings.Builder
	for _, diag := range diags {
		buf.WriteString(format.Diagnostic(
			diag,
			sources,
			&colorstring.Colorize{
				Colors: map[string]string{
					"red":       "",
					"yellow":    "",
					"dark_gray": "",
					"bold":      "",
					"reset":     "",
					"underline": "",
				},
				Disable: true,
			},
			72,
		))
	}
	return buf.String()
}
