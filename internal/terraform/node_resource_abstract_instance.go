package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/objchange"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
		log.Printf("[TRACE] readDiff: No planned change recorded for %s", n.Addr)
		return nil, nil
	}

	change, err := csrc.Decode(schema.ImpliedType())
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned changes for %s: %s", n.Addr, err)
	}

	log.Printf("[TRACE] readDiff: Read %s change from plan for %s", change.Action, n.Addr)

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
			return h.PreApply(n.Addr, change.DeposedKey.Generation(), change.Action, priorState, plannedNewState)
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
			return h.PostApply(n.Addr, nil, newState, err)
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

	if providerSchema == nil {
		// Should never happen, unless our state object is nil
		panic("writeResourceInstanceStateImpl used with nil ProviderSchema")
	}

	if obj != nil {
		log.Printf("[TRACE] %s: writing state object for %s", logFuncName, absAddr)
	} else {
		log.Printf("[TRACE] %s: removing state object for %s", logFuncName, absAddr)
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

	write(src)
	return nil
}

// planDestroy returns a plain destroy diff.
func (n *NodeAbstractResourceInstance) planDestroy(ctx EvalContext, currentState *states.ResourceInstanceObject, deposedKey states.DeposedKey) (*plans.ResourceInstanceChange, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	absAddr := n.Addr

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
			Addr:       absAddr,
			DeposedKey: deposedKey,
			Change: plans.Change{
				Action: plans.NoOp,
				Before: cty.NullVal(cty.DynamicPseudoType),
				After:  cty.NullVal(cty.DynamicPseudoType),
			},
			ProviderAddr: n.ResolvedProvider,
		}
		return noop, nil
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

// writeChange  saves a planned change for an instance object into the set of
// global planned changes.
func (n *NodeAbstractResourceInstance) writeChange(ctx EvalContext, change *plans.ResourceInstanceChange, deposedKey states.DeposedKey) error {
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

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return err
	}

	if change.Addr.String() != n.Addr.String() || change.DeposedKey != deposedKey {
		// Should never happen, and indicates a bug in the caller.
		panic("inconsistent address and/or deposed key in writeChange")
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
		log.Printf("[TRACE] writeChange: recorded %s change for %s", change.Action, n.Addr)
	} else {
		log.Printf("[TRACE] writeChange: recorded %s change for %s deposed object %s", change.Action, n.Addr, deposedKey)
	}

	return nil
}

// refresh does a refresh for a resource
func (n *NodeAbstractResourceInstance) refresh(ctx EvalContext, deposedKey states.DeposedKey, state *states.ResourceInstanceObject) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	absAddr := n.Addr
	if deposedKey == states.NotDeposed {
		log.Printf("[TRACE] NodeAbstractResourceInstance.refresh for %s", absAddr)
	} else {
		log.Printf("[TRACE] NodeAbstractResourceInstance.refresh for %s (deposed object %s)", absAddr, deposedKey)
	}
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
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

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return state, diags
	}

	hookGen := states.CurrentGen
	if deposedKey != states.NotDeposed {
		hookGen = deposedKey
	}

	// Call pre-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, hookGen, state.Value)
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
	if n.Config != nil {
		resp.Diagnostics = resp.Diagnostics.InConfigBody(n.Config.Config, n.Addr.String())
	}

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

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, hookGen, priorVal, ret.Value)
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

