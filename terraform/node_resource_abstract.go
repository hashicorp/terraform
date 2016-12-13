package terraform

import (
	"fmt"

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
		addrCopy.Index = -1
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
			result = append(result, ReferencesFromConfig(p.ConnInfo)...)
			result = append(result, ReferencesFromConfig(p.RawConfig)...)
		}

		return result
	}

	// If we have state, that is our next source
	if s := n.ResourceState; s != nil {
		return s.Dependencies
	}

	return nil
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
