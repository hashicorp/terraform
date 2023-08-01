package command

import (
	"context"
	"fmt"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
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

  Terraform will search for .tftest.hcl files within the current configuration 
  and testing directories. Terraform will then execute the testing run blocks 
  within any testing files in order, and verify conditional checks and 
  assertions against the created infrastructure. 

  This command creates real infrastructure and will attempt to clean up the
  testing infrastructure on completion. Monitor the output carefully to ensure
  this cleanup process is successful.

Options:

  -filter=testfile      If specified, Terraform will only execute the test files
                        specified by this flag. You can use this option multiple
                        times to execute more than one test file.

  -json                 If specified, machine readable output will be printed in
                        JSON format

  -no-color             If specified, output won't contain any color.

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

	runCount := 0
	fileCount := 0

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

					fileCount++

					var runs []*moduletest.Run
					for ix, run := range file.Runs {
						runs = append(runs, &moduletest.Run{
							Config: run,
							Index:  ix,
							Name:   run.Name,
						})
					}

					runCount += len(runs)
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
				fileCount++

				var runs []*moduletest.Run
				for ix, run := range file.Runs {
					runs = append(runs, &moduletest.Run{
						Config: run,
						Index:  ix,
						Name:   run.Name,
					})
				}

				runCount += len(runs)
				files[name] = &moduletest.File{
					Config: file,
					Name:   name,
					Runs:   runs,
				}
			}
			return files
		}(),
	}

	log.Printf("[DEBUG] TestCommand: found %d files with %d run blocks", fileCount, runCount)

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

	// Print out all the diagnostics we have from the setup. These will just be
	// warnings, and we want them out of the way before we start the actual
	// testing.
	view.Diagnostics(nil, nil, diags)

	// We have two levels of interrupt here. A 'stop' and a 'cancel'. A 'stop'
	// is a soft request to stop. We'll finish the current test, do the tidy up,
	// but then skip all remaining tests and run blocks. A 'cancel' is a hard
	// request to stop now. We'll cancel the current operation immediately
	// even if it's a delete operation, and we won't clean up any infrastructure
	// if we're halfway through a test. We'll print details explaining what was
	// stopped so the user can do their best to recover from it.

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
		defer done()
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

			// We'll wait 5 seconds for this operation to finish now, regardless
			// of whether it finishes successfully or not.
			select {
			case <-runningCtx.Done():
			case <-time.After(5 * time.Second):
			}

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
	log.Printf("[TRACE] TestRunner: executing test file %s", file.Name)

	printAll := func() {
		runner.View.File(file)
		for _, run := range file.Runs {
			runner.View.Run(run, file)
		}
	}

	mgr := new(TestStateManager)
	mgr.runner = runner
	mgr.State = states.NewState()

	// We're going to check if the cleanupStates function call will actually
	// work before we start the test.
	mgr.prepare(file, globals)
	if runner.Cancelled {
		return // Don't print anything just stop.
	}

	if file.Diagnostics.HasErrors() || runner.Stopped {
		// We can't run this file, but we still want to do nice printing.
		for _, run := range file.Runs {
			// The prepare function doesn't touch the run blocks, so we'll
			// update those so they make sense.
			run.Status = moduletest.Skip
		}
		printAll()
		return
	}

	// Make sure we clean up any states created during the execution of this
	// file.
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
			state := runner.ExecuteTestRun(mgr, run, file, states.NewState(), run.Config.ConfigUnderTest, globals)
			mgr.States = append(mgr.States, &TestModuleState{
				State: state,
				Run:   run,
			})
		} else {
			mgr.State = runner.ExecuteTestRun(mgr, run, file, mgr.State, runner.Config, globals)
		}
		file.Status = file.Status.Merge(run.Status)
	}

	printAll()
}

