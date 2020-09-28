package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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

func (n *NodeAbstractResource) Name() string {
	return n.ResourceAddr().String()
}

func (n *NodeAbstractResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String()
}

func (n *NodeAbstractResourceInstance) Path() addrs.ModuleInstance {
	return n.Addr.Module
}

// GraphNodeModulePath
func (n *NodeAbstractResource) ModulePath() addrs.Module {
	return n.Addr.Module
}

// GraphNodeReferenceable
func (n *NodeAbstractResource) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Resource}
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

// converts an instance address to the legacy dotted notation
func dottedInstanceAddr(tr addrs.ResourceInstance) string {
	// The legacy state format uses dot-separated instance keys,
	// rather than bracketed as in our modern syntax.
	var suffix string
	switch tk := tr.Key.(type) {
	case addrs.IntKey:
		suffix = fmt.Sprintf(".%d", int(tk))
	case addrs.StringKey:
		suffix = fmt.Sprintf(".%s", string(tk))
	}
	return tr.Resource.String() + suffix
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

// GraphNodeResourceInstance
func (n *NodeAbstractResourceInstance) ResourceInstanceAddr() addrs.AbsResourceInstance {
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

// GraphNodeAttachResourceState
func (n *NodeAbstractResourceInstance) AttachResourceState(s *states.Resource) {
	if s == nil {
		log.Printf("[WARN] attaching nil state to %s", n.Addr)
		return
	}
	n.instanceState = s.Instance(n.Addr.Resource.Key)
	n.storedProviderConfig = s.ProviderConfig
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

// WriteResourceState ensures that a suitable resource-level state record is
// present in the state, if that's required for the "each mode" of that
// resource.
//
// This is important primarily for the situation where count = 0, since this
// eval is the only change we get to set the resource "each mode" to list
// in that case, allowing expression evaluation to see it as a zero-element list
// rather than as not set at all.
func (n *NodeAbstractResource) WriteResourceState(ctx EvalContext, addr addrs.AbsResource) error {
	var diags tfdiags.Diagnostics
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
			return diags.Err()
		}

		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceCount(addr.Module, n.Addr.Resource, count)

	case n.Config.ForEach != nil:
		forEach, forEachDiags := evaluateForEachExpression(n.Config.ForEach, ctx)
		diags = diags.Append(forEachDiags)
		if forEachDiags.HasErrors() {
			return diags.Err()
		}

		// This method takes care of all of the business logic of updating this
		// while ensuring that any existing instances are preserved, etc.
		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceForEach(addr.Module, n.Addr.Resource, forEach)

	default:
		state.SetResourceProvider(addr, n.ResolvedProvider)
		expander.SetResourceSingle(addr.Module, n.Addr.Resource)
	}

	return nil
}

func (n *NodeAbstractResource) ReadResourceInstanceState(ctx EvalContext, addr addrs.AbsResourceInstance) (*states.ResourceInstanceObject, error) {
	provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)

	if provider == nil {
		panic("ReadResourceInstanceState used with no Provider object")
	}
	if providerSchema == nil {
		panic("ReadResourceInstanceState used with no ProviderSchema object")
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

// CheckPreventDestroy returns an error if a resource has PreventDestroy
// configured and the diff would destroy the resource.
func (n *NodeAbstractResource) CheckPreventDestroy(addr addrs.AbsResourceInstance, change *plans.ResourceInstanceChange) error {
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
				addr,
			),
			Subject: &n.Config.DeclRange,
		})
		return diags.Err()
	}

	return nil
}

// ReadDiff returns the planned change for a particular resource instance
// object.
func (n *NodeAbstractResourceInstance) ReadDiff(ctx EvalContext, providerSchema *ProviderSchema) (*plans.ResourceInstanceChange, error) {
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
