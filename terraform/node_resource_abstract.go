package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/lang"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
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

	Schema *configschema.Block // Schema for processing the configuration body
	Config *configs.Resource   // Config is the resource in the config

	ProvisionerSchemas map[string]*configschema.Block

	Targets []addrs.Targetable // Set from GraphNodeTargetable

	// The address of the provider this resource will use
	ResolvedProvider addrs.AbsProviderConfig
}

var (
	_ GraphNodeSubPath              = (*NodeAbstractResource)(nil)
	_ GraphNodeReferenceable        = (*NodeAbstractResource)(nil)
	_ GraphNodeReferencer           = (*NodeAbstractResource)(nil)
	_ GraphNodeProviderConsumer     = (*NodeAbstractResource)(nil)
	_ GraphNodeProvisionerConsumer  = (*NodeAbstractResource)(nil)
	_ GraphNodeResource             = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeAbstractResource)(nil)
	_ GraphNodeAttachResourceSchema = (*NodeAbstractResource)(nil)
	_ GraphNodeTargetable           = (*NodeAbstractResource)(nil)
	_ dag.GraphNodeDotter           = (*NodeAbstractResource)(nil)
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

	ResourceState *ResourceState // the ResourceState for this instance
}

var (
	_ GraphNodeSubPath              = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferenceable        = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeReferencer           = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProviderConsumer     = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer  = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeResource             = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeResourceInstance     = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceState  = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceConfig = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeAttachResourceSchema = (*NodeAbstractResourceInstance)(nil)
	_ GraphNodeTargetable           = (*NodeAbstractResourceInstance)(nil)
	_ dag.GraphNodeDotter           = (*NodeAbstractResourceInstance)(nil)
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
			log.Printf("[WARN] no schema is attached to %s, so references cannot be detected", n.Name())
		}

		refs, _ := lang.ReferencesInExpr(c.Count)
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
	// from configuration.
	if n.Config != nil {
		return n.NodeAbstractResource.References()
	}

	// Otherwise, if we have state then we'll use the values stored in state
	// as a fallback.
	if s := n.ResourceState; s != nil {
		// State is still storing dependencies as old-style strings, so we'll
		// need to do a little work here to massage this to the form we now
		// want.
		var result []*addrs.Reference
		for _, legacyDep := range s.Dependencies {
			traversal, diags := hclsyntax.ParseTraversalAbs([]byte(legacyDep), "", hcl.Pos{})
			if diags.HasErrors() {
				log.Printf("[ERROR] Can't parse %q from dependencies in state as a reference: invalid syntax", legacyDep)
				continue
			}
			ref, err := addrs.ParseRef(traversal)
			if err != nil {
				log.Printf("[ERROR] Can't parse %q from dependencies in state as a reference: invalid syntax", legacyDep)
				continue
			}

			result = append(result, ref)
		}
		return result
	}

	// If we have neither config nor state then we have no references.
	return nil
}

// StateReferences returns the dependencies to put into the state for
// this resource.
func (n *NodeAbstractResource) StateReferences() []string {
	selfAddrs := n.ReferenceableAddrs()

	depsRaw := n.References()
	deps := make([]string, 0, len(depsRaw))
	for _, d := range depsRaw {
		switch tr := d.Subject.(type) {
		case addrs.ResourceInstance:
			// For historical reasons, state uses dot-separated instance keys,
			// rather than bracketed as in our modern syntax.
			var suffix string
			switch tk := tr.Key.(type) {
			case addrs.IntKey:
				suffix = fmt.Sprintf(".%d", int(tk))
			case addrs.StringKey:
				suffix = fmt.Sprintf(".%s", string(tk))
			}
			key := tr.Resource.String() + suffix
			deps = append(deps, key)
		case addrs.Resource:
			depStr := tr.String()
			selfRef := false
			for _, selfAddr := range selfAddrs {
				if selfAddr.String() == depStr {
					selfRef = true
					break
				}
			}
			if !selfRef { // Don't create self-references
				deps = append(deps, tr.String())
			}
		case addrs.ModuleCallInstance:
			deps = append(deps, tr.String())
		case addrs.ModuleCallOutput:
			// For state dependencies, we simplify outputs to just refer
			// to the module as a whole. It's not really clear why we do this,
			// but this logic is preserved from before the 0.12 rewrite of
			// this function.
			deps = append(deps, tr.Call.String())
		default:
			// No other reference types are recorded in the state.
		}
	}

	return deps
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
	return addrs.NewDefaultProviderConfig(n.Addr.Resource.Type).Absolute(n.Addr.Module), false
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResourceInstance) ProvidedBy() (addrs.AbsProviderConfig, bool) {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		relAddr := n.Config.ProviderConfigAddr()
		return relAddr.Absolute(n.Path()), false
	}

	// If we have state, then we will use the provider from there
	if n.ResourceState != nil && n.ResourceState.Provider != "" {
		traversal, parseDiags := hclsyntax.ParseTraversalAbs([]byte(n.ResourceState.Provider), "", hcl.Pos{})
		if parseDiags.HasErrors() {
			log.Printf("[ERROR] %s has syntax-invalid provider address %q", n.Addr, n.ResourceState.Provider)
			goto Guess
		}

		addr, diags := addrs.ParseAbsProviderConfig(traversal)
		if diags.HasErrors() {
			log.Printf("[ERROR] %s has content-invalid provider address %q", n.Addr, n.ResourceState.Provider)
			goto Guess
		}

		// An address from the state must match exactly, since we must ensure
		// we refresh/destroy a resource with the same provider configuration
		// that created it.
		return addr, true
	}

Guess:
	// Use our type and containing module path to guess a provider configuration address
	return addrs.NewDefaultProviderConfig(n.Addr.Resource.Type).Absolute(n.Addr.Module), false
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
func (n *NodeAbstractResourceInstance) AttachResourceState(s *ResourceState) {
	n.ResourceState = s
}

// GraphNodeAttachResourceConfig
func (n *NodeAbstractResource) AttachResourceConfig(c *configs.Resource) {
	n.Config = c
}

// GraphNodeAttachResourceSchema impl
func (n *NodeAbstractResource) AttachResourceSchema(schema *configschema.Block) {
	n.Schema = schema
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
