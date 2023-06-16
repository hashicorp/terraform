package command

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type TestCommand struct {
	Meta
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform [global options] test [options]

  Executes automated integration tests against the current Terraform 
  configuration.

  Terraform will search for .tftest files within the current configuration and 
  testing directories. Terraform will then execute the testing run blocks within
  any testing files in order, and verify conditional checks and assertions 
  against the created infrastructure. 

  This command creates real infrastructure and will attempt to clean up the
  testing infrastructure on completion. Monitor the output carefully to ensure
  this cleanup process is successful.

Options:

  TODO: implement optional arguments.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Execute integration tests for Terraform modules"
}

func (c *TestCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	common, _ := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	view := views.NewTest(arguments.ViewHuman, c.View)

	config, configDiags := c.loadConfigWithTests(".", "tests")
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	suite := moduletest.Suite{
		Files: func() map[string]*moduletest.File {
			files := make(map[string]*moduletest.File)
			for name, file := range config.Module.Tests {
				var runs []*moduletest.Run
				for _, run := range file.Runs {
					runs = append(runs, &moduletest.Run{
						Config: run,
						Name:   run.Name,
					})
				}
				files[name] = &moduletest.File{
					Config: file,
					Name:   name,
					Runs:   runs,
				}
			}
			return files
		}(),
	}

	view.Abstract(&suite)
	c.ExecuteTestSuite(&suite, config, view)
	view.Conclusion(&suite)

	if suite.Status != moduletest.Pass {
		return 1
	}
	return 0
}

func (c *TestCommand) ExecuteTestSuite(suite *moduletest.Suite, config *configs.Config, view views.Test) {
	var diags tfdiags.Diagnostics

	opts, err := c.contextOpts()
	diags = diags.Append(err)
	if err != nil {
		suite.Status = suite.Status.Merge(moduletest.Error)
		view.Diagnostics(nil, nil, diags)
		return
	}

	ctx, ctxDiags := terraform.NewContext(opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		suite.Status = suite.Status.Merge(moduletest.Error)
		view.Diagnostics(nil, nil, diags)
		return
	}
	view.Diagnostics(nil, nil, diags) // Print out any warnings from the setup.

	var files []string
	for name := range suite.Files {
		files = append(files, name)
	}
	sort.Strings(files) // execute the files in alphabetical order

	suite.Status = moduletest.Pass
	for _, name := range files {
		file := suite.Files[name]
		c.ExecuteTestFile(ctx, file, config, view)

		suite.Status = suite.Status.Merge(file.Status)
	}
}

func (c *TestCommand) ExecuteTestFile(ctx *terraform.Context, file *moduletest.File, config *configs.Config, view views.Test) {

	mgr := new(TestStateManager)
	mgr.c = c
	mgr.State = states.NewState()
	defer mgr.cleanupStates(ctx, view, file, config)

	file.Status = file.Status.Merge(moduletest.Pass)
	for _, run := range file.Runs {
		if file.Status == moduletest.Error {
			// If the overall test file has errored, we don't keep trying to
			// execute tests. Instead, we mark all remaining run blocks as
			// skipped.
			run.Status = moduletest.Skip
			continue
		}

		if run.Config.ConfigUnderTest != nil {
			// Then we want to execute a different module under a kind of
			// sandbox.
			state := c.ExecuteTestRun(ctx, run, file, states.NewState(), run.Config.ConfigUnderTest)
			mgr.States = append(mgr.States, &TestModuleState{
				State: state,
				Run:   run,
			})
		} else {
			mgr.State = c.ExecuteTestRun(ctx, run, file, mgr.State, config)
		}
		file.Status = file.Status.Merge(run.Status)
	}

	view.File(file)
	for _, run := range file.Runs {
		view.Run(run, file)
	}
}

func (c *TestCommand) ExecuteTestRun(ctx *terraform.Context, run *moduletest.Run, file *moduletest.File, state *states.State, config *configs.Config) *states.State {

	// Since we don't want to modify the actual plan and apply operations for
	// tests where possible, we insert provider blocks directly into the config
	// under test for each test run.
	//
	// This function transforms the config under test by inserting relevant
	// provider blocks. It returns a reset function which restores the config
	// back to the original state.
	cfgReset, cfgDiags := config.TransformForTest(run.Config, file.Config)
	defer cfgReset()
	run.Diagnostics = run.Diagnostics.Append(cfgDiags)
	if cfgDiags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	var targets []addrs.Targetable
	for _, target := range run.Config.Options.Target {
		addr, diags := addrs.ParseTarget(target)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		targets = append(targets, addr.Subject)
	}

	var replaces []addrs.AbsResourceInstance
	for _, replace := range run.Config.Options.Replace {
		addr, diags := addrs.ParseAbsResourceInstance(replace)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			run.Diagnostics = run.Diagnostics.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "can only target managed resources for forced replacements",
				Detail:   addr.String(),
				Subject:  replace.SourceRange().Ptr(),
			})
			return state
		}

		replaces = append(replaces, addr)
	}

	variables, diags := c.GetInputValues(run.Config.Variables, file.Config.Variables, config)
	run.Diagnostics = run.Diagnostics.Append(diags)
	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	var references []*addrs.Reference
	for _, assert := range run.Config.CheckRules {
		for _, variable := range assert.Condition.Variables() {
			reference, diags := addrs.ParseRef(variable)
			run.Diagnostics = run.Diagnostics.Append(diags)
			references = append(references, reference)
		}
	}
	if run.Diagnostics.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	plan, diags := ctx.Plan(config, state, &terraform.PlanOpts{
		Mode: func() plans.Mode {
			switch run.Config.Options.Mode {
			case configs.RefreshOnlyTestMode:
				return plans.RefreshOnlyMode
			default:
				return plans.NormalMode
			}
		}(),
		SetVariables:       variables,
		Targets:            targets,
		ForceReplace:       replaces,
		SkipRefresh:        !run.Config.Options.Refresh,
		ExternalReferences: references,
	})
	run.Diagnostics = run.Diagnostics.Append(diags)
	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	if run.Config.Command == configs.ApplyTestCommand {
		state, diags = ctx.Apply(plan, config)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		ctx.TestContext(config, state, plan, variables).EvaluateAgainstState(run)
		return state
	}

	ctx.TestContext(config, plan.PlannedState, plan, variables).EvaluateAgainstPlan(run)
	return state
}