func (runner *TestRunner) ExecuteTestRun(mgr *TestStateManager, run *moduletest.Run, file *moduletest.File, state *states.State, config *configs.Config, globals map[string]backend.UnparsedVariableValue) *states.State {
	log.Printf("[TRACE] TestRunner: executing run block %s/%s", file.Name, run.Name)

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

	ctx, plan, state, diags := runner.execute(mgr, run, file, config, state, &terraform.PlanOpts{
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
	if plan != nil {
		// If the returned plan is nil, then the something went wrong before
		// we could even attempt to plan or apply the expected failures, so we
		// won't validate them if the plan is nil.
		diags = run.ValidateExpectedFailures(diags)
	}
	run.Diagnostics = run.Diagnostics.Append(diags)

	if runner.Cancelled {
		log.Printf("[DEBUG] TestRunner: exiting after test execution for %s/%s due to cancellation", file.Name, run.Name)

		// Print out the diagnostics from the run now, since it was cancelled
		// the normal set of diagnostics will not be printed otherwise.
		runner.View.Diagnostics(run, file, run.Diagnostics)
		run.Status = moduletest.Error
		return state
	}
	log.Printf("[DEBUG] TestRunner: completed test execution for %s/%s", file.Name, run.Name)

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

	variables, reset, diags := prepareInputVariablesForAssertions(config, run, file, globals)
	defer reset()

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

func (runner *TestRunner) validateFile(file *moduletest.File) {
	log.Printf("[TRACE] TestRunner: validating config for %s", file.Name)

	config := runner.Config

	reset, transformDiags := config.TransformForTest(nil, file.Config)
	defer reset()
	file.Diagnostics = file.Diagnostics.Append(transformDiags)

	if transformDiags.HasErrors() {
		file.Status = moduletest.Error
		return
	}

	tfCtxOpts, err := runner.command.contextOpts()
	file.Diagnostics = file.Diagnostics.Append(err)
	if err != nil {
		file.Status = moduletest.Error
		return
	}

	tfCtx, ctxDiags := terraform.NewContext(tfCtxOpts)
	file.Diagnostics = file.Diagnostics.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		file.Status = moduletest.Error
		return
	}

	runningCtx, done := context.WithCancel(context.Background())

	var validateDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestRunner: starting validate for %s", file.Name)
		validateDiags = tfCtx.Validate(config)
		log.Printf("[DEBUG] TestRunner: completed validate for %s", file.Name)
	}()
	// We don't pass in a manager or any created resources here since we are
	// only validating. If something goes wrong, there will be no state we need
	// to worry about cleaning up manually. So the manager and created resources
	// can be empty.
	waitDiags, _ := runner.wait(tfCtx, runningCtx, nil, nil, file, nil)

	file.Diagnostics = file.Diagnostics.Append(validateDiags)
	file.Diagnostics = file.Diagnostics.Append(waitDiags)
	if validateDiags.HasErrors() || waitDiags.HasErrors() {
		file.Status = moduletest.Error
	}
}

// execute executes Terraform plan and apply operations for the given arguments.
//
// The command argument decides whether it executes only a plan or also applies
// the plan it creates during the planning.
func (runner *TestRunner) execute(mgr *TestStateManager, run *moduletest.Run, file *moduletest.File, config *configs.Config, state *states.State, opts *terraform.PlanOpts, command configs.TestCommand, globals map[string]backend.UnparsedVariableValue) (*terraform.Context, *plans.Plan, *states.State, tfdiags.Diagnostics) {
	identifier := file.Name
	if run != nil {
		identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
	}
	log.Printf("[TRACE] TestRunner: executing %s for %s", opts.Mode, identifier)

	if opts.Mode == plans.DestroyMode && state.Empty() {
		// Nothing to do!
		log.Printf("[DEBUG] TestRunner: nothing to destroy for %s", identifier)
		return nil, nil, state, nil
	}

	var diags tfdiags.Diagnostics

	// First, do a quick validation of the run blocks config.

	if run != nil {
		diags = diags.Append(run.Config.Validate())
		if diags.HasErrors() {
			return nil, nil, state, diags
		}
	}

	// Second, transform the config for the given test run and test file.

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

	// Third, do a full validation of the now transformed config.

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

	var validateDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestRunner: starting validate for %s", identifier)
		validateDiags = tfCtx.Validate(config)
		log.Printf("[DEBUG] TestRunner: completed validate for %s", identifier)
	}()
	waitDiags, cancelled := runner.wait(tfCtx, runningCtx, mgr, run, file, nil)
	validateDiags = validateDiags.Append(waitDiags)

	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		// Either the plan errored, or we only wanted to see the plan. Either
		// way, just return what we have: The plan and diagnostics from making
		// it and the unchanged state.
		return tfCtx, nil, state, diags
	}

	if cancelled {
		log.Printf("[DEBUG] TestRunner: skipping plan and apply stage for %s due to cancellation", identifier)
		// If the execution was cancelled during the plan, we'll exit here to
		// stop the plan being applied and using more time.
		return tfCtx, nil, state, diags
	}

	// Fourth, gather any variables and give them to the plan options.

	variables, variableDiags := buildInputVariablesForTest(run, file, config, globals)
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		return nil, nil, state, diags
	}
	opts.SetVariables = variables

	// Fifth, execute planning stage.

	tfCtx, ctxDiags = terraform.NewContext(tfCtxOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, nil, state, diags
	}

	runningCtx, done = context.WithCancel(context.Background())

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	go func() {
		defer logging.PanicHandler()
		defer done()

		log.Printf("[DEBUG] TestRunner: starting plan for %s", identifier)
		plan, planDiags = tfCtx.Plan(config, state, opts)
		log.Printf("[DEBUG] TestRunner: completed plan for %s", identifier)
	}()
	waitDiags, cancelled = runner.wait(tfCtx, runningCtx, mgr, run, file, nil)
	planDiags = planDiags.Append(waitDiags)

	diags = diags.Append(planDiags)
	if planDiags.HasErrors() || command == configs.PlanTestCommand {
		// Either the plan errored, or we only wanted to see the plan. Either
		// way, just return what we have: The plan and diagnostics from making
		// it and the unchanged state.
		return tfCtx, plan, state, diags
	}

	if cancelled {
		log.Printf("[DEBUG] TestRunner: skipping apply stage for %s due to cancellation", identifier)
		// If the execution was cancelled during the plan, we'll exit here to
		// stop the plan being applied and using more time.
		return tfCtx, plan, state, diags
	}

	// We're also going to strip out any warnings from check blocks, as we do
	// for normal executions. Since we're going to go ahead and execute the
	// plan immediately, any warnings from the check block are just not relevant
	// any more.
	var filteredDiags tfdiags.Diagnostics
	for _, diag := range diags {
		if rule, ok := addrs.DiagnosticOriginatesFromCheckRule(diag); ok && rule.Container.CheckableKind() == addrs.CheckableCheck {
			continue
		}
		filteredDiags = filteredDiags.Append(diag)
	}
	diags = filteredDiags

	// Sixth, execute apply stage.
	tfCtx, ctxDiags = terraform.NewContext(tfCtxOpts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, nil, state, diags
	}

	runningCtx, done = context.WithCancel(context.Background())

	// If things get cancelled while we are executing the apply operation below
	// we want to print out all the objects that we were creating so the user
	// can verify we managed to tidy everything up possibly.
	//
	// Unfortunately, this creates a race condition as the apply operation can
	// edit the plan (by removing changes once they are applied) while at the
	// same time our cancellation process will try to read the plan.
	//
	// We take a quick copy of the changes we care about here, which will then
	// be used in place of the plan when we print out the objects to be created
	// as part of the cancellation process.
	var created []*plans.ResourceInstanceChangeSrc
	for _, change := range plan.Changes.Resources {
		if change.Action != plans.Create {
			continue
		}
		created = append(created, change)
	}

	var updated *states.State
	var applyDiags tfdiags.Diagnostics

	go func() {
		defer logging.PanicHandler()
		defer done()
		log.Printf("[DEBUG] TestRunner: starting apply for %s", identifier)
		updated, applyDiags = tfCtx.Apply(plan, config)
		log.Printf("[DEBUG] TestRunner: completed apply for %s", identifier)
	}()
	waitDiags, _ = runner.wait(tfCtx, runningCtx, mgr, run, file, created)
	applyDiags = applyDiags.Append(waitDiags)

	diags = diags.Append(applyDiags)
	return tfCtx, plan, updated, diags
}

