package command

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/testconfigs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
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
		if args.MockOnly && scenario.UsesRealProviders() {
			ret[name] = &moduletest.ScenarioResult{
				Name:   name,
				Status: checks.StatusUnknown,
				// TODO: We should probably include some sort of optional
				// "status reason" thing here so that we can be clear
				// that we skipped this because it uses real providers.
			}
			continue
		}
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

	extProviderConfigs, moreDiags := c.externalProviderConfigsForStep(scenarioConfig, stepConfig, coreOpts.Providers)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}
	defer c.terminateExternalProviderInstances(extProviderConfigs, &diags)

	postCondEvalCtx := &testStepPostconditionEvalContext{
		PriorState: state,
		// We'll populate the rest gradually as we go.
	}

	planOpts := &terraform.PlanOpts{
		Mode:                    stepConfig.PlanMode,
		SetVariables:            make(terraform.InputValues, len(stepConfig.RootModule.Variables)),
		ExternalProviderConfigs: extProviderConfigs,
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
				// Terraform Core considers cty.NilVal to mean "not set",
				// thereby allowing the defaults be used instead. This is
				// different from a null value for any variable that does
				// not set nullable = true, since an explicit null can
				// override the default for a nullable variable.
				Value:      cty.NilVal,
				SourceType: terraform.ValueFromCaller,
			}
		}
	}
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
	postCondEvalCtx.PlanOpts = planOpts
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

	postCondEvalCtx.Plan = plan
	if stepConfig.ApplyPlan {
		// If we're going to apply this plan then we'll allow conditions
		// to produce unknown results at this phase as long as they produce
		// known results after apply. For a plan-only step the conditions
		// must all have known results even during the plan step.
		postCondEvalCtx.AllowUnknown = true
	}

	// We'll update our result from just the plan for now, which might give
	// us a partial result. If we reach the apply step below then we'll
	// overwrite these with final results.
	result.Checks = plan.Checks
	result.ExpectedFailures = c.prepareExpectedFailuresReport(result.Checks, stepConfig.ExpectFailure)
	result.Postconditions, moreDiags = c.evalStepPostconditions(scenarioConfig, stepConfig, postCondEvalCtx)
	diags = diags.Append(moreDiags)
	result.Status = c.stepAggregateStatus(result.Checks, result.ExpectedFailures, result.Postconditions)

	if !stepConfig.ApplyPlan {
		// If we're not actually applying the plan then we'll let the caller
		// keep using its previous config and state for the final cleanup
		// step, since swapping out config at this point is more likely to
		// make cleanup fail due to mismatches.
		return nil, diags
	}

	newState, moreDiags := core.Apply(plan, cfg, &terraform.ApplyOpts{
		// TODO: Should we close and then re-open all of these, to make this
		// more realistic for how a real plan and apply would behave?
		// In theory it shouldn't matter but in practice there might be
		// some weird lingering state in the provider that could change
		// its behavior.
		ExternalProviderConfigs: extProviderConfigs,
	})
	result.Diagnostics = result.Diagnostics.Append(moreDiags)
	cleanupCtx := &testCommandCleanupContext{
		Scenario:     scenarioConfig,
		Step:         stepConfig,
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

	postCondEvalCtx.NewState = newState
	postCondEvalCtx.AllowUnknown = false // unknowns are never allowed after apply

	result.Checks = newState.CheckResults
	result.ExpectedFailures = c.prepareExpectedFailuresReport(result.Checks, stepConfig.ExpectFailure)
	result.Postconditions, moreDiags = c.evalStepPostconditions(scenarioConfig, stepConfig, postCondEvalCtx)
	diags = diags.Append(moreDiags)
	result.Status = c.stepAggregateStatus(result.Checks, result.ExpectedFailures, result.Postconditions)

	// TODO: We also need to deal with any expected failures, which should
	// allow the overall step to succeed even if they have StatusFail.

	return cleanupCtx, diags
}

