package moduletest

import (
	"path/filepath"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Scenario is the topmost level in the organizational tree for tests,
// with each instance representing one of the directories under the "tests"
// directory.
type Scenario struct {
	// Path is the relative path from the directory containing the module
	// under test to the directory containing the test scenario, always
	// using forward slashes even on Windows.
	//
	// Due to the current directory layout convention, Path always begins
	// with the literal prefix "tests/", but we include that prefix to
	// make room for additional extension in future, such as "examples/" if
	// we add support for running examples as additional tests in future.
	Path string

	// BaseDir is the path that field Path is relative to.
	// filepath.Join(BaseDir, Path) will produce a path relative to the current
	// working directory where the test scenario's root module should be found.
	BaseDir string

	// Steps are the steps of the scenario. In today's implementation a
	// scenario always has exactly the same steps generated automatically
	// to run the fixed apply+destroy sequence, but we expect to
	// allow some amount of customization of steps in future via an explicit
	// scenario configuration file, so that e.g. authors can test the
	// handling of gradual updates to existing infrastructure over multiple
	// steps.
	Steps []*Step
}

func (s *Scenario) Addr() addrs.ModuleTestScenario {
	return addrs.ModuleTestScenario{Path: s.Path}
}

// RootModulePath returns the filesystem path to the directory containing the
// root module used by this testing scenario, relative to whatever was the
// current working directory at the time of creating the Scenario object.
func (s *Scenario) RootModulePath() string {
	return filepath.Join(s.BaseDir, s.Path)
}

// newInitPseudoStep is a helper for constructing a weirdo Step object that
// represents the "step" of preparing to run our steps, in case we need to
// report errors with the scenario's configuration that we encounter before
// we start running any steps at all.
//
// The result of this should never end up as part of the "Steps" field in
// a Scenario object. It's only for inclusion in test results, when needed.
func (s *Scenario) newInitPseudoStep() *Step {
	return &Step{
		Scenario: s,
		Name:     InitPseudoStepName,
		PlanMode: plans.NormalMode,
	}
}

// newInitPseudoStepResult is a helper for constructing a StepResult that
// reports some diagnostics found during initialization of a test scenario,
// before running any steps.
//
// Call this only when the given diagnostics has at least one element, because
// otherwise there should be no "init pseudo-step".
func (s *Scenario) newInitPseudoStepResult(diags tfdiags.Diagnostics) *StepResult {
	if len(diags) == 0 {
		panic("cannot make init pseudo-step result with no diagnostics")
	}

	aggrStatus := checks.StatusPass
	if diags.HasErrors() {
		aggrStatus = checks.StatusError
	}

	return &StepResult{
		Step:            s.newInitPseudoStep(),
		AggregateStatus: aggrStatus,
		Diagnostics:     diags,
	}
}
