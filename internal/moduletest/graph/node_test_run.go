// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeExecutable    = (*NodeTestRun)(nil)
	_ GraphNodeReferenceable = (*NodeTestRun)(nil)
	_ GraphNodeReferencer    = (*NodeTestRun)(nil)
)

type NodeTestRun struct {
	run       *moduletest.Run
	priorRuns map[string]*moduletest.Run
	opts      *graphOptions
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.opts.File
}

func (n *NodeTestRun) Name() string {
	return fmt.Sprintf("%s.%s", n.opts.File.Name, n.run.Addr().String())
}

func (n *NodeTestRun) Referenceable() addrs.Referenceable {
	return n.run.Addr()
}

func (n *NodeTestRun) References() []*addrs.Reference {
	references, _ := n.run.GetReferences()
	return references
}

// Execute executes the test run block and update the status of the run block
// based on the result of the execution.
func (n *NodeTestRun) Execute(evalCtx *EvalContext) {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", n.File().Name, n.run.Name)
	startTime := time.Now().UTC()
	file, run := n.File(), n.run

	// At the end of the function, we'll update the status of the file based on
	// the status of the run block, and render the run summary.
	defer func() {
		evalCtx.Renderer().Run(run, file, moduletest.Complete, 0)
		file.UpdateStatus(run.Status)
		evalCtx.AddRunBlock(run)
	}()

	if !evalCtx.PriorRunsCompleted(n.priorRuns) || !evalCtx.ReferencesCompleted(n.References()) {
		// If any of our prior runs or references weren't completed successfully
		// then we will just skip this run block.
		run.Status = moduletest.Skip
		return
	}
	if evalCtx.Cancelled() {
		// A cancellation signal has been received.
		// Don't do anything, just give up and return immediately.
		// The surrounding functions should stop this even being called, but in
		// case of race conditions or something we can still verify this.
		return
	}

	if evalCtx.Stopped() {
		// Then the test was requested to be stopped, so we just mark each
		// following test as skipped, print the status, and move on.
		run.Status = moduletest.Skip
		return
	}

	// Create a waiter which handles waiting for terraform operations to complete.
	// While waiting, the wait will also respond to cancellation signals, and
	// handle them appropriately.
	// The test progress is updated periodically, and the progress status
	// depends on the async operation being waited on.
	// Before the terraform operation is started, the operation updates the
	// waiter with the cleanup context on cancellation, as well as the
	// progress status.
	waiter := NewOperationWaiter(nil, evalCtx, n, moduletest.Running, startTime.UnixMilli())
	cancelled := waiter.Run(func() {
		defer logging.PanicHandler()
		n.execute(evalCtx, waiter)
	})

	if cancelled {
		n.run.Diagnostics = n.run.Diagnostics.Append(tfdiags.Sourceless(tfdiags.Error, "Test interrupted", "The test operation could not be completed due to an interrupt signal. Please read the remaining diagnostics carefully for any sign of failed state cleanup or dangling resources."))
	}

	// If we got far enough to actually attempt to execute the run then
	// we'll give the view some additional metadata about the execution.
	n.run.ExecutionMeta = &moduletest.RunExecutionMeta{
		Start:    startTime,
		Duration: time.Since(startTime),
	}
}

func (n *NodeTestRun) execute(ctx *EvalContext, waiter *operationWaiter) {
	file, run := n.File(), n.run
	ctx.Renderer().Run(run, file, moduletest.Starting, 0)
	if run.Config.ConfigUnderTest != nil && run.GetStateKey() == moduletest.MainStateIdentifier {
		// This is bad, and should not happen because the state key is derived from the custom module source.
		panic(fmt.Sprintf("TestFileRunner: custom module %s has the same key as main state", file.Name))
	}

	providers, mocks, providerDiags := n.getProviders(ctx)
	if !ctx.ProvidersCompleted(providers) {
		run.Status = moduletest.Skip
		return
	}

	run.Diagnostics = run.Diagnostics.Append(providerDiags)
	if providerDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	n.testValidate(providers, waiter)
	if run.Diagnostics.HasErrors() {
		return
	}

	variables, variableDiags := n.GetVariables(ctx, true)
	run.Diagnostics = run.Diagnostics.Append(variableDiags)
	if variableDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}

	if run.Config.Command == configs.PlanTestCommand {
		n.testPlan(ctx, variables, providers, mocks, waiter)
	} else {
		n.testApply(ctx, variables, providers, mocks, waiter)
	}
}

// Validating the module config which the run acts on
func (n *NodeTestRun) testValidate(providers map[addrs.RootProviderConfig]providers.Interface, waiter *operationWaiter) {
	run := n.run
	file := n.File()
	config := run.ModuleConfig

	log.Printf("[TRACE] TestFileRunner: called validate for %s/%s", file.Name, run.Name)
	tfCtx, ctxDiags := terraform.NewContext(n.opts.ContextOpts)
	if ctxDiags.HasErrors() {
		return
	}
	waiter.update(tfCtx, moduletest.Running, nil)
	validateDiags := tfCtx.Validate(config, &terraform.ValidateOpts{
		ExternalProviders: providers,
	})
	run.Diagnostics = run.Diagnostics.Append(validateDiags)
	if validateDiags.HasErrors() {
		run.Status = moduletest.Error
		return
	}
}

