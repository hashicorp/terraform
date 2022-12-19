package command

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/testconfigs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// TestCommand is the implementation of "terraform test".
type TestCommand struct {
	Meta
}

func (c *TestCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseTest(rawArgs)
	view := views.NewTest(c.View, args.Output)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		`The "terraform test" command is experimental`,
		"We'd like to invite adventurous module authors to write integration tests for their modules using this command, but all of the behaviors of this command are currently experimental and may change based on feedback.\n\nFor more information on the testing experiment, including ongoing research goals and avenues for feedback, see:\n    https://www.terraform.io/docs/language/modules/testing-experiment.html",
	))

	ctx, cancel := c.InterruptibleContext()
	defer cancel()

	// We'll first make sure the module we're testing is even valid, since that
	// will allow us to give quicker feedback than waiting for all of the
	// scenarios to run and presumably end up hitting the same error.
	//
	// This is philosophically similar to catching complile-time errors before
	// running tests in a ahead-of-time compiled language, although for us it
	// isn't strictly necessary and is instead to just tighten the modify/test
	// loop when using tests to support local development.
	moreDiags := c.preValidate(ctx, args)
	if moreDiags.HasErrors() {
		// NOTE: This is intentionally different than the usual pattern for
		// diagnostics where we'd normally append unconditionally and _then_
		// check for errors, so that we can preserve any warnings.
		//
		// In this situation we're using this early validation check as a
		// timesaver to catch problems that would typically make every test
		// case fail, but if it only generates warnings then we're likely to
		// see those same warnings for every test scenario anyway and so it'd
		// just add noise to report them again here.
		diags = diags.Append(moreDiags)
		view.Diagnostics(diags)
		return 1
	}

	results, moreDiags := c.run(ctx, args)
	diags = diags.Append(moreDiags)

	initFailed := diags.HasErrors()
	view.Diagnostics(diags)
	diags = view.Results(results)
	resultsFailed := diags.HasErrors()
	view.Diagnostics(diags) // possible additional errors from saving the results

	var testsFailed bool
	for _, scenario := range results {
		if scenario.Status == checks.StatusFail || scenario.Status == checks.StatusError {
			testsFailed = true
			break
		}
	}

	// Lots of things can possibly have failed
	if initFailed || resultsFailed || testsFailed {
		return 1
	}
	return 0
}

func (c *TestCommand) preValidate(ctx context.Context, args arguments.Test) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	cfg, cfgDiags := c.loadConfig(".")
	diags = diags.Append(cfgDiags)

	if diags.HasErrors() {
		return diags
	}

	coreOpts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	core, ctxDiags := terraform.NewContext(coreOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return diags
	}

	validateDiags := core.Validate(cfg)
	diags = diags.Append(validateDiags)
	return diags
}

