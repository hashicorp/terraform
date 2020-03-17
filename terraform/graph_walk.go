package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

// GraphWalker is an interface that can be implemented that when used
// with Graph.Walk will invoke the given callbacks under certain events.
type GraphWalker interface {
	EvalContext() EvalContext
	EnterPath(addrs.ModuleInstance) EvalContext
	ExitPath(addrs.ModuleInstance)
	EnterVertex(dag.Vertex)
	ExitVertex(dag.Vertex, tfdiags.Diagnostics)
	EnterEvalTree(dag.Vertex, EvalNode) EvalNode
	ExitEvalTree(dag.Vertex, interface{}, error) tfdiags.Diagnostics
}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

func (NullGraphWalker) EvalContext() EvalContext                        { return new(MockEvalContext) }
func (NullGraphWalker) EnterPath(addrs.ModuleInstance) EvalContext      { return new(MockEvalContext) }
func (NullGraphWalker) ExitPath(addrs.ModuleInstance)                   {}
func (NullGraphWalker) EnterVertex(dag.Vertex)                          {}
func (NullGraphWalker) ExitVertex(dag.Vertex, tfdiags.Diagnostics)      {}
func (NullGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode { return n }
func (NullGraphWalker) ExitEvalTree(dag.Vertex, interface{}, error) tfdiags.Diagnostics {
	return nil
}
