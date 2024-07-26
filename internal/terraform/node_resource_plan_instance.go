// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodePlannableResourceInstance represents a _single_ resource
// instance that is plannable. This means this represents a single
// count index, for example.
type NodePlannableResourceInstance struct {
	*NodeAbstractResourceInstance
	ForceCreateBeforeDestroy bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance

	// replaceTriggeredBy stores references from replace_triggered_by which
	// triggered this instance to be replaced.
	replaceTriggeredBy []*addrs.Reference

	// importTarget, if populated, contains the information necessary to plan
	// an import of this resource.
	importTarget cty.Value
}

var (
	_ GraphNodeModuleInstance       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeConfigResource       = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodePlannableResourceInstance)(nil)
	_ GraphNodeExecutable           = (*NodePlannableResourceInstance)(nil)
)

// GraphNodeEvalable
func (n *NodePlannableResourceInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	addr := n.ResourceInstanceAddr()

	// Eval info is different depending on what kind of resource this is
	switch addr.Resource.Resource.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodePlannableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var change *plans.ResourceInstanceChange

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	checkRuleSeverity := tfdiags.Error
	if n.skipPlanChanges || n.preDestroyRefresh {
		checkRuleSeverity = tfdiags.Warning
	}

	deferrals := ctx.Deferrals()
	change, state, deferred, repeatData, planDiags := n.planDataSource(ctx, checkRuleSeverity, n.skipPlanChanges, deferrals.ShouldDeferResourceInstanceChanges(addr, n.Dependencies))
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	// A nil change here indicates that Terraform is deciding NOT to make a
	// change at all. In which case even if we wanted to try and defer it
	// (because of a dependency) we can't as there is no change to defer.
	//
	// The most common case for this is when the data source is being refreshed
	// but depends on unknown values or dependencies which means we just skip
	// refreshing the data source. We maintain that behaviour here.
	if change != nil && deferred != nil {
		// Then this data source got deferred by the provider during planning.
		deferrals.ReportDataSourceInstanceDeferred(addr, deferred.Reason, change)
	} else {
		// Not deferred; business as usual.

		// write the data source into both the refresh state and the
		// working state
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, refreshState))
		if diags.HasErrors() {
			return diags
		}
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
		if diags.HasErrors() {
			return diags
		}

		diags = diags.Append(n.writeChange(ctx, change, ""))

		// Post-conditions might block further progress. We intentionally do this
		// _after_ writing the state/diff because we want to check against
		// the result of the operation, and to fail on future operations
		// until the user makes the condition succeed.
		checkDiags := evalCheckRules(
			addrs.ResourcePostcondition,
			n.Config.Postconditions,
			ctx, addr, repeatData,
			checkRuleSeverity,
		)
		diags = diags.Append(checkDiags)
	}

	return diags
}

