// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type ApplyOpts struct {
	ProviderFactories ProviderFactories
	DependencyLocks   depsfile.Locks

	// PrevStateDescKeys is a set of all of the state description keys currently
	// known by the caller.
	//
	// The apply phase uses this to perform any broad "description maintenence"
	// that might need to happen to contend with changes to the state
	// description representation over time. For example, if any of the given
	// keys are unrecognized and classifed as needing to be discarded when
	// unrecognized then the apply phase will use this to emit the necessary
	// "discard" events to keep the state consistent.
	PrevStateDescKeys collections.Set[statekeys.Key]

	// InputVariableValues are variable values to use during the apply phase.
	//
	// This should typically include values for only variables that were
	// marked as being "required on apply" in the plan, but for ease of use
	// it's also valid to set other input variables here as long as the
	// given value is exactly equal to what was used during planning.
	InputVariableValues map[stackaddrs.InputVariable]ExternalInputValue

	ExperimentsAllowed bool
}

// Applyable is an interface implemented by types which represent objects
// that can potentially produce diagnostics and object change reports during
// the apply phase.
//
// Unlike [Plannable], Applyable implementations do not actually apply
// changes themselves. Instead, the real changes get driven separately using
// the [ChangeExec] function (see [ApplyPlan]) and then we collect up any
// reports to send to the caller separately using this interface.
type Applyable interface {
	// CheckApply checks the receiver's apply-time result and returns zero
	// or more applied change descriptions and zero or more diagnostics
	// describing any problems that occured for this specific object during
	// the apply phase.
	//
	// CheckApply must not report any diagnostics raised indirectly by
	// evaluating other objects. Those will be collected separately by calling
	// this same method on those other objects.
	CheckApply(ctx context.Context) ([]stackstate.AppliedChange, tfdiags.Diagnostics)

	// Our general async planning helper relies on this to name its
	// tracing span.
	tracingNamer
}

// ApplyableComponentInstance is an interface that represents a single instance
// of a component that can be applied. This is going to be a ComponentInstance
// or a RemovedInstance.
type ApplyableComponentInstance interface {
	ConfigComponentExpressionScope[stackaddrs.AbsComponentInstance]

	// ApplyModuleTreePlan applies the given plan to the module tree of this
	// component instance, returning the result of the apply operation and
	// any diagnostics that were generated.
	ApplyModuleTreePlan(ctx context.Context, plan *plans.Plan) (*ComponentInstanceApplyResult, tfdiags.Diagnostics)

	// PlaceholderApplyResultForSkippedApply returns a placeholder apply result
	// for the case where the apply operation was skipped. This is used to
	// ensure that the apply operation always returns a result, even if it
	// didn't actually do anything.
	PlaceholderApplyResultForSkippedApply(ctx context.Context, plan *plans.Plan) *ComponentInstanceApplyResult
}

