package moduletest

import (
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ScenarioResult represents the overall results of executing a single test
// scenario.
type ScenarioResult struct {
	// Name is the user-selected name for the scenario.
	Name string

	// Status is the aggregate status across all of the steps. This uses the
	// usual rules for check status aggregation, so for example if any
	// one step is failing then the entire scenario has failed.
	Status checks.Status

	// Steps describes the results of each of the scenario's test steps.
	Steps []StepResult
}

// StepResult represents the result of executing a single step within a test
// scenario.
type StepResult struct {
	// Name is the user-selected name for the step, or it's a system-generated
	// implied step name which is then guaranteed to start with "<" and end
	// with ">" to allow distinguishing explicit vs. implied steps.
	Name string

	// Status is the aggregate status across all of the checks in this step.
	//
	// If field Diagnostics includes at least one error diagnostic then Status
	// is always checks.StatusError, regardless of the individual check results.
	//
	// Status unknown represents that the step didn't run to completion but that
	// any partial execution didn't encounter any failures or errors. For
	// example, a step has an unknown result if an earlier step in the same
	// scenario failed and therefore blocked running the remaining steps.
	Status checks.Status

	// Checks describes the results of each of the checkable objects declared
	// in the configuration for this step.
	//
	// Some implied steps don't actually perform normal Terraform plan/apply
	// operations and so do not produce check results. In that case Checks
	// is nil and Status and Diagnostics together describe the outcome of
	// the step.
	//
	// The special implied steps, like the final "terraform destroy" to clean
	// up anything left dangling, are essentially implementation details
	// rather than a real part of the author's test suite, and so UI code may
	// wish to use more muted presentation when reporting them, or perhaps not
	// mention them at all unless they return errors.
	Checks *states.CheckResults

	// Diagnostics reports any diagnostics generated during this step.
	//
	// Diagnostics cannot be unambigously associated with specific checks, so
	// in some cases these diagnostics might be the direct cause of some checks
	// having status error, while in other cases the diagnostics may be totally
	// unrelated to any of the checks and instead describe a more general
	// problem.
	Diagnostics tfdiags.Diagnostics
}