func (c *TestCommand) run(ctx context.Context, args arguments.Test) (results map[string]*moduletest.ScenarioResult, diags tfdiags.Diagnostics) {
	loader, err := c.initConfigLoader()
	if err != nil {
		// It would be strange to get here so for prototype purposes we won't
		// bother with a full diagnostic message.
		// TODO: Make this a decent error message if we make a real version of this.
		diags = diags.Append(err)
		return nil, diags
	}

	suite, moreDiags := testconfigs.LoadSuiteForModule(".", loader.Parser())
	// HACK: Loading the suite implicitly loads the root module for each
	// test scenario step and because smoke_tests is currently an experiment
	// that writes warnings into moreDiags. We're going to see those same
	// warnings again when we load the whole configuration as we run each
	// step for real, so we'll just drop them up here to avoid duplicating
	// them in two places.
	var noWarningsDiags tfdiags.Diagnostics
	if len(moreDiags) > 0 {
		noWarningsDiags = make(tfdiags.Diagnostics, 0, len(moreDiags))
		for _, diag := range moreDiags {
			if diag.Severity() == tfdiags.Warning {
				continue
			}
			noWarningsDiags = append(noWarningsDiags, diag)
		}
	}
	diags = diags.Append(noWarningsDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// NOTE: In a real system (as opposed to this prototype) it should be
	// some other package's responsibility to actually run the tests, rather
	// than it being just inline here.
	ret := make(map[string]*moduletest.ScenarioResult, len(suite.Scenarios))
	for name, scenario := range suite.Scenarios {
		result, moreDiags := c.runScenario(ctx, scenario)
		diags = diags.Append(moreDiags)
		ret[name] = result
	}

	return ret, diags
}

func (c *TestCommand) runScenario(ctx context.Context, config *testconfigs.Scenario) (*moduletest.ScenarioResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	const impliedStepCountPre = 0  // currently we have no fallible initialization
	const impliedStepCountPost = 1 // the final destroy step
	ret := &moduletest.ScenarioResult{
		Name:   config.Name,
		Status: checks.StatusUnknown,
		Steps:  make([]moduletest.StepResult, 0, len(config.Steps)+impliedStepCountPre+impliedStepCountPost),
	}
	for _, stepName := range config.StepsOrder {
		ret.Steps = append(ret.Steps, moduletest.StepResult{
			Name:   stepName,
			Status: checks.StatusUnknown,
		})
	}
	ret.Steps = append(ret.Steps, moduletest.StepResult{
		Name:   "<cleanup>",
		Status: checks.StatusUnknown,
	})
	finalDestroyResult := &ret.Steps[len(ret.Steps)-1]

	var cleanupCtx *testCommandCleanupContext
	state := states.NewState()
	for i, stepName := range config.StepsOrder {
		stepResult := &ret.Steps[i+impliedStepCountPre]
		step := config.Steps[stepName]
		newCleanupCtx, moreDiags := c.runScenarioStep(ctx, config, step, state, stepResult)
		diags = diags.Append(moreDiags) // NOTE: These are test harness errors, not errors from the test step itself
		if newCleanupCtx != nil {
			cleanupCtx = newCleanupCtx
			// (if runScenarioStep returns a nil cleanup context then we assume
			// it didn't get far enough to actually change anything and so
			// we'll just keep the cleanup context from the previous step)
		}
		if moreDiags.HasErrors() || stepResult.Status == checks.StatusFail || stepResult.Status == checks.StatusError {
			// If any step fails or errors then we skip running the others
			// because we assume they will expect the effects of the
			// prior steps.
			break
		}
	}

	diags = diags.Append(
		c.runScenarioCleanup(ctx, cleanupCtx, finalDestroyResult),
	)
	// TODO: Make runScenarioCleanup return its final state and generate a
	// louder message if there are any managed resources left in there
	// after the cleanup step.

	ret.Status = checks.AggregateCheckStatusSlice(
		ret.Steps,
		func(result moduletest.StepResult) checks.Status {
			return result.Status
		},
	)

	return ret, diags
}

func (c *TestCommand) runScenarioStep(ctx context.Context, scenarioConfig *testconfigs.Scenario, stepConfig *testconfigs.Step, state *states.State, result *moduletest.StepResult) (*testCommandCleanupContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// This prototype relies on a hack over in the "terraform init"
	// command which preinstalls all of the modules for each step into
	// a directory following this same naming scheme.
	modulesCacheDir := filepath.Join(c.DataDir(), "test-scenarios", scenarioConfig.Name, stepConfig.Name, "modules")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesCacheDir,
		Services:   c.Services,
	})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}
	loader.AllowLanguageExperiments(c.AllowExperimentalFeatures)

	// From this point onwards we need to be careful about the scope
	// of diagnostics. The variable "diags" is for errors related to the
	// behavior of this test harness itself, whereas anything which is related
	// to the validity or behavior of the module under test or the test step
	// itself belongs in result.Diagnostics so that the UI can report them
	// in proper context.
	//
	// If any errors are added to result.Diagnostics then result.Status
	// should be set to checks.StatusError.

	cfg, hclDiags := loader.LoadConfig(stepConfig.ModuleDir)
	result.Diagnostics = result.Diagnostics.Append(hclDiags)
	if hclDiags.HasErrors() {
		result.Status = checks.StatusError
		// Although we could potentially return cfg as the new config
		// here, we know it's invalid and we've not taken any real actions
		// using it and so the caller has a better chance of using its previous
		// configuration for the final cleanup step.
		return nil, diags
	}

	// TODO: For now we just use the main contextOpts without any changes,
	// but once we want to support mock providers we'll need to swap out
	// the real providers for mocks before we proceed here.
	coreOpts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	core, moreDiags := terraform.NewContext(coreOpts)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	planOpts := &terraform.PlanOpts{
		Mode:         stepConfig.PlanMode,
		SetVariables: make(terraform.InputValues, len(stepConfig.RootModule.Variables)),
	}
	varDefErrors := false
	for name, decl := range stepConfig.RootModule.Variables {
		if defn, defined := stepConfig.VariableDefs[decl.Addr()]; defined {
			rng := tfdiags.SourceRangeFromHCL(defn.Range())

			// TODO: Eventually we'll presumably want to allow references to
			// earlier steps and to other data in these expressions, but
			// for now we just require them to be constant values.
			v, hclDiags := defn.Value(nil)
			result.Diagnostics = result.Diagnostics.Append(hclDiags)
			if hclDiags.HasErrors() {
				varDefErrors = true
				v = cty.DynamicVal
			}
			planOpts.SetVariables[name] = &terraform.InputValue{
				Value:       v,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: rng,
			}
		} else {
			planOpts.SetVariables[name] = &terraform.InputValue{
				Value:      cty.NullVal(cty.DynamicPseudoType),
				SourceType: terraform.ValueFromCaller,
			}
		}
	}
	// TODO: We also need to pass in provider configurations, once such a thing
	// is possible to do.
	// TODO: Make sure that everything listed in expected_failures refers to
	// a static checkable object that we can see in the configuration, and
	// reject the test step as invalid if not. The subsequent logic which
	// checks whether the expected failures actually failed assumes that
	// expected_failures can only refer to checkable objects that exist.
	if varDefErrors {
		// If we get here then the loop above should've added at least one
		// error to result.Diagnostics, and so we'll bail out and return
		// those as an error status.
		result.Status = checks.StatusError
		return nil, diags
	}
	plan, moreDiags := core.Plan(cfg, state, planOpts)
	result.Diagnostics = result.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		result.Status = checks.StatusError
		// Although we could potentially return cfg as the new config
		// here, we've not made any changes using it and so the caller has a
		// better chance of using its previous configuration for the final
		// cleanup step.
		return nil, diags
	}

	// We'll update our result from just the plan for now, which might give
	// us a partial result. If we reach the apply step below then we'll
	// overwrite these with final results.
	result.Checks = plan.Checks
	result.ExpectedFailures = c.prepareExpectedFailuresReport(result.Checks, stepConfig.ExpectFailure)
	result.Status = c.stepAggregateStatus(result.Checks, result.ExpectedFailures)

	if !stepConfig.ApplyPlan {
		// If we're not actually applying the plan then we'll let the caller
		// keep using its previous config and state for the final cleanup
		// step, since swapping out config at this point is more likely to
		// make cleanup fail due to mismatches.
		return nil, diags
	}

	newState, moreDiags := core.Apply(plan, cfg)
	result.Diagnostics = result.Diagnostics.Append(moreDiags)
	cleanupCtx := &testCommandCleanupContext{
		State:        newState,
		Config:       cfg,
		SetVariables: planOpts.SetVariables,
	}
	if moreDiags.HasErrors() {
		result.Status = checks.StatusError
		// Even though this failed we'll still return the new config
		// and state, because we may have taken some real actions before
		// the failure and so the latest config and state is most likely
		// to successfully destroy everything during the cleanup step.
		return cleanupCtx, diags
	}

	result.Checks = newState.CheckResults
	result.ExpectedFailures = c.prepareExpectedFailuresReport(result.Checks, stepConfig.ExpectFailure)
	result.Status = c.stepAggregateStatus(result.Checks, result.ExpectedFailures)

	// TODO: We also need to deal with any expected failures, which should
	// allow the overall step to succeed even if they have StatusFail.

	return cleanupCtx, diags
}

