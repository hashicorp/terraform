package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// graphNodeConfig is an interface that all graph nodes for the
// configuration graph need to implement in order to build the variable
// dependencies properly.
type graphNodeConfig interface {
	dag.NamedVertex

	// All graph nodes should be dependent on other things, and able to
	// be depended on.
	GraphNodeDependable
	GraphNodeDependent

	// ConfigType returns the type of thing in the configuration that
	// this node represents, such as a resource, module, etc.
	ConfigType() GraphNodeConfigType
}

// GraphNodeAddressable is an interface that all graph nodes for the
// configuration graph need to implement in order to be be addressed / targeted
// properly.
type GraphNodeAddressable interface {
	ResourceAddress() *ResourceAddress
}

// GraphNodeTargetable is an interface for graph nodes to implement when they
// need to be told about incoming targets. This is useful for nodes that need
// to respect targets as they dynamically expand. Note that the list of targets
// provided will contain every target provided, and each implementing graph
// node must filter this list to targets considered relevant.
type GraphNodeTargetable interface {
	SetTargets([]ResourceAddress)
}