func (n *NodeAbstractResourceInstance) plan(
	ctx EvalContext,
	plannedChange *plans.ResourceInstanceChange,
	currentState *states.ResourceInstanceObject,
	createBeforeDestroy bool,
	forceReplace []addrs.AbsResourceInstance) (*plans.ResourceInstanceChange, *states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var state *states.ResourceInstanceObject
	var plan *plans.ResourceInstanceChange

	config := *n.Config
	resource := n.Addr.Resource.Resource
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return plan, state, diags.Append(err)
	}

	if plannedChange != nil {
		// If we already planned the action, we stick to that plan
		createBeforeDestroy = plannedChange.Action == plans.CreateThenDelete
	}

	if providerSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema is unavailable for %s", n.Addr))
		return plan, state, diags
	}

	// Evaluate the configuration
	schema, _ := providerSchema.SchemaForResourceAddr(resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", resource.Type))
		return plan, state, diags
	}

	forEach, _ := evaluateForEachExpression(n.Config.ForEach, ctx)

	keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)
	origConfigVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return plan, state, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return plan, state, diags
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
			TypeName: n.Addr.Resource.Resource.Type,
			Config:   unmarkedConfigVal,
		},
	)
	diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return plan, state, diags
	}

	// ignore_changes is meant to only apply to the configuration, so it must
	// be applied before we generate a plan. This ensures the config used for
	// the proposed value, the proposed value itself, and the config presented
	// to the provider in the PlanResourceChange request all agree on the
	// starting values.
	// Here we operate on the marked values, so as to revert any changes to the
	// marks as well as the value.
	configValIgnored, ignoreChangeDiags := n.processIgnoreChanges(priorVal, origConfigVal)
	diags = diags.Append(ignoreChangeDiags)
	if ignoreChangeDiags.HasErrors() {
		return plan, state, diags
	}

	// Create an unmarked version of our config val and our prior val.
	// Store the paths for the config val to re-mark after we've sent things
	// over the wire.
	unmarkedConfigVal, unmarkedPaths := configValIgnored.UnmarkDeepWithPaths()
	unmarkedPriorVal, priorPaths := priorVal.UnmarkDeepWithPaths()

	proposedNewVal := objchange.ProposedNew(schema, unmarkedPriorVal, unmarkedConfigVal)

	// Call pre-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(n.Addr, states.CurrentGen, priorVal, proposedNewVal)
	}))
	if diags.HasErrors() {
		return plan, state, diags
	}

	resp := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         n.Addr.Resource.Resource.Type,
		Config:           unmarkedConfigVal,
		PriorState:       unmarkedPriorVal,
		ProposedNewState: proposedNewVal,
		PriorPrivate:     priorPrivate,
		ProviderMeta:     metaConfigVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return plan, state, diags
	}

	plannedNewVal := resp.PlannedState
	plannedPrivate := resp.PlannedPrivate

	if plannedNewVal == cty.NilVal {
		// Should never happen. Since real-world providers return via RPC a nil
		// is always a bug in the client-side stub. This is more likely caused
		// by an incompletely-configured mock provider in tests, though.
		panic(fmt.Sprintf("PlanResourceChange of %s produced nil value", n.Addr))
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
		return plan, state, diags
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
			return plan, state, diags
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
		plannedNewVal, ignoreChangeDiags = n.processIgnoreChanges(unmarkedPriorVal, plannedNewVal)
		diags = diags.Append(ignoreChangeDiags)
		if ignoreChangeDiags.HasErrors() {
			return plan, state, diags
		}
	}

	// Add the marks back to the planned new value -- this must happen after ignore changes
	// have been processed
	unmarkedPlannedNewVal := plannedNewVal
	if len(unmarkedPaths) > 0 {
		plannedNewVal = plannedNewVal.MarkWithPaths(unmarkedPaths)
	}

	// The provider produces a list of paths to attributes whose changes mean
	// that we must replace rather than update an existing remote object.
	// However, we only need to do that if the identified attributes _have_
	// actually changed -- particularly after we may have undone some of the
	// changes in processIgnoreChanges -- so now we'll filter that list to
	// include only where changes are detected.
	reqRep := cty.NewPathSet()
	if len(resp.RequiresReplace) > 0 {
		for _, path := range resp.RequiresReplace {
			if priorVal.IsNull() {
				// If prior is null then we don't expect any RequiresReplace at all,
				// because this is a Create action.
				continue
			}

			priorChangedVal, priorPathDiags := hcl.ApplyPath(unmarkedPriorVal, path, nil)
			plannedChangedVal, plannedPathDiags := hcl.ApplyPath(plannedNewVal, path, nil)
			if plannedPathDiags.HasErrors() && priorPathDiags.HasErrors() {
				// This means the path was invalid in both the prior and new
				// values, which is an error with the provider itself.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider produced invalid plan",
					fmt.Sprintf(
						"Provider %q has indicated \"requires replacement\" on %s for a non-existent attribute path %#v.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
						n.ResolvedProvider.Provider, n.Addr, path,
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

			// Unmark for this value for the equality test. If only sensitivity has changed,
			// this does not require an Update or Replace
			unmarkedPlannedChangedVal, _ := plannedChangedVal.UnmarkDeep()
			eqV := unmarkedPlannedChangedVal.Equals(priorChangedVal)
			if !eqV.IsKnown() || eqV.False() {
				reqRep.Add(path)
			}
		}
		if diags.HasErrors() {
			return plan, state, diags
		}
	}

	// The user might also ask us to force replacing a particular resource
	// instance, regardless of whether the provider thinks it needs replacing.
	// For example, users typically do this if they learn a particular object
	// has become degraded in an immutable infrastructure scenario and so
	// replacing it with a new object is a viable repair path.
	matchedForceReplace := false
	for _, candidateAddr := range forceReplace {
		if candidateAddr.Equal(n.Addr) {
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
	eqV := unmarkedPlannedNewVal.Equals(unmarkedPriorVal)
	eq := eqV.IsKnown() && eqV.True()

	var action plans.Action
	var actionReason plans.ResourceInstanceChangeActionReason
	switch {
	case priorVal.IsNull():
		action = plans.Create
	case eq && !matchedForceReplace:
		action = plans.NoOp
	case matchedForceReplace || !reqRep.Empty():
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
	default:
		action = plans.Update
		// "Delete" is never chosen here, because deletion plans are always
		// created more directly elsewhere, such as in "orphan" handling.
	}

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

		resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:         n.Addr.Resource.Resource.Type,
			Config:           unmarkedConfigVal,
			PriorState:       nullPriorVal,
			ProposedNewState: proposedNewVal,
			PriorPrivate:     plannedPrivate,
			ProviderMeta:     metaConfigVal,
		})
		// We need to tread carefully here, since if there are any warnings
		// in here they probably also came out of our previous call to
		// PlanResourceChange above, and so we don't want to repeat them.
		// Consequently, we break from the usual pattern here and only
		// append these new diagnostics if there's at least one error inside.
		if resp.Diagnostics.HasErrors() {
			diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
			return plan, state, diags
		}
		plannedNewVal = resp.PlannedState
		plannedPrivate = resp.PlannedPrivate

		if len(unmarkedPaths) > 0 {
			plannedNewVal = plannedNewVal.MarkWithPaths(unmarkedPaths)
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
			return plan, state, diags
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

	// If we plan to write or delete sensitive paths from state,
	// this is an Update action
	if action == plans.NoOp && !marksEqual(unmarkedPaths, priorPaths) {
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
		return h.PostDiff(n.Addr, states.CurrentGen, action, priorVal, plannedNewVal)
	}))
	if diags.HasErrors() {
		return plan, state, diags
	}

	// Update our return plan
	plan = &plans.ResourceInstanceChange{
		Addr:         n.Addr,
		Private:      plannedPrivate,
		ProviderAddr: n.ResolvedProvider,
		Change: plans.Change{
			Action: action,
			Before: priorVal,
			// Pass the marked planned value through in our change
			// to propogate through evaluation.
			// Marks will be removed when encoding.
			After: plannedNewVal,
		},
		ActionReason:    actionReason,
		RequiredReplace: reqRep,
	}

	// Update our return state
	state = &states.ResourceInstanceObject{
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

	return plan, state, diags
}

func (n *NodeAbstractResource) processIgnoreChanges(prior, config cty.Value) (cty.Value, tfdiags.Diagnostics) {
	// ignore_changes only applies when an object already exists, since we
	// can't ignore changes to a thing we've not created yet.
	if prior.IsNull() {
		return config, nil
	}

	ignoreChanges := traversalsToPaths(n.Config.Managed.IgnoreChanges)
	ignoreAll := n.Config.Managed.IgnoreAllChanges

	if len(ignoreChanges) == 0 && !ignoreAll {
		return config, nil
	}
	if ignoreAll {
		return prior, nil
	}
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
func traversalsToPaths(traversals []hcl.Traversal) []cty.Path {
	paths := make([]cty.Path, len(traversals))
	for i, traversal := range traversals {
		path := make(cty.Path, len(traversal))
		for si, step := range traversal {
			switch ts := step.(type) {
			case hcl.TraverseRoot:
				path[si] = cty.GetAttrStep{
					Name: ts.Name,
				}
			case hcl.TraverseAttr:
				path[si] = cty.GetAttrStep{
					Name: ts.Name,
				}
			case hcl.TraverseIndex:
				path[si] = cty.IndexStep{
					Key: ts.Key,
				}
			default:
				panic(fmt.Sprintf("unsupported traversal step %#v", step))
			}
		}
		paths[i] = path
	}
	return paths
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
		if len(configMap) == 0 {
			newVal = cty.MapValEmpty(v.Type().ElementType())
		} else {
			newVal = cty.MapVal(configMap)
		}

		if len(vMarks) > 0 {
			newVal = v.WithMarks(vMarks)
		}

		return newVal, nil
	})
	return ret, nil
}

