// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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

	// Config and RemovedConfig are mutally-exclusive, because a
	// resource can't be both declared and removed at the same time.
	Config        *configs.Resource // Config is the resource in the config, if any
	RemovedConfig *configs.Removed  // RemovedConfig is the "removed" block for this resource, if any

	// ProviderMetas is the provider_meta configs for the module this resource belongs to
	ProviderMetas map[addrs.Provider]*configs.ProviderMeta

	ProvisionerSchemas map[string]*configschema.Block

	// Set from GraphNodeTargetable
	Targets []addrs.Targetable

	// Set from AttachDataResourceDependsOn
	dependsOn []addrs.ConfigResource

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
	// storedProviderConfig is the provider address retrieved from the
	// state. This is defined here for access within the ProvidedBy method, but
	// will be set from the embedding instance type when the state is attached.
	storedProviderConfig addrs.AbsProviderConfig

	// This resource may expand into instances which need to be imported.
	importTargets []*ImportTarget

	// generateConfigPath tells this node which file to write generated config
	// into. If empty, then config should not be generated.
	generateConfigPath string

	forceCreateBeforeDestroy bool
}

var (
	_ GraphNodeReferenceable               = (*NodeAbstractResource)(nil)
	_ GraphNodeReferencer                  = (*NodeAbstractResource)(nil)
	_ GraphNodeImportReferencer            = (*NodeAbstractResource)(nil)
	_ GraphNodeProviderConsumer            = (*NodeAbstractResource)(nil)
	_ GraphNodeProvisionerConsumer         = (*NodeAbstractResource)(nil)
	_ GraphNodeConfigResource              = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceConfig        = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceSchema        = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachProvisionerSchema     = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachProviderMetaConfigs   = (*NodeAbstractResource)(nil)
	_ GraphNodeTargetable                  = (*NodeAbstractResource)(nil)
	_ graphNodeAttachDataResourceDependsOn = (*NodeAbstractResource)(nil)
	_ dag.GraphNodeDotter                  = (*NodeAbstractResource)(nil)
	_ GraphNodeDestroyerCBD                = (*NodeAbstractResource)(nil)
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
	_ GraphNodeOverridable               = (*NodeAbstractResourceInstance)(nil)
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

// CreateBeforeDestroy returns this node's CreateBeforeDestroy status.
func (n *NodeAbstractResource) CreateBeforeDestroy() bool {
	if n.forceCreateBeforeDestroy {
		return n.forceCreateBeforeDestroy
	}

	if n.Config != nil && n.Config.Managed != nil {
		return n.Config.Managed.CreateBeforeDestroy
	}

	return false
}

func (n *NodeAbstractResource) ModifyCreateBeforeDestroy(v bool) error {
	n.forceCreateBeforeDestroy = v
	return nil
}

// GraphNodeReferencer
func (n *NodeAbstractResource) References() []*addrs.Reference {
	var result []*addrs.Reference
	// If we have a config then we prefer to use that.
	if c := n.Config; c != nil {
		result = append(result, n.DependsOn()...)

		if n.Schema == nil {
			// Should never happen, but we'll log if it does so that we can
			// see this easily when debugging.
			log.Printf("[WARN] no schema is attached to %s, so config references cannot be detected", n.Name())
		}

		refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, c.Count)
		result = append(result, refs...)
		refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, c.ForEach)
		result = append(result, refs...)

		for _, expr := range c.TriggersReplacement {
			refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, expr)
			result = append(result, refs...)
		}

		// ReferencesInBlock() requires a schema
		if n.Schema != nil {
			refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Config, n.Schema)
			result = append(result, refs...)
		}

		if c.Managed != nil {
			if c.Managed.Connection != nil {
				refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, c.Managed.Connection.Config, connectionBlockSupersetSchema)
				result = append(result, refs...)
			}

			for _, p := range c.Managed.Provisioners {
				if p.When != configs.ProvisionerWhenCreate {
					continue
				}
				if p.Connection != nil {
					refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, p.Connection.Config, connectionBlockSupersetSchema)
					result = append(result, refs...)
				}

				schema := n.ProvisionerSchemas[p.Type]
				if schema == nil {
					log.Printf("[WARN] no schema for provisioner %q is attached to %s, so provisioner block references cannot be detected", p.Type, n.Name())
				}
				refs, _ = langrefs.ReferencesInBlock(addrs.ParseRef, p.Config, schema)
				result = append(result, refs...)
			}
		}

		for _, check := range c.Preconditions {
			refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.Condition)
			result = append(result, refs...)
			refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, check.ErrorMessage)
			result = append(result, refs...)
		}
		for _, check := range c.Postconditions {
			refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, check.Condition)
			result = append(result, refs...)
			refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, check.ErrorMessage)
			result = append(result, refs...)
		}
	}

	return result
}

