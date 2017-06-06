package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/dag"
)

// DestroyEdge is an edge that represents a standard "destroy" relationship:
// Target depends on Source because Source is destroying.
type DestroyEdge struct {
	S, T dag.Vertex
}

func (e *DestroyEdge) Hashcode() interface{} { return fmt.Sprintf("%p-%p", e.S, e.T) }
func (e *DestroyEdge) Source() dag.Vertex    { return e.S }
func (e *DestroyEdge) Target() dag.Vertex    { return e.T }
