// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// NodeAbstractResourceInstance represents a resource instance with no
// associated operations. It embeds NodeAbstractResource but additionally
// contains an instance key, used to identify one of potentially many
// instances that were created from a resource in configuration, e.g. using
// the "count" or "for_each" arguments.
type NodeAbstractResourceInstance struct {
	NodeAbstractResource
	Addr addrs.AbsResourceInstance

	// These are set via the AttachState method.
	instanceState *states.ResourceInstance

	Dependencies []addrs.ConfigResource

	preDestroyRefresh bool

	// During import we may generate configuration for a resource, which needs
	// to be stored in the final change.
	generatedConfigHCL string

	// override is set by the graph itself, just before this node executes.
	override *configs.Override
}

// NewNodeAbstractResourceInstance creates an abstract resource instance graph
// node for the given absolute resource instance address.
func NewNodeAbstractResourceInstance(addr addrs.AbsResourceInstance) *NodeAbstractResourceInstance {
	// Due to the fact that we embed NodeAbstractResource, the given address
	// actually ends up split between the resource address in the embedded
	// object and the InstanceKey field in our own struct. The
	// ResourceInstanceAddr method will stick these back together again on
	// request.
	r := NewNodeAbstractResource(addr.ContainingResource().Config())
	return &NodeAbstractResourceInstance{
		NodeAbstractResource: *r,
		Addr:                 addr,
	}
}

func (n *NodeAbstractResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String()
}

func (n *NodeAbstractResourceInstance) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

func (n *NodeAbstractResourceInstance) HookResourceIdentity() HookResourceIdentity {
	return HookResourceIdentity{
		Addr:         n.Addr,
		ProviderAddr: n.ResolvedProvider.Provider,
	}
}

// GraphNodeReferenceable
func (n *NodeAbstractResourceInstance) ReferenceableAddrs() []addrs.Referenceable {
	addr := n.ResourceInstanceAddr()
	return []addrs.Referenceable{
		addr.Resource,

		// A resource instance can also be referenced by the address of its
		// containing resource, so that e.g. a reference to aws_instance.foo
		// would match both aws_instance.foo[0] and aws_instance.foo[1].
		addr.ContainingResource().Resource,
	}
}

// GraphNodeReferencer
func (n *NodeAbstractResourceInstance) References() []*addrs.Reference {
	// If we have a configuration attached then we'll delegate to our
	// embedded abstract resource, which knows how to extract dependencies
	// from configuration. If there is no config, then the dependencies will
	// be connected during destroy from those stored in the state.
	if n.Config != nil {
		if n.Schema == nil {
			// We'll produce a log message about this out here so that
			// we can include the full instance address, since the equivalent
			// message in NodeAbstractResource.References cannot see it.
			log.Printf("[WARN] no schema is attached to %s, so config references cannot be detected", n.Name())
			return nil
		}
		return n.NodeAbstractResource.References()
	}

	// If we have neither config nor state then we have no references.
	return nil
}

// StateDependencies returns the dependencies which will be saved in the state
// for managed resources, or the most current dependencies for data resources.
func (n *NodeAbstractResourceInstance) StateDependencies() []addrs.ConfigResource {
	// Managed resources prefer the stored dependencies, to avoid possible
	// conflicts in ordering when refactoring configuration.
	if s := n.instanceState; s != nil {
		if s.Current != nil {
			return s.Current.Dependencies
		}
	}

	// If there are no stored dependencies, this is either a newly created
	// managed resource, or a data source, and we can use the most recently
	// calculated dependencies.
	return n.Dependencies
}

// GraphNodeResourceInstance
func (n *NodeAbstractResourceInstance) ResourceInstanceAddr() addrs.AbsResourceInstance {
	return n.Addr
}

// GraphNodeAttachResourceState
func (n *NodeAbstractResourceInstance) AttachResourceState(s *states.Resource) {
	if s == nil {
		log.Printf("[WARN] attaching nil state to %s", n.Addr)
		return
	}
	log.Printf("[TRACE] NodeAbstractResourceInstance.AttachResourceState for %s", n.Addr)
	n.instanceState = s.Instance(n.Addr.Resource.Key)
	n.storedProviderConfig = s.ProviderConfig
}

// GraphNodeOverridable
func (n *NodeAbstractResourceInstance) ConfigProvider() addrs.AbsProviderConfig {
	return n.ResolvedProvider
}

// GraphNodeOverridable
func (n *NodeAbstractResourceInstance) SetOverride(override *configs.Override) {
	n.override = override
}

// readDiff returns the planned change for a particular resource instance
// object.
func (n *NodeAbstractResourceInstance) readDiff(ctx EvalContext, providerSchema providers.ProviderSchema) (*plans.ResourceInstanceChange, error) {
	changes := ctx.Changes()
	addr := n.ResourceInstanceAddr()

	schema, _ := providerSchema.SchemaForResourceAddr(addr.Resource.Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", addr.Resource.Resource.Type)
	}

	change := changes.GetResourceInstanceChange(addr, addrs.NotDeposed)

	log.Printf("[TRACE] readDiff: Read %s change from plan for %s", change.Action, n.Addr)

	return change, nil
}

func (n *NodeAbstractResourceInstance) checkPreventDestroy(change *plans.ResourceInstanceChange) tfdiags.Diagnostics {
	if change == nil || n.Config == nil || n.Config.Managed == nil {
		return nil
	}

	preventDestroy := n.Config.Managed.PreventDestroy

	if (change.Action == plans.Delete || change.Action.IsReplace()) && preventDestroy {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Instance cannot be destroyed",
			Detail: fmt.Sprintf(
				"Resource %s has lifecycle.prevent_destroy set, but the plan calls for this resource to be destroyed. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or reduce the scope of the plan using the -target option.",
				n.Addr.String(),
			),
			Subject: &n.Config.DeclRange,
		})
		return diags
	}

	return nil
}

// preApplyHook calls the pre-Apply hook
func (n *NodeAbstractResourceInstance) preApplyHook(ctx EvalContext, change *plans.ResourceInstanceChange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if change == nil {
		panic(fmt.Sprintf("preApplyHook for %s called with nil Change", n.Addr))
	}

	// Only managed resources have user-visible apply actions.
	if n.Addr.Resource.Resource.Mode == addrs.ManagedResourceMode {
		priorState := change.Before
		plannedNewState := change.After

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(n.HookResourceIdentity(), change.DeposedKey, change.Action, priorState, plannedNewState)
		}))
		if diags.HasErrors() {
			return diags
		}
	}

	return nil
}

// postApplyHook calls the post-Apply hook
func (n *NodeAbstractResourceInstance) postApplyHook(ctx EvalContext, state *states.ResourceInstanceObject, err error) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Only managed resources have user-visible apply actions.
	if n.Addr.Resource.Resource.Mode == addrs.ManagedResourceMode {
		var newState cty.Value
		if state != nil {
			newState = state.Value
		} else {
			newState = cty.NullVal(cty.DynamicPseudoType)
		}
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(n.HookResourceIdentity(), addrs.NotDeposed, newState, err)
		}))
	}

	return diags
}

type phaseState int

const (
	workingState phaseState = iota
	refreshState
	prevRunState
)

//go:generate go run golang.org/x/tools/cmd/stringer -type phaseState

// writeResourceInstanceState saves the given object as the current object for
// the selected resource instance.
//
// dependencies is a parameter, instead of those directly attacted to the
// NodeAbstractResourceInstance, because we don't write dependencies for
// datasources.
//
// targetState determines which context state we're writing to during plan. The
// default is the global working state.
func (n *NodeAbstractResourceInstance) writeResourceInstanceState(ctx EvalContext, obj *states.ResourceInstanceObject, targetState phaseState) error {
	return n.writeResourceInstanceStateImpl(ctx, states.NotDeposed, obj, targetState)
}

func (n *NodeAbstractResourceInstance) writeResourceInstanceStateDeposed(ctx EvalContext, deposedKey states.DeposedKey, obj *states.ResourceInstanceObject, targetState phaseState) error {
	if deposedKey == states.NotDeposed {
		// Bail out to avoid silently doing something other than what the
		// caller seems to have intended.
		panic("trying to write current state object using writeResourceInstanceStateDeposed")
	}
	return n.writeResourceInstanceStateImpl(ctx, deposedKey, obj, targetState)
}

// (this is the private common body of both writeResourceInstanceState and
// writeResourceInstanceStateDeposed. Don't call it directly; instead, use
// one of the two wrappers to be explicit about which of the instance's
// objects you are intending to write.
func (n *NodeAbstractResourceInstance) writeResourceInstanceStateImpl(ctx EvalContext, deposedKey states.DeposedKey, obj *states.ResourceInstanceObject, targetState phaseState) error {
	absAddr := n.Addr
	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}
	logFuncName := "NodeAbstractResouceInstance.writeResourceInstanceState"
	if deposedKey == states.NotDeposed {
		log.Printf("[TRACE] %s to %s for %s", logFuncName, targetState, absAddr)
	} else {
		logFuncName = "NodeAbstractResouceInstance.writeResourceInstanceStateDeposed"
		log.Printf("[TRACE] %s to %s for %s (deposed key %s)", logFuncName, targetState, absAddr, deposedKey)
	}

	var state *states.SyncState
	switch targetState {
	case workingState:
		state = ctx.State()
	case refreshState:
		state = ctx.RefreshState()
	case prevRunState:
		state = ctx.PrevRunState()
	default:
		panic(fmt.Sprintf("unsupported phaseState value %#v", targetState))
	}
	if state == nil {
		// Should not happen, because we shouldn't ever try to write to
		// a state that isn't applicable to the current operation.
		// (We can also get in here for unit tests which are using
		// EvalContextMock but not populating PrevRunStateState with
		// a suitable state object.)
		return fmt.Errorf("state of type %s is not applicable to the current operation; this is a bug in Terraform", targetState)
	}

	// In spite of the name, this function also handles the non-deposed case
	// via the writeResourceInstanceState wrapper, by setting deposedKey to
	// the NotDeposed value (the zero value of DeposedKey).
	var write func(src *states.ResourceInstanceObjectSrc)
	if deposedKey == states.NotDeposed {
		write = func(src *states.ResourceInstanceObjectSrc) {
			state.SetResourceInstanceCurrent(absAddr, src, n.ResolvedProvider)
		}
	} else {
		write = func(src *states.ResourceInstanceObjectSrc) {
			state.SetResourceInstanceDeposed(absAddr, deposedKey, src, n.ResolvedProvider)
		}
	}

	if obj == nil || obj.Value.IsNull() {
		// No need to encode anything: we'll just write it directly.
		write(nil)
		log.Printf("[TRACE] %s: removing state object for %s", logFuncName, absAddr)
		return nil
	}

	log.Printf("[TRACE] %s: writing state object for %s", logFuncName, absAddr)

	schema, currentVersion := providerSchema.SchemaForResourceAddr(absAddr.ContainingResource().Resource)
	if schema == nil {
		// It shouldn't be possible to get this far in any real scenario
		// without a schema, but we might end up here in contrived tests that
		// fail to set up their world properly.
		return fmt.Errorf("failed to encode %s in state: no resource type schema available", absAddr)
	}

	src, err := obj.Encode(schema.ImpliedType(), currentVersion)
	if err != nil {
		return fmt.Errorf("failed to encode %s in state: %s", absAddr, err)
	}

	write(src)
	return nil
}