func (n *NodeAbstractResource) ImportReferences() []*addrs.Reference {
	var result []*addrs.Reference
	for _, importTarget := range n.importTargets {
		// legacy import won't have any config
		if importTarget.Config == nil {
			continue
		}

		refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, importTarget.Config.ID)
		result = append(result, refs...)
		refs, _ = langrefs.ReferencesInExpr(addrs.ParseRef, importTarget.Config.ForEach)
		result = append(result, refs...)
	}
	return result
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
	// Once the provider is fully resolved, we can return the known value.
	if n.ResolvedProvider.Provider.Type != "" {
		return n.ResolvedProvider, true
	}

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

	// We might have an import target that is providing a specific provider,
	// this is okay as we know there is nothing else potentially providing a
	// provider configuration.
	if len(n.importTargets) > 0 {
		// The import targets should either all be defined via config or none
		// of them should be. They should also all have the same provider, so it
		// shouldn't matter which we check here, as they'll all give the same.
		if n.importTargets[0].Config != nil && n.importTargets[0].Config.ProviderConfigRef != nil {
			return addrs.LocalProviderConfig{
				LocalName: n.importTargets[0].Config.ProviderConfigRef.Name,
				Alias:     n.importTargets[0].Config.ProviderConfigRef.Alias,
			}, false
		}
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
	if n.storedProviderConfig.Provider.Type != "" {
		return n.storedProviderConfig.Provider
	}

	if len(n.importTargets) > 0 {
		// The import targets should either all be defined via config or none
		// of them should be. They should also all have the same provider, so it
		// shouldn't matter which we check here, as they'll all give the same.
		if n.importTargets[0].Config != nil {
			return n.importTargets[0].Config.Provider
		}
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

// graphNodeAttachDataResourceDependsOn
func (n *NodeAbstractResource) AttachDataResourceDependsOn(deps []addrs.ConfigResource) {
	n.dependsOn = deps
}

// GraphNodeAttachResourceConfig
func (n *NodeAbstractResource) AttachResourceConfig(c *configs.Resource, rc *configs.Removed) {
	n.Config = c
	n.RemovedConfig = rc
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

// recordResourceData records some metadata for the resource as a whole in
// various locations. This currently includes adding resource expansion info to
// the instance expander, and recording the provider used in the state.
func (n *NodeAbstractResource) recordResourceData(ctx EvalContext, addr addrs.AbsResource) (diags tfdiags.Diagnostics) {

	// We'll record our expansion decision in the shared "expander" object
	// so that later operations (i.e. DynamicExpand and expression evaluation)
	// can refer to it. Since this node represents the abstract module, we need
	// to expand the module here to create all resources.
	expander := ctx.InstanceExpander()

	// Allowing unknown values in count and for_each is a top-level plan option.
	//
	// If this is false then the codepaths that handle unknown values below
	// become unreachable, because the evaluate functions will reject unknown
	// values as an error.
	allowUnknown := ctx.Deferrals().DeferralAllowed()

	switch {
	case n.Config != nil && n.Config.Count != nil:
		count, countDiags := evaluateCountExpression(n.Config.Count, ctx, allowUnknown)
		diags = diags.Append(countDiags)
		if countDiags.HasErrors() {
			return diags
		}

		if count >= 0 {
			expander.SetResourceCount(addr.Module, n.Addr.Resource, count)
		} else {
			// -1 represents "unknown"
			expander.SetResourceCountUnknown(addr.Module, n.Addr.Resource)
		}

	case n.Config != nil && n.Config.ForEach != nil:
		forEach, known, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx, allowUnknown)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		if known {
			expander.SetResourceForEach(addr.Module, n.Addr.Resource, forEach)
		} else {
			expander.SetResourceForEachUnknown(addr.Module, n.Addr.Resource)
		}

	default:
		expander.SetResourceSingle(addr.Module, n.Addr.Resource)
	}

	if addr.Resource.Mode == addrs.EphemeralResourceMode {
		// ephemeral resources are not included in the state
		return diags
	}

	state := ctx.State()
	state.SetResourceProvider(addr, n.ResolvedProvider)

	return diags
}

// readResourceInstanceState reads the current object for a specific instance in
// the state.
func (n *NodeAbstractResource) readResourceInstanceState(ctx EvalContext, addr addrs.AbsResourceInstance) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	log.Printf("[TRACE] readResourceInstanceState: reading state for %s", addr)

	src := ctx.State().ResourceInstanceObject(addr, addrs.NotDeposed)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] readResourceInstanceState: no state present for %s", addr)
		return nil, nil
	}

	schema, currentVersion := (providerSchema).SchemaForResourceAddr(addr.Resource.ContainingResource())
	if schema == nil {
		// Shouldn't happen since we should've failed long ago if no schema is present
		return nil, diags.Append(fmt.Errorf("no schema available for %s while reading state; this is a bug in Terraform and should be reported", addr))
	}
	src, upgradeDiags := upgradeResourceState(addr, provider, src, schema, currentVersion)
	if n.Config != nil {
		upgradeDiags = upgradeDiags.InConfigBody(n.Config.Config, addr.String())
	}
	diags = diags.Append(upgradeDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		diags = diags.Append(err)
	}

	return obj, diags
}

