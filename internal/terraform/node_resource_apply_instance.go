// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeApplyableResourceInstance represents a resource instance that is
// "applyable": it is ready to be applied and is represented by a diff.
//
// This node is for a specific instance of a resource. It will usually be
// accompanied in the graph by a NodeApplyableResource representing its
// containing resource, and should depend on that node to ensure that the
// state is properly prepared to receive changes to instances.
type NodeApplyableResourceInstance struct {
	*NodeAbstractResourceInstance

	graphNodeDeposer // implementation of GraphNodeDeposerConfig

	// forceReplace indicates that this resource is being replaced for external
	// reasons, like a -replace flag or via replace_triggered_by.
	forceReplace bool

	resolvedActionProviders []addrs.AbsProviderConfig
}

var (
	_ GraphNodeConfigResource         = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeResourceInstance       = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeCreator                = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeReferencer             = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeDeposer                = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeExecutable             = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeAttachDependencies     = (*NodeApplyableResourceInstance)(nil)
	_ GraphNodeActionProviderConsumer = (*NodeApplyableResourceInstance)(nil)
)

// GraphNodeCreator
func (n *NodeApplyableResourceInstance) CreateAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeReferencer, overriding NodeAbstractResourceInstance
func (n *NodeApplyableResourceInstance) References() []*addrs.Reference {
	// Start with the usual resource instance implementation
	ret := n.NodeAbstractResourceInstance.References()

	// Applying a resource must also depend on the destruction of any of its
	// dependencies, since this may for example affect the outcome of
	// evaluating an entire list of resources with "count" set (by reducing
	// the count).
	//
	// However, we can't do this in create_before_destroy mode because that
	// would create a dependency cycle. We make a compromise here of requiring
	// changes to be updated across two applies in this case, since the first
	// plan will use the old values.
	if !n.CreateBeforeDestroy() {
		for _, ref := range ret {
			switch tr := ref.Subject.(type) {
			case addrs.ResourceInstance:
				newRef := *ref // shallow copy so we can mutate
				newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
				newRef.Remaining = nil // can't access attributes of something being destroyed
				ret = append(ret, &newRef)
			case addrs.Resource:
				newRef := *ref // shallow copy so we can mutate
				newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
				newRef.Remaining = nil // can't access attributes of something being destroyed
				ret = append(ret, &newRef)
			}
		}
	}

	return ret
}

// GraphNodeAttachDependencies
func (n *NodeApplyableResourceInstance) AttachDependencies(deps []addrs.ConfigResource) {
	n.Dependencies = deps
}

// GraphNodeExecutable
func (n *NodeApplyableResourceInstance) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	addr := n.ResourceInstanceAddr()

	if n.Config == nil {
		// If there is no config, and there is no change, then we have nothing
		// to do and the change was left in the plan for informational
		// purposes only.
		changes := ctx.Changes()
		csrc := changes.GetResourceInstanceChange(n.ResourceInstanceAddr(), addrs.NotDeposed)
		if csrc == nil || csrc.Action == plans.NoOp {
			log.Printf("[DEBUG] NodeApplyableResourceInstance: No config or planned change recorded for %s", n.Addr)
			return nil
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource node has no configuration attached",
			fmt.Sprintf(
				"The graph node for %s has no configuration attached to it. This suggests a bug in Terraform's apply graph builder; please report it!",
				addr,
			),
		))
		return diags
	}

	// Eval info is different depending on what kind of resource this is
	switch n.Config.Mode {
	case addrs.ManagedResourceMode:
		return n.managedResourceExecute(ctx)
	case addrs.DataResourceMode:
		return n.dataResourceExecute(ctx)
	case addrs.EphemeralResourceMode:
		return n.ephemeralResourceExecute(ctx)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodeApplyableResourceInstance) ephemeralResourceExecute(ctx EvalContext) tfdiags.Diagnostics {
	_, diags := ephemeralResourceOpen(ctx, ephemeralResourceInput{
		addr:           n.Addr,
		config:         n.Config,
		providerConfig: n.ResolvedProvider,
	})

	return diags
}