// planDestroy returns a plain destroy diff.
func (n *NodeAbstractResourceInstance) planDestroy(ctx EvalContext, currentState *states.ResourceInstanceObject, deposedKey states.DeposedKey) (*plans.ResourceInstanceChange, *providers.Deferred, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var deferred *providers.Deferred
	var plan *plans.ResourceInstanceChange

	absAddr := n.Addr
	deferralAllowed := ctx.Deferrals().DeferralAllowed()

	if n.ResolvedProvider.Provider.Type == "" {
		if deposedKey == "" {
			panic(fmt.Sprintf("planDestroy for %s does not have ProviderAddr set", absAddr))
		} else {
			panic(fmt.Sprintf("planDestroy for %s (deposed %s) does not have ProviderAddr set", absAddr, deposedKey))
		}
	}

	// If there is no state or our attributes object is null then we're already
	// destroyed.
	if currentState == nil || currentState.Value.IsNull() {
		// We still need to generate a NoOp change, because that allows
		// outside consumers of the plan to distinguish between us affirming
		// that we checked something and concluded no changes were needed
		// vs. that something being entirely excluded e.g. due to -target.
		noop := &plans.ResourceInstanceChange{
			Addr:        absAddr,
			PrevRunAddr: n.prevRunAddr(ctx),
			DeposedKey:  deposedKey,
			Change: plans.Change{
				Action: plans.NoOp,
				Before: cty.NullVal(cty.DynamicPseudoType),
				After:  cty.NullVal(cty.DynamicPseudoType),
			},
			ProviderAddr: n.ResolvedProvider,
		}
		return noop, deferred, nil
	}

	// If we are in a context where we forget instead of destroying, we can
	// just return the forget change without consulting the provider.
	if ctx.Forget() {
		forget, diags := n.planForget(ctx, currentState, deposedKey)
		return forget, deferred, diags
	}

	unmarkedPriorVal, _ := currentState.Value.UnmarkDeep()

	// The config and new value are null to signify that this is a destroy
	// operation.
	nullVal := cty.NullVal(unmarkedPriorVal.Type())

	provider, _, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return plan, deferred, diags.Append(err)
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return plan, deferred, diags
	}

	var resp providers.PlanResourceChangeResponse
	if n.override != nil {
		// If we have an overridden value from the test framework, that means
		// this value was created without consulting the provider previously.
		// We can just set the planned state to deleted without consulting the
		// provider.
		resp = providers.PlanResourceChangeResponse{
			PlannedState: nullVal,
		}
	} else {
		// Allow the provider to check the destroy plan, and insert any
		// necessary private data.
		resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:           n.Addr.Resource.Resource.Type,
			Config:             nullVal,
			PriorState:         unmarkedPriorVal,
			ProposedNewState:   nullVal,
			PriorPrivate:       currentState.Private,
			ProviderMeta:       metaConfigVal,
			ClientCapabilities: ctx.ClientCapabilities(),
		})
		deferred = resp.Deferred

		// If we don't support deferrals, but the provider reports a deferral and does not
		// emit any error level diagnostics, we should emit an error.
		if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
			diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
		}

		// We may not have a config for all destroys, but we want to reference
		// it in the diagnostics if we do.
		if n.Config != nil {
			resp.Diagnostics = resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String())
		}
		diags = diags.Append(resp.Diagnostics)
		if diags.HasErrors() {
			return plan, deferred, diags
		}

		// Check that the provider returned a null value here, since that is the
		// only valid value for a destroy plan.
		if !resp.PlannedState.IsNull() && deferred == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid plan",
				fmt.Sprintf(
					"Provider %q planned a non-null destroy value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider.Provider, n.Addr),
			),
			)
			return plan, deferred, diags
		}
	}

	// Plan is always the same for a destroy.
	plan = &plans.ResourceInstanceChange{
		Addr:        absAddr,
		PrevRunAddr: n.prevRunAddr(ctx),
		DeposedKey:  deposedKey,
		Change: plans.Change{
			Action: plans.Delete,
			Before: currentState.Value,
			After:  nullVal,
		},
		Private:      resp.PlannedPrivate,
		ProviderAddr: n.ResolvedProvider,
	}

	return plan, deferred, diags
}

// planForget returns a Forget change.
func (n *NodeAbstractResourceInstance) planForget(ctx EvalContext, currentState *states.ResourceInstanceObject, deposedKey states.DeposedKey) (*plans.ResourceInstanceChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var plan *plans.ResourceInstanceChange

	absAddr := n.Addr

	// If there is no state or our attributes object is null then the resource
	// is already removed.
	if currentState == nil || currentState.Value.IsNull() {
		// We still need to generate a NoOp change, because that allows
		// outside consumers of the plan to distinguish between us affirming
		// that we checked something and concluded no changes were needed
		// vs. that something being entirely excluded e.g. due to -target.
		noop := &plans.ResourceInstanceChange{
			Addr:        absAddr,
			PrevRunAddr: n.prevRunAddr(ctx),
			DeposedKey:  deposedKey,
			Change: plans.Change{
				Action: plans.NoOp,
				Before: cty.NullVal(cty.DynamicPseudoType),
				After:  cty.NullVal(cty.DynamicPseudoType),
			},
			ProviderAddr: n.ResolvedProvider,
		}
		return noop, nil
	}

	unmarkedPriorVal, _ := currentState.Value.UnmarkDeep()
	nullVal := cty.NullVal(unmarkedPriorVal.Type())

	// Plan is always the same for a forget.
	plan = &plans.ResourceInstanceChange{
		Addr:        absAddr,
		PrevRunAddr: n.prevRunAddr(ctx),
		DeposedKey:  deposedKey,
		Change: plans.Change{
			Action: plans.Forget,
			Before: currentState.Value,
			After:  nullVal,
		},
		ProviderAddr: n.ResolvedProvider,
	}

	return plan, diags
}

// writeChange saves a planned change for an instance object into the set of
// global planned changes.
func (n *NodeAbstractResourceInstance) writeChange(ctx EvalContext, change *plans.ResourceInstanceChange, deposedKey states.DeposedKey) error {
	changes := ctx.Changes()

	if change == nil {
		// Caller sets nil to indicate that we need to remove a change from
		// the set of changes.
		changes.RemoveResourceInstanceChange(n.Addr, deposedKey)
		return nil
	}

	if change.Addr.String() != n.Addr.String() || change.DeposedKey != deposedKey {
		// Should never happen, and indicates a bug in the caller.
		panic("inconsistent address and/or deposed key in writeChange")
	}
	if change.PrevRunAddr.Resource.Resource.Type == "" {
		// Should never happen, and indicates a bug in the caller.
		// (The change.Encode function actually has its own fixup to just
		// quietly make this match change.Addr in the incorrect case, but we
		// intentionally panic here in order to catch incorrect callers where
		// the stack trace will hopefully be actually useful. The tolerance
		// at the next layer down is mainly to accommodate sloppy input in
		// older tests.)
		panic("unpopulated ResourceInstanceChange.PrevRunAddr in writeChange")
	}

	changes.AppendResourceInstanceChange(change)
	if deposedKey == states.NotDeposed {
		log.Printf("[TRACE] writeChange: recorded %s change for %s", change.Action, n.Addr)
	} else {
		log.Printf("[TRACE] writeChange: recorded %s change for %s deposed object %s", change.Action, n.Addr, deposedKey)
	}

	return nil
}

// refresh does a refresh for a resource
// if the second return value is non-nil, the refresh is deferred
func (n *NodeAbstractResourceInstance) refresh(ctx EvalContext, deposedKey states.DeposedKey, state *states.ResourceInstanceObject, deferralAllowed bool) (*states.ResourceInstanceObject, *providers.Deferred, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var deferred *providers.Deferred
	absAddr := n.Addr
	if deposedKey == states.NotDeposed {
		log.Printf("[TRACE] NodeAbstractResourceInstance.refresh for %s", absAddr)
	} else {
		log.Printf("[TRACE] NodeAbstractResourceInstance.refresh for %s (deposed object %s)", absAddr, deposedKey)
	}
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return state, deferred, diags.Append(err)
	}
	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", absAddr)
		return state, deferred, diags
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.Resource.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Resource.Type))
		return state, deferred, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return state, deferred, diags
	}

	// Call pre-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(n.HookResourceIdentity(), deposedKey, state.Value)
	}))
	if diags.HasErrors() {
		return state, deferred, diags
	}

	// Refresh!
	priorVal := state.Value

	// Unmarked before sending to provider
	var priorMarks []cty.PathValueMarks
	priorVal, priorMarks = priorVal.UnmarkDeepWithPaths()

	var resp providers.ReadResourceResponse
	if n.override != nil {
		// If we have an override set for this resource, we don't want to talk
		// to the provider so we'll just return whatever was in state.
		resp = providers.ReadResourceResponse{
			NewState: priorVal,
		}
	} else {
		resp = provider.ReadResource(providers.ReadResourceRequest{
			TypeName:           n.Addr.Resource.Resource.Type,
			PriorState:         priorVal,
			Private:            state.Private,
			ProviderMeta:       metaConfigVal,
			ClientCapabilities: ctx.ClientCapabilities(),
		})

		// If we don't support deferrals, but the provider reports a deferral and does not
		// emit any error level diagnostics, we should emit an error.
		if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
			diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
		}

		if resp.Deferred != nil {
			deferred = resp.Deferred
		}
	}
	if n.Config != nil {
		resp.Diagnostics = resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String())
	}

	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return state, deferred, diags
	}

	if resp.NewState == cty.NilVal {
		// This ought not to happen in real cases since it's not possible to
		// send NilVal over the plugin RPC channel, but it can come up in
		// tests due to sloppy mocking.
		panic("new state is cty.NilVal")
	}

	// If we have deferred the refresh, we expect the new state not to be wholly known
	// and callers should be prepared to handle this.
	if !resp.NewState.IsWhollyKnown() && deferred == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s during refresh: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider.Provider, absAddr, "The returned state contains unknown values",
			),
		))
	}

	for _, err := range resp.NewState.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s during refresh: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider.Provider, absAddr, tfdiags.FormatError(err),
			),
		))
	}
	if diags.HasErrors() {
		return state, deferred, diags
	}

	// Providers are supposed to return null values for all write-only attributes
	writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
		"Provider produced invalid object",
		func(path cty.Path) string {
			return fmt.Sprintf(
				"Provider %q returned a value for the write-only attribute \"%s%s\" during refresh. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider, n.Addr, tfdiags.FormatCtyPath(path),
			)
		},
		resp.NewState,
		schema,
	)
	diags = diags.Append(writeOnlyDiags)

	if writeOnlyDiags.HasErrors() {
		return state, deferred, diags
	}

	newState := objchange.NormalizeObjectFromLegacySDK(resp.NewState, schema)
	if !newState.RawEquals(resp.NewState) {
		// We had to fix up this object in some way, and we still need to
		// accept any changes for compatibility, so all we can do is log a
		// warning about the change.
		log.Printf("[WARN] Provider %q produced an invalid new value containing null blocks for %q during refresh\n", n.ResolvedProvider.Provider, n.Addr)
	}

	ret := state.DeepCopy()
	ret.Value = newState
	ret.Private = resp.Private

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(n.HookResourceIdentity(), deposedKey, priorVal, ret.Value)
	}))
	if diags.HasErrors() {
		return ret, deferred, diags
	}

	// Mark the value with any prior marks from the state, and the marks from
	// the schema. This ensures we capture any marks from the last
	// configuration, as well as any marks from the schema which were not in
	// the prior state. New marks may appear when the prior state was from an
	// import operation, or if the provider added new marks to the schema.
	ret.Value = ret.Value.MarkWithPaths(priorMarks)
	if moreSensitivePaths := schema.SensitivePaths(ret.Value, nil); len(moreSensitivePaths) != 0 {
		ret.Value = marks.MarkPaths(ret.Value, marks.Sensitive, moreSensitivePaths)
	}

	return ret, deferred, diags
}

