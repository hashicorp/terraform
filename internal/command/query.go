// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
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

	_, configDiags := c.loadConfig(".", configs.MatchQueryFiles())
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, diags)
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

	_, variableDiags := c.collectVariableValues() // TODO: collect query variables?
	diags = diags.Append(variableDiags)
	if variableDiags.HasErrors() {
		view.Diagnostics(nil, diags)
		return 1
	}

	_, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(nil, diags)
		return 1
	}

	view.Diagnostics(nil, diags)

	runningCtx, done := context.WithCancel(context.Background())
	_, stop := context.WithCancel(runningCtx)
	_, cancel := context.WithCancel(context.Background())

	hasCloudBackend := false // TODO fetch from config
	if hasCloudBackend {
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

		// TODO: run cloud query
		_ = renderer
	} else {
		// TODO: run local query
	}

	var queryDiags tfdiags.Diagnostics

	go func() {
		defer logging.PanicHandler()
		defer done()
		defer stop()
		defer cancel()

		// TODO: RUN
	}()

	// Wait for the operation to complete, or for an interrupt to occur.
	select {
	case <-c.ShutdownCh:
		// Nice request to be cancelled.

		view.Interrupted()
		// runner.Stop()
		stop()

		select {
		case <-c.ShutdownCh:
			// The user pressed it again, now we have to get it to stop as
			// fast as possible.

			view.FatalInterrupt()
			// runner.Cancel()
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
		// query finished normally with no interrupts.
	}

	view.Diagnostics(nil, queryDiags)

	return 0
}