func (n *NodePlannableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	config := n.Config
	addr := n.ResourceInstanceAddr()

	var instanceRefreshState *states.ResourceInstanceObject

	checkRuleSeverity := tfdiags.Error
	if n.skipPlanChanges || n.preDestroyRefresh {
		checkRuleSeverity = tfdiags.Warning
	}

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	if config != nil {
		diags = diags.Append(validateSelfRef(addr.Resource, config.Config, providerSchema))
		if diags.HasErrors() {
			return diags
		}
	}

	importing := n.importTarget != cty.NilVal && !n.preDestroyRefresh

	var deferred *providers.Deferred

	// If the resource is to be imported, we now ask the provider for an Import
	// and a Refresh, and save the resulting state to instanceRefreshState.

	if importing {
		if n.importTarget.IsKnown() {
			var importDiags tfdiags.Diagnostics
			instanceRefreshState, deferred, importDiags = n.importState(ctx, addr, n.importTarget.AsString(), provider, providerSchema)
			diags = diags.Append(importDiags)
		} else {
			// Otherwise, just mark the resource as deferred without trying to
			// import it.
			deferred = &providers.Deferred{
				Reason: providers.DeferredReasonResourceConfigUnknown,
			}
			if n.Config == nil && len(n.generateConfigPath) > 0 {
				// Then we're supposed to be generating configuration for this
				// resource, but we can't because the configuration is unknown.
				//
				// Normally, the rest of this function would just be about
				// planning the known configuration to make sure everything we
				// do know about it is correct, but we can't even do that here.
				//
				// What we'll do is write out the address as being deferred with
				// an entirely unknown value. Then we'll skip the rest of this
				// function. (a) We're going to panic later when it complains
				// about having no configuration, and (b) the rest of the
				// function isn't doing anything as there is no configuration
				// to validate.

				impliedType := providerSchema.ResourceTypes[addr.Resource.Resource.Type].Block.ImpliedType()
				ctx.Deferrals().ReportResourceInstanceDeferred(addr, providers.DeferredReasonResourceConfigUnknown, &plans.ResourceInstanceChange{
					Addr:         addr,
					PrevRunAddr:  addr,
					ProviderAddr: n.ResolvedProvider,
					Change: plans.Change{
						Action: plans.NoOp, // assume we'll get the config generation correct.
						Before: cty.NullVal(impliedType),
						After:  cty.UnknownVal(impliedType),
						Importing: &plans.Importing{
							ID: n.importTarget,
						},
					},
				})
				return diags
			}
		}
	} else {
		var readDiags tfdiags.Diagnostics
		instanceRefreshState, readDiags = n.readResourceInstanceState(ctx, addr)
		diags = diags.Append(readDiags)
		if diags.HasErrors() {
			return diags
		}
	}

	if deferred == nil {
		// We'll save a snapshot of what we just read from the state into the
		// prevRunState before we do anything else, since this will capture the
		// result of any schema upgrading that readResourceInstanceState just did,
		// but not include any out-of-band changes we might detect in in the
		// refresh step below.
		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, prevRunState))
		if diags.HasErrors() {
			return diags
		}
		// Also the refreshState, because that should still reflect schema upgrades
		// even if it doesn't reflect upstream changes.
		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	// we may need to detect a change in CreateBeforeDestroy to ensure it's
	// stored when we are not refreshing
	updatedCBD := false
	if n.Config != nil && n.Config.Managed != nil && instanceRefreshState != nil {
		newCBD := n.Config.Managed.CreateBeforeDestroy || n.ForceCreateBeforeDestroy
		updatedCBD = instanceRefreshState.CreateBeforeDestroy != newCBD
		instanceRefreshState.CreateBeforeDestroy = newCBD
	}

	var refreshDeferred *providers.Deferred
	// This is the state of the resource before we refresh the value, we need to keep track
	// of this to report this as the before value if the refresh is deferred.
	priorInstanceRefreshState := instanceRefreshState

	// Refresh, maybe
	// The import process handles its own refresh
	if !n.skipRefresh && !importing {
		var refreshDiags tfdiags.Diagnostics
		instanceRefreshState, refreshDeferred, refreshDiags = n.refresh(ctx, states.NotDeposed, instanceRefreshState, ctx.Deferrals().DeferralAllowed())
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return diags
		}

		if instanceRefreshState != nil {
			// When refreshing we start by merging the stored dependencies and
			// the configured dependencies. The configured dependencies will be
			// stored to state once the changes are applied. If the plan
			// results in no changes, we will re-write these dependencies
			// below.
			instanceRefreshState.Dependencies = mergeDeps(n.Dependencies, instanceRefreshState.Dependencies)
		}

		if deferred == nil && refreshDeferred != nil {
			deferred = refreshDeferred
		}

		if deferred == nil {
			diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
		}
		if diags.HasErrors() {
			return diags
		}
	}

	if n.skipRefresh && !importing && updatedCBD {
		// CreateBeforeDestroy must be set correctly in the state which is used
		// to create the apply graph, so if we did not refresh the state make
		// sure we still update any changes to CreateBeforeDestroy.
		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
		if diags.HasErrors() {
			return diags
		}
	}

	// Plan the instance, unless we're in the refresh-only mode
	if !n.skipPlanChanges {

		// add this instance to n.forceReplace if replacement is triggered by
		// another change
		repData := instances.RepetitionData{}
		switch k := addr.Resource.Key.(type) {
		case addrs.IntKey:
			repData.CountIndex = k.Value()
		case addrs.StringKey:
			repData.EachKey = k.Value()
			repData.EachValue = cty.DynamicVal
		}

		diags = diags.Append(n.replaceTriggered(ctx, repData))
		if diags.HasErrors() {
			return diags
		}

		change, instancePlanState, planDeferred, repeatData, planDiags := n.plan(
			ctx, nil, instanceRefreshState, n.ForceCreateBeforeDestroy, n.forceReplace,
		)
		diags = diags.Append(planDiags)
		if diags.HasErrors() {
			// If we are importing and generating a configuration, we need to
			// ensure the change is written out so the configuration can be
			// captured.
			if planDeferred == nil && len(n.generateConfigPath) > 0 {
				// Update our return plan
				change := &plans.ResourceInstanceChange{
					Addr:         n.Addr,
					PrevRunAddr:  n.prevRunAddr(ctx),
					ProviderAddr: n.ResolvedProvider,
					Change: plans.Change{
						// we only need a placeholder, so this will be a NoOp
						Action:          plans.NoOp,
						Before:          instanceRefreshState.Value,
						After:           instanceRefreshState.Value,
						GeneratedConfig: n.generatedConfigHCL,
					},
				}
				diags = diags.Append(n.writeChange(ctx, change, ""))
			}

			return diags
		}

		if deferred == nil && planDeferred != nil {
			deferred = planDeferred
		}

		if importing {
			change.Importing = &plans.Importing{ID: n.importTarget}
		}

		// FIXME: here we udpate the change to reflect the reason for
		// replacement, but we still overload forceReplace to get the correct
		// change planned.
		if len(n.replaceTriggeredBy) > 0 {
			change.ActionReason = plans.ResourceInstanceReplaceByTriggers
		}

		deferrals := ctx.Deferrals()
		if deferred != nil {
			// Then this resource has been deferred either during the import,
			// refresh or planning stage. We'll report the deferral and
			// store what we could produce in the deferral tracker.
			deferrals.ReportResourceInstanceDeferred(addr, deferred.Reason, change)
		} else if !deferrals.ShouldDeferResourceInstanceChanges(n.Addr, n.Dependencies) {
			// We intentionally write the change before the subsequent checks, because
			// all of the checks below this point are for problems caused by the
			// context surrounding the change, rather than the change itself, and
			// so it's helpful to still include the valid-in-isolation change as
			// part of the plan as additional context in our error output.
			//
			// FIXME: it is currently important that we write resource changes to
			// the plan (n.writeChange) before we write the corresponding state
			// (n.writeResourceInstanceState).
			//
			// This is because the planned resource state will normally have the
			// status of states.ObjectPlanned, which causes later logic to refer to
			// the contents of the plan to retrieve the resource data. Because
			// there is no shared lock between these two data structures, reversing
			// the order of these writes will cause a brief window of inconsistency
			// which can lead to a failed safety check.
			//
			// Future work should adjust these APIs such that it is impossible to
			// update these two data structures incorrectly through any objects
			// reachable via the terraform.EvalContext API.
			diags = diags.Append(n.writeChange(ctx, change, ""))
			if diags.HasErrors() {
				return diags
			}
			diags = diags.Append(n.writeResourceInstanceState(ctx, instancePlanState, workingState))
			if diags.HasErrors() {
				return diags
			}

			diags = diags.Append(n.checkPreventDestroy(change))
			if diags.HasErrors() {
				return diags
			}

			// If this plan resulted in a NoOp, then apply won't have a chance to make
			// any changes to the stored dependencies. Since this is a NoOp we know
			// that the stored dependencies will have no effect during apply, and we can
			// write them out now.
			if change.Action == plans.NoOp && !depsEqual(instanceRefreshState.Dependencies, n.Dependencies) {
				// the refresh state will be the final state for this resource, so
				// finalize the dependencies here if they need to be updated.
				instanceRefreshState.Dependencies = n.Dependencies
				diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
				if diags.HasErrors() {
					return diags
				}
			}

			// Post-conditions might block completion. We intentionally do this
			// _after_ writing the state/diff because we want to check against
			// the result of the operation, and to fail on future operations
			// until the user makes the condition succeed.
			// (Note that some preconditions will end up being skipped during
			// planning, because their conditions depend on values not yet known.)
			checkDiags := evalCheckRules(
				addrs.ResourcePostcondition,
				n.Config.Postconditions,
				ctx, n.ResourceInstanceAddr(), repeatData,
				checkRuleSeverity,
			)
			diags = diags.Append(checkDiags)
		} else {
			// The deferrals tracker says that we must defer changes for
			// this resource instance, presumably due to a dependency on an
			// upstream object that was already deferred. Therefore we just
			// report our own deferral (capturing a placeholder value in the
			// deferral tracker) and don't add anything to the plan or
			// working state.
			// In this case, the expression evaluator should use the placeholder
			// value registered here as the value of this resource instance,
			// instead of using the plan.
			deferrals.ReportResourceInstanceDeferred(n.Addr, providers.DeferredReasonDeferredPrereq, change)
		}
	} else {
		// In refresh-only mode we need to evaluate the for-each expression in
		// order to supply the value to the pre- and post-condition check
		// blocks. This has the unfortunate edge case of a refresh-only plan
		// executing with a for-each map which has the same keys but different
		// values, which could result in a post-condition check relying on that
		// value being inaccurate. Unless we decide to store the value of the
		// for-each expression in state, this is unavoidable.
		forEach, _, _ := evaluateForEachExpression(n.Config.ForEach, ctx, false)
		repeatData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

		checkDiags := evalCheckRules(
			addrs.ResourcePrecondition,
			n.Config.Preconditions,
			ctx, addr, repeatData,
			checkRuleSeverity,
		)
		diags = diags.Append(checkDiags)

		// Even if we don't plan changes, we do still need to at least update
		// the working state to reflect the refresh result. If not, then e.g.
		// any output values refering to this will not react to the drift.
		// (Even if we didn't actually refresh above, this will still save
		// the result of any schema upgrading we did in readResourceInstanceState.)
		diags = diags.Append(n.writeResourceInstanceState(ctx, instanceRefreshState, workingState))
		if diags.HasErrors() {
			return diags
		}

		// Here we also evaluate post-conditions after updating the working
		// state, because we want to check against the result of the refresh.
		// Unlike in normal planning mode, these checks are still evaluated
		// even if pre-conditions generated diagnostics, because we have no
		// planned changes to block.
		checkDiags = evalCheckRules(
			addrs.ResourcePostcondition,
			n.Config.Postconditions,
			ctx, addr, repeatData,
			checkRuleSeverity,
		)
		diags = diags.Append(checkDiags)

		// In this case we skipped planning changes and therefore need to report the deferral
		// here, if there was one.
		if refreshDeferred != nil {
			ctx.Deferrals().ReportResourceInstanceDeferred(addr, deferred.Reason, &plans.ResourceInstanceChange{
				Addr: n.Addr,
				Change: plans.Change{
					Action: plans.Read,
					Before: priorInstanceRefreshState.Value,
					After:  instanceRefreshState.Value,
				},
			})
		}
	}

	return diags
}