func (n *NodeAbstractResourceInstance) plan(
	ctx EvalContext,
	plannedChange *plans.ResourceInstanceChange,
	currentState *states.ResourceInstanceObject,
	createBeforeDestroy bool,
	forceReplace []addrs.AbsResourceInstance,
) (*plans.ResourceInstanceChange, *states.ResourceInstanceObject, *providers.Deferred, instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var keyData instances.RepetitionData
	var deferred *providers.Deferred

	resource := n.Addr.Resource.Resource
	deferralAllowed := ctx.Deferrals().DeferralAllowed()

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, nil, deferred, keyData, diags.Append(err)
	}

	schema, _ := providerSchema.SchemaForResourceAddr(resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", resource.Type))
		return nil, nil, deferred, keyData, diags
	}

	// If we're importing and generating config, generate it now.
	if n.Config == nil {
		// This shouldn't happen. A node that isn't generating config should
		// have embedded config, and the rest of Terraform should enforce this.
		// If, however, we didn't do things correctly the next line will panic,
		// so let's not do that and return an error message with more context.

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource has no configuration",
			fmt.Sprintf("Terraform attempted to process a resource at %s that has no configuration. This is a bug in Terraform; please report it!", n.Addr.String())))
		return nil, nil, deferred, keyData, diags
	}

	config := *n.Config

	checkRuleSeverity := tfdiags.Error
	if n.preDestroyRefresh {
		checkRuleSeverity = tfdiags.Warning
	}

	if plannedChange != nil {
		// If we already planned the action, we stick to that plan
		createBeforeDestroy = plannedChange.Action == plans.CreateThenDelete
	}

	// Evaluate the configuration
	forEach, _, _ := evaluateForEachExpression(n.Config.ForEach, ctx, false)

	keyData = EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

	checkDiags := evalCheckRules(
		addrs.ResourcePrecondition,
		n.Config.Preconditions,
		ctx, n.Addr, keyData,
		checkRuleSeverity,
	)
	diags = diags.Append(checkDiags)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags // failed preconditions prevent further evaluation
	}

	// If we have a previous plan and the action was a noop, then the only
	// reason we're in this method was to evaluate the preconditions. There's
	// no need to re-plan this resource.
	if plannedChange != nil && plannedChange.Action == plans.NoOp {
		return plannedChange, currentState.DeepCopy(), deferred, keyData, diags
	}

	origConfigVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	diags = diags.Append(
		validateResourceForbiddenEphemeralValues(ctx, origConfigVal, schema).InConfigBody(n.Config.Config, n.Addr.String()),
	)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	var priorVal cty.Value
	var priorValTainted cty.Value
	var priorPrivate []byte
	if currentState != nil {
		if currentState.Status != states.ObjectTainted {
			priorVal = currentState.Value
			priorPrivate = currentState.Private
		} else {
			// If the prior state is tainted then we'll proceed below like
			// we're creating an entirely new object, but then turn it into
			// a synthetic "Replace" change at the end, creating the same
			// result as if the provider had marked at least one argument
			// change as "requires replacement".
			priorValTainted = currentState.Value
			priorVal = cty.NullVal(schema.ImpliedType())
		}
	} else {
		priorVal = cty.NullVal(schema.ImpliedType())
	}

	log.Printf("[TRACE] Re-validating config for %q", n.Addr)
	// Allow the provider to validate the final set of values.  The config was
	// statically validated early on, but there may have been unknown values
	// which the provider could not validate at the time.
	//
	// TODO: It would be more correct to validate the config after
	// ignore_changes has been applied, but the current implementation cannot
	// exclude computed-only attributes when given the `all` option.

	// we must unmark and use the original config, since the ignore_changes
	// handling below needs access to the marks.
	unmarkedConfigVal, _ := origConfigVal.UnmarkDeep()
	validateResp := provider.ValidateResourceConfig(
		providers.ValidateResourceConfigRequest{
			TypeName:           n.Addr.Resource.Resource.Type,
			Config:             unmarkedConfigVal,
			ClientCapabilities: ctx.ClientCapabilities(),
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	// ignore_changes is meant to only apply to the configuration, so it must
	// be applied before we generate a plan. This ensures the config used for
	// the proposed value, the proposed value itself, and the config presented
	// to the provider in the PlanResourceChange request all agree on the
	// starting values.
	// Here we operate on the marked values, so as to revert any changes to the
	// marks as well as the value.
	configValIgnored, ignoreChangeDiags := n.processIgnoreChanges(priorVal, origConfigVal, schema)
	diags = diags.Append(ignoreChangeDiags)
	if ignoreChangeDiags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	// Create an unmarked version of our config val and our prior val.
	// Store the paths for the config val to re-mark after we've sent things
	// over the wire.
	unmarkedConfigVal, unmarkedPaths := configValIgnored.UnmarkDeepWithPaths()
	unmarkedPriorVal, _ := priorVal.UnmarkDeepWithPaths()

	proposedNewVal := objchange.ProposedNew(schema, unmarkedPriorVal, unmarkedConfigVal)

	// Call pre-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(n.HookResourceIdentity(), addrs.NotDeposed, priorVal, proposedNewVal)
	}))
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	var resp providers.PlanResourceChangeResponse
	if n.override != nil {
		// Then we have an override to apply for this change. But, overrides
		// only matter when we are creating a resource for the first time as we
		// only apply computed values.
		if priorVal.IsNull() {
			// Then we are actually creating something, so let's populate the
			// computed values from our override value.
			override, overrideDiags := mocking.PlanComputedValuesForResource(proposedNewVal, &mocking.MockedData{
				Value:             n.override.Values,
				Range:             n.override.Range,
				ComputedAsUnknown: !n.override.UseForPlan,
			}, schema)
			resp = providers.PlanResourceChangeResponse{
				PlannedState: override,
				Diagnostics:  overrideDiags,
			}
		} else {
			// This is an update operation, and we don't actually have any
			// computed values that need to be applied.
			resp = providers.PlanResourceChangeResponse{
				PlannedState: proposedNewVal,
			}
		}
	} else {
		resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:           n.Addr.Resource.Resource.Type,
			Config:             unmarkedConfigVal,
			PriorState:         unmarkedPriorVal,
			ProposedNewState:   proposedNewVal,
			PriorPrivate:       priorPrivate,
			ProviderMeta:       metaConfigVal,
			ClientCapabilities: ctx.ClientCapabilities(),
		})
		// If we don't support deferrals, but the provider reports a deferral and does not
		// emit any error level diagnostics, we should emit an error.
		if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
			diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
		}
	}
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	// We mark this node as deferred at a later point when we know the complete change
	if resp.Deferred != nil {
		deferred = resp.Deferred
	}

	plannedNewVal := resp.PlannedState
	plannedPrivate := resp.PlannedPrivate

	// These checks are only relevant if the provider is not deferring the
	// change.
	if deferred == nil {
		if plannedNewVal == cty.NilVal {
			// Should never happen. Since real-world providers return via RPC a nil
			// is always a bug in the client-side stub. This is more likely caused
			// by an incompletely-configured mock provider in tests, though.
			panic(fmt.Sprintf("PlanResourceChange of %s produced nil value", n.Addr))
		}

		// Providers are supposed to return null values for all write-only attributes
		writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
			"Provider produced invalid plan",
			func(path cty.Path) string {
				return fmt.Sprintf(
					"Provider %q returned a value for the write-only attribute \"%s%s\" during planning. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, n.Addr, tfdiags.FormatCtyPath(path),
				)
			},
			plannedNewVal,
			schema,
		)
		diags = diags.Append(writeOnlyDiags)

		if writeOnlyDiags.HasErrors() {
			return nil, nil, deferred, keyData, diags
		}

		// We allow the planned new value to disagree with configuration _values_
		// here, since that allows the provider to do special logic like a
		// DiffSuppressFunc, but we still require that the provider produces
		// a value whose type conforms to the schema.
		for _, err := range plannedNewVal.Type().TestConformance(schema.ImpliedType()) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid plan",
				fmt.Sprintf(
					"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, n.Addr.String()),
				),
			))
		}

		if diags.HasErrors() {
			return nil, nil, deferred, keyData, diags
		}

		if errs := objchange.AssertPlanValid(schema, unmarkedPriorVal, unmarkedConfigVal, plannedNewVal); len(errs) > 0 {
			if resp.LegacyTypeSystem {
				// The shimming of the old type system in the legacy SDK is not precise
				// enough to pass this consistency check, so we'll give it a pass here,
				// but we will generate a warning about it so that we are more likely
				// to notice in the logs if an inconsistency beyond the type system
				// leads to a downstream provider failure.
				var buf strings.Builder
				fmt.Fprintf(&buf,
					"[WARN] Provider %q produced an invalid plan for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:",
					n.ResolvedProvider.Provider, n.Addr,
				)
				for _, err := range errs {
					fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
				}
				log.Print(buf.String())
			} else {
				for _, err := range errs {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Provider produced invalid plan",
						fmt.Sprintf(
							"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
							n.ResolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, n.Addr.String()),
						),
					))
				}
				return nil, nil, deferred, keyData, diags
			}
		}
	}

	if resp.LegacyTypeSystem {
		// Because we allow legacy providers to depart from the contract and
		// return changes to non-computed values, the plan response may have
		// altered values that were already suppressed with ignore_changes.
		// A prime example of this is where providers attempt to obfuscate
		// config data by turning the config value into a hash and storing the
		// hash value in the state. There are enough cases of this in existing
		// providers that we must accommodate the behavior for now, so for
		// ignore_changes to work at all on these values, we will revert the
		// ignored values once more.
		// A nil schema is passed to processIgnoreChanges to indicate that we
		// don't want to fixup a config value according to the schema when
		// ignoring "all", rather we are reverting provider imposed changes.
		plannedNewVal, ignoreChangeDiags = n.processIgnoreChanges(unmarkedPriorVal, plannedNewVal, nil)
		diags = diags.Append(ignoreChangeDiags)
		if ignoreChangeDiags.HasErrors() {
			return nil, nil, deferred, keyData, diags
		}
	}

	// Add the marks back to the planned new value -- this must happen after
	// ignore changes have been processed. We add in the schema marks as well,
	// to ensure that provider defined private attributes are marked correctly
	// here. We remove the ephemeral marks, the provider is expected to return null
	// for write-only attributes (the only place where ephemeral values are allowed).
	// This is verified in objchange.AssertPlanValid already.
	unmarkedPlannedNewVal := plannedNewVal
	_, nonEphemeralMarks := marks.PathsWithMark(unmarkedPaths, marks.Ephemeral)
	plannedNewVal = plannedNewVal.MarkWithPaths(nonEphemeralMarks)
	if sensitivePaths := schema.SensitivePaths(plannedNewVal, nil); len(sensitivePaths) != 0 {
		plannedNewVal = marks.MarkPaths(plannedNewVal, marks.Sensitive, sensitivePaths)
	}

	writeOnlyPaths := schema.WriteOnlyPaths(plannedNewVal, nil)

	reqRep, reqRepDiags := getRequiredReplaces(unmarkedPriorVal, unmarkedPlannedNewVal, writeOnlyPaths, resp.RequiresReplace, n.ResolvedProvider.Provider, n.Addr)
	diags = diags.Append(reqRepDiags)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	woPathSet := cty.NewPathSet(writeOnlyPaths...)
	action, actionReason := getAction(n.Addr, unmarkedPriorVal, unmarkedPlannedNewVal, createBeforeDestroy, woPathSet, forceReplace, reqRep)

	if action.IsReplace() {
		// In this strange situation we want to produce a change object that
		// shows our real prior object but has a _new_ object that is built
		// from a null prior object, since we're going to delete the one
		// that has all the computed values on it.
		//
		// Therefore we'll ask the provider to plan again here, giving it
		// a null object for the prior, and then we'll meld that with the
		// _actual_ prior state to produce a correctly-shaped replace change.
		// The resulting change should show any computed attributes changing
		// from known prior values to unknown values, unless the provider is
		// able to predict new values for any of these computed attributes.
		nullPriorVal := cty.NullVal(schema.ImpliedType())

		// Since there is no prior state to compare after replacement, we need
		// a new unmarked config from our original with no ignored values.
		unmarkedConfigVal := origConfigVal
		if origConfigVal.ContainsMarked() {
			unmarkedConfigVal, _ = origConfigVal.UnmarkDeep()
		}

		// create a new proposed value from the null state and the config
		proposedNewVal = objchange.ProposedNew(schema, nullPriorVal, unmarkedConfigVal)

		if n.override != nil {
			// In this case, we are always creating the resource so we don't
			// do any validation, and just call out to the mocking library.
			override, overrideDiags := mocking.PlanComputedValuesForResource(proposedNewVal, &mocking.MockedData{
				Value:             n.override.Values,
				Range:             n.override.Range,
				ComputedAsUnknown: !n.override.UseForPlan,
			}, schema)
			resp = providers.PlanResourceChangeResponse{
				PlannedState: override,
				Diagnostics:  overrideDiags,
			}
		} else {
			resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
				TypeName:           n.Addr.Resource.Resource.Type,
				Config:             unmarkedConfigVal,
				PriorState:         nullPriorVal,
				ProposedNewState:   proposedNewVal,
				PriorPrivate:       plannedPrivate,
				ProviderMeta:       metaConfigVal,
				ClientCapabilities: ctx.ClientCapabilities(),
			})

			// If we don't support deferrals, but the provider reports a deferral and does not
			// emit any error level diagnostics, we should emit an error.
			if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
				diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
			}
		}
		// We need to tread carefully here, since if there are any warnings
		// in here they probably also came out of our previous call to
		// PlanResourceChange above, and so we don't want to repeat them.
		// Consequently, we break from the usual pattern here and only
		// append these new diagnostics if there's at least one error inside.
		if resp.Diagnostics.HasErrors() {
			diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
			return nil, nil, deferred, keyData, diags
		}

		if deferred == nil && resp.Deferred != nil {
			deferred = resp.Deferred
		}

		plannedNewVal = resp.PlannedState
		plannedPrivate = resp.PlannedPrivate

		if len(nonEphemeralMarks) > 0 {
			plannedNewVal = plannedNewVal.MarkWithPaths(nonEphemeralMarks)
		}

		for _, err := range plannedNewVal.Type().TestConformance(schema.ImpliedType()) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid plan",
				fmt.Sprintf(
					"Provider %q planned an invalid value for %s%s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider.Provider, n.Addr, tfdiags.FormatError(err),
				),
			))
		}
		if diags.HasErrors() {
			return nil, nil, deferred, keyData, diags
		}

		// Providers are supposed to return null values for all write-only attributes
		writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
			"Provider produced invalid plan",
			func(path cty.Path) string {
				return fmt.Sprintf(
					"Provider %q returned a value for the write-only attribute \"%s%s\" during planning. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, n.Addr, tfdiags.FormatCtyPath(path),
				)
			},
			plannedNewVal,
			schema,
		)
		diags = diags.Append(writeOnlyDiags)

		if writeOnlyDiags.HasErrors() {
			return nil, nil, deferred, keyData, diags
		}
	}

	// If our prior value was tainted then we actually want this to appear
	// as a replace change, even though so far we've been treating it as a
	// create.
	if action == plans.Create && !priorValTainted.IsNull() {
		if createBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
		}
		priorVal = priorValTainted
		actionReason = plans.ResourceInstanceReplaceBecauseTainted
	}

	// If we plan to change the sensitivity on some portion of the value, this
	// is an Update action even when the values are otherwise equal.
	//
	// The marks should be normalized by being applied to the new value. We
	// don't try to compare the marks we had collected from the config and
	// schema, because the act of applying marks to a value may result in
	// slightly different marks. For example marks within a set transfer to the
	// entire set, and are not saved on the individual elements.
	if action == plans.NoOp && !valueMarksEqual(plannedNewVal, priorVal) {
		action = plans.Update
	}

	// As a special case, if we have a previous diff (presumably from the plan
	// phases, whereas we're now in the apply phase) and it was for a replace,
	// we've already deleted the original object from state by the time we
	// get here and so we would've ended up with a _create_ action this time,
	// which we now need to paper over to get a result consistent with what
	// we originally intended.
	if plannedChange != nil {
		prevChange := *plannedChange
		if prevChange.Action.IsReplace() && action == plans.Create {
			log.Printf("[TRACE] plan: %s treating Create change as %s change to match with earlier plan", n.Addr, prevChange.Action)
			action = prevChange.Action
			priorVal = prevChange.Before
		}
	}

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(n.HookResourceIdentity(), addrs.NotDeposed, action, priorVal, plannedNewVal)
	}))
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}

	// Update our return plan
	plan := &plans.ResourceInstanceChange{
		Addr:         n.Addr,
		PrevRunAddr:  n.prevRunAddr(ctx),
		Private:      plannedPrivate,
		ProviderAddr: n.ResolvedProvider,
		Change: plans.Change{
			Action: action,
			Before: priorVal,
			// Pass the marked planned value through in our change
			// to propogate through evaluation.
			// Marks will be removed when encoding.
			After:           plannedNewVal,
			GeneratedConfig: n.generatedConfigHCL,
		},
		ActionReason:    actionReason,
		RequiredReplace: reqRep,
	}

	// Update our return state
	state := &states.ResourceInstanceObject{
		// We use the special "planned" status here to note that this
		// object's value is not yet complete. Objects with this status
		// cannot be used during expression evaluation, so the caller
		// must _also_ record the returned change in the active plan,
		// which the expression evaluator will use in preference to this
		// incomplete value recorded in the state.
		Status:  states.ObjectPlanned,
		Value:   plannedNewVal,
		Private: plannedPrivate,
	}

	return plan, state, deferred, keyData, diags
}

