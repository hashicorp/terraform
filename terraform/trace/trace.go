package trace

import (
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/terraform"
)

// A Tracer is an entry-point for tracing.
type Tracer interface {
	// TraceGraphBuild begins tracing the construction of a graph.
	TraceGraphBuild(ty terraform.GraphType) GraphBuildTracer
	TraceGraphWalk(graph terraform.Graph) GraphWalkTracer
}

// A GraphBuildTracer traces the steps of graph construction using
// terraform.BasicGraphBuilder. If a graph is built using some other
// graph builder then no transform traces will be recorded.
type GraphBuildTracer interface {
	// TraceGraphTransform begins tracing of a particular graph transform step.
	TraceGraphTransform(transformer terraform.GraphTransformer) GraphTransformTracer

	// TraceFinalGraph is called once all transforms are complete, to capture
	// the final state of the graph.
	// Since graphs can be modified in-place, the tracer must extract a deep copy
	// of all of the information it is interested in before returning.
	TraceFinalGraph(graph terraform.Graph)
}

// A GraphTransformTracer traces the input and result of a particular graph
// transformer executed during graph construction.
type GraphTransformTracer interface {
	// TraceGraphBefore and TraceGraphAfter are called before and after (respectively)
	// a graph transform is called. Since graph transforms modify the graph in-place,
	// the tracer must extract a deep copy of all of the information it is interested
	// in before returning from these methods.
	TraceGraphBefore(graph terraform.Graph)
	TraceGraphAfter(graph terraform.Graph)

	// TraceTransformError is called instead of TraceGraphAfter if a transform
	// returns an error.
	TraceTransformError(err error)
}

// A GraphWalkTracer traces the nodes visited during a graph walk. Graph walks
// visit several nodes concurrently, so the methods of a GraphWalkTracer and
// any objects they return must be concurrency-safe.
type GraphWalkTracer interface {
	// TraceNode begins tracing of a particular node in the graph.
	TraceNode(v *dag.Vertex) GraphNodeTracer
}

// A GraphNodeTracer traces the evaluation of a particular graph node.
type GraphNodeTracer interface {
	// TraceNodeEval begins the tracing of the evaluation of a particular node
	// within a graph node's evaluation tree.
	// EvalNodes often self-mutate during their evaluation, so a TraceNodeEval
	// implementation must deep-copy any interesting information from the
	// given node before returning.
	TraceNodeEval(n terraform.EvalNode) GraphNodeEvalTracer
}

// A GraphNodeEvalTracer traces the result of of the evaluation of an eval node
// on a particular graph node.
type GraphNodeEvalTracer interface {
	// TraceNodeEvalResult is called after an EvalNode has been evaluated,
	// tracking its result. Since most EvalNodes contain data that is
	// mutated by later runs, a TraceNodeEvalResult implementation must
	// deep-copy any interesting information from the given node
	// before returning.
	TraceNodeEvalResult(n terraform.EvalNode, v interface{}, err error)
}