// readDataSource handles everything needed to call ReadDataSource on the provider.
// A previously evaluated configVal can be passed in, or a new one is generated
// from the resource configuration.
func (n *NodeAbstractResourceInstance) readDataSource(ctx EvalContext, configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var newVal cty.Value

	config := *n.Config

	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return newVal, diags
	}
	if providerSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
		return newVal, diags
	}
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return newVal, diags
	}

	metaConfigVal, metaDiags := n.providerMetas(ctx)
	diags = diags.Append(metaDiags)
	if diags.HasErrors() {
		return newVal, diags
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
		return newVal, diags
	}

	// If we get down here then our configuration is complete and we're read
	// to actually call the provider to read the data.
	log.Printf("[TRACE] readDataSource: %s configuration is complete, so reading from provider", n.Addr)

	resp := provider.ReadDataSource(providers.ReadDataSourceRequest{
		TypeName:     n.Addr.ContainingResource().Resource.Type,
		Config:       configVal,
		ProviderMeta: metaConfigVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config, n.Addr.String()))
	if diags.HasErrors() {
		return newVal, diags
	}
	newVal = resp.State
	if newVal == cty.NilVal {
		// This can happen with incompletely-configured mocks. We'll allow it
		// and treat it as an alias for a properly-typed null value.
		newVal = cty.NullVal(schema.ImpliedType())
	}

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
		return newVal, diags
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

	if len(pvm) > 0 {
		newVal = newVal.MarkWithPaths(pvm)
	}

	return newVal, diags
}