func (c *TestCommand) externalProviderConfigsForStep(scenarioConfig *testconfigs.Scenario, stepConfig *testconfigs.Step, factories map[addrs.Provider]providers.Factory) (map[addrs.RootProviderConfig]providers.Interface, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(map[addrs.RootProviderConfig]providers.Interface, len(stepConfig.Providers))

	for _, passed := range stepConfig.Providers {
		if passed.InParent == nil || passed.InChild == nil {
			diags = diags.Append(fmt.Errorf("scenario %q step %q has invalid provider configuration definition", scenarioConfig.Name, stepConfig.Name))
			continue
		}
		inScenarioAddr := passed.InParent.Addr()
		providerDecl, declared := scenarioConfig.ProviderReqs.RequiredProviders[inScenarioAddr.LocalName]
		if !declared {
			// We shouldn't be able to get here if the config decoder did all
			// of the validation it should have, but we'll handle it anyway
			// to be robust.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undeclared provider local name",
				Detail:   fmt.Sprintf("The scenario's required_providers block does not declare a provider with local name %q.", inScenarioAddr.LocalName),
				Subject:  passed.InParent.NameRange.Ptr(),
			})
			continue
		}
		providerAddr := providerDecl.Type
		factory := factories[providerAddr]
		if factory == nil {
			// This suggests that something went wrong during 'terraform init',
			// since it should've installed all of the requested providers and
			// recorded its selections in the dependency lock file, and then
			// our caller will use the dependency lock file to determine which
			// factories to send us.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider not available",
				Detail:   fmt.Sprintf("This test scenario depends on provider %s, which is not installed.\n\nTo install all providers required for this module, run the following command:\n    terraform init", providerAddr.ForDisplay()),
				Subject:  providerDecl.DeclRange.Ptr(),
			})
			continue
		}
		inModuleAddr := addrs.RootProviderConfig{
			Provider: providerAddr,
			Alias:    passed.InChild.Alias,
		}

		inst, err := factory()
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Provider plugin initialization failed",
				Detail:   fmt.Sprintf("Failed to launch the provider plugin for %s: %s.", inScenarioAddr.String(), err),
				Subject:  providerDecl.DeclRange.Ptr(),
			})
			continue
		}
		schemaResp := inst.GetProviderSchema()
		diags = diags.Append(schemaResp.Diagnostics)
		if schemaResp.Diagnostics.HasErrors() {
			inst.Close()
			continue
		}

		if mockConfig, isMock := scenarioConfig.MockProviderConfigs[inScenarioAddr]; isMock {
			// For mock providers we wrap up an unconfigured real instance
			// inside a mock instance. The mock wrapper uses the real instance
			// for all of the pre-config functionality such as config
			// validation, but intercepts any operations that would normally
			// require a configured provider and stubs them out locally
			// instead.
			mockInst, moreDiags := mockConfig.Config.Instantiate(inst)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				inst.Close()
				continue
			}
			ret[inModuleAddr] = mockInst
		} else if realConfig, isReal := scenarioConfig.RealProviderConfigs[inScenarioAddr]; isReal {
			// For a real provider we'll want to properly configure it using
			// the arguments written in the provider block.
			configSchema := schemaResp.Provider.Block
			decSpec := configSchema.DecoderSpec()
			// NOTE: Unlike in a real Terraform module, a provider configuration
			// for _testing_ must use only literal values, since test scenarios
			// are supposed to be self-contained. Any settings that relate to
			// who is running the test (credentials, etc) must be set using
			// out-of-band techniques like environment variables; the in-scenario
			// configuration is only for configuring behavior that ought to be
			// true regardless of who is running the tests.
			configVal, hclDiags := hcldec.Decode(realConfig.Config, decSpec, nil)
			diags = diags.Append(hclDiags)
			if hclDiags.HasErrors() {
				continue
			}
			validResp := inst.ValidateProviderConfig(providers.ValidateProviderConfigRequest{
				Config: configVal,
			})
			diags = diags.Append(validResp.Diagnostics)
			if validResp.Diagnostics.HasErrors() {
				continue
			}
			configResp := inst.ConfigureProvider(providers.ConfigureProviderRequest{
				TerraformVersion: version.String(),
				Config:           configVal,
			})
			diags = diags.Append(configResp.Diagnostics)
			if configResp.Diagnostics.HasErrors() {
				continue
			}
			ret[inModuleAddr] = inst
		} else {
			// We shouldn't be able to get here if the config decoder did all
			// of the validation it should have, but we'll handle it anyway
			// to be robust.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to undefined provider",
				Detail:   fmt.Sprintf("There is no real provider or mock provider configuration in this test scenario for %s.", inScenarioAddr.String()),
				Subject:  passed.InParent.NameRange.Ptr(),
			})
		}
	}

	return ret, diags
}

