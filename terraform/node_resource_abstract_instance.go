package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	// storedProviderConfig is the provider address retrieved from the
	// state, but since it is only stored in the whole Resource rather than the
	// ResourceInstance, we extract it out here.
	storedProviderConfig addrs.AbsProviderConfig

	Dependencies []addrs.ConfigResource
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

// StateDependencies returns the dependencies saved in the state.
func (n *NodeAbstractResourceInstance) StateDependencies() []addrs.ConfigResource {
	if s := n.instanceState; s != nil {
		if s.Current != nil {
			return s.Current.Dependencies
		}
	}

	return nil
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResourceInstance) ProvidedBy() (addrs.ProviderConfig, bool) {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		relAddr := n.Config.ProviderConfigAddr()
		return addrs.LocalProviderConfig{
			LocalName: relAddr.LocalName,
			Alias:     relAddr.Alias,
		}, false
	}

	// See if we have a valid provider config from the state.
	if n.storedProviderConfig.Provider.Type != "" {
		// An address from the state must match exactly, since we must ensure
		// we refresh/destroy a resource with the same provider configuration
		// that created it.
		return n.storedProviderConfig, true
	}

	// No provider configuration found; return a default address
	return addrs.AbsProviderConfig{
		Provider: n.Provider(),
		Module:   n.ModulePath(),
	}, false
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResourceInstance) Provider() addrs.Provider {
	if n.Config != nil {
		return n.Config.Provider
	}
	return addrs.ImpliedProviderForUnqualifiedType(n.Addr.Resource.ContainingResource().ImpliedProvider())
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
	n.instanceState = s.Instance(n.Addr.Resource.Key)
	n.storedProviderConfig = s.ProviderConfig
}

// readDiff returns the planned change for a particular resource instance
// object.
func (n *NodeAbstractResourceInstance) readDiff(ctx EvalContext, providerSchema *ProviderSchema) (*plans.ResourceInstanceChange, error) {
	changes := ctx.Changes()
	addr := n.ResourceInstanceAddr()

	schema, _ := providerSchema.SchemaForResourceAddr(addr.Resource.Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", addr.Resource.Resource.Type)
	}

	gen := states.CurrentGen
	csrc := changes.GetResourceInstanceChange(addr, gen)
	if csrc == nil {
		log.Printf("[TRACE] EvalReadDiff: No planned change recorded for %s", n.Addr)
		return nil, nil
	}

	change, err := csrc.Decode(schema.ImpliedType())
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned changes for %s: %s", n.Addr, err)
	}

	log.Printf("[TRACE] EvalReadDiff: Read %s change from plan for %s", change.Action, n.Addr)

	return change, nil
}

func (n *NodeAbstractResourceInstance) checkPreventDestroy(change *plans.ResourceInstanceChange) error {
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
				"Resource %s has lifecycle.prevent_destroy set, but the plan calls for this resource to be destroyed. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or reduce the scope of the plan using the -target flag.",
				n.Addr.String(),
			),
			Subject: &n.Config.DeclRange,
		})
		return diags.Err()
	}

	return nil
}

// PreApplyHook calls the pre-Apply hook
func (n *NodeAbstractResourceInstance) PreApplyHook(ctx EvalContext, change *plans.ResourceInstanceChange) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if change == nil {
		panic(fmt.Sprintf("PreApplyHook for %s called with nil Change", n.Addr))
	}

	if resourceHasUserVisibleApply(n.Addr.Resource) {
		priorState := change.Before
		plannedNewState := change.After

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(n.Addr, nil, change.Action, priorState, plannedNewState)
		}))
		if diags.HasErrors() {
			return diags
		}
	}

	return nil
}

// PostApplyHook calls the post-Apply hook
func (n *NodeAbstractResourceInstance) PostApplyHook(ctx EvalContext, state *states.ResourceInstanceObject, err *error) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if resourceHasUserVisibleApply(n.Addr.Resource) {
		var newState cty.Value
		if state != nil {
			newState = state.Value
		} else {
			newState = cty.NullVal(cty.DynamicPseudoType)
		}
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(n.Addr, nil, newState, *err)
		}))
	}

	diags = diags.Append(*err)

	return diags
}

