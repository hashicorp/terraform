package moduletest

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RunEnvironment is a collection of arguments used by Scenario.RunTests to
// interact with the development environment.
type RunEnvironment struct {
	Services           *disco.Disco
	ConfigParser       *configs.Parser
	ExperimentsAllowed bool
}

// RunTests executes all of the testing steps for the receiving scenario, which
// includes also evaluating the test cases within the configuration, and
// returns a description of the test results.
//
// Diagnostics raised in the execution of a particular step are considered to
// be part of that step's result rather than diagnostics for the test run
// itself, and so the returned diagnostics contains only diagnostics relating
// to the testing harness and test scenario configuration itself.
//
// If the returned diagnostics contains errors then the scenario result may
// either be entirely nil or incomplete. If incomplete, a caller may carefully
// inspect and report the partial result.
//
// If the given Context has a deadline or is cancelled, RunTests will try to
// return early if possible, but will first attempt to run the scenario's
// cleanup step to avoid leaving any dangling objects in a remote system.
func (s *Scenario) RunTests(ctx context.Context, env *RunEnvironment) (*ScenarioResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	result := &ScenarioResult{
		Scenario:    s,
		StepResults: make([]*StepResult, 0, len(s.Steps)),
		FinalState:  states.NewState(),
	}

	log.Printf("[TRACE] moduletest.Scenario.RunTests: starting scenario %s", s.Addr())

	core, moreDiags := terraform.NewContext(&terraform.ContextOpts{
		Parallelism: 10, // TODO: Make this configurable? How would it interact with scenario concurrency?

		// TODO: We'll need to populate Providers and Provisioners at least
		// in order for this context to actually be useful!
	})
	diags = diags.Append(moreDiags)

	// If we encountered any errors already then we'll bail out before we
	// actually run any steps. Everything we've done so far is test harness
	// setup and so we return it top-level diagnostics rather than as
	// associated with the scenario itself or with any of its steps.
	if diags.HasErrors() {
		return nil, diags
	}

	config, initDiags := s.LoadMainConfig(env)
	if len(initDiags) != 0 {
		// "Initialization" is a pseudo-step that we use to describe any
		// problems with the configuration itself, since those are not
		// associated with any particular real step.
		pseudoResult := s.newInitPseudoStepResult(initDiags)
		result.StepResults = append(result.StepResults, pseudoResult)

	}
	if initDiags.HasErrors() {
		// If we had any configuration errors then we can't continue, and so
		// we'll just stub out the results for all of the real steps and
		// return early.
		for _, step := range s.Steps {
			stubResult := step.skippedResult(config)
			result.StepResults = append(result.StepResults, stubResult)
		}
		return result, diags
	}

	// The final step of any scenario is always the mandatory cleanup step,
	// which we treat as special and always run even if an earlier step
	// failed.
	finalStepIdx := len(s.Steps) - 1
	if s.Steps[finalStepIdx].Name != CleanupStepName || s.Steps[finalStepIdx].PlanMode != plans.DestroyMode {
		// Safety check: bail if whatever constructed this scenario didn't
		// obey they invariant that the last step must be cleanup, before
		// we take any actions that might need cleaning up.
		panic(fmt.Sprintf("final step of %s is not a cleanup step", s.Addr()))
	}

	for stepIdx := 0; stepIdx < len(s.Steps); stepIdx++ {
		step := s.Steps[stepIdx]
		log.Printf("[TRACE] moduletest.Scenario.RunTests: starting step %s", step.Addr())

		prevState := result.FinalState
		var newState *states.State

		log.Printf("[TRACE] moduletest.Scenario.RunTests: starting planning for step %s (%s)", step.Addr(), step.PlanMode)
		opts := terraform.SimplePlanOpts(step.PlanMode, nil)
		plan, stepDiags := core.Plan(config, prevState, opts)
		var checkResults *states.CheckResults
		if plan != nil {
			checkResults = plan.Checks
		}

		// We'll start with a possibly-incomplete step result based on the
		// plan checks, but if the plan was valid then we'll throw this away
		// below and use an equivalent object taken from the new state
		// instead.
		stepResult := step.buildResult(step.PlanMode, checkResults, stepDiags)
		stepStatus := stepResult.AggregateStatus
		log.Printf("[TRACE] moduletest.Scenario.RunTests: completed planning for step %s, with %s", step.Addr(), stepStatus)
		if stepStatus != checks.StatusPass && stepStatus != checks.StatusUnknown {
			// If planning failed then we'll use the provisional check results
			// from the plan as our test case results, which may be incomplete
			// but still typically better than reporting nothing.
			result.StepResults = append(result.StepResults, stepResult)
			goto BailOut
		}

		if plan == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Plan phase passed but produced no plan",
				fmt.Sprintf("Despite the plan phase of %s not generating any errors or test failures, it still didn't produce a plan. This is a bug in Terraform; please report it!", step.Addr()),
			))
			goto BailOut
		}

		log.Printf("[TRACE] moduletest.Scenario.RunTests: starting apply for step %s", step.Addr())
		newState, stepDiags = core.Apply(plan, config)
		if newState != nil {
			result.FinalState = newState
		}
		checkResults = nil
		if newState != nil {
			checkResults = newState.CheckResults
		}

		// This replaces the possibly-incomplete result we produced based on
		// the plan-time checks above.
		stepResult = step.buildResult(step.PlanMode, checkResults, stepDiags)
		stepStatus = stepResult.AggregateStatus
		result.StepResults = append(result.StepResults, stepResult)
		log.Printf("[TRACE] moduletest.Scenario.RunTests: completed apply for step %s, with %s", step.Addr(), stepStatus)

		if stepStatus != checks.StatusPass {
			goto BailOut
		}

		// Unless we "goto BailOut" above we don't want to enter the BailOut
		// codepath below.
		continue

	BailOut:
		// If something failed or errored then we'll bail out, but we will
		// try to run the cleanup step first.
		// In other words, "goto BailOut" above is acting as a funny sort
		// of "break" that tries to run the last iteration of the loop before,
		// exiting it, if we're not already on the last iteration.
		if stepIdx != finalStepIdx {
			log.Println("[TRACE] moduletest.Scenario.RunTests: skipping to cleanup step due to failure/error")

			// If there are any steps we haven't run yet that we're skipping
			// over then we'll generate wholly-unknown stub results for each
			// one so that our overall results will tend to have a consistent
			// shape even when we encounter a failure or error.
			for stepIdx++; stepIdx < finalStepIdx; stepIdx++ {
				step := s.Steps[stepIdx]
				stubResult := step.skippedResult(config)
				result.StepResults = append(result.StepResults, stubResult)
			}

			// TODO: Generate synthetic StatusUnknown results for any steps
			// we're skipping here, so that we'll produce a reasonably
			// consistent tree of steps and cases on every run, even if there's
			// a failure or error somewhere.

			stepIdx = finalStepIdx - 1 // so that stepIdx++ on our outer loop will then select the final step
		}
		continue
	}

	log.Printf("[TRACE] moduletest.Scenario.RunTests: completed scenario %s with %s", s.Addr(), result.AggregateStatus())
	if !result.FinalState.Empty() {
		log.Printf("[WARN] final state for test scenario %s is not empty", s.Addr())
	}

	return result, diags
}
