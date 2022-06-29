package moduletest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ScenarioResult is the top-level object for describing the result of
// executing a particular test scenario.
type ScenarioResult struct {
	// Scenario is the test scenario that these results are describing.
	Scenario *Scenario

	// StepResults contains the individual results for the scenario's steps,
	// in the order they were executed.
	StepResults []*StepResult

	// FinalState is a snapshot of the state of the scenario at the end of
	// execution. If the scenario succeeded or failed then this should always
	// be an empty state, but this could be non-empty if the test process
	// was interrupted or encountered an error during its cleanup step.
	//
	// A testing driver should typically check whether the FinalState is
	// "empty" and, if not, write a snapshot of the state and/or a description
	// of the resource instances recorded inside it to somewhere an operator
	// can refer to in order to find and manually destroy the remaining
	// objects.
	FinalState *states.State
}

// AggregateStatus returns a single status value summarizing all of the
// statuses of all of the test cases across all of the steps in this scenario.
func (r *ScenarioResult) AggregateStatus() checks.Status {
	// We'll start with a pass since a step that has no steps is
	// an implicit pass. However, a scenario with no steps is impossible
	// so we'll always aggregate with at least one actual result below.
	ret := checks.StatusPass
	for _, stepResult := range r.StepResults {
		ret = checks.AggregateStatus(ret, stepResult.AggregateStatus)
	}
	return ret
}

// StepResult describes the result of running a particular step within
// a testing scenario.
type StepResult struct {
	// Step is the test step that these results are describing.
	Step *Step

	// AggregateStatus is the status representing the outcome of the entire
	// test step.
	AggregateStatus checks.Status

	// TestCaseResults contains the individual results for each of the
	// test cases (static checkable objects) visited as part of the step.
	TestCaseResults addrs.Map[addrs.ConfigCheckable, *TestCaseResult]

	// Diagnostics is the subset of diagnostics that are not associated with
	// any particular test case.
	//
	// Terraform Core annotates some diagnostics to say that they are
	// related to a particular ConfigCheckable, in which case those diagnostics
	// will be excluded from this collection and reported inside the objects
	// in TestCaseResults instead.
	Diagnostics tfdiags.Diagnostics
}

// TestCase result describes the results of checking a particular test case
// (a ConfigCheckable) within a testing step.
type TestCaseResult struct {
	// Step is the test step whose instance of this test case this object
	// is describing.
	Step *Step

	// ConfigObject is the ConfigCheckable address representing the object
	// that this test case is associated with.
	ConfigObject addrs.ConfigCheckable

	// AggregateStatus is the status representing the outcome of the entire
	// test case.
	//
	// If AggregateStatus is checks.StatusError then there will always be at
	// least one error in field Diagnostics, and conversely for any other
	// status we guarantee that there are no error diagnostics.
	AggregateStatus checks.Status

	// ObjectResults gives the statuses of each of the individual objects
	// belonging to this test case, if any.
	//
	// If this map is empty then the meaning depends on AggregateStatus:
	//   - If Passed, this test step didn't include any objects for this test case.
	//   - If Unknown or Error, this test step didn't get a chance to expand at
	//     all because it was blocked by an error.
	ObjectResults addrs.Map[addrs.Checkable, *states.CheckResultObject]

	// Diagnostics is the subset of diagnostics that were explicitly associated
	// with this test case by Terraform Core.
	//
	// Not all diagnostics are annotated with information that allows us to
	// determine which test case they relate to (if any), so any diagnostics
	// unaccounted for by a test result should be returned in the corresponding
	// field of StepResult, as diagnostics for the whole step.
	//
	// Diagnostics does not include any diagnostics that would be redundant
	// with failure messages included in the ObjectResults values, so we can
	// always return actual failures against the specific dynamic object that
	// failed.
	Diagnostics tfdiags.Diagnostics
}

// TestCaseAddr returns the fully-qualified address for this test case in
// its particular step and scenario.
func (r *TestCaseResult) TestCaseAddr() addrs.ModuleTestCase {
	return r.Step.Addr().TestCase(r.ConfigObject)
}