// readResourceInstanceStateDeposed reads the deposed object for a specific
// instance in the state.
func (n *NodeAbstractResource) readResourceInstanceStateDeposed(ctx EvalContext, addr addrs.AbsResourceInstance, key states.DeposedKey) (*states.ResourceInstanceObject, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	provider, providerSchema, err := getProvider(ctx, n.ResolvedProvider)
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	if key == states.NotDeposed {
		return nil, diags.Append(fmt.Errorf("readResourceInstanceStateDeposed used with no instance key; this is a bug in Terraform and should be reported"))
	}

	log.Printf("[TRACE] readResourceInstanceStateDeposed: reading state for %s deposed object %s", addr, key)

	src := ctx.State().ResourceInstanceObject(addr, key)
	if src == nil {
		// Presumably we only have deposed objects, then.
		log.Printf("[TRACE] readResourceInstanceStateDeposed: no state present for %s deposed object %s", addr, key)
		return nil, diags
	}

	schema, currentVersion := (providerSchema).SchemaForResourceAddr(addr.Resource.ContainingResource())
	if schema == nil {
		// Shouldn't happen since we should've failed long ago if no schema is present
		return nil, diags.Append(fmt.Errorf("no schema available for %s while reading state; this is a bug in Terraform and should be reported", addr))

	}

	src, upgradeDiags := upgradeResourceState(addr, provider, src, schema, currentVersion)
	if n.Config != nil {
		upgradeDiags = upgradeDiags.InConfigBody(n.Config.Config, addr.String())
	}
	diags = diags.Append(upgradeDiags)
	if diags.HasErrors() {
		// Note that we don't have any channel to return warnings here. We'll
		// accept that for now since warnings during a schema upgrade would
		// be pretty weird anyway, since this operation is supposed to seem
		// invisible to the user.
		return nil, diags
	}

	obj, err := src.Decode(schema.ImpliedType())
	if err != nil {
		diags = diags.Append(err)
	}

	return obj, diags
}

// graphNodesAreResourceInstancesInDifferentInstancesOfSameModule is an
// annoyingly-task-specific helper function that returns true if and only if
// the following conditions hold:
//   - Both of the given vertices represent specific resource instances, as
//     opposed to unexpanded resources or any other non-resource-related object.
//   - The module instance addresses for both of the resource instances belong
//     to the same static module.
//   - The module instance addresses for both of the resource instances are
//     not equal, indicating that they belong to different instances of the
//     same module.
//
// This result can be used as a way to compensate for the effects of
// conservative analysis passes in our graph builders which make their
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