type phaseState int

const (
	workingState phaseState = iota
	refreshState
)

// writeResourceInstanceState saves the given object as the current object for
// the selected resource instance.
//
// dependencies is a parameter, instead of those directly attacted to the
// NodeAbstractResourceInstance, because we don't write dependencies for
// datasources.
//
// targetState determines which context state we're writing to during plan. The
// default is the global working state.
func (n *NodeAbstractResourceInstance) writeResourceInstanceState(ctx EvalContext, obj *states.ResourceInstanceObject, dependencies []addrs.ConfigResource, targetState phaseState) error {
	absAddr := n.Addr
	_, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	var state *states.SyncState
	switch targetState {
	case refreshState:
		log.Printf("[TRACE] writeResourceInstanceState: using RefreshState for %s", absAddr)
		state = ctx.RefreshState()
	default:
		state = ctx.State()
	}

	if obj == nil || obj.Value.IsNull() {
		// No need to encode anything: we'll just write it directly.
		state.SetResourceInstanceCurrent(absAddr, nil, n.ResolvedProvider)
		log.Printf("[TRACE] writeResourceInstanceState: removing state object for %s", absAddr)
		return nil
	}

	// store the new deps in the state.
	// We check for nil here because don't want to override existing dependencies on orphaned nodes.
	if dependencies != nil {
		obj.Dependencies = dependencies
	}

	if providerSchema == nil {
		// Should never happen, unless our state object is nil
		panic("writeResourceInstanceState used with nil ProviderSchema")
	}

	if obj != nil {
		log.Printf("[TRACE] writeResourceInstanceState: writing current state object for %s", absAddr)
	} else {
		log.Printf("[TRACE] writeResourceInstanceState: removing current state object for %s", absAddr)
	}

	schema, currentVersion := (*providerSchema).SchemaForResourceAddr(absAddr.ContainingResource().Resource)
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

	state.SetResourceInstanceCurrent(absAddr, src, n.ResolvedProvider)
	return nil
}

// PlanDestroy returns a plain destroy diff.
func (n *NodeAbstractResourceInstance) PlanDestroy(ctx EvalContext, currentState *states.ResourceInstanceObject, deposedKey states.DeposedKey) (*plans.ResourceInstanceChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	absAddr := n.Addr

	if n.ResolvedProvider.Provider.Type == "" {
		if deposedKey == "" {
			panic(fmt.Sprintf("DestroyPlan for %s does not have ProviderAddr set", absAddr))
		} else {
			panic(fmt.Sprintf("DestroyPlan for %s (deposed %s) does not have ProviderAddr set", absAddr, deposedKey))
		}
	}

	// If there is no state or our attributes object is null then we're already
	// destroyed.
	if currentState == nil || currentState.Value.IsNull() {
		return nil, nil
	}

	// Call pre-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(
			absAddr, deposedKey.Generation(),
			currentState.Value,
			cty.NullVal(cty.DynamicPseudoType),
		)
	}))
	if diags.HasErrors() {
		return nil, diags
	}

	// Plan is always the same for a destroy. We don't need the provider's
	// help for this one.
	plan := &plans.ResourceInstanceChange{
		Addr:       absAddr,
		DeposedKey: deposedKey,
		Change: plans.Change{
			Action: plans.Delete,
			Before: currentState.Value,
			After:  cty.NullVal(cty.DynamicPseudoType),
		},
		Private:      currentState.Private,
		ProviderAddr: n.ResolvedProvider,
	}

	// Call post-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(
			absAddr,
			deposedKey.Generation(),
			plan.Action,
			plan.Before,
			plan.After,
		)
	}))

	return plan, diags
}