func (n *NodeAbstractResource) processIgnoreChanges(prior, config cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	// ignore_changes only applies when an object already exists, since we
	// can't ignore changes to a thing we've not created yet.
	if prior.IsNull() {
		return config, nil
	}

	ignoreChanges, keys := traversalsToPaths(n.Config.Managed.IgnoreChanges)
	ignoreAll := n.Config.Managed.IgnoreAllChanges

	if len(ignoreChanges) == 0 && !ignoreAll {
		return config, nil
	}

	if ignoreAll {
		log.Printf("[TRACE] processIgnoreChanges: Ignoring all changes for %s", n.Addr)

		// Legacy providers need up to clean up their invalid plans and ensure
		// no changes are passed though, but that also means making an invalid
		// config with computed values. In that case we just don't supply a
		// schema and return the prior val directly.
		if schema == nil {
			return prior, nil
		}

		// If we are trying to ignore all attribute changes, we must filter
		// computed attributes out from the prior state to avoid sending them
		// to the provider as if they were included in the configuration.
		ret, _ := cty.Transform(prior, func(path cty.Path, v cty.Value) (cty.Value, error) {
			attr := schema.AttributeByPath(path)
			if attr != nil && attr.Computed && !attr.Optional {
				return cty.NullVal(v.Type()), nil
			}

			return v, nil
		})

		return ret, nil
	}
	log.Printf("[TRACE] processIgnoreChanges: Ignoring changes for %s at [%s]", n.Addr, strings.Join(keys, ", "))

	if prior.IsNull() || config.IsNull() {
		// Ignore changes doesn't apply when we're creating for the first time.
		// Proposed should never be null here, but if it is then we'll just let it be.
		return config, nil
	}

	ret, diags := processIgnoreChangesIndividual(prior, config, ignoreChanges)

	return ret, diags
}

// Convert the hcl.Traversal values we get form the configuration to the
// cty.Path values we need to operate on the cty.Values
func traversalsToPaths(traversals []hcl.Traversal) ([]cty.Path, []string) {
	paths := make([]cty.Path, len(traversals))
	keys := make([]string, len(traversals))
	for i, traversal := range traversals {
		path, key := traversalToPath(traversal)
		paths[i] = path
		keys[i] = key
	}
	return paths, keys
}

func traversalToPath(traversal hcl.Traversal) (cty.Path, string) {
	path := make(cty.Path, len(traversal))
	var key strings.Builder
	for si, step := range traversal {
		switch ts := step.(type) {
		case hcl.TraverseRoot:
			path[si] = cty.GetAttrStep{
				Name: ts.Name,
			}
			key.WriteString(ts.Name)
		case hcl.TraverseAttr:
			path[si] = cty.GetAttrStep{
				Name: ts.Name,
			}
			key.WriteString(".")
			key.WriteString(ts.Name)
		case hcl.TraverseIndex:
			path[si] = cty.IndexStep{
				Key: ts.Key,
			}
			if ts.Key.Type().IsPrimitiveType() {
				key.WriteString("[")
				key.WriteString(tfdiags.CompactValueStr(ts.Key))
				key.WriteString("]")
			} else {
				key.WriteString("[...]")
			}
		default:
			panic(fmt.Sprintf("unsupported traversal step %#v", step))
		}
	}
	return path, key.String()
}

func processIgnoreChangesIndividual(prior, config cty.Value, ignoreChangesPath []cty.Path) (cty.Value, tfdiags.Diagnostics) {
	type ignoreChange struct {
		// Path is the full path, minus any trailing map index
		path cty.Path
		// Value is the value we are to retain at the above path. If there is a
		// key value, this must be a map and the desired value will be at the
		// key index.
		value cty.Value
		// Key is the index key if the ignored path ends in a map index.
		key cty.Value
	}
	var ignoredValues []ignoreChange

	// Find the actual changes first and store them in the ignoreChange struct.
	// If the change was to a map value, and the key doesn't exist in the
	// config, it would never be visited in the transform walk.
	for _, icPath := range ignoreChangesPath {
		key := cty.NullVal(cty.String)
		// check for a map index, since maps are the only structure where we
		// could have invalid path steps.
		last, ok := icPath[len(icPath)-1].(cty.IndexStep)
		if ok {
			if last.Key.Type() == cty.String {
				icPath = icPath[:len(icPath)-1]
				key = last.Key
			}
		}

		// The structure should have been validated already, and we already
		// trimmed the trailing map index. Any other intermediate index error
		// means we wouldn't be able to apply the value below, so no need to
		// record this.
		p, err := icPath.Apply(prior)
		if err != nil {
			continue
		}
		c, err := icPath.Apply(config)
		if err != nil {
			continue
		}

		// If this is a map, it is checking the entire map value for equality
		// rather than the individual key. This means that the change is stored
		// here even if our ignored key doesn't change. That is OK since it
		// won't cause any changes in the transformation, but allows us to skip
		// breaking up the maps and checking for key existence here too.
		if !p.RawEquals(c) {
			// there a change to ignore at this path, store the prior value
			ignoredValues = append(ignoredValues, ignoreChange{icPath, p, key})
		}
	}

	if len(ignoredValues) == 0 {
		return config, nil
	}

	ret, _ := cty.Transform(config, func(path cty.Path, v cty.Value) (cty.Value, error) {
		// Easy path for when we are only matching the entire value. The only
		// values we break up for inspection are maps.
		if !v.Type().IsMapType() {
			for _, ignored := range ignoredValues {
				if path.Equals(ignored.path) {
					return ignored.value, nil
				}
			}
			return v, nil
		}
		// We now know this must be a map, so we need to accumulate the values
		// key-by-key.

		if !v.IsNull() && !v.IsKnown() {
			// since v is not known, we cannot ignore individual keys
			return v, nil
		}

		// The map values will remain as cty values, so we only need to store
		// the marks from the outer map itself
		v, vMarks := v.Unmark()

		// The configMap is the current configuration value, which we will
		// mutate based on the ignored paths and the prior map value.
		var configMap map[string]cty.Value
		switch {
		case v.IsNull() || v.LengthInt() == 0:
			configMap = map[string]cty.Value{}
		default:
			configMap = v.AsValueMap()
		}

		for _, ignored := range ignoredValues {
			if !path.Equals(ignored.path) {
				continue
			}

			if ignored.key.IsNull() {
				// The map address is confirmed to match at this point,
				// so if there is no key, we want the entire map and can
				// stop accumulating values.
				return ignored.value, nil
			}
			// Now we know we are ignoring a specific index of this map, so get
			// the config map and modify, add, or remove the desired key.

			// We also need to create a prior map, so we can check for
			// existence while getting the value, because Value.Index will
			// return null for a key with a null value and for a non-existent
			// key.
			var priorMap map[string]cty.Value

			// We need to drop the marks from the ignored map for handling. We
			// don't need to store these, as we now know the ignored value is
			// only within the map, not the map itself.
			ignoredVal, _ := ignored.value.Unmark()

			switch {
			case ignored.value.IsNull() || ignoredVal.LengthInt() == 0:
				priorMap = map[string]cty.Value{}
			default:
				priorMap = ignoredVal.AsValueMap()
			}

			key := ignored.key.AsString()
			priorElem, keep := priorMap[key]

			switch {
			case !keep:
				// this didn't exist in the old map value, so we're keeping the
				// "absence" of the key by removing it from the config
				delete(configMap, key)
			default:
				configMap[key] = priorElem
			}
		}

		var newVal cty.Value
		switch {
		case len(configMap) > 0:
			newVal = cty.MapVal(configMap)
		case v.IsNull():
			// if the config value was null, and no values remain in the map,
			// reset the value to null.
			newVal = v
		default:
			newVal = cty.MapValEmpty(v.Type().ElementType())
		}

		if len(vMarks) > 0 {
			newVal = newVal.WithMarks(vMarks)
		}

		return newVal, nil
	})
	return ret, nil
}

