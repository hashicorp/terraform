// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
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

  -cloud-run=source     If specified, Terraform will execute this test run 
                        remotely using HCP Terraform or Terraform Enterprise. 
						You must specify the source of a module registered in 
						a private module registry as the argument to this flag. 
						This allows Terraform to associate the cloud run with 
						the correct HCP Terraform or Terraform Enterprise module 
						and organization.

  -filter=testfile      If specified, Terraform will only execute the test files
                        specified by this flag. You can use this option multiple
                        times to execute more than one test file.

  -json                 If specified, machine readable output will be printed in
                        JSON format

  -junit-xml=path       Saves a test report in JUnit XML format to the specified
                        file. This is currently incompatible with remote test
                        execution using the the -cloud-run option. The file path
                        must be relative or absolute.

  -no-color             If specified, output won't contain any color.

  -parallelism=n        Limit the number of concurrent operations within the 
  						plan/apply operation of a test run. Defaults to 10.

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
	preparation, diags := c.setupTestExecution(moduletest.NormalMode, "test", rawArgs)
	if diags.HasErrors() {
		return 1
	}

	args := preparation.Args
	view := preparation.View
	config := preparation.Config
	variables := preparation.Variables
	testVariables := preparation.TestVariables
	opts := preparation.Opts

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

	var runner moduletest.TestSuiteRunner
	if len(args.CloudRunSource) > 0 {

		var renderer *jsonformat.Renderer
		if args.ViewType == arguments.ViewHuman {
			// We only set the renderer if we want Human-readable output.
			// Otherwise, we just let the runner echo whatever data it receives
			// back from the agent anyway.
			renderer = &jsonformat.Renderer{
				Streams:             c.Streams,
				Colorize:            c.Colorize(),
				RunningInAutomation: c.RunningInAutomation,
			}
		}

		runner = &cloud.TestSuiteRunner{
			ConfigDirectory:      ".", // Always loading from the current directory.
			TestingDirectory:     args.TestDirectory,
			Config:               config,
			Services:             c.Services,
			Source:               args.CloudRunSource,
			GlobalVariables:      variables,
			Stopped:              false,
			Cancelled:            false,
			StoppedCtx:           stopCtx,
			CancelledCtx:         cancelCtx,
			Verbose:              args.Verbose,
			OperationParallelism: args.OperationParallelism,
			Filters:              args.Filter,
			Renderer:             renderer,
			View:                 view,
			Streams:              c.Streams,
		}
	} else {
		localRunner := &local.TestSuiteRunner{
			BackendFactory: backendInit.Backend,
			Config:         config,
			// The GlobalVariables are loaded from the
			// main configuration directory
			// The GlobalTestVariables are loaded from the
			// test directory
			GlobalVariables:     variables,
			GlobalTestVariables: testVariables,
			TestingDirectory:    args.TestDirectory,
			Opts:                opts,
			View:                view,
			Stopped:             false,
			Cancelled:           false,
			StoppedCtx:          stopCtx,
			CancelledCtx:        cancelCtx,
			Filter:              args.Filter,
			Verbose:             args.Verbose,
			Concurrency:         args.RunParallelism,
			DeferralAllowed:     args.DeferralAllowed,
		}

		// JUnit output is only compatible with local test execution
		if args.JUnitXMLFile != "" {
			// Make sure TestCommand's calls loadConfigWithTests before this code, so configLoader is not nil
			localRunner.JUnit = junit.NewTestJUnitXMLFile(args.JUnitXMLFile, c.configLoader, localRunner)
		}

		runner = localRunner
	}

	var testDiags tfdiags.Diagnostics
	var status moduletest.Status

	go func() {
		defer logging.PanicHandler()
		defer done()
		defer stop()
		defer cancel()

		status, testDiags = runner.Test(c.AllowExperimentalFeatures)
	}()

	// Wait for the operation to complete, or for an interrupt to occur.
	select {
	case <-c.ShutdownCh:
		// Nice request to be cancelled.

		view.Interrupted()
		runner.Stop()
		stop()

		select {
		case <-c.ShutdownCh:
			// The user pressed it again, now we have to get it to stop as
			// fast as possible.

			view.FatalInterrupt()
			runner.Cancel()
			cancel()

			waitTime := 5 * time.Second
			if len(args.CloudRunSource) > 0 {
				// We wait longer for cloud runs because the agent should force
				// kill the remote job after 5 seconds (as defined above).
				//
				// This can take longer as the remote agent doesn't receive the
				// interrupt immediately. So for cloud runs, we'll wait a minute
				// which should give the remote process enough to receive the
				// signal, process it, and exit.
				//
				// If after a minute, the job still hasn't finished then we
				// assume something else has gone wrong and we'll just have to
				// live with the consequences.
				waitTime = time.Minute
			}

			// We'll wait 5 seconds for this operation to finish now, regardless
			// of whether it finishes successfully or not.
			select {
			case <-runningCtx.Done():
			case <-time.After(waitTime):
			}

		case <-runningCtx.Done():
			// The application finished nicely after the request was stopped.
		}
	case <-runningCtx.Done():
		// tests finished normally with no interrupts.
	}

	view.Diagnostics(nil, nil, testDiags)

	if status != moduletest.Pass {
		return 1
	}
	return 0
}

