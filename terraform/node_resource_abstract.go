package terraform

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ConcreteResourceNodeFunc is a callback type used to convert an
// abstract resource to a concrete one of some type.
type ConcreteResourceNodeFunc func(*NodeAbstractResource) dag.Vertex

// GraphNodeConfigResource is implemented by any nodes that represent a resource.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeConfigResource interface {
	ResourceAddr() addrs.ConfigResource
}

// ConcreteResourceInstanceNodeFunc is a callback type used to convert an
// abstract resource instance to a concrete one of some type.
type ConcreteResourceInstanceNodeFunc func(*NodeAbstractResourceInstance) dag.Vertex

// GraphNodeResourceInstance is implemented by any nodes that represent
// a resource instance. A single resource may have multiple instances if,
// for example, the "count" or "for_each" argument is used for it in
// configuration.
type GraphNodeResourceInstance interface {
	ResourceInstanceAddr() addrs.AbsResourceInstance

	// StateDependencies returns any inter-resource dependencies that are
	// stored in the state.
	StateDependencies() []addrs.ConfigResource
}

// NodeAbstractResource represents a resource that has no associated
// operations. It registers all the interfaces for a resource that common
// across multiple operation types.
type NodeAbstractResource struct {
	Addr addrs.ConfigResource

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Schema        *configschema.Block // Schema for processing the configuration body
	SchemaVersion uint64              // Schema version of "Schema", as decided by the provider
	Config        *configs.Resource   // Config is the resource in the config

	// ProviderMetas is the provider_meta configs for the module this resource belongs to
	ProviderMetas map[addrs.Provider]*configs.ProviderMeta

	ProvisionerSchemas map[string]*configschema.Block

	// Set from GraphNodeTargetable
	Targets []addrs.Targetable

	// Set from AttachResourceDependencies
	dependsOn      []addrs.ConfigResource
	forceDependsOn bool

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeReferenceable              = (*NodeAbstractResource)(nil)
	_ GraphNodeReferencer                 = (*NodeAbstractResource)(nil)
	_ GraphNodeProviderConsumer           = (*NodeAbstractResource)(nil)
	_ GraphNodeProvisionerConsumer        = (*NodeAbstractResource)(nil)
	_ GraphNodeConfigResource             = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceConfig       = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceSchema       = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachProvisionerSchema    = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachProviderMetaConfigs  = (*NodeAbstractResource)(nil)
	_ GraphNodeTargetable                 = (*NodeAbstractResource)(nil)
	_ graphNodeAttachResourceDependencies = (*NodeAbstractResource)(nil)
	_ dag.GraphNodeDotter                 = (*NodeAbstractResource)(nil)
)

// NewNodeAbstractResource creates an abstract resource graph node for
// the given absolute resource address.
func NewNodeAbstractResource(addr addrs.ConfigResource) *NodeAbstractResource {
	return &NodeAbstractResource{
		Addr: addr,
	}
}

var (
	_ GraphNodeModuleInstance            = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferenceable             = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferencer                = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProviderConsumer          = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer       = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeConfigResource            = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeResourceInstance          = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceState       = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig      = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceSchema      = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachProvisionerSchema   = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachProviderMetaConfigs = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeTargetable                = (*NodeAbstractResourceInstance)(nil)
	_ dag.GraphNodeDotter                = (*NodeAbstractResourceInstance)(nil)
)

func (n *NodeAbstractResource) Name() string {
	return n.ResourceAddr().String()
}

// GraphNodeModulePath
func (n *NodeAbstractResource) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeReferenceable
func (n *NodeAbstractResource) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Resource}
}

// GraphNodeReferencer
func (n *NodeAbstractResource) References() []*addrs.Reference {
	// If we have a config then we prefer to use that.
	if c := n.Config; c != nil {
		var result []*addrs.Reference

		result = append(result, n.DependsOn()...)

		if n.Schema == nil {
			// Should never happen, but we'll log if it does so that we can
			// see this easily when debugging.
			log.Printf("[WARN] no schema is attached to %s, so config references cannot be detected", n.Name())
		}

		refs, _ := lang.ReferencesInExpr(c.Count)
		result = append(result, refs...)
		refs, _ = lang.ReferencesInExpr(c.ForEach)
		result = append(result, refs...)

		// ReferencesInBlock() requires a schema
		if n.Schema != nil {
			refs, _ = lang.ReferencesInBlock(c.Config, n.Schema)
		}

		result = append(result, refs...)
		if c.Managed != nil {
			if c.Managed.Connection != nil {
				refs, _ = lang.ReferencesInBlock(c.Managed.Connection.Config, connectionBlockSupersetSchema)
				result = append(result, refs...)
			}

			for _, p := range c.Managed.Provisioners {
				if p.When != configs.ProvisionerWhenCreate {
					continue
				}
				if p.Connection != nil {
					refs, _ = lang.ReferencesInBlock(p.Connection.Config, connectionBlockSupersetSchema)
					result = append(result, refs...)
				}

				schema := n.ProvisionerSchemas[p.Type]
				if schema == nil {
					log.Printf("[WARN] no schema for provisioner %q is attached to %s, so provisioner block references cannot be detected", p.Type, n.Name())
				}
				refs, _ = lang.ReferencesInBlock(p.Config, schema)
				result = append(result, refs...)
			}
		}
		return result
	}

	// Otherwise, we have no references.
	return nil
}

