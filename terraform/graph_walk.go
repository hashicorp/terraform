package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphWalker is an interface that can be implemented that when used
// with Graph.Walk will invoke the given callbacks under certain events.
type GraphWalker interface {
	EnterPath([]string) EvalContext
	ExitPath([]string)
	EnterVertex(dag.Vertex)
	ExitVertex(dag.Vertex, error)
	EnterEvalTree(dag.Vertex, EvalNode) EvalNode
	ExitEvalTree(dag.Vertex, interface{}, error) error
}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

func (NullGraphWalker) EnterPath([]string) EvalContext                  { return nil }
func (NullGraphWalker) ExitPath([]string)                               {}
func (NullGraphWalker) EnterVertex(dag.Vertex)                          {}
func (NullGraphWalker) ExitVertex(dag.Vertex, error)                    {}
func (NullGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode { return n }
func (NullGraphWalker) ExitEvalTree(dag.Vertex, interface{}, error) error {
	return nil
}
