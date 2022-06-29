package moduletest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

// Step represents one of the steps in a testing scenario. Each step represents
// creating a plan and then optionally applying it.
type Step struct {
	// Scenario points to the scenario this step belongs to.
	Scenario *Scenario

	// Name is a user-specified name for a custom step, or a special name for
	// any synthetic step created by the testing harness itself.
	//
	// A user-specified name is always a valid HCL identifier, and so any
	// synthetic steps will have non-identifier names. Currently, those are:
	//   - Empty string represents the synthetic main plan+apply step.
	//   - "(cleanup)" represents the final destroy step we use to clean
	//     up any remaining infrastructure still present at the end of the
	//     scenario.
	//
	// The special step name "(init)" is also reserved as a special case to
	// describe any problems that prevent even starting the scenario, such
	// as a syntax error in the scenario configuration. Initialization isn't
	// really a "step" in the sense of a plan+apply, but we need to be able
	// to talk about it in our test result reports as if it were one.
	Name string

	// PlanMode describes the kind of plan we'll generate in this step.
	PlanMode plans.Mode

	// We don't currently track anything else about a step, but in future
	// we might also track whether a particular step is a plan-only step
	// (skips the apply phase), a set of addresses of checkable objects that
	// are _expected_ to produce errors, etc. All of this extra information
	// would require defining a mini-language for describing test scenarios,
	// which could perhaps appear as an optional file "terraform-test.hcl"
	// inside the scenario directory. In the absense of such a file, each
	// scenario is implied to have a default main step which is a full
	// plan+apply with no expected errors.
}

const (
	// DefaultStepName is the reserved step name used for the synthetic main
	// testing step generated when there is no explicit scenario configuration.
	DefaultStepName string = ""

	// CleanupStepName is the reserved step name used for the mandatory
	// cleanup step that always appears as the final step of any testing
	// scenario.
	CleanupStepName string = "(cleanup)"

	// InitPseudoStepName is a name that is reserved to pretend that various
	// initialization failures happened in a "step" when we're reporting test
	// results, even though initialization isn't really a step in the
	// usual sense of the term.
	InitPseudoStepName string = "(init)"
)

func (s *Step) Addr() addrs.ModuleTestStep {
	scenarioAddr := s.Scenario.Addr()
	return scenarioAddr.TestStep(s.Name)
}

func (s *Step) IsCleanup() bool {
	return s.Name == CleanupStepName
}