func (n *NodeApplyableResourceInstance) dataResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	change, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}
	// Stop early if we don't actually have a diff
	if change == nil {
		return diags
	}
	if change.Action != plans.Read && change.Action != plans.NoOp {
		diags = diags.Append(fmt.Errorf("nonsensical planned action %#v for %s; this is a bug in Terraform", change.Action, n.Addr))
	}

	// In this particular call to applyDataSource we include our planned
	// change, which signals that we expect this read to complete fully
	// with no unknown values; it'll produce an error if not.
	state, repeatData, applyDiags := n.applyDataSource(ctx, change)
	diags = diags.Append(applyDiags)
	if diags.HasErrors() {
		return diags
	}

	if state != nil {
		// If n.applyDataSource returned a nil state object with no accompanying
		// errors then it determined that the given change doesn't require
		// actually reading the data (e.g. because it was already read during
		// the plan phase) and so we're only running through here to get the
		// extra details like precondition/postcondition checks.
		diags = diags.Append(n.writeResourceInstanceState(ctx, state, workingState))
		if diags.HasErrors() {
			return diags
		}
	}

	diags = diags.Append(n.writeChange(ctx, nil, ""))

	diags = diags.Append(updateStateHook(ctx))

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state/diff because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		n.Config.Postconditions,
		ctx, n.ResourceInstanceAddr(),
		repeatData,
		tfdiags.Error,
	)
	diags = diags.Append(checkDiags)

	return diags
}