func (n *NodeAbstractResourceInstance) providerMetas(ctx EvalContext) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return metaConfigVal, diags.Append(err)
	}
	if providerSchema == nil {
		return metaConfigVal, diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
	}
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ResolvedProvider.Provider]; ok && m != nil {
			// if the provider doesn't support this feature, throw an error
			if providerSchema.ProviderMeta == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ResolvedProvider.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr.Resource),
					Subject:  &m.ProviderRange,
				})
			} else {
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, providerSchema.ProviderMeta, nil, EvalDataForNoInstanceKey)
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
func (n *NodeAbstractResourceInstance) planDataSource(ctx EvalContext, currentState *states.ResourceInstanceObject) (*plans.ResourceInstanceChange, *states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var configVal cty.Value

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, nil, diags.Append(err)
	}
	if providerSchema == nil {
		return nil, nil, diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
	}

	config := *n.Config
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return nil, nil, diags
	}

	objTy := schema.ImpliedType()
	priorVal := cty.NullVal(objTy)
	if currentState != nil {
		priorVal = currentState.Value
	}

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, nil, diags
	}

	unmarkedConfigVal, configMarkPaths := configVal.UnmarkDeepWithPaths()
	// We drop marks on the values used here as the result is only
	// temporarily used for validation.
	unmarkedPriorVal, _ := priorVal.UnmarkDeep()

	configKnown := configVal.IsWhollyKnown()
	// If our configuration contains any unknown values, or we depend on any
	// unknown values then we must defer the read to the apply phase by
	// producing a "Read" change for this resource, and a placeholder value for
	// it in the state.
	if n.forcePlanReadData(ctx) || !configKnown {
		if configKnown {
			log.Printf("[TRACE] planDataSource: %s configuration is fully known, but we're forcing a read plan to be created", n.Addr)
		} else {
			log.Printf("[TRACE] planDataSource: %s configuration not fully known yet, so deferring to apply phase", n.Addr)
		}

		proposedNewVal := objchange.PlannedDataResourceObject(schema, unmarkedConfigVal)

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(n.Addr, states.CurrentGen, priorVal, proposedNewVal)
		}))
		if diags.HasErrors() {
			return nil, nil, diags
		}
		proposedNewVal = proposedNewVal.MarkWithPaths(configMarkPaths)

		// Apply detects that the data source will need to be read by the After
		// value containing unknowns from PlanDataResourceObject.
		plannedChange := &plans.ResourceInstanceChange{
			Addr:         n.Addr,
			ProviderAddr: n.ResolvedProvider,
			Change: plans.Change{
				Action: plans.Read,
				Before: priorVal,
				After:  proposedNewVal,
			},
		}

		plannedNewState := &states.ResourceInstanceObject{
			Value:  proposedNewVal,
			Status: states.ObjectPlanned,
		}

		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(n.Addr, states.CurrentGen, plans.Read, priorVal, proposedNewVal)
		}))

		return plannedChange, plannedNewState, diags
	}

	// While this isn't a "diff", continue to call this for data sources.
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(n.Addr, states.CurrentGen, priorVal, configVal)
	}))
	if diags.HasErrors() {
		return nil, nil, diags
	}
	// We have a complete configuration with no dependencies to wait on, so we
	// can read the data source into the state.
	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	// if we have a prior value, we can check for any irregularities in the response
	if !priorVal.IsNull() {
		// While we don't propose planned changes for data sources, we can
		// generate a proposed value for comparison to ensure the data source
		// is returning a result following the rules of the provider contract.
		proposedVal := objchange.ProposedNew(schema, unmarkedPriorVal, unmarkedConfigVal)
		if errs := objchange.AssertObjectCompatible(schema, proposedVal, newVal); len(errs) > 0 {
			// Resources have the LegacyTypeSystem field to signal when they are
			// using an SDK which may not produce precise values. While data
			// sources are read-only, they can still return a value which is not
			// compatible with the config+schema. Since we can't detect the legacy
			// type system, we can only warn about this for now.
			var buf strings.Builder
			fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s.",
				n.ResolvedProvider, n.Addr)
			for _, err := range errs {
				fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
			}
			log.Print(buf.String())
		}
	}

	plannedNewState := &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(n.Addr, states.CurrentGen, plans.Update, priorVal, newVal)
	}))
	return nil, plannedNewState, diags
}

