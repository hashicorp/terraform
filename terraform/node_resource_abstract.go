package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// ConcreteResourceNodeFunc is a callback type used to convert an
// abstract resource to a concrete one of some type.
type ConcreteResourceNodeFunc func(*NodeAbstractResource) dag.Vertex

// GraphNodeResource is implemented by any nodes that represent a resource.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeResource interface {
	ResourceAddr() addrs.AbsResource
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
}

// NodeAbstractResource represents a resource that has no associated
// operations. It registers all the interfaces for a resource that common
// across multiple operation types.
type NodeAbstractResource struct {
	Addr addrs.AbsResource // Addr is the address for this resource

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Schema        *configschema.Block // Schema for processing the configuration body
	SchemaVersion uint64              // Schema version of "Schema", as decided by the provider
	Config        *configs.Resource   // Config is the resource in the config

	ProvisionerSchemas map[string]*configschema.Block

	Targets []addrs.Targetable // Set from GraphNodeTargetable

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeSubPath                 = (*NodeAbstractResource)(nil)
	_ GraphNodeReferenceable           = (*NodeAbstractResource)(nil)
	_ GraphNodeReferencer              = (*NodeAbstractResource)(nil)
	_ GraphNodeProviderConsumer        = (*NodeAbstractResource)(nil)
	_ GraphNodeProvisionerConsumer     = (*NodeAbstractResource)(nil)
	_ GraphNodeResource                = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceConfig    = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceSchema    = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachProvisionerSchema = (*NodeAbstractResource)(nil)
	_ GraphNodeTargetable              = (*NodeAbstractResource)(nil)
	_ dag.GraphNodeDotter              = (*NodeAbstractResource)(nil)
)

// NewNodeAbstractResource creates an abstract resource graph node for
// the given absolute resource address.
func NewNodeAbstractResource(addr addrs.AbsResource) *NodeAbstractResource {
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
	InstanceKey addrs.InstanceKey

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.
	ResourceState *states.Resource
	Dependencies  []addrs.AbsResource
}

var (
	_ GraphNodeSubPath                 = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferenceable           = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferencer              = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProviderConsumer        = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer     = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeResource                = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeResourceInstance        = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceState     = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig    = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceSchema    = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachProvisionerSchema = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeTargetable              = (*NodeAbstractResourceInstance)(nil)
	_ dag.GraphNodeDotter              = (*NodeAbstractResourceInstance)(nil)
)

// NewNodeAbstractResourceInstance creates an abstract resource instance graph
// node for the given absolute resource instance address.
func NewNodeAbstractResourceInstance(addr addrs.AbsResourceInstance) *NodeAbstractResourceInstance {
	// Due to the fact that we embed NodeAbstractResource, the given address
	// actually ends up split between the resource address in the embedded
	// object and the InstanceKey field in our own struct. The
	// ResourceInstanceAddr method will stick these back together again on
	// request.
	return &NodeAbstractResourceInstance{
		NodeAbstractResource: NodeAbstractResource{
			Addr: addr.ContainingResource(),
		},
		InstanceKey: addr.Resource.Key,
	}
}

func (n *NodeAbstractResource) Name() string {
	return n.ResourceAddr().String()
}

func (n *NodeAbstractResourceInstance) Name() string {
	return n.ResourceInstanceAddr().String()
}