// buildResult projects our generic idea of check results and our
// full set of diagnostics from a step into a StepResult object suitable
// for inclusion as an element of ScenarioResult.StepResults.
func (s *Step) buildResult(planMode plans.Mode, checkResults *states.CheckResults, allDiags tfdiags.Diagnostics) *StepResult {
	caseResults := addrs.MakeMap[addrs.ConfigCheckable, *TestCaseResult]()
	var stepDiags tfdiags.Diagnostics

	// We'll work on the checks first, because the presence of a particular
	// ConfigCheckable in the check results is what establishes the existence
	// of a particular test case.
	aggrStatus := checks.StatusPass
	if checkResults != nil {
		for _, elem := range checkResults.ConfigResults.Elems {
			addr := elem.Key
			checkResult := elem.Value

			// A TestCaseResult is essentially the same as a check result, but
			// additionally has a record of the step it came from and, once we
			// populate it with the other loop below, any diagnostics that are
			// related to this configuration object in particular.
			caseResult := &TestCaseResult{
				Step:            s,
				ConfigObject:    addr,
				AggregateStatus: checkResult.Status,
				ObjectResults:   checkResult.ObjectResults,
			}
			caseResults.Put(addr, caseResult)
			aggrStatus = checks.AggregateStatus(aggrStatus, checkResult.Status)
		}
	}

	if aggrStatus == checks.StatusError && !allDiags.HasErrors() {
		// We should not get error status if we don't have any errors, so
		// this is always a bug but we'll generate a message about it just
		// to avoid the result being confusing.
		allDiags = nil
		allDiags = allDiags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error result without any errors",
			"This test step produced an error result but didn't produce any errors, which suggests a bug in Terraform. Please report it!",
		))
	}

	// Now we can try to fit the diagnostics to the test cases where possible,
	// based on the annotations.
	errorCount := 0
	for _, diag := range allDiags {
		if addr, _ := CheckStatusForDiagnostic(diag); addr != nil {
			// If this diagnostic is reporting the status of a particular
			// check then we'll skip it altogether, because we should already
			// have it recorded in the corresponding test case result and so
			// it would be redundant to re-report it as a diagnostic.
			continue
		}

		if diag.Severity() == tfdiags.Error {
			errorCount++ // NOTE: We only count non-check-status errors
		}

		if addr := ConfigCheckableForDiagnostic(diag); addr != nil {
			if caseResult, ok := caseResults.GetOk(addr); ok {
				// If the diagnostic belongs to a ConfigCheckable and it's
				// one of the ones we already counted as a test case then
				// the diagnostic belongs to that test case.
				caseResult.Diagnostics = append(caseResult.Diagnostics, diag)
				continue
			}
		}

		// If we get here then we've not found a more specific home for
		// the diagnostic, and so it can just go in our whole-step collection.
		stepDiags = append(stepDiags, diag)
	}

	if errorCount > 0 {
		// If we have any step-level errors then they always take priority
		// over whatever the per-test-case results were.
		aggrStatus = checks.StatusError
	}

	if planMode == plans.DestroyMode && aggrStatus == checks.StatusUnknown {
		// TRICKY: Terraform Core doesn't evaluate checks when creating or
		// applying a destroy plan, so it's correct to say that the
		// individual test cases were all skipped, but misleading to say
		// that the whole step was skipped. Therefore we have an awkward
		// special case here to treat this particular situation as a funny
		// sort of aggregate pass.
		aggrStatus = checks.StatusPass
	}

	return &StepResult{
		Step:            s,
		AggregateStatus: aggrStatus,
		TestCaseResults: caseResults,
		Diagnostics:     stepDiags,
	}
}

// skippedResult is a weird variant of buildResult for generating an
// entirely-placeholder step result for a step we skipped over entirely due
// to the failure of an earlier step.
//
// The idea here is to still describe all of the checks we would have expected
// to run if we hadn't skipped the step, so that our test result shape stays
// relatively consistent between runs even if we encounter a failure or error.
func (s *Step) skippedResult(config *configs.Config) *StepResult {
	// The initial value of a newly-created *checks.State already tracks
	// all of the configured checkable objects, with each one initially
	// recorded as StatusUnknown.
	emptyState := checks.NewState(config)

	// Converting that "empty" state to a *states.CheckResults freezes all
	// of those unknown-status checks for us.
	checkResults := states.NewCheckResults(emptyState)

	// Finally we can use our buildResult logic to project that into a
	// *StepResult for us to return. (No diagnostics here, because we didn't
	// actually do anything that could potentially generate any.)
	ret := s.buildResult(plans.NormalMode, checkResults, nil)

	// Skipped steps always have unknown aggregate status to differentiate
	// them from steps that succeeded by doing nothing.
	ret.AggregateStatus = checks.StatusUnknown

	return ret
}
