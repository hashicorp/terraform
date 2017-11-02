package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// ConcreteResourceNodeFunc is a callback type used to convert an
// abstract resource to a concrete one of some type.
type ConcreteResourceNodeFunc func(*NodeAbstractResource) dag.Vertex

// GraphNodeResource is implemented by any nodes that represent a resource.
// The type of operation cannot be assumed, only that this node represents
// the given resource.
type GraphNodeResource interface {
	ResourceAddr() *ResourceAddress
}

// NodeAbstractResource represents a resource that has no associated
// operations. It registers all the interfaces for a resource that common
// across multiple operation types.
type NodeAbstractResource struct {
	Addr *ResourceAddress // Addr is the address for this resource

	// The fields below will be automatically set using the Attach
	// interfaces if you're running those transforms, but also be explicitly
	// set if you already have that information.

	Config        *config.Resource // Config is the resource in the config
	ResourceState *ResourceState   // ResourceState is the ResourceState for this

	Targets []ResourceAddress // Set from GraphNodeTargetable

	// The address of the provider this resource will use
	ResolvedProvider string
}

func (n *NodeAbstractResource) Name() string {
	return n.Addr.String()
}

// GraphNodeSubPath
func (n *NodeAbstractResource) Path() []string {
	return n.Addr.Path
}

// GraphNodeReferenceable
func (n *NodeAbstractResource) ReferenceableName() []string {
	// We always are referenceable as "type.name" as long as
	// we have a config or address. Determine what that value is.
	var id string
	if n.Config != nil {
		id = n.Config.Id()
	} else if n.Addr != nil {
		addrCopy := n.Addr.Copy()
		addrCopy.Path = nil // ReferenceTransformer handles paths
		addrCopy.Index = -1 // We handle indexes below
		id = addrCopy.String()
	} else {
		// No way to determine our type.name, just return
		return nil
	}

	var result []string

	// Always include our own ID. This is primarily for backwards
	// compatibility with states that didn't yet support the more
	// specific dep string.
	result = append(result, id)

	// We represent all multi-access
	result = append(result, fmt.Sprintf("%s.*", id))

	// We represent either a specific number, or all numbers
	suffix := "N"
	if n.Addr != nil {
		idx := n.Addr.Index
		if idx == -1 {
			idx = 0
		}

		suffix = fmt.Sprintf("%d", idx)
	}
	result = append(result, fmt.Sprintf("%s.%s", id, suffix))

	return result
}

// GraphNodeReferencer
func (n *NodeAbstractResource) References() []string {
	// If we have a config, that is our source of truth
	if c := n.Config; c != nil {
		// Grab all the references
		var result []string
		result = append(result, c.DependsOn...)
		result = append(result, ReferencesFromConfig(c.RawCount)...)
		result = append(result, ReferencesFromConfig(c.RawConfig)...)
		for _, p := range c.Provisioners {
			if p.When == config.ProvisionerWhenCreate {
				result = append(result, ReferencesFromConfig(p.ConnInfo)...)
				result = append(result, ReferencesFromConfig(p.RawConfig)...)
			}
		}

		return uniqueStrings(result)
	}

	// If we have state, that is our next source
	if s := n.ResourceState; s != nil {
		return s.Dependencies
	}

	return nil
}

// StateReferences returns the dependencies to put into the state for
// this resource.
func (n *NodeAbstractResource) StateReferences() []string {
	self := n.ReferenceableName()

	// Determine what our "prefix" is for checking for references to
	// ourself.
	addrCopy := n.Addr.Copy()
	addrCopy.Index = -1
	selfPrefix := addrCopy.String() + "."

	depsRaw := n.References()
	deps := make([]string, 0, len(depsRaw))
	for _, d := range depsRaw {
		// Ignore any variable dependencies
		if strings.HasPrefix(d, "var.") {
			continue
		}

		// If this has a backup ref, ignore those for now. The old state
		// file never contained those and I'd rather store the rich types we
		// add in the future.
		if idx := strings.IndexRune(d, '/'); idx != -1 {
			d = d[:idx]
		}

		// If we're referencing ourself, then ignore it
		found := false
		for _, s := range self {
			if d == s {
				found = true
			}
		}
		if found {
			continue
		}

		// If this is a reference to ourself and a specific index, we keep
		// it. For example, if this resource is "foo.bar" and the reference
		// is "foo.bar.0" then we keep it exact. Otherwise, we strip it.
		if strings.HasSuffix(d, ".0") && !strings.HasPrefix(d, selfPrefix) {
			d = d[:len(d)-2]
		}

		// This is sad. The dependencies are currently in the format of
		// "module.foo.bar" (the full field). This strips the field off.
		if strings.HasPrefix(d, "module.") {
			parts := strings.SplitN(d, ".", 3)
			d = strings.Join(parts[0:2], ".")
		}

		deps = append(deps, d)
	}

	return deps
}

func (n *NodeAbstractResource) SetProvider(p string) {
	n.ResolvedProvider = p
}

// GraphNodeProviderConsumer
func (n *NodeAbstractResource) ProvidedBy() []string {
	// If we have a config we prefer that above all else
	if n.Config != nil {
		return []string{resourceProvider(n.Config.Type, n.Config.Provider)}
	}

	// If we have state, then we will use the provider from there
	if n.ResourceState != nil && n.ResourceState.Provider != "" {
		return []string{n.ResourceState.Provider}
	}

	// Use our type
	return []string{resourceProvider(n.Addr.Type, "")}
}

// GraphNodeProvisionerConsumer
func (n *NodeAbstractResource) ProvisionedBy() []string {
	// If we have no configuration, then we have no provisioners
	if n.Config == nil {
		return nil
	}

	// Build the list of provisioners we need based on the configuration.
	// It is okay to have duplicates here.
	result := make([]string, len(n.Config.Provisioners))
	for i, p := range n.Config.Provisioners {
		result[i] = p.Type
	}

	return result
}

// GraphNodeResource, GraphNodeAttachResourceState
func (n *NodeAbstractResource) ResourceAddr() *ResourceAddress {
	return n.Addr
}

// GraphNodeAddressable, TODO: remove, used by target, should unify
func (n *NodeAbstractResource) ResourceAddress() *ResourceAddress {
	return n.ResourceAddr()
}

// GraphNodeTargetable
func (n *NodeAbstractResource) SetTargets(targets []ResourceAddress) {
	n.Targets = targets
}

// GraphNodeAttachResourceState
func (n *NodeAbstractResource) AttachResourceState(s *ResourceState) {
	n.ResourceState = s
}

// GraphNodeAttachResourceConfig
func (n *NodeAbstractResource) AttachResourceConfig(c *config.Resource) {
	n.Config = c
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