// WriteChange  saves a planned change for an instance object into the set of
// global planned changes.
func (n *NodeAbstractResourceInstance) WriteChange(ctx EvalContext, change *plans.ResourceInstanceChange, deposedKey states.DeposedKey) error {
	changes := ctx.Changes()

	if change == nil {
		// Caller sets nil to indicate that we need to remove a change from
		// the set of changes.
		gen := states.CurrentGen
		if deposedKey != states.NotDeposed {
			gen = deposedKey
		}
		changes.RemoveResourceInstanceChange(n.Addr, gen)
		return nil
	}

	_, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	if change.Addr.String() != n.Addr.String() || change.DeposedKey != deposedKey {
		// Should never happen, and indicates a bug in the caller.
		panic("inconsistent address and/or deposed key in WriteChange")
	}

	ri := n.Addr.Resource
	schema, _ := providerSchema.SchemaForResourceAddr(ri.Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return fmt.Errorf("provider does not support resource type %q", ri.Resource.Type)
	}

	csrc, err := change.Encode(schema.ImpliedType())
	if err != nil {
		return fmt.Errorf("failed to encode planned changes for %s: %s", n.Addr, err)
	}

	changes.AppendResourceInstanceChange(csrc)
	if deposedKey == states.NotDeposed {
		log.Printf("[TRACE] WriteChange: recorded %s change for %s", change.Action, n.Addr)
	} else {
		log.Printf("[TRACE] WriteChange: recorded %s change for %s deposed object %s", change.Action, n.Addr, deposedKey)
	}

	return nil
}

// refresh does a refresh for a resource
func (n *NodeAbstractResourceInstance) refresh(ctx EvalContext, state *states.ResourceInstanceObject) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	absAddr := n.Addr
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return state, diags.Append(err)
	}
	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, so not refreshing", absAddr)
		return state, diags
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.Resource.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Resource.Type))
		return state, diags
	}

	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ResolvedProvider.Provider]; ok && m != nil {
			log.Printf("[DEBUG] EvalRefresh: ProviderMeta config value set")
			// if the provider doesn't support this feature, throw an error
			if providerSchema.ProviderMeta == nil {
				log.Printf("[DEBUG] EvalRefresh: no ProviderMeta schema")
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ResolvedProvider.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr.Resource),
					Subject:  &m.ProviderRange,
				})
			} else {
				log.Printf("[DEBUG] EvalRefresh: ProviderMeta schema found: %+v", providerSchema.ProviderMeta)
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, providerSchema.ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
				if configDiags.HasErrors() {
					return state, diags
				}
			}
		}
	}

	// Call pre-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, state.Value)
	}))
	if diags.HasErrors() {
		return state, diags
	}

	// Refresh!
	priorVal := state.Value

	// Unmarked before sending to provider
	var priorPaths []cty.PathValueMarks
	if priorVal.ContainsMarked() {
		priorVal, priorPaths = priorVal.UnmarkDeepWithPaths()
	}

	providerReq := providers.ReadResourceRequest{
		TypeName:     n.Addr.Resource.Resource.Type,
		PriorState:   priorVal,
		Private:      state.Private,
		ProviderMeta: metaConfigVal,
	}

	resp := provider.ReadResource(providerReq)
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return state, diags
	}

	if resp.NewState == cty.NilVal {
		// This ought not to happen in real cases since it's not possible to
		// send NilVal over the plugin RPC channel, but it can come up in
		// tests due to sloppy mocking.
		panic("new state is cty.NilVal")
	}

	for _, err := range resp.NewState.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s during refresh: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ResolvedProvider.Provider.String(), absAddr, tfdiags.FormatError(err),
			),
		))
	}
	if diags.HasErrors() {
		return state, diags
	}

	// We have no way to exempt provider using the legacy SDK from this check,
	// so we can only log inconsistencies with the updated state values.
	// In most cases these are not errors anyway, and represent "drift" from
	// external changes which will be handled by the subsequent plan.
	if errs := objchange.AssertObjectCompatible(schema, priorVal, resp.NewState); len(errs) > 0 {
		var buf strings.Builder
		fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s during refresh.", n.ResolvedProvider.Provider.String(), absAddr)
		for _, err := range errs {
			fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
		}
		log.Print(buf.String())
	}

	ret := state.DeepCopy()
	ret.Value = resp.NewState
	ret.Private = resp.Private
	ret.Dependencies = state.Dependencies
	ret.CreateBeforeDestroy = state.CreateBeforeDestroy

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, ret.Value)
	}))
	if diags.HasErrors() {
		return ret, diags
	}

	// Mark the value if necessary
	if len(priorPaths) > 0 {
		ret.Value = ret.Value.MarkWithPaths(priorPaths)
	}

	return ret, diags
}
