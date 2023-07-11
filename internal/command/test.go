package command

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
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

  -filter=testfile      If specified, Terraform will only execute the test files
                        specified by this flag. You can use this option multiple
                        times to execute more than one test file.

  -json                 If specified, machine readable output will be printed in
                        JSON format

  -test-directory=path	Set the Terraform test directory, defaults to "tests".    

  -var 'foo=bar'        Set a value for one of the input variables in the root
                        module of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.

  -verbose              Print the plan or state for each test run block as it
                        executes.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Execute integration tests for Terraform modules"
}

func (c *TestCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseTest(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("test")
		return 1
	}

	view := views.NewTest(args.ViewType, c.View)

	config, configDiags := c.loadConfigWithTests(".", args.TestDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	var fileDiags tfdiags.Diagnostics
	suite := moduletest.Suite{
		Files: func() map[string]*moduletest.File {
			files := make(map[string]*moduletest.File)

			if len(args.Filter) > 0 {
				for _, name := range args.Filter {
					file, ok := config.Module.Tests[name]
					if !ok {
						// If the filter is invalid, we'll simply skip this
						// entry and print a warning. But we could still execute
						// any other tests within the filter.
						fileDiags.Append(tfdiags.Sourceless(
							tfdiags.Warning,
							"Unknown test file",
							fmt.Sprintf("The specified test file, %s, could not be found.", name)))
						continue
					}

					var runs []*moduletest.Run
					for ix, run := range file.Runs {
						runs = append(runs, &moduletest.Run{
							Config: run,
							Index:  ix,
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
			}

			// Otherwise, we'll just do all the tests in the directory!
			for name, file := range config.Module.Tests {
				var runs []*moduletest.Run
				for ix, run := range file.Runs {
					runs = append(runs, &moduletest.Run{
						Config: run,
						Index:  ix,
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

	diags = diags.Append(fileDiags)
	if fileDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	// Users can also specify variables via the command line, so we'll parse
	// all that here.
	var items []rawFlag
	for _, variable := range args.Vars.All() {
		items = append(items, rawFlag{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	c.variableArgs = rawFlags{items: &items}

	variables, variableDiags := c.collectVariableValues()
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	runningCtx, done := context.WithCancel(context.Background())
	stopCtx, stop := context.WithCancel(runningCtx)
	cancelCtx, cancel := context.WithCancel(context.Background())

	runner := &TestRunner{
		command: c,

		Suite:  &suite,
		Config: config,
		View:   view,

		CancelledCtx: cancelCtx,
		StoppedCtx:   stopCtx,

		// Just to be explicit, we'll set the following fields even though they
		// default to these values.
		Cancelled: false,
		Stopped:   false,

		Verbose: args.Verbose,
	}

	view.Abstract(&suite)

	go func() {
		defer logging.PanicHandler()
		defer done() // We completed successfully.
		defer stop()
		defer cancel()

		runner.Start(variables)
	}()

	// Wait for the operation to complete, or for an interrupt to occur.
	select {
	case <-c.ShutdownCh:
		// Nice request to be cancelled.

		view.Interrupted()
		runner.Stopped = true
		stop()

		select {
		case <-c.ShutdownCh:
			// The user pressed it again, now we have to get it to stop as
			// fast as possible.

			view.FatalInterrupt()
			runner.Cancelled = true
			cancel()

			// TODO(liamcervante): Should we add a timer here? That would mean
			//   after 5 seconds we just give up and don't even print out the
			//   lists of resources left behind?
			<-runningCtx.Done() // Nothing left to do now but wait.

		case <-runningCtx.Done():
			// The application finished nicely after the request was stopped.
		}
	case <-runningCtx.Done():
		// tests finished normally with no interrupts.
	}

	if runner.Cancelled {
		// Don't print out the conclusion if the test was cancelled.
		return 1
	}

	view.Conclusion(&suite)

	if suite.Status != moduletest.Pass {
		return 1
	}
	return 0
}

// test runner

type TestRunner struct {
	command *TestCommand

	Suite  *moduletest.Suite
	Config *configs.Config

	View views.Test

	// Stopped and Cancelled track whether the user requested the testing
	// process to be interrupted. Stopped is a nice graceful exit, we'll still
	// tidy up any state that was created and mark the tests with relevant
	// `skipped` status updates. Cancelled is a hard stop right now exit, we
	// won't attempt to clean up any state left hanging, and tests will just
	// be left showing `pending` as the status. We will still print out the
	// destroy summary diagnostics that tell the user what state has been left
	// behind and needs manual clean up.
	Stopped   bool
	Cancelled bool

	// StoppedCtx and CancelledCtx allow in progress Terraform operations to
	// respond to external calls from the test command.
	StoppedCtx   context.Context
	CancelledCtx context.Context

	// Verbose tells the runner to print out plan files during each test run.
	Verbose bool
}

func (runner *TestRunner) Start(globals map[string]backend.UnparsedVariableValue) {
	var files []string
	for name := range runner.Suite.Files {
		files = append(files, name)
	}
	sort.Strings(files) // execute the files in alphabetical order

	runner.Suite.Status = moduletest.Pass
	for _, name := range files {
		if runner.Cancelled {
			return
		}

		file := runner.Suite.Files[name]
		runner.ExecuteTestFile(file, globals)
		runner.Suite.Status = runner.Suite.Status.Merge(file.Status)
	}
}

func (runner *TestRunner) ExecuteTestFile(file *moduletest.File, globals map[string]backend.UnparsedVariableValue) {
	mgr := new(TestStateManager)
	mgr.runner = runner
	mgr.State = states.NewState()
	defer mgr.cleanupStates(file, globals)

	file.Status = file.Status.Merge(moduletest.Pass)
	for _, run := range file.Runs {
		if runner.Cancelled {
			// This means a hard stop has been requested, in this case we don't
			// even stop to mark future tests as having been skipped. They'll
			// just show up as pending in the printed summary.
			return
		}

		if runner.Stopped {
			// Then the test was requested to be stopped, so we just mark each
			// following test as skipped and move on.
			run.Status = moduletest.Skip
			continue
		}

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
			state := runner.ExecuteTestRun(run, file, states.NewState(), run.Config.ConfigUnderTest, globals)
			mgr.States = append(mgr.States, &TestModuleState{
				State: state,
				Run:   run,
			})
		} else {
			mgr.State = runner.ExecuteTestRun(run, file, mgr.State, runner.Config, globals)
		}
		file.Status = file.Status.Merge(run.Status)
	}

	runner.View.File(file)
	for _, run := range file.Runs {
		runner.View.Run(run, file)
	}
}

func (runner *TestRunner) ExecuteTestRun(run *moduletest.Run, file *moduletest.File, state *states.State, config *configs.Config, globals map[string]backend.UnparsedVariableValue) *states.State {
	if runner.Cancelled {
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return state
	}

	if runner.Stopped {
		// Basically the same as above, except we'll be a bit nicer.
		run.Status = moduletest.Skip
		return state
	}

	targets, diags := run.GetTargets()
	run.Diagnostics = run.Diagnostics.Append(diags)

	replaces, diags := run.GetReplaces()
	run.Diagnostics = run.Diagnostics.Append(diags)

	references, diags := run.GetReferences()
	run.Diagnostics = run.Diagnostics.Append(diags)

	if run.Diagnostics.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	ctx, plan, state, diags := runner.execute(run, file, config, state, &terraform.PlanOpts{
		Mode: func() plans.Mode {
			switch run.Config.Options.Mode {
			case configs.RefreshOnlyTestMode:
				return plans.RefreshOnlyMode
			default:
				return plans.NormalMode
			}
		}(),
		Targets:            targets,
		ForceReplace:       replaces,
		SkipRefresh:        !run.Config.Options.Refresh,
		ExternalReferences: references,
	}, run.Config.Command, globals)
	run.Diagnostics = run.Diagnostics.Append(diags)

	if runner.Cancelled {
		// Print out the diagnostics from the run now, since it was cancelled
		// the normal set of diagnostics will not be printed otherwise.
		runner.View.Diagnostics(run, file, run.Diagnostics)
		run.Status = moduletest.Error
		return state
	}

	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	if runner.Stopped {
		run.Status = moduletest.Skip
		return state
	}

	// If the user wants to render the plans as part of the test output, we
	// track that here.
	if runner.Verbose {
		schemas, diags := ctx.Schemas(config, state)

		// If we're going to fail to render the plan, let's not fail the overall
		// test. It can still have succeeded. So we'll add the diagnostics, but
		// still report the test status as a success.
		if diags.HasErrors() {
			// This is very unlikely.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Failed to print verbose output",
				fmt.Sprintf("Terraform failed to print the verbose output for %s, other diagnostics will contain more details as to why.", path.Join(file.Name, run.Name))))
		} else {
			run.Verbose = &moduletest.Verbose{
				Plan:         plan,
				State:        state,
				Config:       config,
				Providers:    schemas.Providers,
				Provisioners: schemas.Provisioners,
			}
		}

		run.Diagnostics = run.Diagnostics.Append(diags)
	}

	variables, diags := buildInputVariablesForAssertions(run, file, config, globals)
	run.Diagnostics = run.Diagnostics.Append(diags)
	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	if run.Config.Command == configs.ApplyTestCommand {
		ctx.TestContext(config, state, plan, variables).EvaluateAgainstState(run)
		return state
	}

	ctx.TestContext(config, plan.PlannedState, plan, variables).EvaluateAgainstPlan(run)
	return state
}

// execute executes Terraform plan and apply operations for the given arguments.
//
// The command argument decides whether it executes only a plan or also applies
// the plan it creates during the planning.
func (runner *TestRunner) execute(run *moduletest.Run, file *moduletest.File, config *configs.Config, state *states.State, opts *terraform.PlanOpts, command configs.TestCommand, globals map[string]backend.UnparsedVariableValue) (*terraform.Context, *plans.Plan, *states.State, tfdiags.Diagnostics) {
	if opts.Mode == plans.DestroyMode && state.Empty() {
		// Nothing to do!
		return nil, nil, state, nil
	}

	identifier := file.Name
	if run != nil {
		identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
	}

	// First, transform the config for the given test run and test file.

	var diags tfdiags.Diagnostics
	if run == nil {
		reset, cfgDiags := config.TransformForTest(nil, file.Config)
		defer reset()
		diags = diags.Append(cfgDiags)
	} else {
		reset, cfgDiags := config.TransformForTest(run.Config, file.Config)
		defer reset()
		diags = diags.Append(cfgDiags)
	}
	if diags.HasErrors() {
		return nil, nil, state, diags
	}

	// Second, gather any variables and give them to the plan options.

	variables, variableDiags := buildInputVariablesForTest(run, file, config, globals)
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		return nil, nil, state, diags
	}
	opts.SetVariables = variables

	// Third, execute planning stage.

	tfCtxOpts, err := runner.command.contextOpts()
	diags = diags.Append(err)
	if err != nil {
		return nil, nil, state, diags
	}

	tfCtx, ctxDiags := terraform.NewContext(tfCtxOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, nil, state, diags
	}

	runningCtx, done := context.WithCancel(context.Background())

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	go func() {
		defer done()
		plan, planDiags = tfCtx.Plan(config, state, opts)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, opts, identifier)
	planDiags = planDiags.Append(waitDiags)

	diags = diags.Append(planDiags)
	if planDiags.HasErrors() || command == configs.PlanTestCommand {
		// Either the plan errored, or we only wanted to see the plan. Either
		// way, just return what we have: The plan and diagnostics from making
		// it and the unchanged state.
		return tfCtx, plan, state, diags
	}

	if cancelled {
		// If the execution was cancelled during the plan, we'll exit here to
		// stop the plan being applied and using more time.
		return tfCtx, plan, state, diags
	}

	// Fourth, execute apply stage.
	tfCtx, ctxDiags = terraform.NewContext(tfCtxOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, nil, state, diags
	}

	runningCtx, done = context.WithCancel(context.Background())

	var updated *states.State
	var applyDiags tfdiags.Diagnostics

	go func() {
		defer done()
		updated, applyDiags = tfCtx.Apply(plan, config)
	}()
	waitDiags, _ = runner.wait(tfCtx, runningCtx, opts, identifier)
	applyDiags = applyDiags.Append(waitDiags)

	diags = diags.Append(applyDiags)
	return tfCtx, plan, updated, diags
}

func (runner *TestRunner) wait(ctx *terraform.Context, runningCtx context.Context, opts *terraform.PlanOpts, identifier string) (diags tfdiags.Diagnostics, cancelled bool) {
	select {
	case <-runner.StoppedCtx.Done():

		if opts.Mode != plans.DestroyMode {
			// It takes more impetus from the user to cancel the cleanup
			// operations, so we only do this during the actual tests.
			cancelled = true
			go ctx.Stop()
		}

		select {
		case <-runner.CancelledCtx.Done():

			// If the user still really wants to cancel, then we'll oblige
			// even during the destroy mode at this point.
			if opts.Mode == plans.DestroyMode {
				cancelled = true
				go ctx.Stop()
			}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Terraform Test Interrupted",
				fmt.Sprintf("Terraform test was interrupted while executing %s. This means resources that were created during the test may have been left active, please monitor the rest of the output closely as any dangling resources will be listed.", identifier)))

			// It is actually quite disastrous if we exist early at this
			// point as it means we'll have created resources that we
			// haven't tracked at all. So for now, we won't ever actually
			// forcibly terminate the test. When cancelled, we make the
			// clean up faster by not performing it but we should still
			// always manage it give an accurate list of resources left
			// alive.
			// TODO(liamcervante): Consider adding a timer here, so that we
			//   exit early even if that means some resources are just lost
			//   forever.
			<-runningCtx.Done() // Just wait for things to finish now.

		case <-runningCtx.Done():
			// The operation exited nicely when asked!
		}
	case <-runner.CancelledCtx.Done():
		// This shouldn't really happen, as we'd expect to see the StoppedCtx
		// being triggered first. But, just in case.
		cancelled = true
		go ctx.Stop()

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Terraform Test Interrupted",
			fmt.Sprintf("Terraform test was interrupted while executing %s. This means resources that were created during the test may have been left active, please monitor the rest of the output closely as any dangling resources will be listed.", identifier)))

		// It is actually quite disastrous if we exist early at this
		// point as it means we'll have created resources that we
		// haven't tracked at all. So for now, we won't ever actually
		// forcibly terminate the test. When cancelled, we make the
		// clean up faster by not performing it but we should still
		// always manage it give an accurate list of resources left
		// alive.
		// TODO(liamcervante): Consider adding a timer here, so that we
		//   exit early even if that means some resources are just lost
		//   forever.
		<-runningCtx.Done() // Just wait for things to finish now.

	case <-runningCtx.Done():
		// The operation exited normally.
	}

	return diags, cancelled
}

// state management

// TestStateManager is a helper struct to maintain the various state objects
// that a test file has to keep track of.
type TestStateManager struct {
	runner *TestRunner

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

	// Run is the config for the given run block, that contains the config
	// under test and the variable values.
	Run *moduletest.Run
}

func (manager *TestStateManager) cleanupStates(file *moduletest.File, globals map[string]backend.UnparsedVariableValue) {
	if manager.runner.Cancelled {

		// We are still going to print out the resources that we have left
		// even though the user asked for an immediate exit.

		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Test cleanup skipped due to immediate exit", "Terraform could not clean up the state left behind due to immediate interrupt."))
		manager.runner.View.DestroySummary(diags, nil, file, manager.State)

		for _, module := range manager.States {
			manager.runner.View.DestroySummary(diags, module.Run, file, module.State)
		}

		return
	}

	// First, we'll clean up the main state.
	_, _, state, diags := manager.runner.execute(nil, file, manager.runner.Config, manager.State, &terraform.PlanOpts{
		Mode: plans.DestroyMode,
	}, configs.ApplyTestCommand, globals)
	manager.runner.View.DestroySummary(diags, nil, file, state)

	// Then we'll clean up the additional states for custom modules in reverse
	// order.
	for ix := len(manager.States); ix > 0; ix-- {
		module := manager.States[ix-1]

		if manager.runner.Cancelled {
			// In case the cancellation came while a previous state was being
			// destroyed.
			manager.runner.View.DestroySummary(diags, module.Run, file, module.State)
			continue
		}

		_, _, state, diags := manager.runner.execute(module.Run, file, module.Run.Config.ConfigUnderTest, module.State, &terraform.PlanOpts{
			Mode: plans.DestroyMode,
		}, configs.ApplyTestCommand, globals)
		manager.runner.View.DestroySummary(diags, module.Run, file, state)
	}
}

// helper functions

// buildInputVariablesForTest creates a terraform.InputValues mapping for
// variable values that are relevant to the config being tested.
//
// Crucially, it differs from buildInputVariablesForAssertions in that it only
// includes variables that are reference by the config and not everything that
// is defined within the test run block and test file.
func buildInputVariablesForTest(run *moduletest.Run, file *moduletest.File, config *configs.Config, globals map[string]backend.UnparsedVariableValue) (terraform.InputValues, tfdiags.Diagnostics) {
	variables := make(map[string]backend.UnparsedVariableValue)
	for name := range config.Module.Variables {
		if run != nil {
			if expr, exists := run.Config.Variables[name]; exists {
				// Local variables take precedence.
				variables[name] = unparsedVariableValueExpression{
					expr:       expr,
					sourceType: terraform.ValueFromConfig,
				}
				continue
			}
		}

		if file != nil {
			if expr, exists := file.Config.Variables[name]; exists {
				// If it's not set locally, it maybe set for the entire file.
				variables[name] = unparsedVariableValueExpression{
					expr:       expr,
					sourceType: terraform.ValueFromConfig,
				}
				continue
			}
		}

		if globals != nil {
			// If it's not set locally or at the file level, maybe it was
			// defined globally.
			if variable, exists := globals[name]; exists {
				variables[name] = variable
			}
		}

		// If it's not set at all that might be okay if the variable is optional
		// so we'll just not add anything to the map.
	}

	return backend.ParseVariableValues(variables, config.Module.Variables)
}

// buildInputVariablesForAssertions creates a terraform.InputValues mapping that
// contains all the variables defined for a given run and file, alongside any
// unset variables that have defaults within the provided config.
//
// Crucially, it differs from buildInputVariablesForTest in that the returned
// input values include all variables available even if they are not defined
// within the config.
//
// This does mean the returned diags might contain warnings about variables not
// defined within the config. We might want to remove these warnings in the
// future, since it is actually okay for test files to have variables defined
// outside the configuration.
func buildInputVariablesForAssertions(run *moduletest.Run, file *moduletest.File, config *configs.Config, globals map[string]backend.UnparsedVariableValue) (terraform.InputValues, tfdiags.Diagnostics) {
	variables := make(map[string]backend.UnparsedVariableValue)

	if run != nil {
		for name, expr := range run.Config.Variables {
			variables[name] = unparsedVariableValueExpression{
				expr:       expr,
				sourceType: terraform.ValueFromConfig,
			}
		}
	}

	if file != nil {
		for name, expr := range file.Config.Variables {
			if _, exists := variables[name]; exists {
				// Then this variable was defined at the run level and we want
				// that value to take precedence.
				continue
			}
			variables[name] = unparsedVariableValueExpression{
				expr:       expr,
				sourceType: terraform.ValueFromConfig,
			}
		}
	}

	if globals != nil {
		for name, variable := range globals {
			if _, exists := variables[name]; exists {
				// Then this value was already defined at either the run level
				// or the file level, and we want those values to take
				// precedence.
				continue
			}
			variables[name] = variable
		}
	}

	return backend.ParseVariableValues(variables, config.Module.Variables)
}
