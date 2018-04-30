package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
)

// GraphWalker is an interface that can be implemented that when used
// with Graph.Walk will invoke the given callbacks under certain events.
type GraphWalker interface {
	EnterPath(addrs.ModuleInstance) EvalContext
	ExitPath(addrs.ModuleInstance)
	EnterVertex(dag.Vertex)
	ExitVertex(dag.Vertex, tfdiags.Diagnostics)
	EnterEvalTree(dag.Vertex, EvalNode) EvalNode
	ExitEvalTree(dag.Vertex, interface{}, error) tfdiags.Diagnostics
}

// GrpahWalkerPanicwrapper can be optionally implemented to catch panics
// that occur while walking the graph. This is not generally recommended
// since panics should crash Terraform and result in a bug report. However,
// this is particularly useful for situations like the shadow graph where
// you don't ever want to cause a panic.
type GraphWalkerPanicwrapper interface {
	GraphWalker

	// Panic is called when a panic occurs. This will halt the panic from
	// propogating so if the walker wants it to crash still it should panic
	// again. This is called from within a defer so runtime/debug.Stack can
	// be used to get the stack trace of the panic.
	Panic(dag.Vertex, interface{})
}

// GraphWalkerPanicwrap wraps an existing Graphwalker to wrap and swallow
// the panics. This doesn't lose the panics since the panics are still
// returned as errors as part of a graph walk.
func GraphWalkerPanicwrap(w GraphWalker) GraphWalkerPanicwrapper {
	return &graphWalkerPanicwrapper{
		GraphWalker: w,
	}
}

type graphWalkerPanicwrapper struct {
	GraphWalker
}

func (graphWalkerPanicwrapper) Panic(dag.Vertex, interface{}) {}

// NullGraphWalker is a GraphWalker implementation that does nothing.
// This can be embedded within other GraphWalker implementations for easily
// implementing all the required functions.
type NullGraphWalker struct{}

func (NullGraphWalker) EnterPath(addrs.ModuleInstance) EvalContext      { return new(MockEvalContext) }
func (NullGraphWalker) ExitPath(addrs.ModuleInstance)                   {}
func (NullGraphWalker) EnterVertex(dag.Vertex)                          {}
func (NullGraphWalker) ExitVertex(dag.Vertex, tfdiags.Diagnostics)      {}
func (NullGraphWalker) EnterEvalTree(v dag.Vertex, n EvalNode) EvalNode { return n }
func (NullGraphWalker) ExitEvalTree(dag.Vertex, interface{}, error) tfdiags.Diagnostics {
	return nil
}