func (n *NodeApplyableResourceInstance) managedResourceExecute(ctx EvalContext) (diags tfdiags.Diagnostics) {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var state *states.ResourceInstanceObject
	var createBeforeDestroyEnabled bool
	var deposedKey states.DeposedKey

	addr := n.ResourceInstanceAddr().Resource
	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Get the saved diff for apply
	diffApply, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Get & Order the actions for this resource
	var beforeActions, afterActions map[int]actionInvocationInstanceSrcs
	if len(n.plannedActions) > 0 {
		actions := n.plannedActions
		beforeActions, afterActions = sortAndOrderActions(actions)
	}

	// We don't want to do any destroys
	// (these are handled by NodeDestroyResourceInstance instead)
	if diffApply == nil || diffApply.Action == plans.Delete {
		return diags
	}
	if diffApply.Action == plans.Read {
		diags = diags.Append(fmt.Errorf("nonsensical planned action %#v for %s; this is a bug in Terraform", diffApply.Action, n.Addr))
	}

	destroy := (diffApply.Action == plans.Delete || diffApply.Action.IsReplace())
	// Get the stored action for CBD if we have a plan already
	createBeforeDestroyEnabled = diffApply.Change.Action == plans.CreateThenDelete

	if destroy && n.CreateBeforeDestroy() {
		createBeforeDestroyEnabled = true
	}

	if createBeforeDestroyEnabled {
		state := ctx.State()
		if n.PreallocatedDeposedKey == states.NotDeposed {
			deposedKey = state.DeposeResourceInstanceObject(n.Addr)
		} else {
			deposedKey = n.PreallocatedDeposedKey
			state.DeposeResourceInstanceObjectForceKey(n.Addr, deposedKey)
		}
		log.Printf("[TRACE] managedResourceExecute: prior object for %s now deposed with key %s", n.Addr, deposedKey)
	}

	state, readDiags := n.readResourceInstanceState(ctx, n.ResourceInstanceAddr())
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return diags
	}

	// Get the saved diff
	diff, err := n.readDiff(ctx, providerSchema)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return diags
	}

	// Make a new diff, in case we've learned new values in the state
	// during apply which we can now incorporate.
	diffApply, _, deferred, repeatData, planDiags := n.plan(ctx, diff, state, false, n.forceReplace)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return diags
	}

	if deferred != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource deferred during apply, but not during plan",
			Detail: fmt.Sprintf(
				"Terraform has encountered a bug where a provider would mark the resource %q as deferred during apply, but not during plan. This is most likely a bug in the provider. Please file an issue with the provider.", n.Addr,
			),
			Subject: n.Config.DeclRange.Ptr(),
		})
		return diags
	}

	// Compare the diffs
	diags = diags.Append(n.checkPlannedChange(ctx, diff, diffApply, providerSchema))
	if diags.HasErrors() {
		return diags
	}

	diffApply = reducePlan(addr, diffApply, false)
	// reducePlan may have simplified our planned change
	// into a NoOp if it only requires destroying, since destroying
	// is handled by NodeDestroyResourceInstance. If so, we'll
	// still run through most of the logic here because we do still
	// need to deal with other book-keeping such as marking the
	// change as "complete", and running the author's postconditions.

	diags = diags.Append(n.preApplyHook(ctx, diffApply))
	if diags.HasErrors() {
		return diags
	}

	// If there is no change, there was nothing to apply, and we don't need to
	// re-write the state, but we do need to re-evaluate postconditions.
	if diffApply.Action == plans.NoOp {
		return diags.Append(n.managedResourcePostconditions(ctx, repeatData))
	}

	// Re-evaluate the condition and trigger any before_* actions
	if len(beforeActions) > 0 {
		actionDiags := n.applyActions(ctx, beforeActions)
		diags = diags.Append(actionDiags)
		if diags.HasErrors() {
			// quit if any before action failed
			return diags
		}
	}

	state, applyDiags := n.apply(ctx, state, diffApply, n.Config, repeatData, n.CreateBeforeDestroy())
	diags = diags.Append(applyDiags)

	// We clear the change out here so that future nodes don't see a change
	// that is already complete.
	err = n.writeChange(ctx, nil, "")
	if err != nil {
		return diags.Append(err)
	}

	state = maybeTainted(addr.Absolute(ctx.Path()), state, diffApply, diags.Err())

	if state != nil {
		// dependencies are always updated to match the configuration during apply
		state.Dependencies = n.Dependencies
	}
	err = n.writeResourceInstanceState(ctx, state, workingState)
	if err != nil {
		return diags.Append(err)
	}

	// Run Provisioners
	createNew := (diffApply.Action == plans.Create || diffApply.Action.IsReplace())
	applyProvisionersDiags := n.evalApplyProvisioners(ctx, state, createNew, configs.ProvisionerWhenCreate)
	// the provisioner errors count as port of the apply error, so we can bundle the diags
	diags = diags.Append(applyProvisionersDiags)

	state = maybeTainted(addr.Absolute(ctx.Path()), state, diffApply, diags.Err())

	err = n.writeResourceInstanceState(ctx, state, workingState)
	if err != nil {
		return diags.Append(err)
	}

	if createBeforeDestroyEnabled && diags.HasErrors() {
		if deposedKey == states.NotDeposed {
			// This should never happen, and so it always indicates a bug.
			// We should evaluate this node only if we've previously deposed
			// an object as part of the same operation.
			if diffApply != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Attempt to restore non-existent deposed object",
					fmt.Sprintf(
						"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This occurred during a %s action. This is a bug in Terraform; please report it!",
						addr, diffApply.Action,
					),
				))
			} else {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Attempt to restore non-existent deposed object",
					fmt.Sprintf(
						"Terraform has encountered a bug where it would need to restore a deposed object for %s without knowing a deposed object key for that object. This is a bug in Terraform; please report it!",
						addr,
					),
				))
			}
		} else {
			restored := ctx.State().MaybeRestoreResourceInstanceDeposed(addr.Absolute(ctx.Path()), deposedKey)
			if restored {
				log.Printf("[TRACE] managedResourceExecute: %s deposed object %s was restored as the current object", addr, deposedKey)
			} else {
				log.Printf("[TRACE] managedResourceExecute: %s deposed object %s remains deposed", addr, deposedKey)
			}
		}
	}

	diags = diags.Append(n.postApplyHook(ctx, state, diags.Err()))
	diags = diags.Append(updateStateHook(ctx))

	// now run the after actions
	if len(afterActions) > 0 {
		actionDiags := n.applyActions(ctx, afterActions)
		diags = diags.Append(actionDiags)
		if diags.HasErrors() {
			// quit if any before action failed
			return diags
		}
	}

	// Post-conditions might block further progress. We intentionally do this
	// _after_ writing the state because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	return diags.Append(n.managedResourcePostconditions(ctx, repeatData))
}

func (n *NodeApplyableResourceInstance) managedResourcePostconditions(ctx EvalContext, repeatData instances.RepetitionData) (diags tfdiags.Diagnostics) {

	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		n.Config.Postconditions,
		ctx, n.ResourceInstanceAddr(), repeatData,
		tfdiags.Error,
	)
	return diags.Append(checkDiags)
}

