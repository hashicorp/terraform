package webcommand

import (
	"fmt"
)

type TargetObject interface {
	targetObjectSigil()

	// UIDescription is a short string describing what was selected for use
	// in error messages in the UI reporting when the target object is not
	// supported by the backend.
	UIDescription() string
}

type targetObjectSimple string

func (to targetObjectSimple) targetObjectSigil() {}

var (
	TargetObjectCurrentWorkspace TargetObject
	TargetObjectLatestRun        TargetObject
)

func init() {
	TargetObjectCurrentWorkspace = targetObjectSimple("current_workspace")
	TargetObjectLatestRun = targetObjectSimple("latest_run")
}

func (to targetObjectSimple) UIDescription() string {
	switch to {
	case TargetObjectCurrentWorkspace:
		return "the currently-selected workspace"
	case TargetObjectLatestRun:
		return "the latest run for the current workspace"
	default:
		// We should not get here because the above should be exhaustive
		// for all of our exported webTargetObjectSimple values.
		return "the selected object"
	}
}

type TargetObjectRun struct {
	RunID string
}

func (wto TargetObjectRun) targetObjectSigil() {}

func (wto TargetObjectRun) UIDescription() string {
	return fmt.Sprintf("run %q", wto.RunID)
}