func (c *TestCommand) prepareExpectedFailuresReport(checkResults *states.CheckResults, expectedFail addrs.Set[addrs.Checkable]) addrs.Map[addrs.Checkable, checks.Status] {
	var ret addrs.Map[addrs.Checkable, checks.Status]
	if len(expectedFail) == 0 {
		return ret
	}
	ret = addrs.MakeMap[addrs.Checkable, checks.Status]()
	for _, addr := range expectedFail {
		result := checkResults.GetObjectResult(addr)
		status := checks.StatusUnknown
		if result != nil {
			status = result.Status
		}
		ret.Put(addr, status)
	}
	return ret
}

func (c *TestCommand) stepAggregateStatus(checkResults *states.CheckResults, expectedFails addrs.Map[addrs.Checkable, checks.Status]) checks.Status {
	return checks.AggregateCheckStatusAddrsMap(
		checkResults.ConfigResults,
		func(k addrs.ConfigCheckable, aggrResult *states.CheckResultAggregate) checks.Status {
			return checks.AggregateCheckStatusAddrsMap(
				aggrResult.ObjectResults,
				func(addr addrs.Checkable, result *states.CheckResultObject) checks.Status {
					if expectedFails.Has(addr) {
						return result.Status.ForExpectedFailure()
					}
					return result.Status
				},
			)
		},
	)
}

