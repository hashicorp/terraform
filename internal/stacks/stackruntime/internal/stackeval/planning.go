// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type PlanOpts struct {
	PlanningMode plans.Mode

	InputVariableValues map[stackaddrs.InputVariable]ExternalInputValue

	ProviderFactories ProviderFactories

	PlanTimestamp time.Time

	DependencyLocks depsfile.Locks
}

// Plannable is implemented by objects that can participate in planning.
type Plannable interface {
	// PlanChanges produces zero or more [stackplan.PlannedChange] objects
	// representing changes needed to converge the current and desired states
	// for the reciever, and zero or more diagnostics that represent any
	// problems encountered while calcuating the changes.
	//
	// The diagnostics returned by PlanChanges must be shallow, which is to
	// say that in particular they _must not_ call the PlanChanges methods
	// of other objects that implement Plannable, and should also think
	// very hard about calling any planning-related methods of other objects,
	// to avoid generating duplicate diagnostics via two different return
	// paths.
	//
	// In general, assume that _all_ objects that implement Plannable will
	// have their Validate methods called at some point during planning, and
	// so it's unnecessary and harmful to for one object to try to handle
	// planning (or plan-time validation) on behalf of some other object.
	PlanChanges(ctx context.Context) ([]stackplan.PlannedChange, tfdiags.Diagnostics)

	// Our general async planning helper relies on this to name its
	// tracing span.
	tracingNamer
}

func PlanComponentInstance(ctx context.Context, main *Main, state *states.State, opts *terraform.PlanOpts, scope ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance]) (*plans.Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	addr := scope.Addr()

	h := hooksFromContext(ctx)
	hookSingle(ctx, hooksFromContext(ctx).PendingComponentInstancePlan, addr)
	seq, ctx := hookBegin(ctx, h.BeginComponentInstancePlan, h.ContextAttach, addr)

	// This is our main bridge from the stacks language into the main Terraform
	// module language during the planning phase. We need to ask the main
	// language runtime to plan the module tree associated with this
	// component and return the result.

	moduleTree := scope.ModuleTree(ctx)
	if moduleTree == nil {
		// Presumably the configuration is invalid in some way, so
		// we can't create a plan and the relevant diagnostics will
		// get reported when the plan driver visits the ComponentConfig
		// object.
		return nil, diags
	}

	providerSchemas, moreDiags, _ := neededProviderSchemas(ctx, main, PlanPhase, scope)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	// We're actually going to provide two sets of providers to Core
	// for Stacks operations.
	//
	// First, we provide the basic set of factories here. These are used
	// by Terraform Core to handle operations that require an
	// unconfigured provider, such as cross-provider move operations and
	// provider functions. The provider factories return the shared
	// unconfigured client that stacks holds for the same reasons. The
	// factories will lazily request the unconfigured clients here as
	// they are requested by Terraform.
	//
	// Second, we provide provider clients that are already configured
	// for any operations that require configured clients. This is
	// because we want to provide the clients built using the provider
	// configurations from the stack that exist outside of Terraform's
	// concerns. There are provided directly in the PlanOpts argument.

	providerFactories := make(map[addrs.Provider]providers.Factory, len(providerSchemas))
	for addr := range providerSchemas {
		providerFactories[addr] = func() (providers.Interface, error) {
			// Lazily fetch the unconfigured client for the provider
			// as and when we need it.
			provider, err := main.ProviderType(ctx, addr).UnconfiguredClient()
			if err != nil {
				return nil, err
			}
			// this provider should only be used for selected operations
			return stubs.OfflineProvider(provider), nil
		}
	}

	tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
		Hooks: []terraform.Hook{
			&componentInstanceTerraformHook{
				ctx:   ctx,
				seq:   seq,
				hooks: hooksFromContext(ctx),
				addr:  addr,
			},
		},
		Providers:                providerFactories,
		PreloadedProviderSchemas: providerSchemas,
		Provisioners:             main.availableProvisioners(),
	})
	if err != nil {
		// Should not get here because we should always pass a valid
		// ContextOpts above.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to instantiate Terraform modules runtime",
			fmt.Sprintf("Could not load the main Terraform language runtime: %s.\n\nThis is a bug in Terraform; please report it!", err),
		))
		return nil, diags
	}

	// When our given context is cancelled, we want to instruct the
	// modules runtime to stop the running operation. We use this
	// nested context to ensure that we don't leak a goroutine when the
	// parent context isn't cancelled.
	operationCtx, operationCancel := context.WithCancel(ctx)
	defer operationCancel()
	go func() {
		<-operationCtx.Done()
		if ctx.Err() == context.Canceled {
			tfCtx.Stop()
		}
	}()

	plan, moreDiags := tfCtx.Plan(moduleTree, state, opts)
	diags = diags.Append(moreDiags)

	if plan != nil {
		cic := &hooks.ComponentInstanceChange{
			Addr: addr,
		}

		for _, rsrcChange := range plan.DriftedResources {
			hookMore(ctx, seq, h.ReportResourceInstanceDrift, &hooks.ResourceInstanceChange{
				Addr: stackaddrs.AbsResourceInstanceObject{
					Component: addr,
					Item:      rsrcChange.ObjectAddr(),
				},
				Change: rsrcChange,
			})
		}
		for _, rsrcChange := range plan.Changes.Resources {
			if rsrcChange.Importing != nil {
				cic.Import++
			}
			if rsrcChange.Moved() {
				cic.Move++
			}
			cic.CountNewAction(rsrcChange.Action)

			hookMore(ctx, seq, h.ReportResourceInstancePlanned, &hooks.ResourceInstanceChange{
				Addr: stackaddrs.AbsResourceInstanceObject{
					Component: addr,
					Item:      rsrcChange.ObjectAddr(),
				},
				Change: rsrcChange,
			})
		}
		for _, rsrcChange := range plan.DeferredResources {
			cic.Defer++
			hookMore(ctx, seq, h.ReportResourceInstanceDeferred, &hooks.DeferredResourceInstanceChange{
				Reason: rsrcChange.DeferredReason,
				Change: &hooks.ResourceInstanceChange{
					Addr: stackaddrs.AbsResourceInstanceObject{
						Component: addr,
						Item:      rsrcChange.ChangeSrc.ObjectAddr(),
					},
					Change: rsrcChange.ChangeSrc,
				},
			})
		}
		hookMore(ctx, seq, h.ReportComponentInstancePlanned, cic)
	}

	if diags.HasErrors() {
		hookMore(ctx, seq, h.ErrorComponentInstancePlan, addr)
	} else {
		if plan.Complete {
			hookMore(ctx, seq, h.EndComponentInstancePlan, addr)

		} else {
			hookMore(ctx, seq, h.DeferComponentInstancePlan, addr)
		}
	}

	return plan, diags
}