// readDataSource handles everything needed to call ReadDataSource on the provider.
// A previously evaluated configVal can be passed in, or a new one is generated
// from the resource configuration.
func (n *NodeAbstractResourceInstance) readDataSource(ctx EvalContext, configVal cty.Value) (cty.Value, *providers.Deferred, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var newVal cty.Value
	var deferred *providers.Deferred

	config := *n.Config
	deferralAllowed := ctx.Deferrals().DeferralAllowed()

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return newVal, deferred, diags
	}
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return newVal, deferred, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return newVal, deferred, diags
	}

	// Unmark before sending to provider, will re-mark before returning
	var pvm []cty.PathValueMarks
	configVal, pvm = configVal.UnmarkDeepWithPaths()

	log.Printf("[TRACE] readDataSource: Re-validating config for %s", n.Addr)
	validateResp := provider.ValidateDataResourceConfig(
		providers.ValidateDataResourceConfigRequest{
			TypeName: n.Addr.ContainingResource().Resource.Type,
			Config:   configVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return newVal, deferred, diags
	}

	// If we get down here then our configuration is complete and we're read
	// to actually call the provider to read the data.
	log.Printf("[TRACE] readDataSource: %s configuration is complete, so reading from provider", n.Addr)

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreApply(n.HookResourceIdentity(), addrs.NotDeposed, plans.Read, cty.NullVal(configVal.Type()), configVal)
	}))
	if diags.HasErrors() {
		return newVal, deferred, diags
	}

	var resp providers.ReadDataSourceResponse
	if n.override != nil {
		override, overrideDiags := mocking.ComputedValuesForDataSource(configVal, &mocking.MockedData{
			Value:             n.override.Values,
			Range:             n.override.Range,
			ComputedAsUnknown: false,
		}, schema)
		resp = providers.ReadDataSourceResponse{
			State:       override,
			Diagnostics: overrideDiags,
		}
	} else {
		resp = provider.ReadDataSource(providers.ReadDataSourceRequest{
			TypeName:           n.Addr.ContainingResource().Resource.Type,
			Config:             configVal,
			ProviderMeta:       metaConfigVal,
			ClientCapabilities: ctx.ClientCapabilities(),
		})

		// If we don't support deferrals, but the provider reports a deferral and does not
		// emit any error level diagnostics, we should emit an error.
		if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
			diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(n.Addr))
		}

		if resp.Deferred != nil {
			deferred = resp.Deferred
		}
	}
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return newVal, deferred, diags
	}
	newVal = resp.State
	if newVal == cty.NilVal {
		// This can happen with incompletely-configured mocks. We'll allow it
		// and treat it as an alias for a properly-typed null value.
		newVal = cty.NullVal(schema.ImpliedType())
	}

	// We don't want to run the checks if the data source read is deferred
	if deferred == nil {
		for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid object",
				fmt.Sprintf(
					"Provider %q produced an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, tfdiags.FormatErrorPrefixed(err, n.Addr.String()),
				),
			))
		}
		if diags.HasErrors() {
			return newVal, deferred, diags
		}

		if newVal.IsNull() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced null object",
				fmt.Sprintf(
					"Provider %q produced a null value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, n.Addr,
				),
			))
		}

		if !newVal.IsNull() && !newVal.IsWhollyKnown() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid object",
				fmt.Sprintf(
					"Provider %q produced a value for %s that is not wholly known.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider, n.Addr,
				),
			))

			// We'll still save the object, but we need to eliminate any unknown
			// values first because we can't serialize them in the state file.
			// Note that this may cause set elements to be coalesced if they
			// differed only by having unknown values, but we don't worry about
			// that here because we're saving the value only for inspection
			// purposes; the error we added above will halt the graph walk.
			newVal = cty.UnknownAsNull(newVal)
		}
	}
	newVal = newVal.MarkWithPaths(pvm)
	if sensitivePaths := schema.SensitivePaths(newVal, nil); len(sensitivePaths) != 0 {
		newVal = marks.MarkPaths(newVal, marks.Sensitive, sensitivePaths)
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostApply(n.HookResourceIdentity(), addrs.NotDeposed, newVal, diags.Err())
	}))

	return newVal, deferred, diags
}

func (n *NodeAbstractResourceInstance) providerMetas(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return metaConfigVal, diags.Append(err)
	}
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ResolvedProvider.Provider]; ok && m != nil {
			// if the provider doesn't support this feature, throw an error
			if providerSchema.ProviderMeta.Block == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ResolvedProvider.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr.Resource),
					Subject:  &m.ProviderRange,
				})
			} else {
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, providerSchema.ProviderMeta.Block, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
			}
		}
	}
	return metaConfigVal, diags
}

// planDataSource deals with the main part of the data resource lifecycle:
// either actually reading from the data source or generating a plan to do so.
//
// currentState is the current state for the data source, and the new state is
// returned. While data sources are read-only, we need to start with the prior
// state to determine if we have a change or not.  If we needed to read a new
// value, but it still matches the previous state, then we can record a NoNop
// change. If the states don't match then we record a Read change so that the
// new value is applied to the state.
//
// The cases where a data source will generate a planned change instead
// of finishing during the plan are:
//
//   - Its config has unknown values or it depends on a resource with pending changes.
//     (Note that every data source that is DeferredPrereq should also fit this description.)
//   - We attempted a read request, but the provider says we're deferred.
//   - It's nested in a check block, and should always read again during apply.
func (n *NodeAbstractResourceInstance) planDataSource(ctx EvalContext, checkRuleSeverity tfdiags.Severity, skipPlanChanges, dependencyDeferred bool) (*plans.ResourceInstanceChange, *states.ResourceInstanceObject, *providers.Deferred, instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var keyData instances.RepetitionData
	var configVal cty.Value

	var deferred *providers.Deferred
	if dependencyDeferred {
		// If a dependency of this data source was deferred, then we're going
		// to end up deferring this whatever happens. So, our default status
		// is deferred. If the provider indicates this resource should be
		// deferred for another reason, that reason should take priority over
		// this one.
		deferred = &providers.Deferred{
			Reason: providers.DeferredReasonDeferredPrereq,
		}
	}

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, nil, deferred, keyData, diags.Append(err)
	}

	config := *n.Config
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return nil, nil, deferred, keyData, diags
	}

	objTy := schema.ImpliedType()
	priorVal := cty.NullVal(objTy)

	forEach, _, _ := evaluateForEachExpression(config.ForEach, ctx, false)
	keyData = EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

	checkDiags := evalCheckRules(
		addrs.ResourcePrecondition,
		n.Config.Preconditions,
		ctx, n.Addr, keyData,
		checkRuleSeverity,
	)
	diags = diags.Append(checkDiags)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags // failed preconditions prevent further evaluation
	}

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	diags = diags.Append(
		validateResourceForbiddenEphemeralValues(ctx, configVal, schema).InConfigBody(n.Config.Config, n.Addr.String()),
	)
	if diags.HasErrors() {
		return nil, nil, deferred, keyData, diags
	}
	unmarkedConfigVal, unmarkedPaths := configVal.UnmarkDeepWithPaths()

	check, nested := n.nestedInCheckBlock()
	if nested {
		// Going forward from this point, the only reason we will fail is
		// that the data source fails to load its data. Normally, this would
		// cancel the entire plan and this error message would bubble its way
		// back up to the user.
		//
		// But, if we are in a check block then we don't want this data block to
		// cause the plan to fail. We also need to report a status on the data
		// block so the check processing later on knows whether to attempt to
		// process the checks. Either we'll report the data block as failed
		// if/when we load the data block later, or we want to report it as a
		// success overall.
		//
		// Therefore, we create a deferred function here that will check if the
		// status for the check has been updated yet, and if not we will set it
		// to be StatusPass. The rest of this function will only update the
		// status if it should be StatusFail.
		defer func() {
			status := ctx.Checks().ObjectCheckStatus(check.Addr().Absolute(n.Addr.Module))
			if status == checks.StatusUnknown {
				ctx.Checks().ReportCheckResult(check.Addr().Absolute(n.Addr.Module), addrs.CheckDataResource, 0, checks.StatusPass)
			}
		}()
	}

	configKnown := configVal.IsWhollyKnown()
	depsPending := n.dependenciesHavePendingChanges(ctx)
	// If our configuration contains any unknown values, or we depend on any
	// unknown values then we must defer the read to the apply phase by
	// producing a "Read" change for this resource, and a placeholder value for
	// it in the state.
	if depsPending || !configKnown {
		// We can't plan any changes if we're only refreshing, so the only
		// value we can set here is whatever was in state previously.
		if skipPlanChanges {
			plannedNewState := &states.ResourceInstanceObject{
				Value:  priorVal,
				Status: states.ObjectReady,
			}

			return nil, plannedNewState, deferred, keyData, diags
		}

		var reason plans.ResourceInstanceChangeActionReason
		switch {
		case !configKnown:
			log.Printf("[TRACE] planDataSource: %s configuration not fully known yet, so deferring to apply phase", n.Addr)
			reason = plans.ResourceInstanceReadBecauseConfigUnknown
		case depsPending:
			// NOTE: depsPending can be true at the same time as configKnown
			// is false; configKnown takes precedence because it's more
			// specific.
			log.Printf("[TRACE] planDataSource: %s configuration is fully known, at least one dependency has changes pending", n.Addr)
			reason = plans.ResourceInstanceReadBecauseDependencyPending
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, unmarkedConfigVal)

		// even though we are only returning the config value because we can't
		// yet read the data source, we need to incorporate the schema marks so
		// that downstream consumers can detect them when planning.
		proposedNewVal = proposedNewVal.MarkWithPaths(unmarkedPaths)
		if sensitivePaths := schema.SensitivePaths(proposedNewVal, nil); len(sensitivePaths) != 0 {
			proposedNewVal = marks.MarkPaths(proposedNewVal, marks.Sensitive, sensitivePaths)
		}

		// Apply detects that the data source will need to be read by the After
		// value containing unknowns from PlanDataResourceObject.
		plannedChange := &plans.ResourceInstanceChange{
			Addr:         n.Addr,
			PrevRunAddr:  n.prevRunAddr(ctx),
			ProviderAddr: n.ResolvedProvider,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  proposedNewVal,
			},
			ActionReason: reason,
		}

		plannedNewState := &states.ResourceInstanceObject{
			Value:  proposedNewVal,
			Status: states.ObjectPlanned,
		}

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(n.HookResourceIdentity(), addrs.NotDeposed, plans.Read, priorVal, proposedNewVal)
		}))

		return plannedChange, plannedNewState, deferred, keyData, diags
	}

	// We have a complete configuration with no dependencies to wait on, so we
	// can read the data source into the state.
	// newVal is fully marked by the readDataSource method.
	newVal, readDeferred, readDiags := n.readDataSource(ctx, configVal)

	if readDeferred != nil {
		// This will either be null or a value that indicates we're deferred
		// because of a dependency. In both cases we're happy to just overwrite
		// that with the more relevant information directly from the provider.
		deferred = readDeferred
	}

	// Now we've loaded the data, and diags tells us whether we were successful
	// or not, we are going to create our plannedChange and our
	// proposedNewState.
	var plannedChange *plans.ResourceInstanceChange
	var plannedNewState *states.ResourceInstanceObject

	// If we are a nested block, then we want to create a plannedChange that
	// tells Terraform to reload the data block during the apply stage even if
	// we managed to get the data now.
	// Another consideration is that if we failed to load the data, we need to
	// disguise that for a nested block. Nested blocks will report the overall
	// check as failed but won't affect the rest of the plan operation or block
	// an apply operation.

	if nested {
		addr := check.Addr().Absolute(n.Addr.Module)

		// Let's fix things up for a nested data block.
		//
		// A nested data block doesn't error, and creates a planned change. So,
		// if we encountered an error we'll tidy up newVal so it makes sense
		// and handle the error. We'll also create the plannedChange if
		// appropriate.

		if readDiags.HasErrors() {
			// If we had errors, then we can cover that up by marking the new
			// state as unknown.
			newVal = objchange.PlannedDataResourceObject(schema, unmarkedConfigVal)

			// not only do we want to ensure this synthetic value has the marks,
			// but since this is the value being returned from the data source
			// we need to ensure the schema marks are added as well.
			newVal = newVal.MarkWithPaths(unmarkedPaths)
			if sensitivePaths := schema.SensitivePaths(newVal, nil); len(sensitivePaths) != 0 {
				newVal = marks.MarkPaths(newVal, marks.Sensitive, sensitivePaths)
			}

			// We still want to report the check as failed even if we are still
			// letting it run again during the apply stage.
			ctx.Checks().ReportCheckFailure(addr, addrs.CheckDataResource, 0, readDiags.Err().Error())
		}

		// Any warning or error diagnostics we'll wrap with some special checks
		// diagnostics. This is so we can identify them later, and so they'll
		// only report as warnings.
		readDiags = tfdiags.OverrideAll(readDiags, tfdiags.Warning, func() tfdiags.DiagnosticExtraWrapper {
			return &addrs.CheckRuleDiagnosticExtra{
				CheckRule: addrs.NewCheckRule(addr, addrs.CheckDataResource, 0),
			}
		})

		// refreshOnly plans cannot produce planned changes, so we only do
		// this if skipPlanChanges is false. Conversely, provider-deferred data
		// sources always generate a planned change with a different ActionReason.
		if !skipPlanChanges && deferred == nil {
			plannedChange = &plans.ResourceInstanceChange{
				Addr:         n.Addr,
				PrevRunAddr:  n.prevRunAddr(ctx),
				ProviderAddr: n.ResolvedProvider,
				Change: plans.Change{
					Action: plans.Read,
					Before: priorVal,
					After:  newVal,
				},
				ActionReason: plans.ResourceInstanceReadBecauseCheckNested,
			}
		}
	}

	// Provider-deferred data sources always generate a planned change.
	if deferred != nil {
		plannedChange = &plans.ResourceInstanceChange{
			Addr:         n.Addr,
			PrevRunAddr:  n.prevRunAddr(ctx),
			ProviderAddr: n.ResolvedProvider,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  newVal,
			},
			// The caller should be more interested in the deferral reason, but this
			// action reason is a reasonable description of what's happening.
			ActionReason: plans.ResourceInstanceReadBecauseDependencyPending,
		}

		plannedNewState = &states.ResourceInstanceObject{
			Value:  newVal,
			Status: states.ObjectPlanned,
		}
	}

	diags = diags.Append(readDiags)
	if !diags.HasErrors() && deferred == nil {
		// Finally, let's make our new state.
		plannedNewState = &states.ResourceInstanceObject{
			Value:  newVal,
			Status: states.ObjectReady,
		}
	}

	return plannedChange, plannedNewState, deferred, keyData, diags
}