func (c *TestCommand) runScenarioCleanup(ctx context.Context, cleanupCtx *testCommandCleanupContext, result *moduletest.StepResult) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Our goal here is to make a best effort to create a destroy plan and
	// apply it with the most recent config and state, assuming that we
	// got far enough to have a config and state.
	if !cleanupCtx.NeedsCleanup() {
		// No cleanup necessary
		result.Status = checks.StatusPass
		return diags
	}

	// TODO: For now we just use the main contextOpts without any changes,
	// but once we want to support mock providers we'll need to swap out
	// the real providers for mocks before we proceed here.
	coreOpts, err := c.contextOpts()
	if err != nil {
		result.Status = checks.StatusError
		diags = diags.Append(err)
		return diags
	}

	core, moreDiags := terraform.NewContext(coreOpts)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		result.Status = checks.StatusError
		return diags
	}

	planOpts := &terraform.PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: cleanupCtx.SetVariables,
		// TODO: Must also set the passed in provider configurations, once
		// that is possible to do.
	}
	plan, moreDiags := core.Plan(cleanupCtx.Config, cleanupCtx.State, planOpts)
	result.Diagnostics = result.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		result.Status = checks.StatusError
		// Although we could potentially return cfg as the new config
		// here, we've not made any changes using it and so the caller has a
		// better chance of using its previous configuration for the final
		// cleanup step.
		return diags
	}

	_, moreDiags = core.Apply(plan, cleanupCtx.Config)
	result.Diagnostics = result.Diagnostics.Append(moreDiags)
	if moreDiags.HasErrors() {
		result.Status = checks.StatusError
		// Even though this failed we'll still return the new config
		// and state, because we may have taken some real actions before
		// the failure and so the latest config and state is most likely
		// to successfully destroy everything during the cleanup step.
		return diags
	}

	result.Status = checks.StatusPass
	return diags
}

type testCommandCleanupContext struct {
	// State is the state from the most recent apply.
	State *states.State

	// Config is the configuration that was applied to create State.
	Config *configs.Config

	// SetVariables are the variables that were set in the plan which
	// was applied to create State.
	SetVariables terraform.InputValues
}

func (c *testCommandCleanupContext) NeedsCleanup() bool {
	return c != nil && c.State != nil && !c.State.Empty()
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform test [options]

  This is an experimental command to help with automated integration
  testing of shared modules. The usage and behavior of this command is
  likely to change in breaking ways in subsequent releases, as we
  are currently using this command primarily for research purposes.

Options:

  -compact-warnings  Use a more compact representation for warnings, if
                     this command produces only warnings and no errors.

  -junit-xml=FILE    In addition to the usual output, also write test
                     results to the given file path in JUnit XML format.
                     This format is commonly supported by CI systems, and
                     they typically expect to be given a filename to search
                     for in the test workspace after the test run finishes.

  -no-color          Don't include virtual terminal formatting sequences in
                     the output.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Experimental support for module integration testing"
}