func (n *NodeAbstractResource) DependsOn() []*addrs.Reference {
	var result []*addrs.Reference
	if c := n.Config; c != nil {

		for _, traversal := range c.DependsOn {
			ref, diags := addrs.ParseRef(traversal)
			if diags.HasErrors() {
				// We ignore this here, because this isn't a suitable place to return
				// errors. This situation should be caught and rejected during
				// validation.
				log.Printf("[ERROR] Can't parse %#v from depends_on as reference: %s", traversal, diags.Err())
				continue
			}

			result = append(result, ref)
		}
	}
	return result
}

func (n *NodeAbstractResource) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResource) ProvidedBy() (addrs.ProviderConfig, bool) {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		relAddr := n.Config.ProviderConfigAddr()
		return addrs.LocalProviderConfig{
			LocalName: relAddr.LocalName,
			Alias:     relAddr.Alias,
		}, false
	}

	// No provider configuration found; return a default address
	return addrs.AbsProviderConfig{
		Provider: n.Provider(),
		Module:   n.ModulePath(),
	}, false
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResource) Provider() addrs.Provider {
	if n.Config != nil {
		return n.Config.Provider
	}
	return addrs.ImpliedProviderForUnqualifiedType(n.Addr.Resource.ImpliedProvider())
}

// GraphNodeProvisionerConsumer
func (n *NodeAbstractResource) ProvisionedBy() []string {
	// If we have no configuration, then we have no provisioners
	if n.Config == nil || n.Config.Managed == nil {
		return nil
	}

	// Build the list of provisioners we need based on the configuration.
	// It is okay to have duplicates here.
	result := make([]string, len(n.Config.Managed.Provisioners))
	for i, p := range n.Config.Managed.Provisioners {
		result[i] = p.Type
	}

	return result
}

// GraphNodeProvisionerConsumer
func (n *NodeAbstractResource) AttachProvisionerSchema(name string, schema *configschema.Block) {
	if n.ProvisionerSchemas == nil {
		n.ProvisionerSchemas = make(map[string]*configschema.Block)
	}
	n.ProvisionerSchemas[name] = schema
}

// GraphNodeResource
func (n *NodeAbstractResource) ResourceAddr() addrs.ConfigResource {
	return n.Addr
}

// GraphNodeTargetable
func (n *NodeAbstractResource) SetTargets(targets []addrs.Targetable) {
	n.Targets = targets
}

// graphNodeAttachResourceDependencies
func (n *NodeAbstractResource) AttachResourceDependencies(deps []addrs.ConfigResource, force bool) {
	n.dependsOn = deps
	n.forceDependsOn = force
}

// GraphNodeAttachResourceConfig
func (n *NodeAbstractResource) AttachResourceConfig(c *configs.Resource) {
	n.Config = c
}

// GraphNodeAttachResourceSchema impl
func (n *NodeAbstractResource) AttachResourceSchema(schema *configschema.Block, version uint64) {
	n.Schema = schema
	n.SchemaVersion = version
}

// GraphNodeAttachProviderMetaConfigs impl
func (n *NodeAbstractResource) AttachProviderMetaConfigs(c map[addrs.Provider]*configs.ProviderMeta) {
	n.ProviderMetas = c
}

// GraphNodeDotter impl.
func (n *NodeAbstractResource) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "box",
		},
	}
}