// nestedInCheckBlock determines if this resource is nested in a Check config
// block. If so, this resource will be loaded during both plan and apply
// operations to make sure the check is always giving the latest information.
func (n *NodeAbstractResourceInstance) nestedInCheckBlock() (*configs.Check, bool) {
	if n.Config.Container != nil {
		check, ok := n.Config.Container.(*configs.Check)
		return check, ok
	}
	return nil, false
}

// dependenciesHavePendingChanges determines whether any managed resource the
// receiver depends on has a change pending in the plan, in which case we'd
// need to override the usual behavior of immediately reading from the data
// source where possible, and instead defer the read until the apply step.
func (n *NodeAbstractResourceInstance) dependenciesHavePendingChanges(ctx EvalContext) bool {
	nModInst := n.Addr.Module
	nMod := nModInst.Module()

	// Check and see if any depends_on dependencies have
	// changes, since they won't show up as changes in the
	// configuration.
	changes := ctx.Changes()

	depsToUse := n.dependsOn

	if n.Addr.Resource.Resource.Mode == addrs.DataResourceMode {
		if n.Config.HasCustomConditions() {
			// For a data resource with custom conditions we need to look at
			// the full set of resource dependencies -- both direct and
			// indirect -- because an upstream update might be what's needed
			// in order to make a condition pass.
			depsToUse = n.Dependencies
		}
	}

	for _, d := range depsToUse {
		if d.Resource.Mode == addrs.DataResourceMode {
			// Data sources have no external side effects, so they pose a need
			// to delay this read. If they do have a change planned, it must be
			// because of a dependency on a managed resource, in which case
			// we'll also encounter it in this list of dependencies.
			continue
		}

		for _, change := range changes.GetChangesForConfigResource(d) {
			changeModInst := change.Addr.Module
			changeMod := changeModInst.Module()

			if changeMod.Equal(nMod) && !changeModInst.Equal(nModInst) {
				// Dependencies are tracked by configuration address, which
				// means we may have changes from other instances of parent
				// modules. The actual reference can only take effect within
				// the same module instance, so skip any that aren't an exact
				// match
				continue
			}

			if change != nil && change.Action != plans.NoOp {
				return true
			}
		}
	}
	return false
}

// apply deals with the main part of the data resource lifecycle: either
// actually reading from the data source or generating a plan to do so.
func (n *NodeAbstractResourceInstance) applyDataSource(ctx EvalContext, planned *plans.ResourceInstanceChange) (*states.ResourceInstanceObject, instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var keyData instances.RepetitionData

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, keyData, diags.Append(err)
	}
	if planned != nil && planned.Action != plans.Read && planned.Action != plans.NoOp {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		diags = diags.Append(fmt.Errorf(
			"invalid action %s for %s: only Read is supported (this is a bug in Terraform; please report it!)",
			planned.Action, n.Addr,
		))
		return nil, keyData, diags
	}

	config := *n.Config
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return nil, keyData, diags
	}

	forEach, _, _ := evaluateForEachExpression(config.ForEach, ctx, false)
	keyData = EvalDataForInstanceKey(n.Addr.Resource.Key, forEach)

	checkDiags := evalCheckRules(
		addrs.ResourcePrecondition,
		n.Config.Preconditions,
		ctx, n.Addr, keyData,
		tfdiags.Error,
	)
	diags = diags.Append(checkDiags)
	if diags.HasErrors() {
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(n.HookResourceIdentity(), addrs.NotDeposed, planned.Before, diags.Err())
		}))
		return nil, keyData, diags // failed preconditions prevent further evaluation
	}

	if planned.Action == plans.NoOp {
		// If we didn't actually plan to read this then we have nothing more
		// to do; we're evaluating this only for incidentals like the
		// precondition/postcondition checks.
		return nil, keyData, diags
	}

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, keyData, diags
	}

	newVal, readDeferred, readDiags := n.readDataSource(ctx, configVal)
	if check, nested := n.nestedInCheckBlock(); nested {
		addr := check.Addr().Absolute(n.Addr.Module)

		// We're just going to jump in here and hide away any errors for nested
		// data blocks.
		if readDiags.HasErrors() {
			ctx.Checks().ReportCheckFailure(addr, addrs.CheckDataResource, 0, readDiags.Err().Error())
			diags = diags.Append(tfdiags.OverrideAll(readDiags, tfdiags.Warning, func() tfdiags.DiagnosticExtraWrapper {
				return &addrs.CheckRuleDiagnosticExtra{
					CheckRule: addrs.NewCheckRule(addr, addrs.CheckDataResource, 0),
				}
			}))
			return nil, keyData, diags
		}

		// Even though we know there are no errors here, we still want to
		// identify these diags has having been generated from a check block.
		readDiags = tfdiags.OverrideAll(readDiags, tfdiags.Warning, func() tfdiags.DiagnosticExtraWrapper {
			return &addrs.CheckRuleDiagnosticExtra{
				CheckRule: addrs.NewCheckRule(addr, addrs.CheckDataResource, 0),
			}
		})

		// If no errors, just remember to report this as a success and continue
		// as normal.
		ctx.Checks().ReportCheckResult(addr, addrs.CheckDataResource, 0, checks.StatusPass)
	}

	diags = diags.Append(readDiags)
	if readDiags.HasErrors() {
		return nil, keyData, diags
	}

	if readDeferred != nil {
		// Just skip data sources that are being deferred. Nothing, that
		// references them should be calling them.
		return nil, keyData, diags
	}

	state := &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	return state, keyData, diags
}

// evalApplyProvisioners determines if provisioners need to be run, and if so
// executes the provisioners for a resource and returns an updated error if
// provisioning fails.
func (n *NodeAbstractResourceInstance) evalApplyProvisioners(ctx EvalContext, state *states.ResourceInstanceObject, createNew bool, when configs.ProvisionerWhen) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if state == nil {
		log.Printf("[TRACE] evalApplyProvisioners: %s has no state, so skipping provisioners", n.Addr)
		return nil
	}
	if when == configs.ProvisionerWhenCreate && !createNew {
		// If we're not creating a new resource, then don't run provisioners
		log.Printf("[TRACE] evalApplyProvisioners: %s is not freshly-created, so no provisioning is required", n.Addr)
		return nil
	}
	if state.Status == states.ObjectTainted {
		// No point in provisioning an object that is already tainted, since
		// it's going to get recreated on the next apply anyway.
		log.Printf("[TRACE] evalApplyProvisioners: %s is tainted, so skipping provisioning", n.Addr)
		return nil
	}

	var allProvs []*configs.Provisioner
	switch {
	case n.Config != nil && n.Config.Managed != nil:
		allProvs = n.Config.Managed.Provisioners
	case n.RemovedConfig != nil && n.RemovedConfig.Managed != nil:
		allProvs = n.RemovedConfig.Managed.Provisioners
	}

	provs := filterProvisioners(allProvs, when)
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil
	}

	// Call pre hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreProvisionInstance(n.HookResourceIdentity(), state.Value)
	}))
	if diags.HasErrors() {
		return diags
	}

	// If there are no errors, then we append it to our output error
	// if we have one, otherwise we just output it.
	diags = diags.Append(n.applyProvisioners(ctx, state, when, provs))
	if diags.HasErrors() {
		log.Printf("[TRACE] evalApplyProvisioners: %s provisioning failed, but we will continue anyway at the caller's request", n.Addr)
		return diags
	}

	// Call post hook
	return diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostProvisionInstance(n.HookResourceIdentity(), state.Value)
	}))
}

// filterProvisioners filters the provisioners on the resource to only
// the provisioners specified by the "when" option.
func filterProvisioners(configured []*configs.Provisioner, when configs.ProvisionerWhen) []*configs.Provisioner {
	if len(configured) == 0 {
		return nil
	}

	result := make([]*configs.Provisioner, 0, len(configured))
	for _, p := range configured {
		if p.When == when {
			result = append(result, p)
		}
	}

	return result
}

