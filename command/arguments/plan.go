package arguments

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// Plan represents the command-line arguments for the plan command.
type Plan struct {
	// State, Operation, and Vars are the common extended flags
	State     *State
	Operation *Operation
	Vars      *Vars

	// Destroy can be set to generate a plan to destroy all infrastructure.
	Destroy bool

	// RefreshOnly can be set to plan only the effect of refreshing existing
	// objects to update the state, without also planning actions to change
	// objects to match the current configuration.
	//
	// This mode is not compatible with Destroy.
	RefreshOnly bool

	// TaintInstances is an optional set of resource instance addresses to
	// consider as tainted when creating the plan. This allows planning the
	// effect of a taint while making that effect visible only after the
	// plan is applied.
	TaintInstances []addrs.AbsResourceInstance

	// DetailedExitCode enables different exit codes for error, success with
	// changes, and success with no changes.
	DetailedExitCode bool

	// InputEnabled is used to disable interactive input for unspecified
	// variable and backend config values. Default is true.
	InputEnabled bool

	// OutPath contains an optional path to store the plan file
	OutPath string

	// ViewType specifies which output format to use
	ViewType ViewType
}

// ParsePlan processes CLI arguments, returning a Plan value and errors.
// If errors are encountered, a Plan value is still returned representing
// the best effort interpretation of the arguments.
func ParsePlan(args []string) (*Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	plan := &Plan{
		State:     &State{},
		Operation: &Operation{},
		Vars:      &Vars{},
	}

	var taintInstancesRaw []string // we'll try to parse these later

	cmdFlags := extendedFlagSet("plan", plan.State, plan.Operation, plan.Vars)
	cmdFlags.BoolVar(&plan.Destroy, "destroy", false, "destroy")
	cmdFlags.BoolVar(&plan.RefreshOnly, "refresh-only", false, "refresh-only")
	cmdFlags.Var((*flagStringSlice)(&taintInstancesRaw), "taint", "taint")
	cmdFlags.BoolVar(&plan.DetailedExitCode, "detailed-exitcode", false, "detailed-exitcode")
	cmdFlags.BoolVar(&plan.InputEnabled, "input", true, "input")
	cmdFlags.StringVar(&plan.OutPath, "out", "", "out")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
	}

	args = cmdFlags.Args()

	if len(args) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"To specify a working directory for the plan, use the global -chdir flag.",
		))
	}

	if plan.RefreshOnly && plan.Destroy {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Impossible plan mode",
			"A plan can't be both -refresh-only and -destroy at the same time, because -refresh-only disables any changes to remote objects.",
		))
	}

	if plan.RefreshOnly && !plan.Operation.Refresh {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Impossible plan mode",
			"A plan can't be both -refresh=false and -refresh-only at the same time, because it would then have nothing to do at all.",
		))
	}

	if len(taintInstancesRaw) > 0 {
		plan.TaintInstances = make([]addrs.AbsResourceInstance, 0, len(taintInstancesRaw))
		for _, rawAddr := range taintInstancesRaw {
			addr, moreDiags := addrs.ParseAbsResourceInstanceStr(rawAddr)
			diags = diags.Append(moreDiags)
			if !diags.HasErrors() {
				plan.TaintInstances = append(plan.TaintInstances, addr)
			}
		}
		if len(plan.TaintInstances) == 0 {
			plan.TaintInstances = nil // don't return a non-nil empty slice
		}
	}

	diags = diags.Append(plan.Operation.Parse())

	switch {
	default:
		plan.ViewType = ViewHuman
	}

	return plan, diags
}