func (runner *TestRunner) wait(ctx *terraform.Context, runningCtx context.Context, mgr *TestStateManager, run *moduletest.Run, file *moduletest.File, created []*plans.ResourceInstanceChangeSrc) (diags tfdiags.Diagnostics, cancelled bool) {
	var identifier string
	if file == nil {
		identifier = "validate"
	} else {
		identifier = file.Name
		if run != nil {
			identifier = fmt.Sprintf("%s/%s", identifier, run.Name)
		}
	}
	log.Printf("[TRACE] TestRunner: waiting for execution during %s", identifier)

	// This function handles what happens when the user presses the second
	// interrupt. This is a "hard cancel", we are going to stop doing whatever
	// it is we're doing. This means even if we're halfway through creating or
	// destroying infrastructure we just give up.
	handleCancelled := func() {
		log.Printf("[DEBUG] TestRunner: test execution cancelled during %s", identifier)

		if mgr != nil {

			// The state manager might be nil if we are waiting for a validate
			// call to finish. This is fine, it just means there's no state
			// that might be need to be cleaned up.

			states := make(map[*moduletest.Run]*states.State)
			states[nil] = mgr.State
			for _, module := range mgr.States {
				states[module.Run] = module.State
			}
			runner.View.FatalInterruptSummary(run, file, states, created)

		}

		cancelled = true
		go ctx.Stop()

		// Just wait for things to finish now, the overall test execution will
		// exit early if this takes too long.
		<-runningCtx.Done()
	}

	// This function handles what happens when the user presses the first
	// interrupt. This is essentially a "soft cancel", we're not going to do
	// anything but just wait for things to finish safely. But, we do listen
	// for the crucial second interrupt which will prompt a hard stop / cancel.
	handleStopped := func() {
		log.Printf("[DEBUG] TestRunner: test execution stopped during %s", identifier)

		select {
		case <-runner.CancelledCtx.Done():
			// We've been asked again. This time we stop whatever we're doing
			// and abandon all attempts to do anything reasonable.
			handleCancelled()
		case <-runningCtx.Done():
			// Do nothing, we finished safely and skipping the remaining tests
			// will be handled elsewhere.
		}

	}

	select {
	case <-runner.StoppedCtx.Done():
		handleStopped()
	case <-runner.CancelledCtx.Done():
		handleCancelled()
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

// prepare makes some simple checks that increase our confidence that a later
// clean up operation will succeed.
//
// When it comes time to execute cleanupStates below, we only have the
// information available at the file level. Our run blocks may have executed
// with additional data and configuration, so it's possible that we could
// successfully execute all our run blocks and then find we cannot perform any
// cleanup. We want to use this function to check that our cleanup can happen
// using only the information available within the file.
func (manager *TestStateManager) prepare(file *moduletest.File, globals map[string]backend.UnparsedVariableValue) {

	// First, we're going to check we have definitions for variables at the
	// file level.

	_, diags := buildInputVariablesForTest(nil, file, manager.runner.Config, globals)

	file.Diagnostics = file.Diagnostics.Append(diags)
	if diags.HasErrors() {
		file.Status = moduletest.Error
	}

	// Second, we'll validate that the default provider configurations actually
	// pass a validate operation.

	manager.runner.validateFile(file)
}

func (manager *TestStateManager) cleanupStates(file *moduletest.File, globals map[string]backend.UnparsedVariableValue) {
	log.Printf("[TRACE] TestStateManager: cleaning up state for %s", file.Name)

	if manager.runner.Cancelled {
		// Don't try and clean anything up if the execution has been cancelled.
		log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s due to cancellation", file.Name)
		return
	}

	// First, we'll clean up the main state.
	_, _, state, diags := manager.runner.execute(manager, nil, file, manager.runner.Config, manager.State, &terraform.PlanOpts{
		Mode: plans.DestroyMode,
	}, configs.ApplyTestCommand, globals)
	manager.runner.View.DestroySummary(diags, nil, file, state)

	if manager.runner.Cancelled {
		// In case things were cancelled during the last execution.
		return
	}

	// Then we'll clean up the additional states for custom modules in reverse
	// order.
	for ix := len(manager.States); ix > 0; ix-- {
		module := manager.States[ix-1]

		log.Printf("[DEBUG] TestStateManager: cleaning up state for %s/%s", file.Name, module.Run.Name)

		if manager.runner.Cancelled {
			// In case the cancellation came while a previous state was being
			// destroyed.
			log.Printf("[DEBUG] TestStateManager: skipping state cleanup for %s/%s due to cancellation", file.Name, module.Run.Name)
			return
		}

		_, _, state, diags := manager.runner.execute(manager, module.Run, file, module.Run.Config.ConfigUnderTest, module.State, &terraform.PlanOpts{
			Mode: plans.DestroyMode,
		}, configs.ApplyTestCommand, globals)
		manager.runner.View.DestroySummary(diags, module.Run, file, state)
	}
}

// helper functions

// buildInputVariablesForTest creates a terraform.InputValues mapping for
// variable values that are relevant to the config being tested.
//
// Crucially, it differs from prepareInputVariablesForAssertions in that it only
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

// prepareInputVariablesForAssertions creates a terraform.InputValues mapping
// that contains all the variables defined for a given run and file, alongside
// any unset variables that have defaults within the provided config.
//
// Crucially, it differs from buildInputVariablesForTest in that the returned
// input values include all variables available even if they are not defined
// within the config. This allows the assertions to refer to variables defined
// solely within the test file, and not only those within the configuration.
//
// In addition, it modifies the provided config so that any variables that are
// available are also defined in the config. It returns a function that resets
// the config which must be called so the config can be reused going forward.
func prepareInputVariablesForAssertions(config *configs.Config, run *moduletest.Run, file *moduletest.File, globals map[string]backend.UnparsedVariableValue) (terraform.InputValues, func(), tfdiags.Diagnostics) {
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

	for name, variable := range globals {
		if _, exists := variables[name]; exists {
			// Then this value was already defined at either the run level
			// or the file level, and we want those values to take
			// precedence.
			continue
		}
		variables[name] = variable
	}

	// We've gathered all the values we have, let's convert them into
	// terraform.InputValues so they can be passed into the Terraform graph.

	inputs := make(terraform.InputValues, len(variables))
	var diags tfdiags.Diagnostics
	for name, variable := range variables {
		value, valueDiags := variable.ParseVariableValue(configs.VariableParseLiteral)
		diags = diags.Append(valueDiags)
		inputs[name] = value
	}

	// Next, we're going to apply any default values from the configuration.
	// We do this after the conversion into terraform.InputValues, as the
	// defaults have already been converted into cty.Value objects.

	for name, variable := range config.Module.Variables {
		if _, exists := variables[name]; exists {
			// Then we don't want to apply the default for this variable as we
			// already have a value.
			continue
		}

		if variable.Default != cty.NilVal {
			inputs[name] = &terraform.InputValue{
				Value:       variable.Default,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
			}
		}
	}

	// Finally, we're going to do a some modifications to the config.
	// If we have got variable values from the test file we need to make sure
	// they have an equivalent entry in the configuration. We're going to do
	// that dynamically here.

	// First, take a backup of the existing configuration so we can easily
	// restore it later.
	currentVars := make(map[string]*configs.Variable)
	for name, variable := range config.Module.Variables {
		currentVars[name] = variable
	}

	// Next, let's go through our entire inputs and add any that aren't already
	// defined into the config.
	for name, value := range inputs {
		if _, exists := config.Module.Variables[name]; exists {
			continue
		}

		config.Module.Variables[name] = &configs.Variable{
			Name:           name,
			Type:           value.Value.Type(),
			ConstraintType: value.Value.Type(),
			DeclRange:      value.SourceRange.ToHCL(),
		}
	}

	// We return our input values, a function that will reset the variables
	// within the config so it can be used again, and any diagnostics reporting
	// variables that we couldn't parse.

	return inputs, func() {
		config.Module.Variables = currentVars
	}, diags
}