// replaceTriggered checks if this instance needs to be replace due to a change
// in a replace_triggered_by reference. If replacement is required, the
// instance address is added to forceReplace
func (n *NodePlannableResourceInstance) replaceTriggered(ctx EvalContext, repData instances.RepetitionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if n.Config == nil {
		return diags
	}

	for _, expr := range n.Config.TriggersReplacement {
		ref, replace, evalDiags := ctx.EvaluateReplaceTriggeredBy(expr, repData)
		diags = diags.Append(evalDiags)
		if diags.HasErrors() {
			continue
		}

		if replace {
			// FIXME: forceReplace accomplishes the same goal, however we may
			// want to communicate more information about which resource
			// triggered the replacement in the plan.
			// Rather than further complicating the plan method with more
			// options, we can refactor both of these features later.
			n.forceReplace = append(n.forceReplace, n.Addr)
			log.Printf("[DEBUG] ReplaceTriggeredBy forcing replacement of %s due to change in %s", n.Addr, ref.DisplayString())

			n.replaceTriggeredBy = append(n.replaceTriggeredBy, ref)
			break
		}
	}

	return diags
}

func (n *NodePlannableResourceInstance) importState(ctx EvalContext, addr addrs.AbsResourceInstance, importId string, provider providers.Interface, providerSchema providers.ProviderSchema) (*states.ResourceInstanceObject, *providers.Deferred, tfdiags.Diagnostics) {
	deferralAllowed := ctx.Deferrals().DeferralAllowed()
	var diags tfdiags.Diagnostics
	absAddr := addr.Resource.Absolute(ctx.Path())
	hookResourceID := HookResourceIdentity{
		Addr:         absAddr,
		ProviderAddr: n.ResolvedProvider.Provider,
	}

	var deferred *providers.Deferred

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PrePlanImport(hookResourceID, importId)
	}))
	if diags.HasErrors() {
		return nil, deferred, diags
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.Resource.Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type for %q", n.Addr))
		return nil, deferred, diags
	}

	var resp providers.ImportResourceStateResponse
	if n.override != nil {
		// For overriding resources that are being imported, we cheat a little
		// bit and look ahead at the configuration the user has provided and
		// we'll use that as the basis for the resource we're going to make up
		// that is due to be overridden.

		// Note, we know we have configuration as it's impossible to enable
		// config generation during tests, and the validation that config exists
		// if configuration generation is off has already happened.
		if n.Config == nil {
			// But, just in case we change this at some point in the future,
			// let's add a specific error message here we can test for to
			// document the expectation somewhere. This shouldn't happen in
			// production, so we don't bother with a pretty error.
			diags = diags.Append(fmt.Errorf("override blocks do not support config generation"))
			return nil, deferred, diags
		}

		forEach, _, _ := evaluateForEachExpression(n.Config.ForEach, ctx, false)
		keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)
		configVal, _, configDiags := ctx.EvaluateBlock(n.Config.Config, schema, nil, keyData)
		if configDiags.HasErrors() {
			// We have an overridden resource so we're definitely in a test and
			// the users config is not valid. So give up and just report the
			// problems in the users configuration. Normally, we'd import the
			// resource before giving up but for a test it doesn't matter, the
			// test fails in the same way and the state is just lost anyway.
			//
			// If there were only warnings from the config then we'll duplicate
			// them if we include them (as the config will be loaded again
			// later), so only add the configDiags into the main diags if we
			// found actual errors.
			diags = diags.Append(configDiags)
			return nil, deferred, diags
		}
		configVal, _ = configVal.UnmarkDeep()

		// Let's pretend we're reading the value as a data source so we
		// pre-compute values now as if the resource has already been created.
		override, overrideDiags := mocking.ComputedValuesForDataSource(configVal, mocking.MockedData{
			Value: n.override.Values,
			Range: n.override.ValuesRange,
		}, schema)
		resp = providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: addr.Resource.Resource.Type,
					State:    override,
				},
			},
			Diagnostics: overrideDiags.InConfigBody(n.Config.Config, absAddr.String()),
		}
	} else {
		resp = provider.ImportResourceState(providers.ImportResourceStateRequest{
			TypeName: addr.Resource.Resource.Type,
			ID:       importId,
			ClientCapabilities: providers.ClientCapabilities{
				DeferralAllowed: deferralAllowed,
			},
		})
	}
	// If we don't support deferrals, but the provider reports a deferral and does not
	// emit any error level diagnostics, we should emit an error.
	if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
		diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
	}
	diags = diags.Append(resp.Diagnostics)
	deferred = resp.Deferred
	if diags.HasErrors() {
		return nil, deferred, diags
	}

	imported := resp.ImportedResources

	if len(imported) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Multiple import states not supported",
			fmt.Sprintf("While attempting to import with ID %s, the provider "+
				"returned multiple resource instance states. This "+
				"is not currently supported.",
				importId,
			),
		))
	}

	if len(imported) == 0 {

		// Sanity check against the providers. If the provider defers the response, it may not have been able to return a state, so we'll only error if no deferral was returned.
		if deferred == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Import returned no resources",
				fmt.Sprintf("While attempting to import with ID %s, the provider"+
					"returned no instance states.",
					importId,
				),
			))
			return nil, deferred, diags
		}

		// If we were deferred, then let's make up a resource to represent the
		// state we're going to import.
		state := providers.ImportedResource{
			TypeName: addr.Resource.Resource.Type,
			State:    cty.NullVal(schema.ImpliedType()),
		}

		// We skip the read and further validation since we make up the state
		// of the imported resource anyways.
		return state.AsInstanceObject(), deferred, diags
	}

	for _, obj := range imported {
		log.Printf("[TRACE] graphNodeImportState: import %s %q produced instance object of type %s", absAddr.String(), importId, obj.TypeName)
	}

	importedState := imported[0].AsInstanceObject()

	// We can only call the hooks and validate the imported state if we have
	// actually done the import.
	if resp.Deferred == nil {
		// call post-import hook
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostPlanImport(hookResourceID, imported)
		}))
	}

	if imported[0].TypeName == "" {
		diags = diags.Append(fmt.Errorf("import of %s didn't set type", n.Addr.String()))
		return nil, deferred, diags
	}

	if deferred == nil && importedState.Value.IsNull() {
		// It's actually okay for a deferred import to have returned a null.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Import returned null resource",
			fmt.Sprintf("While attempting to import with ID %s, the provider"+
				"returned an instance with no state.",
				importId,
			),
		))

	}

	// refresh
	riNode := &NodeAbstractResourceInstance{
		Addr: n.Addr,
		NodeAbstractResource: NodeAbstractResource{
			ResolvedProvider: n.ResolvedProvider,
		},
		override: n.override,
	}
	instanceRefreshState, refreshDeferred, refreshDiags := riNode.refresh(ctx, states.NotDeposed, importedState, ctx.Deferrals().DeferralAllowed())
	diags = diags.Append(refreshDiags)
	if diags.HasErrors() {
		return instanceRefreshState, deferred, diags
	}

	// report the refresh was deferred, we don't need to error since the import step succeeded
	if deferred == nil && refreshDeferred != nil {
		deferred = refreshDeferred
	}

	// verify the existence of the imported resource
	if refreshDeferred == nil && instanceRefreshState.Value.IsNull() {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot import non-existent remote object",
			fmt.Sprintf(
				"While attempting to import an existing object to %q, "+
					"the provider detected that no object exists with the given id. "+
					"Only pre-existing objects can be imported; check that the id "+
					"is correct and that it is associated with the provider's "+
					"configured region or endpoint, or use \"terraform apply\" to "+
					"create a new remote object for this resource.",
				n.Addr,
			),
		))
		return instanceRefreshState, deferred, diags
	}

	// If we're importing and generating config, generate it now. We only
	// generate config if the import isn't being deferred. We should generate
	// the configuration in the plan that the import is actually happening in.
	if deferred == nil && len(n.generateConfigPath) > 0 {
		if n.Config != nil {
			return instanceRefreshState, nil, diags.Append(fmt.Errorf("tried to generate config for %s, but it already exists", n.Addr))
		}

		// Generate the HCL string first, then parse the HCL body from it.
		// First we generate the contents of the resource block for use within
		// the planning node. Then we wrap it in an enclosing resource block to
		// pass into the plan for rendering.
		generatedHCLAttributes, generatedDiags := n.generateHCLStringAttributes(n.Addr, instanceRefreshState, schema)
		diags = diags.Append(generatedDiags)

		n.generatedConfigHCL = genconfig.WrapResourceContents(n.Addr, generatedHCLAttributes)

		// parse the "file" as HCL to get the hcl.Body
		synthHCLFile, hclDiags := hclsyntax.ParseConfig([]byte(generatedHCLAttributes), filepath.Base(n.generateConfigPath), hcl.Pos{Byte: 0, Line: 1, Column: 1})
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			return instanceRefreshState, nil, diags
		}

		// We have to do a kind of mini parsing of the content here to correctly
		// mark attributes like 'provider' as hidden. We only care about the
		// resulting content, so it's remain that gets passed into the resource
		// as the config.
		_, remain, resourceDiags := synthHCLFile.Body.PartialContent(configs.ResourceBlockSchema)
		diags = diags.Append(resourceDiags)
		if resourceDiags.HasErrors() {
			return instanceRefreshState, nil, diags
		}

		n.Config = &configs.Resource{
			Mode:     addrs.ManagedResourceMode,
			Type:     n.Addr.Resource.Resource.Type,
			Name:     n.Addr.Resource.Resource.Name,
			Config:   remain,
			Managed:  &configs.ManagedResource{},
			Provider: n.ResolvedProvider.Provider,
		}
	}

	if deferred == nil {
		// Only write the state if the change isn't being deferred. We're also
		// reporting the deferred status to the caller, so they should know
		// not to read from the state.
		diags = diags.Append(riNode.writeResourceInstanceState(ctx, instanceRefreshState, refreshState))
	}
	return instanceRefreshState, deferred, diags
}