// forcePlanReadData determines if we need to override the usual behavior of
// immediately reading from the data source where possible, instead forcing us
// to generate a plan.
func (n *NodeAbstractResourceInstance) forcePlanReadData(ctx EvalContext) bool {
	nModInst := n.Addr.Module
	nMod := nModInst.Module()

	// Check and see if any depends_on dependencies have
	// changes, since they won't show up as changes in the
	// configuration.
	changes := ctx.Changes()
	for _, d := range n.dependsOn {
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
func (n *NodeAbstractResourceInstance) applyDataSource(ctx EvalContext, planned *plans.ResourceInstanceChange) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	_, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, diags.Append(err)
	}
	if providerSchema == nil {
		return nil, diags.Append(fmt.Errorf("provider schema not available for %s", n.Addr))
	}

	if planned != nil && planned.Action != plans.Read {
		// If any other action gets in here then that's always a bug; this
		// EvalNode only deals with reading.
		diags = diags.Append(fmt.Errorf(
			"invalid action %s for %s: only Read is supported (this is a bug in Terraform; please report it!)",
			planned.Action, n.Addr,
		))
		return nil, diags
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreApply(n.Addr, states.CurrentGen, planned.Action, planned.Before, planned.After)
	}))
	if diags.HasErrors() {
		return nil, diags
	}

	config := *n.Config
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource().Resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider %q does not support data source %q", n.ResolvedProvider, n.Addr.ContainingResource().Resource.Type))
		return nil, diags
	}

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Resource.Key, forEach)

	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	state := &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectReady,
	}

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostApply(n.Addr, states.CurrentGen, newVal, diags.Err())
	}))

	return state, diags
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

	provs := filterProvisioners(n.Config, when)
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil
	}

	// Call pre hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreProvisionInstance(n.Addr, state.Value)
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
		return h.PostProvisionInstance(n.Addr, state.Value)
	}))
}