// GraphNodeSubPath
func (n *NodeAbstractResource) Path() addrs.ModuleInstance {
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

		for _, traversal := range c.DependsOn {
			ref, err := addrs.ParseRef(traversal)
			if err != nil {
				// We ignore this here, because this isn't a suitable place to return
				// errors. This situation should be caught and rejected during
				// validation.
				log.Printf("[ERROR] Can't parse %#v from depends_on as reference: %s", traversal, err)
				continue
			}

			result = append(result, ref)
		}

		if n.Schema == nil {
			// Should never happens, but we'll log if it does so that we can
			// see this easily when debugging.
			log.Printf("[WARN] no schema is attached to %s, so config references cannot be detected", n.Name())
		}

		refs, _ := lang.ReferencesInExpr(c.Count)
		result = append(result, refs...)
		refs, _ = lang.ReferencesInExpr(c.ForEach)
		result = append(result, refs...)
		refs, _ = lang.ReferencesInBlock(c.Config, n.Schema)
		result = append(result, refs...)
		if c.Managed != nil {
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

	// FIXME: remove once the deprecated DependsOn values have been removed from state
	// The state dependencies are now connected in a separate transformation as
	// absolute addresses, but we need to keep this here until we can be sure
	// that no state will need to use the old depends_on references.
	if rs := n.ResourceState; rs != nil {
		if s := rs.Instance(n.InstanceKey); s != nil {
			// State is still storing dependencies as old-style strings, so we'll
			// need to do a little work here to massage this to the form we now
			// want.
			var result []*addrs.Reference

			// It is (apparently) possible for s.Current to be nil. This proved
			// difficult to reproduce, so we will fix the symptom here and hope
			// to find the root cause another time.
			//
			// https://github.com/hashicorp/terraform/issues/21407
			if s.Current == nil {
				log.Printf("[WARN] no current state found for %s", n.Name())
				return nil
			}
			for _, addr := range s.Current.DependsOn {
				if addr == nil {
					// Should never happen; indicates a bug in the state loader
					panic(fmt.Sprintf("dependencies for current object on %s contains nil address", n.ResourceInstanceAddr()))
				}

				// This is a little weird: we need to manufacture an addrs.Reference
				// with a fake range here because the state isn't something we can
				// make source references into.
				result = append(result, &addrs.Reference{
					Subject: addr,
					SourceRange: tfdiags.SourceRange{
						Filename: "(state file)",
					},
				})
			}
			return result
		}
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
func (n *NodeAbstractResourceInstance) StateDependencies() []addrs.AbsResource {
	if rs := n.ResourceState; rs != nil {
		if s := rs.Instance(n.InstanceKey); s != nil {
			if s.Current != nil {
				return s.Current.Dependencies
			}
		}
	}

	return nil
}

func (n *NodeAbstractResource) SetProvider(p addrs.AbsProviderConfig) {
	n.ResolvedProvider = p
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResource) ProvidedBy() (addrs.AbsProviderConfig, bool) {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		relAddr := n.Config.ProviderConfigAddr()
		return relAddr.Absolute(n.Path()), false
	}

	// Use our type and containing module path to guess a provider configuration address
	return n.Addr.Resource.DefaultProviderConfig().Absolute(n.Addr.Module), false
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResourceInstance) ProvidedBy() (addrs.AbsProviderConfig, bool) {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		relAddr := n.Config.ProviderConfigAddr()
		return relAddr.Absolute(n.Path()), false
	}

	// If we have state, then we will use the provider from there
	if n.ResourceState != nil {
		// An address from the state must match exactly, since we must ensure
		// we refresh/destroy a resource with the same provider configuration
		// that created it.
		return n.ResourceState.ProviderConfig, true
	}

	// Use our type and containing module path to guess a provider configuration address
	return n.Addr.Resource.DefaultProviderConfig().Absolute(n.Path()), false
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
func (n *NodeAbstractResource) ResourceAddr() addrs.AbsResource {
	return n.Addr
}

// GraphNodeResourceInstance
func (n *NodeAbstractResourceInstance) ResourceInstanceAddr() addrs.AbsResourceInstance {
	return n.NodeAbstractResource.Addr.Instance(n.InstanceKey)
}

// GraphNodeAddressable, TODO: remove, used by target, should unify
func (n *NodeAbstractResource) ResourceAddress() *ResourceAddress {
	return NewLegacyResourceAddress(n.Addr)
}

// GraphNodeTargetable
func (n *NodeAbstractResource) SetTargets(targets []addrs.Targetable) {
	n.Targets = targets
}

// GraphNodeAttachResourceState
func (n *NodeAbstractResourceInstance) AttachResourceState(s *states.Resource) {
	n.ResourceState = s
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