// generateHCLStringAttributes produces a string in HCL format for the given
// resource state and schema without the surrounding block.
func (n *NodePlannableResourceInstance) generateHCLStringAttributes(addr addrs.AbsResourceInstance, state *states.ResourceInstanceObject, schema *configschema.Block) (string, tfdiags.Diagnostics) {
	filteredSchema := schema.Filter(
		configschema.FilterOr(
			configschema.FilterReadOnlyAttribute,
			configschema.FilterDeprecatedAttribute,

			// The legacy SDK adds an Optional+Computed "id" attribute to the
			// resource schema even if not defined in provider code.
			// During validation, however, the presence of an extraneous "id"
			// attribute in config will cause an error.
			// Remove this attribute so we do not generate an "id" attribute
			// where there is a risk that it is not in the real resource schema.
			//
			// TRADEOFF: Resources in which there actually is an
			// Optional+Computed "id" attribute in the schema will have that
			// attribute missing from generated config.
			configschema.FilterHelperSchemaIdAttribute,
		),
		configschema.FilterDeprecatedBlock,
	)

	providerAddr := addrs.LocalProviderConfig{
		LocalName: n.ResolvedProvider.Provider.Type,
		Alias:     n.ResolvedProvider.Alias,
	}

	return genconfig.GenerateResourceContents(addr, filteredSchema, providerAddr, state.Value)
}