// applyProvisioners executes the provisioners for a resource.
func (n *NodeAbstractResourceInstance) applyProvisioners(ctx EvalContext, state *states.ResourceInstanceObject, when configs.ProvisionerWhen, provs []*configs.Provisioner) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// this self is only used for destroy provisioner evaluation, and must
	// refer to the last known value of the resource.
	self := state.Value

	var evalScope func(EvalContext, hcl.Body, cty.Value, *configschema.Block) (cty.Value, tfdiags.Diagnostics)
	switch when {
	case configs.ProvisionerWhenDestroy:
		evalScope = n.evalDestroyProvisionerConfig
	default:
		evalScope = n.evalProvisionerConfig
	}

	// If there's a connection block defined directly inside the resource block
	// then it'll serve as a base connection configuration for all of the
	// provisioners.
	var baseConn hcl.Body
	switch {
	case n.Config != nil && n.Config.Managed != nil && n.Config.Managed.Connection != nil:
		baseConn = n.Config.Managed.Connection.Config
	case n.RemovedConfig != nil && n.RemovedConfig.Managed != nil && n.RemovedConfig.Managed.Connection != nil:
		baseConn = n.RemovedConfig.Managed.Connection.Config
	}

	for _, prov := range provs {
		log.Printf("[TRACE] applyProvisioners: provisioning %s with %q", n.Addr, prov.Type)

		// Get the provisioner
		provisioner, err := ctx.Provisioner(prov.Type)
		if err != nil {
			return diags.Append(err)
		}

		schema, err := ctx.ProvisionerSchema(prov.Type)
		if err != nil {
			// This error probably won't be a great diagnostic, but in practice
			// we typically catch this problem long before we get here, so
			// it should be rare to return via this codepath.
			diags = diags.Append(err)
			return diags
		}

		config, configDiags := evalScope(ctx, prov.Config, self, schema)
		diags = diags.Append(configDiags)
		if diags.HasErrors() {
			return diags
		}

		// If the provisioner block contains a connection block of its own then
		// it can override the base connection configuration, if any.
		var localConn hcl.Body
		if prov.Connection != nil {
			localConn = prov.Connection.Config
		}

		var connBody hcl.Body
		switch {
		case baseConn != nil && localConn != nil:
			// Our standard merging logic applies here, similar to what we do
			// with _override.tf configuration files: arguments from the
			// base connection block will be masked by any arguments of the
			// same name in the local connection block.
			connBody = configs.MergeBodies(baseConn, localConn)
		case baseConn != nil:
			connBody = baseConn
		case localConn != nil:
			connBody = localConn
		}

		// start with an empty connInfo
		connInfo := cty.NullVal(connectionBlockSupersetSchema.ImpliedType())

		if connBody != nil {
			var connInfoDiags tfdiags.Diagnostics
			connInfo, connInfoDiags = evalScope(ctx, connBody, self, connectionBlockSupersetSchema)
			diags = diags.Append(connInfoDiags)
			if diags.HasErrors() {
				return diags
			}
		}

		{
			// Call pre hook
			err := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PreProvisionInstanceStep(n.HookResourceIdentity(), prov.Type)
			})
			if err != nil {
				return diags.Append(err)
			}
		}

		// The output function
		outputFn := func(msg string) {
			ctx.Hook(func(h Hook) (HookAction, error) {
				h.ProvisionOutput(n.HookResourceIdentity(), prov.Type, msg)
				return HookActionContinue, nil
			})
		}

		// If our config or connection info contains any marked values, ensure
		// those are stripped out before sending to the provisioner. Unlike
		// resources, we have no need to capture the marked paths and reapply
		// later.
		unmarkedConfig, configMarks := config.UnmarkDeep()
		unmarkedConnInfo, _ := connInfo.UnmarkDeep()

		// Marks on the config might result in leaking sensitive values through
		// provisioner logging, so we conservatively suppress all output in
		// this case. This should not apply to connection info values, which
		// provisioners ought not to be logging anyway.
		if len(configMarks) > 0 {
			outputFn = func(msg string) {
				ctx.Hook(func(h Hook) (HookAction, error) {
					h.ProvisionOutput(n.HookResourceIdentity(), prov.Type, "(output suppressed due to sensitive value in config)")
					return HookActionContinue, nil
				})
			}
		}

		output := CallbackUIOutput{OutputFn: outputFn}
		resp := provisioner.ProvisionResource(provisioners.ProvisionResourceRequest{
			Config:     unmarkedConfig,
			Connection: unmarkedConnInfo,
			UIOutput:   &output,
		})
		applyDiags := resp.Diagnostics.InConfigBody(prov.Config, n.Addr.String())

		// Call post hook
		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostProvisionInstanceStep(n.HookResourceIdentity(), prov.Type, applyDiags.Err())
		})

		switch prov.OnFailure {
		case configs.ProvisionerOnFailureContinue:
			if applyDiags.HasErrors() {
				log.Printf("[WARN] Errors while provisioning %s with %q, but continuing as requested in configuration", n.Addr, prov.Type)
			} else {
				// Maybe there are warnings that we still want to see
				diags = diags.Append(applyDiags)
			}
		default:
			diags = diags.Append(applyDiags)
			if applyDiags.HasErrors() {
				log.Printf("[WARN] Errors while provisioning %s with %q, so aborting", n.Addr, prov.Type)
				return diags
			}
		}

		// Deal with the hook
		if hookErr != nil {
			return diags.Append(hookErr)
		}
	}

	return diags
}

func (n *NodeAbstractResourceInstance) evalProvisionerConfig(ctx EvalContext, body hcl.Body, self cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	forEach, _, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx, false)
	diags = diags.Append(forEachDiags)

	keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

	config, _, configDiags := ctx.EvaluateBlock(body, schema, n.ResourceInstanceAddr().Resource, keyData)
	diags = diags.Append(configDiags)

	return config, diags
}

// during destroy a provisioner can only evaluate within the scope of the parent resource
func (n *NodeAbstractResourceInstance) evalDestroyProvisionerConfig(ctx EvalContext, body hcl.Body, self cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// For a destroy-time provisioner forEach is intentionally nil here,
	// which EvalDataForInstanceKey responds to by not populating EachValue
	// in its result. That's okay because each.value is prohibited for
	// destroy-time provisioners.
	keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, nil)

	evalScope := ctx.EvaluationScope(n.ResourceInstanceAddr().Resource, nil, keyData)
	config, evalDiags := evalScope.EvalSelfBlock(body, self, schema, keyData)
	diags = diags.Append(evalDiags)

	return config, diags
}