// filterProvisioners filters the provisioners on the resource to only
// the provisioners specified by the "when" option.
func filterProvisioners(config *configs.Resource, when configs.ProvisionerWhen) []*configs.Provisioner {
	// Fast path the zero case
	if config == nil || config.Managed == nil {
		return nil
	}

	if len(config.Managed.Provisioners) == 0 {
		return nil
	}

	result := make([]*configs.Provisioner, 0, len(config.Managed.Provisioners))
	for _, p := range config.Managed.Provisioners {
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
	if n.Config.Managed != nil && n.Config.Managed.Connection != nil {
		baseConn = n.Config.Managed.Connection.Config
	}

	for _, prov := range provs {
		log.Printf("[TRACE] applyProvisioners: provisioning %s with %q", n.Addr, prov.Type)

		// Get the provisioner
		provisioner, err := ctx.Provisioner(prov.Type)
		if err != nil {
			return diags.Append(err)
		}

		schema := ctx.ProvisionerSchema(prov.Type)

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
				return h.PreProvisionInstanceStep(n.Addr, prov.Type)
			})
			if err != nil {
				return diags.Append(err)
			}
		}

		// The output function
		outputFn := func(msg string) {
			ctx.Hook(func(h Hook) (HookAction, error) {
				h.ProvisionOutput(n.Addr, prov.Type, msg)
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
					h.ProvisionOutput(n.Addr, prov.Type, "(output suppressed due to sensitive value in config)")
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
			return h.PostProvisionInstanceStep(n.Addr, prov.Type, applyDiags.Err())
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

	forEach, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx)
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

	evalScope := ctx.EvaluationScope(n.ResourceInstanceAddr().Resource, keyData)
	config, evalDiags := evalScope.EvalSelfBlock(body, self, schema, keyData)
	diags = diags.Append(evalDiags)

	return config, diags
}

// apply accepts an applyConfig, instead of using n.Config, so destroy plans can
// send a nil config. Most of the errors generated in apply are returned as
// diagnostics, but if provider.ApplyResourceChange itself fails, that error is
// returned as an error and nil diags are returned.
func (n *NodeAbstractResourceInstance) apply(
	ctx EvalContext,
	state *states.ResourceInstanceObject,
	change *plans.ResourceInstanceChange,
	applyConfig *configs.Resource,
	createBeforeDestroy bool) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {

	var diags tfdiags.Diagnostics
	if state == nil {
		state = &states.ResourceInstanceObject{}
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
		forEach, _ := evaluateForEachExpression(applyConfig.ForEach, ctx)
		keyData := EvalDataForInstanceKey(n.ResourceInstanceAddr().Resource.Key, forEach)
		configVal, _, configDiags = ctx.EvaluateBlock(applyConfig.Config, schema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags
		}
	}

	if !configVal.IsWhollyKnown() {
		diags = diags.Append(fmt.Errorf(
			"configuration for %s still contains unknown values during apply (this is a bug in Terraform; please report it!)",
			n.Addr,
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
	if change.Action == plans.Update && eq && !marksEqual(beforePaths, afterPaths) {
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

	resp := provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:       n.Addr.Resource.Resource.Type,
		PriorState:     unmarkedBefore,
		Config:         unmarkedConfigVal,
		PlannedState:   unmarkedAfter,
		PlannedPrivate: change.Private,
		ProviderMeta:   metaConfigVal,
	})
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

	// If we have paths to mark, mark those on this new value
	if len(afterPaths) > 0 {
		newVal = newVal.MarkWithPaths(afterPaths)
	}

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
