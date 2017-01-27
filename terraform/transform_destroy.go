package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeDestroy is the interface that must implemented by
// nodes that destroy.
type GraphNodeDestroy interface {
	dag.Vertex

	// CreateBeforeDestroy is called to check whether this node
	// should be created before it is destroyed. The CreateBeforeDestroy
	// transformer uses this information to setup the graph.
	CreateBeforeDestroy() bool

	// CreateNode returns the node used for the create side of this
	// destroy. This must already exist within the graph.
	CreateNode() dag.Vertex
}