func (n *NodeTestRun) getProviders(ctx *EvalContext) (map[addrs.RootProviderConfig]providers.Interface, map[addrs.RootProviderConfig]*configs.MockData, tfdiags.Diagnostics) {
	run := n.run

	var diags tfdiags.Diagnostics

	if len(run.Config.Providers) > 0 {
		// Then we'll only provide the specific providers asked for by the run
		// block.

		providers := make(map[addrs.RootProviderConfig]providers.Interface, len(run.Config.Providers))
		mocks := make(map[addrs.RootProviderConfig]*configs.MockData)

		for _, ref := range run.Config.Providers {

			testAddr := addrs.RootProviderConfig{
				Provider: ctx.ProviderForConfigAddr(ref.InParent.Addr()),
				Alias:    ref.InParent.Alias,
			}

			moduleAddr := addrs.RootProviderConfig{
				Provider: run.ModuleConfig.ProviderForConfigAddr(ref.InChild.Addr()),
				Alias:    ref.InChild.Alias,
			}

			if !testAddr.Provider.Equals(moduleAddr.Provider) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Mismatched provider configuration",
					Detail:   fmt.Sprintf("Expected %q but was %q.", moduleAddr.Provider, testAddr.Provider),
					Subject:  ref.InChild.NameRange.Ptr(),
				})
				continue
			}

			if provider, ok := ctx.GetProvider(testAddr); ok {
				providers[moduleAddr] = provider

				config := n.File().Config.Providers[ref.InParent.String()]
				if config.Mock {
					mocks[moduleAddr] = config.MockData
				}

			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing provider",
					Detail:   fmt.Sprintf("Provider %q was not defined within the test file.", ref.InParent.String()),
					Subject:  ref.InParent.NameRange.Ptr(),
				})
			}

		}
		return providers, mocks, diags

	} else {
		// Otherwise, let's copy over all the relevant providers.

		providers := make(map[addrs.RootProviderConfig]providers.Interface)
		mocks := make(map[addrs.RootProviderConfig]*configs.MockData)

		for addr := range requiredProviders(run.ModuleConfig) {
			if provider, ok := ctx.GetProvider(addr); ok {
				providers[addr] = provider

				local := ctx.LocalNameForProvider(addr)
				if len(addr.Alias) > 0 {
					local = fmt.Sprintf("%s.%s", local, addr.Alias)
				}
				config := n.File().Config.Providers[local]
				if config.Mock {
					mocks[addr] = config.MockData
				}
			}
		}
		return providers, mocks, diags
	}
}

func requiredProviders(config *configs.Config) map[addrs.RootProviderConfig]bool {
	providers := make(map[addrs.RootProviderConfig]bool)

	// First, let's look at the required providers first.
	for _, provider := range config.Module.ProviderRequirements.RequiredProviders {
		providers[addrs.RootProviderConfig{
			Provider: provider.Type,
		}] = true
		for _, alias := range provider.Aliases {
			providers[addrs.RootProviderConfig{
				Provider: provider.Type,
				Alias:    alias.Alias,
			}] = true
		}
	}

	// Second, we look at the defined provider configs.
	for _, provider := range config.Module.ProviderConfigs {
		providers[addrs.RootProviderConfig{
			Provider: config.ProviderForConfigAddr(provider.Addr()),
			Alias:    provider.Alias,
		}] = true
	}

	// Third, we look at the resources and data sources.
	for _, resource := range config.Module.ManagedResources {
		if resource.ProviderConfigRef != nil {
			providers[addrs.RootProviderConfig{
				Provider: config.ProviderForConfigAddr(resource.ProviderConfigRef.Addr()),
				Alias:    resource.ProviderConfigRef.Alias,
			}] = true
			continue
		}
		providers[addrs.RootProviderConfig{
			Provider: resource.Provider,
		}] = true
	}
	for _, datasource := range config.Module.DataResources {
		if datasource.ProviderConfigRef != nil {
			providers[addrs.RootProviderConfig{
				Provider: config.ProviderForConfigAddr(datasource.ProviderConfigRef.Addr()),
				Alias:    datasource.ProviderConfigRef.Alias,
			}] = true
			continue
		}
		providers[addrs.RootProviderConfig{
			Provider: datasource.Provider,
		}] = true
	}

	// Finally, we look at any module calls to see if any providers are used
	// in there.
	for _, module := range config.Module.ModuleCalls {
		for _, provider := range module.Providers {
			providers[addrs.RootProviderConfig{
				Provider: config.ProviderForConfigAddr(provider.InParent.Addr()),
				Alias:    provider.InParent.Alias,
			}] = true
		}
	}

	return providers
}
