package terraform

import (
	"github.com/hashicorp/terraform/internal/dag"
)

func DestroyPlanGraphBuilder(p *PlanGraphBuilder) GraphBuilder {
	p.ConcreteResourceInstance = func(a *NodeAbstractResourceInstance) dag.Vertex {
		return &NodePlanDestroyableResourceInstance{
			NodeAbstractResourceInstance: a,
			skipRefresh:                  p.skipRefresh,
		}
	}
	p.destroy = true

	return p
}