func (c *TestCommand) terminateExternalProviderInstances(insts map[addrs.RootProviderConfig]providers.Interface, diagsPtr *tfdiags.Diagnostics) {
	for addr, inst := range insts {
		err := inst.Close()
		if err != nil {
			*diagsPtr = diagsPtr.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to terminate provider plugin",
				fmt.Sprintf("Error when asking %s to shut down: %s.", addr.String(), err),
			))
			continue
		}
	}
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

type testStepPostconditionEvalContext struct {
	PriorState   *states.State
	PlanOpts     *terraform.PlanOpts
	Plan         *plans.Plan
	NewState     *states.State
	AllowUnknown bool
}

func (c *TestCommand) evalStepPostconditions(scenarioConfig *testconfigs.Scenario, stepConfig *testconfigs.Step, evalCtx *testStepPostconditionEvalContext) (*states.CheckResultObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if len(stepConfig.Postconditions) == 0 {
		// a nil result represents that there aren't any postconditions at
		// all, which our UI layer might use to avoid mentioning the
		// postcondition result in that case.
		return nil, diags
	}

	// If we have at least one postcondition then we will generate a non-nil
	// result that summarizes them all together, as if the test step itself
	// were just another checkable object.
	ret := &states.CheckResultObject{
		Status: checks.StatusUnknown,
	}

	// For initial prototyping we'll focus only on checking output values.
	// In later iterations it'd be interesting to also support other situations
	// that are unique to the test step context, such as:
	//    - The actions chosen for specific resources, so that e.g. an author
	//      can assert that a particular update must not cause an object to
	//      be replaced.
	//    - The specific mock responses that were used to construct individual
	//      resources in the state, if any, so authors can make sure their
	//      mocks are being exercised in the way that they intended.
	//      (The mock provider system stashes information about that in the
	//      "Private" field in the state, so the providermocks package could
	//      offer an API to get that information given a resource instance
	//      object from evalCtx.NewState.)

	hclCtx := &hcl.EvalContext{
		Variables: make(map[string]cty.Value),
	}

	// HACK: Our "lang" package isn't designed to provide functions for use
	// in languages other than the main Terraform language, so we'll need to
	// goad it into giving us the functions by pretending we're trying to
	// evaluate stuff in the main language.
	scope := &lang.Scope{
		BaseDir:  filepath.Dir(stepConfig.DeclRange.Filename),
		PureOnly: false,
	}
	hclCtx.Functions = scope.Functions()

	if evalCtx.PlanOpts != nil {
		variableVals := make(map[string]cty.Value)
		for name, def := range evalCtx.PlanOpts.SetVariables {
			variableVals[name] = def.Value
		}
		hclCtx.Variables["variables"] = cty.ObjectVal(variableVals)
	}
	if evalCtx.NewState != nil {
		outputVals := make(map[string]cty.Value)
		for name, ov := range evalCtx.NewState.RootModule().OutputValues {
			outputVals[name] = ov.Value
		}
		hclCtx.Variables["outputs"] = cty.ObjectVal(outputVals)
	} else if evalCtx.Plan != nil {
		// If we only planned but didn't apply then the plan is an alternative
		// source of (possibly-incomplete) output values.
		outputVals := make(map[string]cty.Value)
		for _, ovc := range evalCtx.Plan.Changes.Outputs {
			addr := ovc.Addr
			if !addr.Module.IsRoot() {
				continue
			}
			name := addr.OutputValue.Name
			val, err := ovc.After.Decode(cty.DynamicPseudoType)
			if err != nil {
				// FIXME: Should handle this error properly
				continue
			}
			outputVals[name] = val
		}
		hclCtx.Variables["outputs"] = cty.ObjectVal(outputVals)
	}

	statuses := make([]checks.Status, len(stepConfig.Postconditions))
	for i, cond := range stepConfig.Postconditions {
		resultVal, hclDiags := cond.Condition.Value(hclCtx)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			statuses[i] = checks.StatusError
			continue
		}

		resultVal, err := convert.Convert(resultVal, cty.Bool)
		const invalidCondResult = "Invalid condition result"
		if err != nil {
			statuses[i] = checks.StatusError
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     invalidCondResult,
				Detail:      fmt.Sprintf("Condition expression produced an unsuitable result: %s.", tfdiags.FormatError(err)),
				Subject:     cond.Condition.Range().Ptr(),
				EvalContext: hclCtx,
				Expression:  cond.Condition,
			})
			continue
		}
		if resultVal.IsNull() {
			statuses[i] = checks.StatusError
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     invalidCondResult,
				Detail:      "Condition expression produced an unsuitable result: must not be null.",
				Subject:     cond.Condition.Range().Ptr(),
				EvalContext: hclCtx,
				Expression:  cond.Condition,
			})
			continue
		}

		statuses[i] = checks.StatusForCtyValue(resultVal)

		// The message expression must be valid regardless of whether the
		// condition actually passes.
		msgVal, hclDiags := cond.ErrorMessage.Value(hclCtx)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			statuses[i] = checks.StatusError
			continue
		}
		var msg string
		if msgVal.IsKnown() {
			if err := gocty.FromCtyValue(msgVal, &msg); err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid condition error message",
					Detail:      fmt.Sprintf("Error message expression produced an unsuitable result: %s.", tfdiags.FormatError(err)),
					Subject:     cond.ErrorMessage.Range().Ptr(),
					EvalContext: hclCtx,
					Expression:  cond.ErrorMessage,
				})
			}
		} else {
			// We'll only complain about an unknown error message if the
			// result is known, because we have a separate message complaining
			// about an unknown result below and one error seems like enough.
			if statuses[i] != checks.StatusUnknown {
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     "Invalid condition error message",
					Detail:      "Error message expression produced an unsuitable result: must refer only to known values.",
					Subject:     cond.ErrorMessage.Range().Ptr(),
					EvalContext: hclCtx,
					Expression:  cond.ErrorMessage,
				})
			}
		}

		if statuses[i] == checks.StatusFail {
			// We need to evaluate the error message and insert it into our
			// set of failure messages, then.
			ret.FailureMessages = append(ret.FailureMessages, msg)
			continue
		}
		if statuses[i] == checks.StatusUnknown {
			if !evalCtx.AllowUnknown {
				statuses[i] = checks.StatusError
				diags = diags.Append(&hcl.Diagnostic{
					Severity:    hcl.DiagError,
					Summary:     invalidCondResult,
					Detail:      "Condition expression produced an unsuitable result: depends on values known only after apply, but this step did not apply the changes.",
					Subject:     cond.Condition.Range().Ptr(),
					EvalContext: hclCtx,
					Expression:  cond.Condition,
				})
			}
			continue
		}
	}

	ret.Status = checks.AggregateCheckStatus(statuses...)

	return ret, diags
}