// writeResourceState ensures that a suitable resource-level state record is
// present in the state, if that's required for the "each mode" of that
// resource.
//
// This is important primarily for the situation where count = 0, since this
// eval is the only change we get to set the resource "each mode" to list
// in that case, allowing expression evaluation to see it as a zero-element list
// rather than as not set at all.
func (n *NodeAbstractResource) writeResourceState(ctx EvalContext, addr addrs.AbsResource) (diags tfdiags.Diagnostics) {
	state := ctx.State()

	// We'll record our expansion decision in the shared "expander" object
	// so that later operations (i.e. DynamicExpand and expression evaluation)
	// can refer to it. Since this node represents the abstract module, we need
	// to expand the module here to create all resources.
	expander := ctx.InstanceExpander()

	switch {
	case n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return diags
		}

		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceCount(addr.Module, n.Addr.Resource, count)

	case n.Config.ForEach != nil:
		forEach, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceForEach(addr.Module, n.Addr.Resource, forEach)

	default:
		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceSingle(addr.Module, n.Addr.Resource)
	}

	return diags
}

// ReadResourceInstanceState reads the current object for a specific instance in
// the state.
func (n *NodeAbstractResource) ReadResourceInstanceState(ctx EvalContext, addr addrs.AbsResourceInstance) (*states.ResourceInstanceObject, error) {
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] ReadResourceInstanceState: reading state for %s", addr)

	src := ctx.State().ResourceInstanceObject(addr, states.CurrentGen)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] ReadResourceInstanceState: no state present for %s", addr)
		return nil, nil
	}

	schema, currentVersion := (providerSchema).SchemaForResourceAddr(addr.Resource.ContainingResource())
	if schema == nil {
		// Shouldn't happen since we should've failed long ago if no schema is present
		return nil, fmt.Errorf("no schema available for %s while reading state; this is a bug in Terraform and should be reported", addr)
	}
	var diags tfdiags.Diagnostics
	src, diags = UpgradeResourceState(addr, provider, src, schema, currentVersion)
	if diags.HasErrors() {
		// Note that we don't have any channel to return warnings here. We'll
		// accept that for now since warnings during a schema upgrade would
		// be pretty weird anyway, since this operation is supposed to seem
		// invisible to the user.
		return nil, diags.Err()
	}

	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// ReadResourceInstanceStateDeposed reads the deposed object for a specific
// instance in the state.
func (n *NodeAbstractResource) ReadResourceInstanceStateDeposed(ctx EvalContext, addr addrs.AbsResourceInstance, key states.DeposedKey) (*states.ResourceInstanceObject, error) {
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
	if err != nil {
		return nil, err
	}

	if key == states.NotDeposed {
		return nil, fmt.Errorf("EvalReadStateDeposed used with no instance key; this is a bug in Terraform and should be reported")
	}

	log.Printf("[TRACE] EvalReadStateDeposed: reading state for %s deposed object %s", addr, key)

	src := ctx.State().ResourceInstanceObject(addr, key)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] EvalReadStateDeposed: no state present for %s deposed object %s", addr, key)
		return nil, nil
	}

	schema, currentVersion := (providerSchema).SchemaForResourceAddr(addr.Resource.ContainingResource())
	if schema == nil {
		// Shouldn't happen since we should've failed long ago if no schema is present
		return nil, fmt.Errorf("no schema available for %s while reading state; this is a bug in Terraform and should be reported", addr)

	}

	src, diags := UpgradeResourceState(addr, provider, src, schema, currentVersion)
	if diags.HasErrors() {
		// Note that we don't have any channel to return warnings here. We'll
		// accept that for now since warnings during a schema upgrade would
		// be pretty weird anyway, since this operation is supposed to seem
		// invisible to the user.
		return nil, diags.Err()
	}

	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		return nil, err
	}

	return obj, nil
}

type EvalPlanRequest struct {
	Addr                addrs.ResourceInstance
	CreateBeforeDestroy bool
	Provider            providers.Interface
	ProviderSchema      *ProviderSchema
	PlannedChange       *plans.ResourceInstanceChange
	State               *states.ResourceInstanceObject
}