// apply accepts an applyConfig, instead of using n.Config, so destroy plans can
// send a nil config. The keyData information can be empty if the config is
// nil, since it is only used to evaluate the configuration.
func (n *NodeAbstractResourceInstance) apply(
	ctx EvalContext,
	state *states.ResourceInstanceObject,
	change *plans.ResourceInstanceChange,
	applyConfig *configs.Resource,
	keyData instances.RepetitionData,
	createBeforeDestroy bool) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {

	var diags tfdiags.Diagnostics
	if state == nil {
		state = &states.ResourceInstanceObject{}
	}

	if change.Action == plans.NoOp {
		// If this is a no-op change then we don't want to actually change
		// anything, so we'll just echo back the state we were given and
		// let our internal checks and updates proceed.
		log.Printf("[TRACE] NodeAbstractResourceInstance.apply: skipping %s because it has no planned action", n.Addr)
		return state, diags
	}

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, diags.Append(err)
	}
	schema, _ := providerSchema.SchemaForResourceType(n.Addr.Resource.Resource.Mode, n.Addr.Resource.Resource.Type)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Resource.Type))
		return nil, diags
	}

	log.Printf("[INFO] Starting apply for %s", n.Addr)

	configVal := cty.NullVal(cty.DynamicPseudoType)
	if applyConfig != nil {
		var configDiags tfdiags.Diagnostics
		configVal, _, configDiags = ctx.EvaluateBlock(applyConfig.Config, schema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags
		}
	}

	if !configVal.IsWhollyKnown() {
		// We don't have a pretty format function for a path, but since this is
		// such a rare error, we can just drop the raw GoString values in here
		// to make sure we have something to debug with.
		var unknownPaths []string
		cty.Transform(configVal, func(p cty.Path, v cty.Value) (cty.Value, error) {
			if !v.IsKnown() {
				unknownPaths = append(unknownPaths, fmt.Sprintf("%#v", p))
			}
			return v, nil
		})

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Configuration contains unknown value",
			fmt.Sprintf("configuration for %s still contains unknown values during apply (this is a bug in Terraform; please report it!)\n"+
				"The following paths in the resource configuration are unknown:\n%s",
				n.Addr,
				strings.Join(unknownPaths, "\n"),
			),
		))
		return nil, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	log.Printf("[DEBUG] %s: applying the planned %s change", n.Addr, change.Action)

	// If our config, Before or After value contain any marked values,
	// ensure those are stripped out before sending
	// this to the provider
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	unmarkedBefore, beforePaths := change.Before.UnmarkDeepWithPaths()
	unmarkedAfter, afterPaths := change.After.UnmarkDeepWithPaths()

	// If we have an Update action, our before and after values are equal,
	// and only differ on their sensitivity, the newVal is the after val
	// and we should not communicate with the provider. We do need to update
	// the state with this new value, to ensure the sensitivity change is
	// persisted.
	eqV := unmarkedBefore.Equals(unmarkedAfter)
	eq := eqV.IsKnown() && eqV.True()
	if change.Action == plans.Update && eq && !marks.MarksEqual(beforePaths, afterPaths) {
		// Copy the previous state, changing only the value
		newState := &states.ResourceInstanceObject{
			CreateBeforeDestroy: state.CreateBeforeDestroy,
			Dependencies:        state.Dependencies,
			Private:             state.Private,
			Status:              state.Status,
			Value:               change.After,
		}
		return newState, diags
	}

	var resp providers.ApplyResourceChangeResponse
	if n.override != nil {
		// As with the planning stage, we only need to worry about computed
		// values the first time the object is created. Otherwise, we're happy
		// to just apply whatever the user asked for.
		if change.Action == plans.Create {
			override, overrideDiags := mocking.ApplyComputedValuesForResource(unmarkedAfter, &mocking.MockedData{
				Value:             n.override.Values,
				Range:             n.override.Range,
				ComputedAsUnknown: false,
			}, schema)
			resp = providers.ApplyResourceChangeResponse{
				NewState:    override,
				Diagnostics: overrideDiags,
			}
		} else {
			resp = providers.ApplyResourceChangeResponse{
				NewState: unmarkedAfter,
			}
		}
	} else {
		resp = provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
			TypeName:       n.Addr.Resource.Resource.Type,
			PriorState:     unmarkedBefore,
			Config:         unmarkedConfigVal,
			PlannedState:   unmarkedAfter,
			PlannedPrivate: change.Private,
			ProviderMeta:   metaConfigVal,
		})
	}
	applyDiags := resp.Diagnostics
	if applyConfig != nil {
		applyDiags = applyDiags.InConfigBody(applyConfig.Config, n.Addr.String())
	}
	diags = diags.Append(applyDiags)

	// Even if there are errors in the returned diagnostics, the provider may
	// have returned a _partial_ state for an object that already exists but
	// failed to fully configure, and so the remaining code must always run
	// to completion but must be defensive against the new value being
	// incomplete.
	newVal := resp.NewState

	if newVal == cty.NilVal {
		// Providers are supposed to return a partial new value even when errors
		// occur, but sometimes they don't and so in that case we'll patch that up
		// by just using the prior state, so we'll at least keep track of the
		// object for the user to retry.
		newVal = change.Before

		// As a special case, we'll set the new value to null if it looks like
		// we were trying to execute a delete, because the provider in this case
		// probably left the newVal unset intending it to be interpreted as "null".
		if change.After.IsNull() {
			newVal = cty.NullVal(schema.ImpliedType())
		}

		if !diags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid object",
				fmt.Sprintf(
					"Provider %q produced an invalid nil value after apply for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ResolvedProvider.String(), n.Addr.String(),
				),
			))
		}
	}

	var conformDiags tfdiags.Diagnostics
	for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
		conformDiags = conformDiags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q produced an invalid value after apply for %s. The result cannot not be saved in the Terraform state.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider.String(), tfdiags.FormatErrorPrefixed(err, n.Addr.String()),
			),
		))
	}
	diags = diags.Append(conformDiags)
	if conformDiags.HasErrors() {
		// Bail early in this particular case, because an object that doesn't
		// conform to the schema can't be saved in the state anyway -- the
		// serializer will reject it.
		return nil, diags
	}

	// Providers are supposed to return null values for all write-only attributes
	writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
		"Provider produced invalid object",
		func(path cty.Path) string {
			return fmt.Sprintf(
				"Provider %q returned a value for the write-only attribute \"%s%s\" after apply. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider, n.Addr, tfdiags.FormatCtyPath(path),
			)
		},
		newVal,
		schema,
	)
	diags = diags.Append(writeOnlyDiags)

	if writeOnlyDiags.HasErrors() {
		return nil, diags
	}

	// After this point we have a type-conforming result object and so we
	// must always run to completion to ensure it can be saved. If n.Error
	// is set then we must not return a non-nil error, in order to allow
	// evaluation to continue to a later point where our state object will
	// be saved.

	// By this point there must not be any unknown values remaining in our
	// object, because we've applied the change and we can't save unknowns
	// in our persistent state. If any are present then we will indicate an
	// error (which is always a bug in the provider) but we will also replace
	// them with nulls so that we can successfully save the portions of the
	// returned value that are known.
	if !newVal.IsWhollyKnown() {
		// To generate better error messages, we'll go for a walk through the
		// value and make a separate diagnostic for each unknown value we
		// find.
		cty.Walk(newVal, func(path cty.Path, val cty.Value) (bool, error) {
			if !val.IsKnown() {
				pathStr := tfdiags.FormatCtyPath(path)
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider returned invalid result object after apply",
					fmt.Sprintf(
						"After the apply operation, the provider still indicated an unknown value for %s%s. All values must be known after apply, so this is always a bug in the provider and should be reported in the provider's own repository. Terraform will still save the other known object values in the state.",
						n.Addr, pathStr,
					),
				))
			}
			return true, nil
		})

		// NOTE: This operation can potentially be lossy if there are multiple
		// elements in a set that differ only by unknown values: after
		// replacing with null these will be merged together into a single set
		// element. Since we can only get here in the presence of a provider
		// bug, we accept this because storing a result here is always a
		// best-effort sort of thing.
		newVal = cty.UnknownAsNull(newVal)
	}

	// If we have paths to mark, mark those on this new value we need to
	// re-check the value against the schema, because nested computed values
	// won't be included in afterPaths, which are only what was read from the
	// After plan value.
	newVal = newVal.MarkWithPaths(afterPaths)
	if sensitivePaths := schema.SensitivePaths(newVal, nil); len(sensitivePaths) != 0 {
		newVal = marks.MarkPaths(newVal, marks.Sensitive, sensitivePaths)
	}

	if change.Action != plans.Delete && !diags.HasErrors() {
		// Only values that were marked as unknown in the planned value are allowed
		// to change during the apply operation. (We do this after the unknown-ness
		// check above so that we also catch anything that became unknown after
		// being known during plan.)
		//
		// If we are returning other errors anyway then we'll give this
		// a pass since the other errors are usually the explanation for
		// this one and so it's more helpful to let the user focus on the
		// root cause rather than distract with this extra problem.
		if errs := objchange.AssertObjectCompatible(schema, change.After, newVal); len(errs) > 0 {
			if resp.LegacyTypeSystem {
				// The shimming of the old type system in the legacy SDK is not precise
				// enough to pass this consistency check, so we'll give it a pass here,
				// but we will generate a warning about it so that we are more likely
				// to notice in the logs if an inconsistency beyond the type system
				// leads to a downstream provider failure.
				var buf strings.Builder
				fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:", n.ResolvedProvider.String(), n.Addr)
				for _, err := range errs {
					fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
				}
				log.Print(buf.String())

				// The sort of inconsistency we won't catch here is if a known value
				// in the plan is changed during apply. That can cause downstream
				// problems because a dependent resource would make its own plan based
				// on the planned value, and thus get a different result during the
				// apply phase. This will usually lead to a "Provider produced invalid plan"
				// error that incorrectly blames the downstream resource for the change.

			} else {
				for _, err := range errs {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Provider produced inconsistent result after apply",
						fmt.Sprintf(
							"When applying changes to %s, provider %q produced an unexpected new value: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
							n.Addr, n.ResolvedProvider.String(), tfdiags.FormatError(err),
						),
					))
				}
			}
		}
	}

	// If a provider returns a null or non-null object at the wrong time then
	// we still want to save that but it often causes some confusing behaviors
	// where it seems like Terraform is failing to take any action at all,
	// so we'll generate some errors to draw attention to it.
	if !diags.HasErrors() {
		if change.Action == plans.Delete && !newVal.IsNull() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider returned invalid result object after apply",
				fmt.Sprintf(
					"After applying a %s plan, the provider returned a non-null object for %s. Destroying should always produce a null value, so this is always a bug in the provider and should be reported in the provider's own repository. Terraform will still save this errant object in the state for debugging and recovery.",
					change.Action, n.Addr,
				),
			))
		}
		if change.Action != plans.Delete && newVal.IsNull() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider returned invalid result object after apply",
				fmt.Sprintf(
					"After applying a %s plan, the provider returned a null object for %s. Only destroying should always produce a null value, so this is always a bug in the provider and should be reported in the provider's own repository.",
					change.Action, n.Addr,
				),
			))
		}
	}

	switch {
	case diags.HasErrors() && newVal.IsNull():
		// Sometimes providers return a null value when an operation fails for
		// some reason, but we'd rather keep the prior state so that the error
		// can be corrected on a subsequent run. We must only do this for null
		// new value though, or else we may discard partial updates the
		// provider was able to complete. Otherwise, we'll continue using the
		// prior state as the new value, making this effectively a no-op.  If
		// the item really _has_ been deleted then our next refresh will detect
		// that and fix it up.
		return state.DeepCopy(), diags

	case diags.HasErrors() && !newVal.IsNull():
		// if we have an error, make sure we restore the object status in the new state
		newState := &states.ResourceInstanceObject{
			Status:              state.Status,
			Value:               newVal,
			Private:             resp.Private,
			CreateBeforeDestroy: createBeforeDestroy,
		}

		// if the resource was being deleted, the dependencies are not going to
		// be recalculated and we need to restore those as well.
		if change.Action == plans.Delete {
			newState.Dependencies = state.Dependencies
		}

		return newState, diags

	case !newVal.IsNull():
		// Non error case with a new state
		newState := &states.ResourceInstanceObject{
			Status:              states.ObjectReady,
			Value:               newVal,
			Private:             resp.Private,
			CreateBeforeDestroy: createBeforeDestroy,
		}
		return newState, diags

	default:
		// Non error case, were the object was deleted
		return nil, diags
	}
}

func (n *NodeAbstractResourceInstance) prevRunAddr(ctx EvalContext) addrs.AbsResourceInstance {
	return resourceInstancePrevRunAddr(ctx, n.Addr)
}

func resourceInstancePrevRunAddr(ctx EvalContext, currentAddr addrs.AbsResourceInstance) addrs.AbsResourceInstance {
	table := ctx.MoveResults()
	return table.OldAddr(currentAddr)
}

func getAction(addr addrs.AbsResourceInstance, priorVal, plannedNewVal cty.Value, createBeforeDestroy bool, writeOnly cty.PathSet, forceReplace []addrs.AbsResourceInstance, reqRep cty.PathSet) (action plans.Action, actionReason plans.ResourceInstanceChangeActionReason) {
	// The user might also ask us to force replacing a particular resource
	// instance, regardless of whether the provider thinks it needs replacing.
	// For example, users typically do this if they learn a particular object
	// has become degraded in an immutable infrastructure scenario and so
	// replacing it with a new object is a viable repair path.
	matchedForceReplace := false
	for _, candidateAddr := range forceReplace {
		if candidateAddr.Equal(addr) {
			matchedForceReplace = true
			break
		}

		// For "force replace" purposes we require an exact resource instance
		// address to match. If a user forgets to include the instance key
		// for a multi-instance resource then it won't match here, but we
		// have an earlier check in NodePlannableResource.Execute that should
		// prevent us from getting here in that case.
	}

	// Unmark for this test for value equality.
	eqV := plannedNewVal.Equals(priorVal)
	eq := eqV.IsKnown() && eqV.True()

	switch {
	case priorVal.IsNull():
		action = plans.Create
	case matchedForceReplace || !reqRep.Empty() || !writeOnly.Intersection(reqRep).Empty():
		// If the user "forced replace" of this instance of if there are any
		// "requires replace" paths left _after our filtering above_ then this
		// is a replace action.
		if createBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
		}
		switch {
		case matchedForceReplace:
			actionReason = plans.ResourceInstanceReplaceByRequest
		case !reqRep.Empty():
			actionReason = plans.ResourceInstanceReplaceBecauseCannotUpdate
		}
	case eq && !matchedForceReplace:
		action = plans.NoOp
	default:
		action = plans.Update
		// "Delete" is never chosen here, because deletion plans are always
		// created more directly elsewhere, such as in "orphan" handling.
	}

	return
}

// getRequiredReplaces returns a list of paths to attributes whose changes mean
// that we must replace rather than update an existing remote object.
//
// The provider produces a list of paths to attributes whose changes mean
// that we must replace rather than update an existing remote object.
// However, we only need to do that if the identified attributes _have_
// actually changed -- particularly after we may have undone some of the
// changes in processIgnoreChanges -- so now we'll filter that list to
// include only where changes are detected.
//
// Both the priorVal and plannedNewVal should be unmarked before calling this
// function. This function exposes nothing about the priorVal or plannedVal
// except for the paths that require replacement which can be deduced from the
// type with or without marks.
func getRequiredReplaces(priorVal, plannedNewVal cty.Value, writeOnly []cty.Path, requiredReplaces []cty.Path, providerAddr tfaddr.Provider, addr addrs.AbsResourceInstance) (cty.PathSet, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	reqRep := cty.NewPathSet()
	if len(requiredReplaces) > 0 {
		for _, path := range requiredReplaces {
			if priorVal.IsNull() {
				// If prior is null then we don't expect any RequiresReplace at all,
				// because this is a Create action.
				continue
			}

			priorChangedVal, priorPathDiags := hcl.ApplyPath(priorVal, path, nil)
			plannedChangedVal, plannedPathDiags := hcl.ApplyPath(plannedNewVal, path, nil)
			if plannedPathDiags.HasErrors() && priorPathDiags.HasErrors() {
				// This means the path was invalid in both the prior and new
				// values, which is an error with the provider itself.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider produced invalid plan",
					fmt.Sprintf(
						"Provider %q has indicated \"requires replacement\" on %s for a non-existent attribute path %#v.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
						providerAddr, addr, path,
					),
				))
				continue
			}

			// Make sure we have valid Values for both values.
			// Note: if the opposing value was of the type
			// cty.DynamicPseudoType, the type assigned here may not exactly
			// match the schema. This is fine here, since we're only going to
			// check for equality, but if the NullVal is to be used, we need to
			// check the schema for th true type.
			switch {
			case priorChangedVal == cty.NilVal && plannedChangedVal == cty.NilVal:
				// this should never happen without ApplyPath errors above
				panic("requires replace path returned 2 nil values")
			case priorChangedVal == cty.NilVal:
				priorChangedVal = cty.NullVal(plannedChangedVal.Type())
			case plannedChangedVal == cty.NilVal:
				plannedChangedVal = cty.NullVal(priorChangedVal.Type())
			}

			eqV := plannedChangedVal.Equals(priorChangedVal)

			// if attribute/path is writeOnly we have no values to compare
			// but still respect the required replacement
			isWriteOnly := false
			for _, woPath := range writeOnly {
				if path.Equals(woPath) {
					isWriteOnly = true
				}
			}
			if !eqV.IsKnown() || eqV.False() || isWriteOnly {
				reqRep.Add(path)
			}
		}
		if diags.HasErrors() {
			return reqRep, diags
		}
	}

	return reqRep, diags
}
