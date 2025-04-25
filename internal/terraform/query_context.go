// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ListStates = collections.Map[addrs.List, []*states.ResourceInstanceObjectSrc]

type QueryRunner struct {
	State addrs.Map[addrs.List, []*states.ResourceInstanceObjectSrc]
	View  QueryViews
}

type QueryViews interface {
	List(ListStates)
	Resource(addrs.List, *states.ResourceInstanceObjectSrc)
}

// QueryOpts are the various options that affect the details of how Terraform
// will build a plan.
type QueryOpts struct {
	// Mode defines what variety of plan the caller wishes to create.
	// Refer to the documentation of the plans.Mode type and its values
	// for more information.
	Mode plans.Mode

	// SetVariables are the raw values for root module variables as provided
	// by the user who is requesting the run, prior to any normalization or
	// substitution of defaults. See the documentation for the InputValue
	// type for more information on how to correctly populate this.
	SetVariables InputValues

	Variables any

	// If Targets has a non-zero length then it activates targeted planning
	// mode, where Terraform will take actions only for resource instances
	// mentioned in this set and any other objects those resource instances
	// depend on.
	//
	// Targeted planning mode is intended for exceptional use only,
	// and so populating this field will cause Terraform to generate extra
	// warnings as part of the planning result.
	Targets []addrs.Targetable

	// GenerateConfigPath tells Terraform where to write any generated
	// configuration for any ImportTargets that do not have configuration
	// already.
	//
	// If empty, then no config will be generated.
	GenerateConfigPath string

	// ExternalProviders are clients for pre-configured providers that are
	// treated as being passed into the root module from the caller. This
	// is equivalent to writing a "providers" argument inside a "module"
	// block in the Terraform language, but for the root module the caller
	// is written in Go rather than the Terraform language.
	//
	// Terraform Core will NOT call ValidateProviderConfig or ConfigureProvider
	// on any providers in this map; it's the caller's responsibility to
	// configure these providers based on information outside the scope of
	// the root module.
	ExternalProviders map[addrs.RootProviderConfig]providers.Interface

	View QueryViews

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
}

// QueryEval is like [Context.Plan] except that it additionally makes a
// best effort to return a [lang.Scope] which can evaluate expressions in the
// root module based on the content of the generated plan.
//
// The scope will be nil if the planning process doesn't complete successfully
// enough to produce a valid evaluation scope. If the returned plan is nil
// then the scope will always be nil, but it's also possible for the scope
// to be nil even when the plan isn't, if the plan is not complete enough for
// the evaluation scope to produce consistent results.
func (c *Context) QueryEval(config *configs.Config, opts *QueryOpts) (*QueryRunner, tfdiags.Diagnostics) {
	defer c.acquireRun("nomeaning")()
	var diags tfdiags.Diagnostics

	// Save the downstream functions from needing to deal with these broken situations.
	// No real callers should rely on these, but we have a bunch of old and
	// sloppy tests that don't always populate arguments properly.
	if config == nil {
		config = configs.NewEmptyConfig()
	}
	if opts == nil {
		opts = &QueryOpts{}
	}

	moreDiags := c.checkConfigDependencies(config)
	diags = diags.Append(moreDiags)

	// If required dependencies are not available then we'll bail early since
	// otherwise we're likely to just see a bunch of other errors related to
	// incompatibilities, which could be overwhelming for the user.
	if diags.HasErrors() {
		return nil, diags
	}

	providerCfgDiags := checkExternalProviders(config, nil, nil, opts.ExternalProviders)
	diags = diags.Append(providerCfgDiags)
	if providerCfgDiags.HasErrors() {
		return nil, diags
	}

	// By the time we get here, we should have values defined for all of
	// the root module variables, even if some of them are "unknown". It's the
	// caller's responsibility to have already handled the decoding of these
	// from the various ways the CLI allows them to be set and to produce
	// user-friendly error messages if they are not all present, and so
	// the error message from checkInputVariables should never be seen and
	// includes language asking the user to report a bug.
	varDiags := checkInputVariables(config.Module.Variables, opts.SetVariables)
	diags = diags.Append(varDiags)

	querier, walkDiags := c.queryWalk(config, opts)
	diags = diags.Append(walkDiags)

	return querier, diags
}

func (c *Context) queryWalk(config *configs.Config, opts *QueryOpts) (*QueryRunner, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	log.Printf("[DEBUG] Building and walking query graph for %s", opts.Mode)
	graph, moreDiags := c.queryGraph(config, opts)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	var externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
	if opts != nil {
		externalProviderConfigs = opts.ExternalProviders
	}

	// If we get here then we should definitely have a non-nil "graph", which
	// we can now walk.

	// Initialize the results table to validate provider function calls.
	// Hold reference to this so we can store the table data in the plan file.
	providerFuncResults := providers.NewFunctionResultsTable(nil)

	querier, walkDiags := c.qwalk(graph, opts, &graphWalkOpts{
		Config: config,
		// InputState:                 prevRunState,
		ExternalProviderConfigs: externalProviderConfigs,
		ProviderFuncResults:     providerFuncResults,
	})
	diags = diags.Append(walkDiags)

	return querier, diags
}

func (c *Context) queryGraph(config *configs.Config, opts *QueryOpts) (*Graph, tfdiags.Diagnostics) {
	var externalProviderConfigs map[addrs.RootProviderConfig]providers.Interface
	if opts != nil {
		externalProviderConfigs = opts.ExternalProviders
	}
	graph, diags := (&QueryGraphBuilder{
		Config:                  config,
		RootVariableValues:      opts.SetVariables,
		ExternalProviderConfigs: externalProviderConfigs,
		Plugins:                 c.plugins,
		Targets:                 opts.Targets,
		Operation:               queryWalkEval,
		GenerateConfigPath:      opts.GenerateConfigPath,
	}).Build()
	return graph, diags
}

func (c *Context) qwalk(graph *Graph, queryOpts *QueryOpts, opts *graphWalkOpts) (*QueryRunner, tfdiags.Diagnostics) {
	log.Printf("[DEBUG] Starting query graph walk")

	walker := &ContextGraphWalker{
		Context:                 c,
		Config:                  opts.Config,
		Overrides:               opts.Overrides,
		NamedValues:             namedvals.NewState(),
		EphemeralResources:      ephemeral.NewResources(),
		InstanceExpander:        instances.NewExpander(opts.Overrides),
		ExternalProviderConfigs: opts.ExternalProviderConfigs,
		MoveResults:             opts.MoveResults,
		StopContext:             c.runContext,
		PlanTimestamp:           opts.PlanTimeTimestamp,
		providerFuncResults:     opts.ProviderFuncResults,
		Forget:                  opts.Forget,
		QueryRunner: &QueryRunner{
			State: addrs.MakeMap[addrs.List, []*states.ResourceInstanceObjectSrc](),
			View:  queryOpts.View,
		},
	}

	// Watch for a stop so we can call the provider Stop() API.
	watchStop, watchWait := c.watchStop(walker)

	// Walk the real graph, this will block until it completes
	ctx := walker.EvalContext()
	diags := graph.walk(ctx, walker)
	diags = diags.Append(walker.NonFatalDiagnostics)

	// Close the channel so the watcher stops, and wait for it to return.
	close(watchStop)
	<-watchWait

	return ctx.Querier(), diags
}