func (n *NodeAbstractResource) Plan(ctx EvalContext, req EvalPlanRequest) (*plans.ResourceInstanceChange, *states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	currentState := req.State
	config := *n.Config
	provider := req.Provider
	providerSchema := req.ProviderSchema
	resource := n.Addr.Resource

	var state *states.ResourceInstanceObject
	var plan *plans.ResourceInstanceChange

	createBeforeDestroy := req.CreateBeforeDestroy
	if req.PlannedChange != nil {
		// If we already planned the action, we stick to that plan
		createBeforeDestroy = req.PlannedChange.Action == plans.CreateThenDelete
	}

	if providerSchema == nil {
		diags = diags.Append(fmt.Errorf("provider schema is unavailable for %s", n.Addr))
		return plan, state, diags
	}
	if n.ResolvedProvider.Provider.Type == "" {
		panic(fmt.Sprintf("EvalDiff for %s does not have ProviderAddr set", n.Addr.Absolute(ctx.Path())))
	}

	// Evaluate the configuration
	schema, _ := providerSchema.SchemaForResourceAddr(resource)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", resource.Type))
		return plan, state, diags
	}

	forEach, _ := evaluateForEachExpression(n.Config.ForEach, ctx)

	keyData := EvalDataForInstanceKey(req.Addr.Key, forEach)
	origConfigVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return plan, state, diags
	}

	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if n.ProviderMetas != nil {
		if m, ok := n.ProviderMetas[n.ResolvedProvider.Provider]; ok && m != nil {
			// if the provider doesn't support this feature, throw an error
			if req.ProviderSchema.ProviderMeta == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ResolvedProvider.Provider),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", req.Addr),
					Subject:  &m.ProviderRange,
				})
			} else {
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, req.ProviderSchema.ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
				if configDiags.HasErrors() {
					return plan, state, diags
				}
			}
		}
	}

	absAddr := req.Addr.Absolute(ctx.Path())
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

	// Create an unmarked version of our config val and our prior val.
	// Store the paths for the config val to re-markafter
	// we've sent things over the wire.
	unmarkedConfigVal, unmarkedPaths := origConfigVal.UnmarkDeepWithPaths()
	unmarkedPriorVal, priorPaths := priorVal.UnmarkDeepWithPaths()

	log.Printf("[TRACE] Re-validating config for %q", n.Addr.Absolute(ctx.Path()))
	// Allow the provider to validate the final set of values.
	// The config was statically validated early on, but there may have been
	// unknown values which the provider could not validate at the time.
	// TODO: It would be more correct to validate the config after
	// ignore_changes has been applied, but the current implementation cannot
	// exclude computed-only attributes when given the `all` option.
	validateResp := provider.ValidateResourceTypeConfig(
		providers.ValidateResourceTypeConfigRequest{
			TypeName: n.Addr.Resource.Type,
			Config:   unmarkedConfigVal,
		},
	)
	if validateResp.Diagnostics.HasErrors() {
		diags = diags.Append(validateResp.Diagnostics.InConfigBody(config.Config))
		return plan, state, diags
	}

	// ignore_changes is meant to only apply to the configuration, so it must
	// be applied before we generate a plan. This ensures the config used for
	// the proposed value, the proposed value itself, and the config presented
	// to the provider in the PlanResourceChange request all agree on the
	// starting values.
	configValIgnored, ignoreChangeDiags := n.processIgnoreChanges(unmarkedPriorVal, unmarkedConfigVal)
	diags = diags.Append(ignoreChangeDiags)
	if ignoreChangeDiags.HasErrors() {
		return plan, state, diags
	}

	proposedNewVal := objchange.ProposedNewObject(schema, unmarkedPriorVal, configValIgnored)

	// Call pre-diff hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
	}))
	if diags.HasErrors() {
		return plan, state, diags
	}

	resp := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         n.Addr.Resource.Type,
		Config:           configValIgnored,
		PriorState:       unmarkedPriorVal,
		ProposedNewState: proposedNewVal,
		PriorPrivate:     priorPrivate,
		ProviderMeta:     metaConfigVal,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
	if diags.HasErrors() {
		return plan, state, diags
	}

	plannedNewVal := resp.PlannedState
	plannedPrivate := resp.PlannedPrivate

	if plannedNewVal == cty.NilVal {
		// Should never happen. Since real-world providers return via RPC a nil
		// is always a bug in the client-side stub. This is more likely caused
		// by an incompletely-configured mock provider in tests, though.
		panic(fmt.Sprintf("PlanResourceChange of %s produced nil value", absAddr.String()))
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
				n.ResolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return plan, state, diags
	}

	if errs := objchange.AssertPlanValid(schema, unmarkedPriorVal, configValIgnored, plannedNewVal); len(errs) > 0 {
		if resp.LegacyTypeSystem {
			// The shimming of the old type system in the legacy SDK is not precise
			// enough to pass this consistency check, so we'll give it a pass here,
			// but we will generate a warning about it so that we are more likely
			// to notice in the logs if an inconsistency beyond the type system
			// leads to a downstream provider failure.
			var buf strings.Builder
			fmt.Fprintf(&buf,
				"[WARN] Provider %q produced an invalid plan for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:",
				n.ResolvedProvider.Provider, absAddr,
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
						n.ResolvedProvider.Provider, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
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
						n.ResolvedProvider.Provider, absAddr, path,
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

	// Unmark for this test for value equality.
	eqV := unmarkedPlannedNewVal.Equals(unmarkedPriorVal)
	eq := eqV.IsKnown() && eqV.True()

	var action plans.Action
	switch {
	case priorVal.IsNull():
		action = plans.Create
	case eq:
		action = plans.NoOp
	case !reqRep.Empty():
		// If there are any "requires replace" paths left _after our filtering
		// above_ then this is a replace action.
		if createBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
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
		proposedNewVal = objchange.ProposedNewObject(schema, nullPriorVal, unmarkedConfigVal)

		resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:         n.Addr.Resource.Type,
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
			diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
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
					n.ResolvedProvider.Provider, absAddr, tfdiags.FormatError(err),
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
	if action == plans.Create && priorValTainted != cty.NilVal {
		if createBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
		}
		priorVal = priorValTainted
	}

	// If we plan to write or delete sensitive paths from state,
	// this is an Update action
	if action == plans.NoOp && !reflect.DeepEqual(priorPaths, unmarkedPaths) {
		action = plans.Update
	}

	// As a special case, if we have a previous diff (presumably from the plan
	// phases, whereas we're now in the apply phase) and it was for a replace,
	// we've already deleted the original object from state by the time we
	// get here and so we would've ended up with a _create_ action this time,
	// which we now need to paper over to get a result consistent with what
	// we originally intended.
	if req.PlannedChange != nil {
		prevChange := *req.PlannedChange
		if prevChange.Action.IsReplace() && action == plans.Create {
			log.Printf("[TRACE] EvalDiff: %s treating Create change as %s change to match with earlier plan", absAddr, prevChange.Action)
			action = prevChange.Action
			priorVal = prevChange.Before
		}
	}

	// Call post-refresh hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(absAddr, states.CurrentGen, action, priorVal, plannedNewVal)
	}))
	if diags.HasErrors() {
		return plan, state, diags
	}

	// Update our return plan
	plan = &plans.ResourceInstanceChange{
		Addr:         absAddr,
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

	ignoreChanges := n.Config.Managed.IgnoreChanges
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

	return processIgnoreChangesIndividual(prior, config, ignoreChanges)
}