// checkPlannedChange produces errors if the _actual_ expected value is not
// compatible with what was recorded in the plan.
//
// Errors here are most often indicative of a bug in the provider, so our error
// messages will report with that in mind. It's also possible that there's a bug
// in Terraform's Core's own "proposed new value" code in EvalDiff.
func (n *NodeApplyableResourceInstance) checkPlannedChange(ctx EvalContext, plannedChange, actualChange *plans.ResourceInstanceChange, providerSchema providers.ProviderSchema) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	addr := n.ResourceInstanceAddr().Resource

	schema := providerSchema.SchemaForResourceAddr(addr.ContainingResource())
	if schema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support %q", addr.Resource.Type))
		return diags
	}

	absAddr := addr.Absolute(ctx.Path())

	log.Printf("[TRACE] checkPlannedChange: Verifying that actual change (action %s) matches planned change (action %s)", actualChange.Action, plannedChange.Action)

	if plannedChange.Action != actualChange.Action {
		switch {
		case plannedChange.Action == plans.Update && actualChange.Action == plans.NoOp:
			// It's okay for an update to become a NoOp once we've filled in
			// all of the unknown values, since the final values might actually
			// match what was there before after all.
			log.Printf("[DEBUG] After incorporating new values learned so far during apply, %s change has become NoOp", absAddr)

		case (plannedChange.Action == plans.CreateThenDelete && actualChange.Action == plans.DeleteThenCreate) ||
			(plannedChange.Action == plans.DeleteThenCreate && actualChange.Action == plans.CreateThenDelete):
			// If the order of replacement changed, then that is a bug in terraform
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Terraform produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, the planned action changed from %s to %s.\n\nThis is a bug in Terraform and should be reported.",
					absAddr, plannedChange.Action, actualChange.Action,
				),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, provider %q changed the planned action from %s to %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					absAddr, n.ResolvedProvider.Provider.String(),
					plannedChange.Action, actualChange.Action,
				),
			))
		}
	}

	errs := objchange.AssertObjectCompatible(schema.Body, plannedChange.After, actualChange.After)
	for _, err := range errs {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced inconsistent final plan",
			fmt.Sprintf(
				"When expanding the plan for %s to include new values learned so far during apply, provider %q produced an invalid new value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				absAddr, n.ResolvedProvider.Provider.String(), tfdiags.FormatError(err),
			),
		))
	}
	return diags
}

// maybeTainted takes the resource addr, new value, planned change, and possible
// error from an apply operation and return a new instance object marked as
// tainted if it appears that a create operation has failed.
func maybeTainted(addr addrs.AbsResourceInstance, state *states.ResourceInstanceObject, change *plans.ResourceInstanceChange, err error) *states.ResourceInstanceObject {
	if state == nil || change == nil || err == nil {
		return state
	}
	if state.Status == states.ObjectTainted {
		log.Printf("[TRACE] maybeTainted: %s was already tainted, so nothing to do", addr)
		return state
	}
	if change.Action == plans.Create {
		// If there are errors during a _create_ then the object is
		// in an undefined state, and so we'll mark it as tainted so
		// we can try again on the next run.
		//
		// We don't do this for other change actions because errors
		// during updates will often not change the remote object at all.
		// If there _were_ changes prior to the error, it's the provider's
		// responsibility to record the effect of those changes in the
		// object value it returned.
		log.Printf("[TRACE] maybeTainted: %s encountered an error during creation, so it is now marked as tainted", addr)
		return state.AsTainted()
	}
	return state
}

