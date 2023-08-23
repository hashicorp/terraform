// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	MainStateIdentifier = ""
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
                        remotely using Terraform Cloud. You must specify the 
                        source of a module registered in a private module
                        registry as the argument to this flag. This allows 
                        Terraform to associate the cloud run with the correct 
                        Terraform Cloud module and organization.

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
		panic("Cloud runs are not yet supported.")
	} else {
		runner = &local.TestSuiteRunner{
			Config:          config,
			GlobalVariables: variables,
			Opts:            opts,
			View:            view,
			Stopped:         false,
			Cancelled:       false,
			StoppedCtx:      stopCtx,
			CancelledCtx:    cancelCtx,
			Filter:          args.Filter,
			Verbose:         args.Verbose,
		}
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

	view.Diagnostics(nil, nil, testDiags)

	if status != moduletest.Pass {
		return 1
	}
	return 0
}