func (c *TestCommand) GetInputValues(locals map[string]hcl.Expression, globals map[string]hcl.Expression, config *configs.Config) (terraform.InputValues, tfdiags.Diagnostics) {
	variables := make(map[string]hcl.Expression)
	for name := range config.Module.Variables {
		if expr, exists := locals[name]; exists {
			// Local variables take precedence.
			variables[name] = expr
			continue
		}

		if expr, exists := globals[name]; exists {
			// If it's not set locally, it maybe set globally.
			variables[name] = expr
			continue
		}

		// If it's not set at all that might be okay if the variable is optional
		// so we'll just not add anything to the map.
	}

	unparsed := make(map[string]backend.UnparsedVariableValue)
	for key, value := range variables {
		unparsed[key] = unparsedVariableValueExpression{
			expr:       value,
			sourceType: terraform.ValueFromConfig,
		}
	}
	return backend.ParseVariableValues(unparsed, config.Module.Variables)
}

func (c *TestCommand) cleanupState(ctx *terraform.Context, view views.Test, run *moduletest.Run, file *moduletest.File, config *configs.Config, state *states.State) {
	if state.Empty() {
		// Nothing to do.
		return
	}

	var locals, globals map[string]hcl.Expression
	if run != nil {
		locals = run.Config.Variables
	}
	if file != nil {
		globals = file.Config.Variables
	}

	var cfgDiags tfdiags.Diagnostics
	if run == nil {
		cfgReset, diags := config.TransformForTest(nil, file.Config)
		defer cfgReset()
		cfgDiags = cfgDiags.Append(diags)
	} else {
		cfgReset, diags := config.TransformForTest(run.Config, file.Config)
		defer cfgReset()
		cfgDiags = cfgDiags.Append(diags)
	}
	if cfgDiags.HasErrors() {
		// This shouldn't really trigger, as we will have applied this transform
		// earlier and it will have worked so a problem now would be strange.
		// To be safe, we'll handle it anyway.
		view.DestroySummary(cfgDiags, run, file, state)
		return
	}
	c.View.Diagnostics(cfgDiags)

	variables, variableDiags := c.GetInputValues(locals, globals, config)
	if variableDiags.HasErrors() {
		// This shouldn't really trigger, as we will have created something
		// using these variables at an earlier stage so for them to have a
		// problem now would be strange. But just to be safe we'll handle this.
		view.DestroySummary(variableDiags, run, file, state)
		return
	}
	view.Diagnostics(nil, file, variableDiags)

	plan, planDiags := ctx.Plan(config, state, &terraform.PlanOpts{
		Mode:         plans.DestroyMode,
		SetVariables: variables,
	})
	if planDiags.HasErrors() {
		// This is bad, we need to tell the user that we couldn't clean up
		// and they need to go and manually delete some resources.
		view.DestroySummary(planDiags, run, file, state)
		return
	}
	view.Diagnostics(nil, file, planDiags) // Print out any warnings from the destroy plan.

	finalState, applyDiags := ctx.Apply(plan, config)
	view.DestroySummary(applyDiags, run, file, finalState)
}

// TestStateManager is a helper struct to maintain the various state objects
// that a test file has to keep track of.
type TestStateManager struct {
	c *TestCommand

	// State is the main state of the module under test during a single test
	// file execution. This state will be updated by every run block without
	// a modifier module block within the test file. At the end of the test
	// file's execution everything in this state should be executed.
	State *states.State

	// States contains the states of every run block within a test file that
	// executed using an alternative module. Any resources created by these
	// run blocks also need to be tidied up, but only after the main state file
	// has been handled.
	States []*TestModuleState
}

// TestModuleState holds the config and the state for a given run block that
// executed with a custom module.
type TestModuleState struct {
	// State is the state after the module executed.
	State *states.State

	// File is the config for the file containing the Run.
	File *moduletest.File

	// Run is the config for the given run block, that contains the config
	// under test and the variable values.
	Run *moduletest.Run
}

func (manager *TestStateManager) cleanupStates(ctx *terraform.Context, view views.Test, file *moduletest.File, config *configs.Config) {
	// First, we'll clean up the main state.
	manager.c.cleanupState(ctx, view, nil, file, config, manager.State)

	// Then we'll clean up the additional states for custom modules in reverse
	// order.
	for ix := len(manager.States); ix > 0; ix-- {
		state := manager.States[ix-1]
		manager.c.cleanupState(ctx, view, state.Run, file, state.Run.Config.ConfigUnderTest, state.State)
	}
}