func (n *NodeApplyableResourceInstance) applyActions(ctx EvalContext, actions map[int]actionInvocationInstanceSrcs) tfdiags.Diagnostics {

	// the map keys correlate to the actionTriggerBlockIndex, so start by making a map of index position -> map key
	keys := make([]int, 0)
	for k := range actions {
		keys = append(keys, k)
	}

	var diags tfdiags.Diagnostics
	// for each action_trigger block
	for i := 0; i < len(actions); i++ {
		// for each action
		for j := 0; j < len(actions[keys[i]]); j++ {
			// evaluate condition again
			triggerConfig := n.Config.Managed.ActionTriggers[i]
			if triggerConfig == nil {
				panic("well at least you didn't expect that to work")
			}
			aiSrc := actions[keys[i]][j]

			actionInstanceNode, ok := n.actionInstances.GetOk(aiSrc.Addr.ConfigAction())
			if !ok {
				panic("HOW")
			}

			aschema := actionInstanceNode.ActionSchema()
			ai, err := aiSrc.Decode(aschema)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to decode ", fmt.Sprintf("Terraform failed to decode a planned action invocation: %v\n\nThis is a bug in Terraform; please report it!", err)))
			}
			at := ai.ActionTrigger.(*plans.LifecycleActionTrigger)
			if triggerConfig.Condition != nil {
				condition, conditionDiags := evaluateActionCondition(ctx, actionConditionContext{
					// For applying the triggering event is sufficient, if the condition could not have
					// been evaluated due to in invalid mix of events we would have caught it during planning.
					events:          []configs.ActionTriggerEvent{at.ActionTriggerEvent},
					conditionExpr:   triggerConfig.Condition,
					resourceAddress: at.TriggeringResourceAddr,
				})
				diags = diags.Append(conditionDiags)
				if diags.HasErrors() {
					return diags
				}

				if !condition {
					return diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Condition changed evaluation during apply",
						Detail:   "The condition evaluated to false during apply, but was true during planning. This may lead to unexpected behavior.",
						Subject:  triggerConfig.Condition.Range().Ptr(),
					})
				}
			}

			actionData, ok := ctx.Actions().GetActionInstance(ai.Addr)
			if !ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Action instance not found",
					Detail:   "Could not find action instance for address " + ai.Addr.String(),
					//Subject:
				})
				return diags
			}

			provider, schema, err := getProvider(ctx, actionData.ProviderAddr)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Failed to get action provider for %s", ai.Addr),
					Detail:   fmt.Sprintf("Failed to get action provider: %s", err),
					//Subject:  n.ActionTriggerRange,
				})
				return diags
			}

			actionSchema, ok := schema.Actions[ai.Addr.Action.Action.Type]
			if !ok {
				// This should have been caught earlier
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Action %s not found in provider schema", ai.Addr),
					Detail:   fmt.Sprintf("The action %s was not found in the provider schema for %s", ai.Addr.Action.Action.Type, actionData.ProviderAddr),
					//Subject:  n.ActionTriggerRange,
				})
				return diags
			}

			configValue := actionData.ConfigValue

			// Validate that what we planned matches the action data we have.
			errs := objchange.AssertObjectCompatible(actionSchema.ConfigSchema, ai.ConfigValue, ephemeral.RemoveEphemeralValues(configValue))
			for _, err := range errs {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Provider produced inconsistent final plan",
					Detail: fmt.Sprintf("When expanding the plan for %s to include new values learned so far during apply, Terraform produced an invalid new value for %s.\n\nThis is a bug in Terraform, which should be reported.",
						ai.Addr, tfdiags.FormatError(err)),
					//Subject: n.ActionTriggerRange,
				})
			}

			if !configValue.IsWhollyKnown() {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Action configuration unknown during apply",
					Detail:   fmt.Sprintf("The action %s was not fully known during apply.\n\nThis is a bug in Terraform, please report it.", ai.Addr.Action.String()),
					//Subject:  n.ActionTriggerRange,
				})
			}

			hookIdentity := HookActionIdentity{
				Addr:          ai.Addr,
				ActionTrigger: ai.ActionTrigger,
			}

			diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
				return h.StartAction(hookIdentity)
			}))
			if diags.HasErrors() {
				return diags
			}

			// We don't want to send the marks, but all marks are okay in the context
			// of an action invocation. We can't reuse our ephemeral free value from
			// above because we want the ephemeral values to be included.
			unmarkedConfigValue, _ := configValue.UnmarkDeep()
			resp := provider.InvokeAction(providers.InvokeActionRequest{
				ActionType:         ai.Addr.Action.Action.Type,
				PlannedActionData:  unmarkedConfigValue,
				ClientCapabilities: ctx.ClientCapabilities(),
			})

			//respDiags := n.AddSubjectToDiagnostics(resp.Diagnostics)
			respDiags := resp.Diagnostics
			diags = diags.Append(respDiags)
			if respDiags.HasErrors() {
				diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
					return h.CompleteAction(hookIdentity, respDiags.Err())
				}))
				return diags
			}

			if resp.Events != nil { // should only occur in misconfigured tests
				for event := range resp.Events {
					switch ev := event.(type) {
					case providers.InvokeActionEvent_Progress:
						diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
							return h.ProgressAction(hookIdentity, ev.Message)
						}))
						if diags.HasErrors() {
							return diags
						}
					case providers.InvokeActionEvent_Completed:
						// Enhance the diagnostics
						//diags = diags.Append(n.AddSubjectToDiagnostics(ev.Diagnostics))
						diags = diags.Append(ev.Diagnostics)
						diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
							return h.CompleteAction(hookIdentity, ev.Diagnostics.Err())
						}))
						if ev.Diagnostics.HasErrors() {
							return diags
						}
						if diags.HasErrors() {
							return diags
						}
					default:
						panic(fmt.Sprintf("unexpected action event type %T", ev))
					}
				}
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Provider return invalid response",
					Detail:   "Provider response did not include any events",
					//Subject:  n.ActionTriggerRange,
				})
			}
		}
	}
	return diags
}