type TestRunnerSetup struct {
	Args          *arguments.Test
	View          views.Test
	Config        *configs.Config
	Variables     map[string]backendrun.UnparsedVariableValue
	TestVariables map[string]backendrun.UnparsedVariableValue
	Opts          *terraform.ContextOpts
}

func (m *Meta) setupTestExecution(mode moduletest.CommandMode, command string, rawArgs []string) (preparation TestRunnerSetup, diags tfdiags.Diagnostics) {
	common, rawArgs := arguments.ParseView(rawArgs)
	m.View.Configure(common)

	var moreDiags tfdiags.Diagnostics

	// Since we build the colorizer for the cloud runner outside the views
	// package we need to propagate our no-color setting manually. Once the
	// cloud package is fully migrated over to the new streams IO we should be
	// able to remove this.
	m.color = !common.NoColor
	m.Color = m.color

	preparation.Args, moreDiags = arguments.ParseTest(rawArgs)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		m.View.Diagnostics(diags)
		m.View.HelpPrompt(command)
		return
	}
	if preparation.Args.Repair && mode != moduletest.CleanupMode {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid command mode",
			"The -repair flag is only valid for the 'test cleanup' command."))
		m.View.Diagnostics(diags)
		return preparation, diags
	}

	m.parallelism = preparation.Args.OperationParallelism

	view := views.NewTest(preparation.Args.ViewType, m.View)
	preparation.View = view

	// EXPERIMENTAL: maybe enable deferred actions
	if !m.AllowExperimentalFeatures && preparation.Args.DeferralAllowed {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			"The -allow-deferral flag is only valid in experimental builds of Terraform.",
		))
		view.Diagnostics(nil, nil, diags)
		return
	}

	// The specified testing directory must be a relative path, and it must
	// point to a directory that is a descendant of the configuration directory.
	if !filepath.IsLocal(preparation.Args.TestDirectory) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid testing directory",
			"The testing directory must be a relative path pointing to a directory local to the configuration directory."))

		view.Diagnostics(nil, nil, diags)
		return
	}

	preparation.Config, moreDiags = m.loadConfigWithTests(".", preparation.Args.TestDirectory)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return
	}

	// Per file, ensure backends:
	// * aren't reused
	// * are valid types
	var backendDiags tfdiags.Diagnostics
	for _, tf := range preparation.Config.Module.Tests {
		bucketHashes := make(map[int]string)
		// Use an ordered list of backends, so that errors are raised by 2nd+ time
		// that a backend config is used in a file.
		for _, bc := range orderBackendsByDeclarationLine(tf.BackendConfigs) {
			f := backendInit.Backend(bc.Backend.Type)
			if f == nil {
				detail := fmt.Sprintf("There is no backend type named %q.", bc.Backend.Type)
				if msg, removed := backendInit.RemovedBackends[bc.Backend.Type]; removed {
					detail = msg
				}
				backendDiags = backendDiags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Unsupported backend type",
					Detail:   detail,
					Subject:  &bc.Backend.TypeRange,
				})
				continue
			}

			b := f()
			schema := b.ConfigSchema()
			hash := bc.Backend.Hash(schema)

			if runName, exists := bucketHashes[hash]; exists {
				// This backend's been encountered before
				backendDiags = backendDiags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Repeat use of the same backend block",
						Detail:   fmt.Sprintf("The run %q contains a backend configuration that's already been used in run %q. Sharing the same backend configuration between separate runs will result in conflicting state updates.", bc.Run.Name, runName),
						Subject:  bc.Backend.TypeRange.Ptr(),
					},
				)
				continue
			}
			bucketHashes[bc.Backend.Hash(schema)] = bc.Run.Name
		}
	}
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return
	}

	// Users can also specify variables via the command line, so we'll parse
	// all that here.
	var items []arguments.FlagNameValue
	for _, variable := range preparation.Args.Vars.All() {
		items = append(items, arguments.FlagNameValue{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	m.variableArgs = arguments.FlagNameValueSlice{Items: &items}

	// Collect variables for "terraform test"
	preparation.TestVariables, moreDiags = m.collectVariableValuesForTests(preparation.Args.TestDirectory)
	diags = diags.Append(moreDiags)

	preparation.Variables, moreDiags = m.collectVariableValues()
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return
	}

	opts, err := m.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(nil, nil, diags)
		return
	}
	preparation.Opts = opts

	// Print out all the diagnostics we have from the setup. These will just be
	// warnings, and we want them out of the way before we start the actual
	// testing.
	view.Diagnostics(nil, nil, diags)
	return
}

// orderBackendsByDeclarationLine takes in a map of state keys to backend configs and returns a list of
// those backend configs, sorted by the line their declaration range starts on. This allows identification
// of the 2nd+ time that a backend configuration is used in the same file.
func orderBackendsByDeclarationLine(backendConfigs map[string]configs.RunBlockBackend) []configs.RunBlockBackend {
	bcs := slices.Collect(maps.Values(backendConfigs))
	sort.Slice(bcs, func(i, j int) bool {
		return bcs[i].Run.DeclRange.Start.Line < bcs[j].Run.DeclRange.Start.Line
	})
	return bcs
}
