// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type QueryCommand struct {
	Meta
}

func (c *QueryCommand) Help() string {
	helpText := `
Usage: terraform [global options] query [options]

  TBD

Options:

  -json                 If specified, machine readable output will be printed in
                        JSON format

  -no-color             If specified, output won't contain any color.

  -var 'foo=bar'        Set a value for one of the input variables in the root
                        module of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.
`
	return strings.TrimSpace(helpText)
}

func (c *QueryCommand) Synopsis() string {
	return "Search and list remote infrastructure with Terraform"
}

func (c *QueryCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseQuery(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("query")
		return 1
	}

	view := views.NewQuery(args.ViewType, c.View)

	config, configDiags := c.loadConfig(".", configs.MatchQueryFiles())
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(addrs.List{}, diags)
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
	variables, variableDiags := c.collectVariableValues()
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		view.Diagnostics(addrs.List{}, diags)
		return 1
	}

	opts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(addrs.List{}, diags)
		return 1
	}

	// Print out all the diagnostics we have from the setup. These will just be
	// warnings, and we want them out of the way before we start the actual
	// testing.
	view.Diagnostics(addrs.List{}, diags)

	// We have two levels of interrupt here. A 'stop' and a 'cancel'. A 'stop'
	// is a soft request to stop. We'll finish the current test, do the tidy up,
	// but then skip all remaining tests and run blocks. A 'cancel' is a hard
	// request to stop now. We'll cancel the current operation immediately
	// even if it's a delete operation, and we won't clean up any infrastructure
	// if we're halfway through a test. We'll print details explaining what was
	// stopped so the user can do their best to recover from it.
	tfCtx, ctxDiags := terraform.NewContext(opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		view.Diagnostics(addrs.List{}, diags)
		return 1
	}

	runningCtx, done := context.WithCancel(context.Background())
	stopCtx, stop := context.WithCancel(runningCtx)
	cancelCtx, cancel := context.WithCancel(context.Background())

	var qDiags tfdiags.Diagnostics

	go func() {
		defer logging.PanicHandler()
		defer done()
		defer stop()
		defer cancel()

		_, qDiags = tfCtx.QueryEval(config, &terraform.QueryOpts{
			View:         view,
			Variables:    variables,
			StoppedCtx:   stopCtx,
			CancelledCtx: cancelCtx,
		})
	}()

	// Wait for the operation to complete, or for an interrupt to occur.
	select {
	case <-c.ShutdownCh:
		// Nice request to be cancelled.

		view.Interrupted()
		stop()

		select {
		case <-c.ShutdownCh:
			// The user pressed it again, now we have to get it to stop as
			// fast as possible.

			view.FatalInterrupt()
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

	view.Diagnostics(addrs.List{}, qDiags)
	if qDiags.HasErrors() {
		return 1
	}
	return 0
}