func processIgnoreChangesIndividual(prior, config cty.Value, ignoreChanges []hcl.Traversal) (cty.Value, tfdiags.Diagnostics) {
	// When we walk below we will be using cty.Path values for comparison, so
	// we'll convert our traversals here so we can compare more easily.
	ignoreChangesPath := make([]cty.Path, len(ignoreChanges))
	for i, traversal := range ignoreChanges {
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
		ignoreChangesPath[i] = path
	}

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
		eq := p.Equals(c)
		if eq.IsKnown() && eq.False() {
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
			switch {
			case ignored.value.IsNull() || ignored.value.LengthInt() == 0:
				priorMap = map[string]cty.Value{}
			default:
				priorMap = ignored.value.AsValueMap()
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

		if len(configMap) == 0 {
			return cty.MapValEmpty(v.Type().ElementType()), nil
		}

		return cty.MapVal(configMap), nil
	})
	return ret, nil
}

// graphNodesAreResourceInstancesInDifferentInstancesOfSameModule is an
// annoyingly-task-specific helper function that returns true if and only if
// the following conditions hold:
// - Both of the given vertices represent specific resource instances, as
//   opposed to unexpanded resources or any other non-resource-related object.
// - The module instance addresses for both of the resource instances belong
//   to the same static module.
// - The module instance addresses for both of the resource instances are
//   not equal, indicating that they belong to different instances of the
//   same module.
//
// This result can be used as a way to compensate for the effects of
// conservative analyses passes in our graph builders which make their
// decisions based only on unexpanded addresses, often so that they can behave
// correctly for interactions between expanded and not-yet-expanded objects.
//
// Callers of this helper function will typically skip adding an edge between
// the two given nodes if this function returns true.
func graphNodesAreResourceInstancesInDifferentInstancesOfSameModule(a, b dag.Vertex) bool {
	aRI, aOK := a.(GraphNodeResourceInstance)
	bRI, bOK := b.(GraphNodeResourceInstance)
	if !(aOK && bOK) {
		return false
	}
	aModInst := aRI.ResourceInstanceAddr().Module
	bModInst := bRI.ResourceInstanceAddr().Module
	aMod := aModInst.Module()
	bMod := bModInst.Module()
	if !aMod.Equal(bMod) {
		return false
	}
	return !aModInst.Equal(bModInst)
}