func ApplyComponentPlan(ctx context.Context, main *Main, plan *plans.Plan, requiredProviders map[addrs.LocalProviderConfig]hcl.Expression, inst ApplyableComponentInstance) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// NOTE WELL: This function MUST either successfully apply the component
	// instance's plan or return at least one error diagnostic explaining why
	// it cannot.
	//
	// All return paths must include a non-nil ComponentInstanceApplyResult.
	// If an error occurs before we even begin applying the plan then the
	// result should report that the changes are incomplete and that the
	// new state is exactly the previous run state.
	//
	// If the underlying modules runtime raises errors when asked to apply the
	// plan, then this function should pass all of those errors through to its
	// own diagnostics while still returning the presumably-partially-updated
	// result state.

	// This is the result to return along with any errors that prevent us from
	// even starting the modules runtime apply phase. It reports that nothing
	// changed at all.
	noOpResult := inst.PlaceholderApplyResultForSkippedApply(ctx, plan)

	stackPlan := main.PlanBeingApplied().Components.Get(inst.Addr())

	// We'll gather up our set of potentially-affected objects before we do
	// anything else, because the modules runtime tends to mutate the objects
	// accessible through the given plan pointer while it does its work and
	// so we're likely to get a different/incomplete answer if we ask after
	// work has already been done.
	affectedResourceInstanceObjects := resourceInstanceObjectsAffectedByStackPlan(stackPlan)

	h := hooksFromContext(ctx)
	hookSingle(ctx, hooksFromContext(ctx).PendingComponentInstanceApply, inst.Addr())
	seq, ctx := hookBegin(ctx, h.BeginComponentInstanceApply, h.ContextAttach, inst.Addr())

	moduleTree := inst.ModuleTree(ctx)
	if moduleTree == nil {
		// We should not get here because if the configuration was statically
		// invalid then we should've detected that during the plan phase.
		// We'll emit a diagnostic about it just to make sure we're explicit
		// that the plan didn't get applied, but if anyone sees this error
		// it suggests a bug in whatever calling system sent us the plan
		// and configuration -- it's sent us the wrong configuration, perhaps --
		// and so we cannot know exactly what to blame with only the information
		// we have here.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Component configuration is invalid during apply",
			fmt.Sprintf(
				"Despite apparently successfully creating a plan earlier, %s seems to have an invalid configuration during the apply phase. This should not be possible, and suggests a bug in whatever subsystem is managing the plan and apply workflow.",
				inst.Addr(),
			),
		))
		return noOpResult, diags
	}

	providerSchemas, moreDiags, _ := neededProviderSchemas(ctx, main, ApplyPhase, inst)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return noOpResult, diags
	}

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

	tfHook := &componentInstanceTerraformHook{
		ctx:   ctx,
		seq:   seq,
		hooks: hooksFromContext(ctx),
		addr:  inst.Addr(),
	}
	tfCtx, err := terraform.NewContext(&terraform.ContextOpts{
		Hooks: []terraform.Hook{
			tfHook,
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
		return noOpResult, diags
	}

	known, unknown, moreDiags := EvalProviderValues(ctx, main, requiredProviders, ApplyPhase, inst)
	if moreDiags.HasErrors() {
		// We won't actually add the diagnostics here, they should be
		// exposed via a different return path.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot apply component plan",
			Detail:   fmt.Sprintf("Cannot apply the plan for %s because the configured provider configuration assignments are invalid.", inst.Addr()),
			Subject:  inst.DeclRange(ctx),
		})
		return nil, diags
	}

	providerClients := configuredProviderClients(ctx, main, known, unknown, ApplyPhase)

	var newState *states.State
	if plan.Applyable {
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

		// NOTE: tfCtx.Apply tends to make changes to the given plan while it
		// works, and so code after this point should not make any further use
		// of either "modifiedPlan" or "plan" (since they share lots of the same
		// pointers to mutable objects and so both can get modified together.)
		newState, moreDiags = tfCtx.Apply(plan, moduleTree, &terraform.ApplyOpts{
			ExternalProviders: providerClients,
		})
		diags = diags.Append(moreDiags)
	} else {
		// For a non-applyable plan, we just skip trying to apply it altogether
		// and just propagate the prior state (including any refreshing we
		// did during the plan phase) forward.
		newState = plan.PriorState
	}

	if newState != nil {
		cic := &hooks.ComponentInstanceChange{
			Addr: inst.Addr(),

			// We'll increment these gradually as we visit each change below.
			Add:    0,
			Change: 0,
			Import: 0,
			Remove: 0,
			Move:   0,
			Forget: 0,

			// The defer changes amount is a bit funny - we just copy over the
			// count of deferred changes from the plan, but we're not actually
			// making changes for this so the "true" count is zero.
			Defer: stackPlan.DeferredResourceInstanceChanges.Len(),
		}

		// We need to report what changes were applied, which is mostly just
		// re-announcing what was planned but we'll check to see if our
		// terraform.Hook implementation saw a "successfully applied" event
		// for each resource instance object before counting it.
		applied := tfHook.ResourceInstanceObjectsSuccessfullyApplied()
		for _, rioAddr := range applied {
			action := tfHook.ResourceInstanceObjectAppliedAction(rioAddr)
			cic.CountNewAction(action)
		}

		// The state management actions (move, import, forget) don't emit
		// actions during an apply so they're not being counted by looking
		// at the ResourceInstanceObjectAppliedAction above.
		//
		// Instead, we'll recheck the planned actions here to count them.
		for _, rioAddr := range affectedResourceInstanceObjects {
			if applied.Has(rioAddr) {
				// Then we processed this above.
				continue
			}

			change, exists := stackPlan.ResourceInstancePlanned.GetOk(rioAddr)
			if !exists {
				// This is a bit weird, but not something we should prevent
				// the apply from continuing for. We'll just ignore it and
				// assume that the plan was incomplete in some way.
				continue
			}

			// Otherwise, we have a change that wasn't successfully applied
			// for some reason. If the change was a no-op and a move or import
			// then it was still successful so we'll count it as such. Also,
			// forget actions don't count as applied changes but still happened
			// so we'll count them here.

			switch change.Action {
			case plans.NoOp:
				if change.Importing != nil {
					cic.Import++
				}
				if change.Moved() {
					cic.Move++
				}
			case plans.Forget:
				cic.Forget++
			}
		}

		hookMore(ctx, seq, h.ReportComponentInstanceApplied, cic)
	}

	if diags.HasErrors() {
		hookMore(ctx, seq, h.ErrorComponentInstanceApply, inst.Addr())
	} else {
		hookMore(ctx, seq, h.EndComponentInstanceApply, inst.Addr())
	}

	if newState == nil {
		// The modules runtime returns a nil state only if an error occurs
		// so early that it couldn't take any actions at all, and so we
		// must assume that the state is totally unchanged in that case.
		newState = plan.PrevRunState
		affectedResourceInstanceObjects = nil
	}

	return &ComponentInstanceApplyResult{
		FinalState:                      newState,
		AffectedResourceInstanceObjects: affectedResourceInstanceObjects,

		// Currently our definition of "complete" is that the apply phase
		// didn't return any errors, since we expect the modules runtime
		// to either perform all of the actions that were planned or
		// return errors explaining why it cannot.
		Complete: !diags.HasErrors(),
	}, diags
}
