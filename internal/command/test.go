// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
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
	var diags tfdiags.Diagnostics

	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Since we build the colorizer for the cloud runner outside the views
	// package we need to propagate our no-color setting manually. Once the
	// cloud package is fully migrated over to the new streams IO we should be
	// able to remove this.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	args, diags := arguments.ParseTest(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("test")
		return 1
	}
	c.Meta.parallelism = args.OperationParallelism

	view := views.NewTest(args.ViewType, c.View)

	// The specified testing directory must be a relative path, and it must
	// point to a directory that is a descendant of the configuration directory.
	if !filepath.IsLocal(args.TestDirectory) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid testing directory",
			"The testing directory must be a relative path pointing to a directory local to the configuration directory."))

		view.Diagnostics(nil, nil, diags)
		return 1
	}

	config, configDiags := c.loadConfigWithTests(".", args.TestDirectory)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	// Users can also specify variables via the command line, so we'll parse
	// all that here.
	var items []arguments.FlagNameValue
	for _, variable := range args.Vars.All() {
		items = append(items, arguments.FlagNameValue{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	c.variableArgs = arguments.FlagNameValueSlice{Items: &items}

	// Collect variables for "terraform test"
	testVariables, variableDiags := c.collectVariableValuesForTests(args.TestDirectory)
	diags = diags.Append(variableDiags)

	variables, variableDiags := c.collectVariableValues()
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	opts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
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
			Config: config,
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

		status, testDiags = runner.Test()
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