func (c *TestCommand) stepAggregateStatus(checkResults *states.CheckResults, expectedFails addrs.Map[addrs.Checkable, checks.Status], postconds *states.CheckResultObject) checks.Status {
	postcondStatus := checks.StatusPass // default if there aren't any postconditions
	if postconds != nil {
		postcondStatus = postconds.Status
	}
	return checks.AggregateCheckStatus(
		checks.AggregateCheckStatusAddrsMap(
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
		),
		postcondStatus,
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

	// We'll use provider instances equivalent to the ones used to run the
	// step that established this state, so that we'll have the best chance
	// of succeeding.
	extProviderConfigs, moreDiags := c.externalProviderConfigsForStep(
		cleanupCtx.Scenario,
		cleanupCtx.Step,
		coreOpts.Providers,
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}
	defer c.terminateExternalProviderInstances(extProviderConfigs, &diags)

	planOpts := &terraform.PlanOpts{
		Mode:                    plans.DestroyMode,
		SetVariables:            cleanupCtx.SetVariables,
		ExternalProviderConfigs: extProviderConfigs,
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

	_, moreDiags = core.Apply(plan, cleanupCtx.Config, &terraform.ApplyOpts{
		// TODO: Should we close and then re-open all of these, to make this
		// more realistic for how a real plan and apply would behave?
		// In theory it shouldn't matter but in practice there might be
		// some weird lingering state in the provider that could change
		// its behavior.
		ExternalProviderConfigs: extProviderConfigs,
	})
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
	// Scenario is the configuration for the scenario that the step
	// in [Step] belongs to.
	Scenario *testconfigs.Scenario

	// Step is the configuration for the step that established the
	// other values in this object.
	Step *testconfigs.Step

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

  -mock-only         Skip any test scenarios that use real provider
                     configurations. Terraform will still run scenarios which
                     exclusively use mock providers.

  -no-color          Don't include virtual terminal formatting sequences in
                     the output.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Experimental support for module integration testing"
}