// sortAndOrderActions iterates through actions planned for this resource and
// returns two maps with before and after actions, with int keys indicating the ActionTriggerBlockIndex.
func sortAndOrderActions(actions []*plans.ActionInvocationInstanceSrc) (map[int]actionInvocationInstanceSrcs, map[int]actionInvocationInstanceSrcs) {
	var before, after []*plans.ActionInvocationInstanceSrc
	// sort into before and after actions
	for _, a := range actions {
		trigger, ok := a.ActionTrigger.(*plans.LifecycleActionTrigger)
		if !ok {
			panic("this action does not belong here") // this should not be possible
		}

		if isBeforeAction(trigger) {
			before = append(before, a)
		} else {
			after = append(after, a)
		}
	}
	return orderActions(before), orderActions(after)
}

// The map key is the action trigger block index, and the value is an ordered list of actions for that trigger block.
func orderActions(actions []*plans.ActionInvocationInstanceSrc) map[int]actionInvocationInstanceSrcs {
	sorted := make(map[int]actionInvocationInstanceSrcs)
	seenTriggers := make([]int, 0) // for sorting the map by keys

	for _, actionInstance := range actions {
		trigger, ok := actionInstance.ActionTrigger.(*plans.LifecycleActionTrigger)
		if !ok {
			panic("this action does not belong here") // this should not be possible
		}
		if _, ok := sorted[trigger.ActionTriggerBlockIndex]; !ok {
			seenTriggers = append(seenTriggers, trigger.ActionTriggerBlockIndex)
			sorted[trigger.ActionTriggerBlockIndex] = make(actionInvocationInstanceSrcs, 0)
		}
		sorted[trigger.ActionTriggerBlockIndex] = append(sorted[trigger.ActionTriggerBlockIndex], actionInstance)
	}

	sort.Ints(seenTriggers)

	for _, k := range seenTriggers {
		// sort the actions by actionListIndex
		sort.Sort(sorted[k])
	}

	return sorted
}

func isBeforeAction(a *plans.LifecycleActionTrigger) bool {
	switch a.ActionTriggerEvent {
	case configs.BeforeCreate, configs.BeforeUpdate:
		return true
	case configs.AfterCreate, configs.AfterUpdate:
		return false
	default: // this should be impossible: did you implement destroy and forget me?
		panic("unknown action event")
	}
}

type actionInvocationInstanceSrcs []*plans.ActionInvocationInstanceSrc

func (o actionInvocationInstanceSrcs) Len() int      { return len(o) }
func (o actionInvocationInstanceSrcs) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o actionInvocationInstanceSrcs) Less(i, j int) bool {
	itrigger, _ := o[i].ActionTrigger.(*plans.LifecycleActionTrigger)
	jtrigger, _ := o[j].ActionTrigger.(*plans.LifecycleActionTrigger)
	return itrigger.ActionsListIndex < jtrigger.ActionsListIndex
}

func (n *NodeApplyableResourceInstance) ActionsProvidedBy() []addrs.AbsProviderConfig {
	providers := make([]addrs.AbsProviderConfig, 0)
	if n.plannedActions != nil {
		for _, action := range n.plannedActions {
			providers = append(providers, action.ProviderAddr)
		}
	}
	return providers
}

func (n *NodeApplyableResourceInstance) AppendProvider(provider addrs.AbsProviderConfig) {
	if n.resolvedActionProviders == nil {
		n.resolvedActionProviders = make([]addrs.AbsProviderConfig, 0)
	}
	n.resolvedActionProviders = append(n.resolvedActionProviders, provider)
}