// mergeDeps returns the union of 2 sets of dependencies
func mergeDeps(a, b []addrs.ConfigResource) []addrs.ConfigResource {
	switch {
	case len(a) == 0:
		return b
	case len(b) == 0:
		return a
	}

	set := make(map[string]addrs.ConfigResource)

	for _, dep := range a {
		set[dep.String()] = dep
	}

	for _, dep := range b {
		set[dep.String()] = dep
	}

	newDeps := make([]addrs.ConfigResource, 0, len(set))
	for _, dep := range set {
		newDeps = append(newDeps, dep)
	}

	return newDeps
}

func depsEqual(a, b []addrs.ConfigResource) bool {
	if len(a) != len(b) {
		return false
	}

	// Because we need to sort the deps to compare equality, make shallow
	// copies to prevent concurrently modifying the array values on
	// dependencies shared between expanded instances.
	copyA, copyB := make([]addrs.ConfigResource, len(a)), make([]addrs.ConfigResource, len(b))
	copy(copyA, a)
	copy(copyB, b)
	a, b = copyA, copyB

	less := func(s []addrs.ConfigResource) func(i, j int) bool {
		return func(i, j int) bool {
			return s[i].String() < s[j].String()
		}
	}

	sort.Slice(a, less(a))
	sort.Slice(b, less(b))

	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
