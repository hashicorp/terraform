// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"strings"
	"time"

	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestCleanupCommand is a command that cleans up left-over resources created
// during Terraform test runs. It basically runs the test command in cleanup mode.
type TestCleanupCommand struct {
	Meta
}

func (c *TestCleanupCommand) Help() string {
	helpText := `
Usage: terraform [global options] test cleanup [options]

  Cleans up left-over resources in states that were created during Terraform test runs.

  By default, this command ignores the skip_cleanup attributes in the manifest
  file. Use the -repair flag to override this behavior, which will ensure that
  resources that were intentionally left-over are exempt from cleanup.

Options:

  -repair               Overrides the skip_cleanup attribute in the manifest
                        file and attempts to clean up all resources.

  -no-color             If specified, output won't contain any color.

  -verbose              Print detailed output during the cleanup process.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCleanupCommand) Synopsis() string {
	return "Clean up left-over resources created during Terraform test runs"
}

func (c *TestCleanupCommand) Run(rawArgs []string) int {
	setup, diags := c.setupTestExecution(moduletest.CleanupMode, "test cleanup", rawArgs)
	if diags.HasErrors() {
		return 1
	}

	args := setup.Args
	view := setup.View
	config := setup.Config
	variables := setup.Variables
	testVariables := setup.TestVariables
	opts := setup.Opts

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

	runner := &local.TestSuiteRunner{
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
		Repair:              args.Repair,
		CommandMode:         moduletest.CleanupMode,
	}

	var testDiags tfdiags.Diagnostics

	go func() {
		defer logging.PanicHandler()
		defer done()
		defer stop()
		defer cancel()

		_, testDiags = runner.Test()
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

	return 0
}
