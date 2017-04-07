package trace

import (
	"github.com/hashicorp/terraform/terraform"
)

// NoOpTracer is a Tracer implementation that does absolutely nothing.
type NoOpTracer struct{}

type noopGraphBuildTracer struct{}
type noopGraphTransformTracer struct{}
type noopGraphWalkTracer struct{}
type noopGraphNodeTracer struct{}
type noopGraphNodeEvalTracer struct{}

func (t NoOpTracer) TraceGraphBuild(terraform.GraphType) GraphBuildTracer {
	return noopGraphBuildTracer{}
}

func (t NoOpTracer) TraceFinalGraph(terraform.Graph) GraphWalkTracer {
	return noopGraphWalkTracer{}
}

func (t noopGraphBuildTracer) TraceGraphTransform(terraform.GraphTransformer) GraphTransformTracer {
	return noopGraphTransformTracer{}
}

func (t noopGraphBuildTracer) TraceFinalGraph(terraform.Graph) {
}
