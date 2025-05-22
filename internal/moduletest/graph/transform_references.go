package graph

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
)

type GraphNodeReferenceable interface {
	Referenceable() addrs.Referenceable
}

type GraphNodeReferences interface {
	References() []*addrs.Reference
}

var _ terraform.GraphTransformer = (*ReferenceTransformer)(nil)

type ReferenceTransformer struct{}

func (r *ReferenceTransformer) Transform(graph *terraform.Graph) error {
	nodes := addrs.MakeMap[addrs.Referenceable, dag.Vertex]()
	for _, v := range graph.Vertices() {
		if referenceable, ok := v.(GraphNodeReferenceable); ok {
			nodes.Put(referenceable.Referenceable(), v)
		}
	}

	for _, v := range graph.Vertices() {
		if references, ok := v.(GraphNodeReferences); ok {
			for _, reference := range references.References() {
				if target, ok := nodes.GetOk(reference.Subject); ok {
					graph.Connect(dag.BasicEdge(v, target))
				}
			}
		}
	}

	return nil
}
